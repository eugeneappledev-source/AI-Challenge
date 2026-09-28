package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var ErrInvalidPipelinePlan = errors.New("agent created an invalid MCP pipeline")

type LLMNewsSummarizer struct{ client ReasoningClient }

func NewLLMNewsSummarizer(client ReasoningClient) *LLMNewsSummarizer {
	return &LLMNewsSummarizer{client: client}
}

func (s *LLMNewsSummarizer) Summarize(ctx context.Context, query string, stories []domain.BusinessStory) (domain.BusinessBrief, error) {
	raw, _ := json.Marshal(stories)
	response, err := s.client.Generate(ctx, domain.ModelRequest{
		SystemPrompt: `Ты редактор Startup Radar. Используй только переданные источники. Верни JSON: {"summary":"2-3 предложения","keySignals":["сигнал"],"risks":["риск"],"nextQuestion":"что исследовать дальше"}. Не обещай доход и не выдумывай факты. Пиши по-русски.`,
		UserPrompt:   "Исследовательский запрос:\n" + query + "\n\nИсточники:\n" + string(raw), JSON: true, MaxTokens: 700,
	})
	if err != nil {
		return domain.BusinessBrief{}, fmt.Errorf("summarize business news: %w", err)
	}
	var brief domain.BusinessBrief
	if err := json.Unmarshal([]byte(response.Content), &brief); err != nil {
		return domain.BusinessBrief{}, fmt.Errorf("decode business brief: %w", err)
	}
	brief.Model, brief.Usage = response.Model, response.Usage
	return brief, nil
}

type pipelinePlanPayload struct {
	Tools           []string       `json:"tools"`
	Rationale       string         `json:"rationale"`
	SearchArguments map[string]any `json:"searchArguments"`
}

type BusinessPipelineService struct {
	client   ReasoningClient
	endpoint string
	now      func() time.Time
}

func NewBusinessPipelineService(client ReasoningClient, endpoint string) *BusinessPipelineService {
	return &BusinessPipelineService{client: client, endpoint: endpoint, now: time.Now}
}

func (s *BusinessPipelineService) Run(ctx context.Context, request string) (domain.BusinessPipelineResult, error) {
	request = strings.TrimSpace(request)
	if request == "" {
		return domain.BusinessPipelineResult{}, ErrEmptyResearchRequest
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "startup-pipeline-agent", Version: "v1.0.0"}, &mcp.ClientOptions{Capabilities: &mcp.ClientCapabilities{}})
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: s.endpoint}, nil)
	if err != nil {
		return domain.BusinessPipelineResult{}, fmt.Errorf("connect pipeline MCP: %w", err)
	}
	defer session.Close()
	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		return domain.BusinessPipelineResult{}, fmt.Errorf("list pipeline tools: %w", err)
	}
	descriptors := toolDescriptors(listed.Tools)
	toolJSON, _ := json.Marshal(descriptors)
	planResponse, err := s.client.Generate(ctx, domain.ModelRequest{
		SystemPrompt: `Ты планировщик MCP. Для исследования новостей составь цепочку из зарегистрированных tools. Верни только JSON: {"tools":["tool1","tool2"],"searchArguments":{"query":"...","limit":5},"rationale":"почему такой порядок"}. Учитывай зависимости из descriptions, не выдумывай tools.`,
		UserPrompt:   "Задача:\n" + request + "\n\nTools:\n" + string(toolJSON), JSON: true, MaxTokens: 450,
	})
	if err != nil {
		return domain.BusinessPipelineResult{}, fmt.Errorf("plan tool pipeline: %w", err)
	}
	var plan pipelinePlanPayload
	if err := json.Unmarshal([]byte(planResponse.Content), &plan); err != nil {
		return domain.BusinessPipelineResult{}, fmt.Errorf("decode pipeline plan: %w", err)
	}
	expected := []string{"search_business_news", "summarize_business_news", "save_business_report"}
	if !sameStrings(plan.Tools, expected) {
		return domain.BusinessPipelineResult{}, ErrInvalidPipelinePlan
	}
	if plan.SearchArguments == nil {
		plan.SearchArguments = map[string]any{"query": request, "limit": 5}
	}

	stages := make([]domain.PipelineStage, 0, 3)
	started := time.Now()
	searchResult, err := session.CallTool(ctx, &mcp.CallToolParams{Name: expected[0], Arguments: plan.SearchArguments})
	if err != nil || searchResult.IsError {
		return domain.BusinessPipelineResult{}, fmt.Errorf("search stage failed: %w", err)
	}
	var searched struct {
		Stories []domain.BusinessStory `json:"stories"`
	}
	if err := decodeStructured(searchResult.StructuredContent, &searched); err != nil {
		return domain.BusinessPipelineResult{}, err
	}
	stages = append(stages, domain.PipelineStage{Order: 1, Server: "startup-research-mcp", Tool: expected[0], InputSummary: request, OutputSummary: fmt.Sprintf("%d structured stories", len(searched.Stories)), DurationMS: time.Since(started).Milliseconds()})

	started = time.Now()
	summaryArgs := map[string]any{"query": request, "stories": searched.Stories}
	summaryResult, err := session.CallTool(ctx, &mcp.CallToolParams{Name: expected[1], Arguments: summaryArgs})
	if err != nil || summaryResult.IsError {
		return domain.BusinessPipelineResult{}, fmt.Errorf("summary stage failed: %w", err)
	}
	var summarized struct {
		Brief domain.BusinessBrief `json:"brief"`
	}
	if err := decodeStructured(summaryResult.StructuredContent, &summarized); err != nil {
		return domain.BusinessPipelineResult{}, err
	}
	stages = append(stages, domain.PipelineStage{Order: 2, Server: "startup-research-mcp", Tool: expected[1], InputSummary: fmt.Sprintf("handoff: %d stories", len(searched.Stories)), OutputSummary: fmt.Sprintf("brief: %d signals, %d risks", len(summarized.Brief.KeySignals), len(summarized.Brief.Risks)), DurationMS: time.Since(started).Milliseconds()})

	started = time.Now()
	saveArgs := map[string]any{"query": request, "brief": summarized.Brief, "sourceCount": len(searched.Stories)}
	saveResult, err := session.CallTool(ctx, &mcp.CallToolParams{Name: expected[2], Arguments: saveArgs})
	if err != nil || saveResult.IsError {
		return domain.BusinessPipelineResult{}, fmt.Errorf("save stage failed: %w", err)
	}
	var saved struct {
		Report domain.RadarReport `json:"report"`
	}
	if err := decodeStructured(saveResult.StructuredContent, &saved); err != nil {
		return domain.BusinessPipelineResult{}, err
	}
	stages = append(stages, domain.PipelineStage{Order: 3, Server: "startup-research-mcp", Tool: expected[2], InputSummary: "handoff: validated brief", OutputSummary: "persisted as " + saved.Report.ID, DurationMS: time.Since(started).Milliseconds()})

	return domain.BusinessPipelineResult{Request: request, Plan: domain.PipelinePlan{Tools: plan.Tools, Rationale: plan.Rationale}, Stages: stages, Stories: searched.Stories, Brief: summarized.Brief, SavedReport: saved.Report, AvailableTools: descriptors, Usage: addUsage(planResponse.Usage, summarized.Brief.Usage), CompletedAt: s.now().UTC()}, nil
}

func decodeStructured(value any, target any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("decode structured MCP output: %w", err)
	}
	return nil
}
func sameStrings(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i := range expected {
		if actual[i] != expected[i] {
			return false
		}
	}
	return true
}
