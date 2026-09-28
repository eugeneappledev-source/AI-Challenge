package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	ErrEmptyResearchRequest   = errors.New("research request is required")
	ErrResearchRequestTooLong = errors.New("research request is too long")
	ErrInvalidToolSelection   = errors.New("agent selected an unavailable tool")
)

type BusinessResearchService struct {
	client          ReasoningClient
	endpoint        string
	maxRequestRunes int
	now             func() time.Time
}

func NewBusinessResearchService(client ReasoningClient, endpoint string, maxRequestRunes int) *BusinessResearchService {
	return &BusinessResearchService{client: client, endpoint: endpoint, maxRequestRunes: maxRequestRunes, now: time.Now}
}

type toolSelectionPayload struct {
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
	Rationale string         `json:"rationale"`
}

func (s *BusinessResearchService) Research(ctx context.Context, request string) (domain.BusinessResearchResult, error) {
	request = strings.TrimSpace(request)
	if request == "" {
		return domain.BusinessResearchResult{}, ErrEmptyResearchRequest
	}
	if utf8.RuneCountInString(request) > s.maxRequestRunes {
		return domain.BusinessResearchResult{}, ErrResearchRequestTooLong
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "startup-radar-agent", Version: "v1.0.0"}, &mcp.ClientOptions{Capabilities: &mcp.ClientCapabilities{}})
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: s.endpoint}, nil)
	if err != nil {
		return domain.BusinessResearchResult{}, fmt.Errorf("connect research MCP: %w", err)
	}
	defer session.Close()
	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		return domain.BusinessResearchResult{}, fmt.Errorf("list research MCP tools: %w", err)
	}
	descriptors := toolDescriptors(listed.Tools)
	toolsJSON, _ := json.Marshal(descriptors)

	plannerResponse, err := s.client.Generate(ctx, domain.ModelRequest{
		SystemPrompt: `Ты маршрутизатор MCP-инструментов Startup Radar. Выбери ровно один подходящий инструмент из фактически зарегистрированного списка. Верни только JSON: {"tool":"name","arguments":{},"rationale":"короткое объяснение"}. Не выдумывай инструменты. Для новостей сохрани смысл запроса в query и установи limit от 3 до 6.`,
		UserPrompt:   "Запрос пользователя:\n" + request + "\n\nЗарегистрированные MCP tools:\n" + string(toolsJSON),
		JSON:         true,
		MaxTokens:    350,
	})
	if err != nil {
		return domain.BusinessResearchResult{}, fmt.Errorf("plan MCP call: %w", err)
	}
	var selection toolSelectionPayload
	if err := json.Unmarshal([]byte(plannerResponse.Content), &selection); err != nil {
		return domain.BusinessResearchResult{}, fmt.Errorf("decode MCP selection: %w", err)
	}
	if selection.Tool != "search_business_news" || !containsTool(listed.Tools, selection.Tool) {
		return domain.BusinessResearchResult{}, ErrInvalidToolSelection
	}
	if selection.Arguments == nil {
		selection.Arguments = map[string]any{"query": request, "limit": 5}
	}

	toolResult, err := session.CallTool(ctx, &mcp.CallToolParams{Name: selection.Tool, Arguments: selection.Arguments})
	if err != nil {
		return domain.BusinessResearchResult{}, fmt.Errorf("call MCP tool: %w", err)
	}
	if toolResult.IsError {
		return domain.BusinessResearchResult{}, errors.New("MCP news tool returned an error")
	}
	var output struct {
		Stories []domain.BusinessStory `json:"stories"`
	}
	structured, err := json.Marshal(toolResult.StructuredContent)
	if err != nil || json.Unmarshal(structured, &output) != nil {
		return domain.BusinessResearchResult{}, errors.New("MCP tool returned an invalid structured result")
	}

	storiesJSON, _ := json.Marshal(output.Stories)
	answerResponse, err := s.client.Generate(ctx, domain.ModelRequest{
		SystemPrompt: `Ты осторожный аналитик стартапов. Используй только переданные новости. Верни JSON строго вида {"summary":"2-3 предложения","opportunities":[{"title":"идея","whyNow":"почему актуально","firstStep":"маленький проверяемый шаг","risk":"главный риск"}],"caveat":"оговорка"}. Дай 2-3 реалистичные гипотезы, не обещай доход и не выдавай финансовую рекомендацию. Отвечай по-русски.`,
		UserPrompt:   "Исходный запрос:\n" + request + "\n\nНовости MCP:\n" + string(storiesJSON),
		JSON:         true,
		MaxTokens:    900,
	})
	if err != nil {
		return domain.BusinessResearchResult{}, fmt.Errorf("analyze business news: %w", err)
	}
	var advice domain.BusinessAdvice
	if err := json.Unmarshal([]byte(answerResponse.Content), &advice); err != nil {
		return domain.BusinessResearchResult{}, fmt.Errorf("decode business advice: %w", err)
	}

	return domain.BusinessResearchResult{
		Request:   request,
		Selection: domain.MCPToolSelection{Server: "startup-research-mcp", Tool: selection.Tool, Arguments: selection.Arguments, Rationale: selection.Rationale},
		Stories:   output.Stories, Advice: advice,
		Model: answerResponse.Model, FinishReason: answerResponse.FinishReason,
		Usage: addUsage(plannerResponse.Usage, answerResponse.Usage), AvailableTools: descriptors,
		Trace: []string{
			"Agent connected to startup-research-mcp",
			"Agent loaded registered tool schemas with tools/list",
			"LLM selected " + selection.Tool + " and prepared validated arguments",
			"Agent invoked the MCP tool and received structured news with source links",
			"LLM converted tool data into cautious opportunity hypotheses",
		},
		CompletedAt: s.now().UTC(),
	}, nil
}

func toolDescriptors(tools []*mcp.Tool) []domain.MCPToolDescriptor {
	result := make([]domain.MCPToolDescriptor, 0, len(tools))
	for _, tool := range tools {
		schema := map[string]any{}
		raw, _ := json.Marshal(tool.InputSchema)
		_ = json.Unmarshal(raw, &schema)
		result = append(result, domain.MCPToolDescriptor{Name: tool.Name, Title: tool.Title, Description: tool.Description, InputSchema: schema})
	}
	return result
}

func containsTool(tools []*mcp.Tool, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}
