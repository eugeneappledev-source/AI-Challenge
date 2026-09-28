package mcpserver

import (
	"context"
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

func NewResearchServer(providers ...NewsProvider) *mcp.Server {
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
	if len(providers) > 0 && providers[0] != nil {
		provider := providers[0]
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
	return server
}

func boolPointer(value bool) *bool { return &value }

func StreamableHandler(server *mcp.Server) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{
		JSONResponse: true,
		Stateless:    true,
	})
}
