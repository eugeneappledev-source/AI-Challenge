package mcpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CapabilitiesInput struct {
	Topic string `json:"topic,omitempty" jsonschema:"optional business topic to describe"`
}

type CapabilitiesOutput struct {
	Server  string   `json:"server" jsonschema:"MCP server name"`
	Purpose string   `json:"purpose" jsonschema:"what this server is responsible for"`
	Sources []string `json:"sources" jsonschema:"trusted external sources used by the server"`
}

type NewsProvider interface {
	Search(ctx context.Context, query string, limit int) ([]domain.BusinessStory, error)
}

type SearchNewsInput struct {
	Query string `json:"query" jsonschema:"business, startup, market, or technology topic to research"`
	Limit int    `json:"limit,omitempty" jsonschema:"number of stories from 1 to 8"`
}

type SearchNewsOutput struct {
	Query     string                 `json:"query" jsonschema:"normalized research query"`
	Source    string                 `json:"source" jsonschema:"external API used for research"`
	Stories   []domain.BusinessStory `json:"stories" jsonschema:"ranked current stories with source links"`
	FetchedAt string                 `json:"fetchedAt" jsonschema:"UTC timestamp of the API call"`
}

type NewsSummarizer interface {
	Summarize(ctx context.Context, query string, stories []domain.BusinessStory) (domain.BusinessBrief, error)
}

type RadarReportStore interface {
	SaveRadarReport(ctx context.Context, report domain.RadarReport) error
}

type SummarizeNewsInput struct {
	Query   string                 `json:"query" jsonschema:"original research question"`
	Stories []domain.BusinessStory `json:"stories" jsonschema:"structured stories returned by search_business_news"`
}

type SummarizeNewsOutput struct {
	Brief domain.BusinessBrief `json:"brief" jsonschema:"evidence-based executive brief"`
}

type SaveReportInput struct {
	Query       string               `json:"query" jsonschema:"original research question"`
	Brief       domain.BusinessBrief `json:"brief" jsonschema:"brief returned by summarize_business_news"`
	SourceCount int                  `json:"sourceCount" jsonschema:"number of source stories used"`
}

type SaveReportOutput struct {
	Report domain.RadarReport `json:"report" jsonschema:"persisted report metadata"`
}

func NewResearchServer(providers ...NewsProvider) *mcp.Server {
	var provider NewsProvider
	if len(providers) > 0 {
		provider = providers[0]
	}
	return newResearchServer(provider, nil, nil)
}

func NewResearchPipelineServer(provider NewsProvider, summarizer NewsSummarizer, store RadarReportStore) *mcp.Server {
	return newResearchServer(provider, summarizer, store)
}

func newResearchServer(provider NewsProvider, summarizer NewsSummarizer, store RadarReportStore) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "startup-research-mcp", Version: "v1.0.0"}, &mcp.ServerOptions{
		Instructions: "Trusted read-only tools for the Startup & Business Radar.",
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "describe_business_radar",
		Title:       "Business Radar capabilities",
		Description: "Describe this trusted MCP server and the business research capabilities it exposes.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}, func(_ context.Context, _ *mcp.CallToolRequest, input CapabilitiesInput) (*mcp.CallToolResult, CapabilitiesOutput, error) {
		purpose := "Inspect startup and business research capabilities"
		if input.Topic != "" {
			purpose += " for " + input.Topic
		}
		return nil, CapabilitiesOutput{
			Server: "startup-research-mcp", Purpose: purpose,
			Sources: []string{"Hacker News official API"},
		}, nil
	})
	if provider != nil {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "search_business_news",
			Title:       "Search current business news",
			Description: "Search current startup, business, market and technology stories from the official Hacker News API. Use this whenever the user asks for fresh news, trends, opportunities or examples. Returns source URLs; never invent links.",
			Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true, OpenWorldHint: boolPointer(true)},
		}, func(ctx context.Context, _ *mcp.CallToolRequest, input SearchNewsInput) (*mcp.CallToolResult, SearchNewsOutput, error) {
			if input.Limit == 0 {
				input.Limit = 5
			}
			stories, err := provider.Search(ctx, input.Query, input.Limit)
			if err != nil {
				return nil, SearchNewsOutput{}, err
			}
			return nil, SearchNewsOutput{Query: input.Query, Source: "Hacker News official API", Stories: stories, FetchedAt: time.Now().UTC().Format(time.RFC3339)}, nil
		})
	}
	if summarizer != nil {
		mcp.AddTool(server, &mcp.Tool{
			Name: "summarize_business_news", Title: "Summarize researched stories",
			Description: "Convert structured stories from search_business_news into a concise evidence-based brief. Call only after search_business_news and pass its stories unchanged.",
			Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
		}, func(ctx context.Context, _ *mcp.CallToolRequest, input SummarizeNewsInput) (*mcp.CallToolResult, SummarizeNewsOutput, error) {
			brief, err := summarizer.Summarize(ctx, input.Query, input.Stories)
			return nil, SummarizeNewsOutput{Brief: brief}, err
		})
	}
	if store != nil {
		mcp.AddTool(server, &mcp.Tool{
			Name: "save_business_report", Title: "Save completed business report",
			Description: "Persist the final brief after research and summarization. Call only after summarize_business_news.",
			Annotations: &mcp.ToolAnnotations{DestructiveHint: boolPointer(false), IdempotentHint: false},
		}, func(ctx context.Context, _ *mcp.CallToolRequest, input SaveReportInput) (*mcp.CallToolResult, SaveReportOutput, error) {
			report := domain.RadarReport{ID: fmt.Sprintf("report-%d", time.Now().UTC().UnixNano()), Query: input.Query, Brief: input.Brief, SourceCount: input.SourceCount, CreatedAt: time.Now().UTC()}
			if err := store.SaveRadarReport(ctx, report); err != nil {
				return nil, SaveReportOutput{}, err
			}
			return nil, SaveReportOutput{Report: report}, nil
		})
	}
	return server
}

func boolPointer(value bool) *bool { return &value }

func StreamableHandler(server *mcp.Server) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{
		JSONResponse: true,
		Stateless:    true,
	})
}
