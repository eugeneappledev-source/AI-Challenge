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

type LLMOpportunityAdvisor struct{ client ReasoningClient }

func NewLLMOpportunityAdvisor(client ReasoningClient) *LLMOpportunityAdvisor {
	return &LLMOpportunityAdvisor{client: client}
}

func (a *LLMOpportunityAdvisor) Score(ctx context.Context, query string, brief domain.BusinessBrief, profile domain.FounderProfile) (domain.OpportunityScores, error) {
	input, _ := json.Marshal(map[string]any{"query": query, "brief": brief, "profile": profile})
	response, err := a.client.Generate(ctx, domain.ModelRequest{
		SystemPrompt: `Ты прагматичный startup-аналитик. Верни JSON {"items":[{"title":"...","evidence":"сигнал из brief","fitScore":0,"effort":"low|medium|high","rationale":"...","risk":"..."}]}. Дай ровно 3 варианта, оцени fitScore от 0 до 100 с учётом бюджета, времени и навыков. Не обещай доход.`,
		UserPrompt:   string(input), JSON: true, MaxTokens: 800,
	})
	if err != nil {
		return domain.OpportunityScores{}, fmt.Errorf("score opportunities: %w", err)
	}
	var scores domain.OpportunityScores
	if err := json.Unmarshal([]byte(response.Content), &scores); err != nil {
		return domain.OpportunityScores{}, fmt.Errorf("decode opportunity scores: %w", err)
	}
	scores.Model, scores.Usage = response.Model, response.Usage
	return scores, nil
}

func (a *LLMOpportunityAdvisor) BuildPlan(ctx context.Context, query string, selected domain.ScoredOpportunity, profile domain.FounderProfile) (domain.FounderActionPlan, error) {
	input, _ := json.Marshal(map[string]any{"query": query, "selected": selected, "profile": profile})
	response, err := a.client.Generate(ctx, domain.ModelRequest{
		SystemPrompt: `Ты product advisor. Верни JSON {"selectedTitle":"...","goal":"проверяемая цель","steps":["шаг"],"stopConditions":["условие остановки"],"budgetNote":"..."}. Дай 4 коротких шага на одну неделю, соблюдай бюджет и время пользователя. Это проверка гипотезы, не обещание заработка.`,
		UserPrompt:   string(input), JSON: true, MaxTokens: 700,
	})
	if err != nil {
		return domain.FounderActionPlan{}, fmt.Errorf("build action plan: %w", err)
	}
	var plan domain.FounderActionPlan
	if err := json.Unmarshal([]byte(response.Content), &plan); err != nil {
		return domain.FounderActionPlan{}, fmt.Errorf("decode action plan: %w", err)
	}
	plan.Model, plan.Usage = response.Model, response.Usage
	return plan, nil
}

type multiServerPlan struct {
	Route []struct {
		Server string `json:"server"`
		Tool   string `json:"tool"`
	} `json:"route"`
	Rationale string `json:"rationale"`
}

type BusinessNetworkService struct {
	client                            ReasoningClient
	researchEndpoint, advisorEndpoint string
	now                               func() time.Time
}

func NewBusinessNetworkService(client ReasoningClient, researchEndpoint, advisorEndpoint string) *BusinessNetworkService {
	return &BusinessNetworkService{client: client, researchEndpoint: researchEndpoint, advisorEndpoint: advisorEndpoint, now: time.Now}
}

func (s *BusinessNetworkService) Run(ctx context.Context, request string, profile domain.FounderProfile) (domain.MultiServerResult, error) {
	request = strings.TrimSpace(request)
	if request == "" {
		return domain.MultiServerResult{}, ErrEmptyResearchRequest
	}
	researchSession, researchTools, err := connectAndList(ctx, "network-orchestrator", s.researchEndpoint)
	if err != nil {
		return domain.MultiServerResult{}, err
	}
	defer researchSession.Close()
	advisorSession, advisorTools, err := connectAndList(ctx, "network-orchestrator", s.advisorEndpoint)
	if err != nil {
		return domain.MultiServerResult{}, err
	}
	defer advisorSession.Close()
	servers := []domain.MCPServerSnapshot{{Name: "startup-research-mcp", Tools: toolDescriptors(researchTools)}, {Name: "business-advisor-mcp", Tools: toolDescriptors(advisorTools)}}
	toolsJSON, _ := json.Marshal(servers)
	planResponse, err := s.client.Generate(ctx, domain.ModelRequest{
		SystemPrompt: `Ты оркестратор нескольких MCP-серверов. Верни JSON {"route":[{"server":"...","tool":"..."}],"rationale":"..."}. Построй зависимую цепочку: сначала собрать факты и brief, затем оценить варианты и составить plan, затем сохранить отчёт. Используй только перечисленные server/tool.`,
		UserPrompt:   "Задача:\n" + request + "\n\nMCP servers:\n" + string(toolsJSON), JSON: true, MaxTokens: 600,
	})
	if err != nil {
		return domain.MultiServerResult{}, fmt.Errorf("plan MCP network: %w", err)
	}
	var planned multiServerPlan
	if err := json.Unmarshal([]byte(planResponse.Content), &planned); err != nil {
		return domain.MultiServerResult{}, fmt.Errorf("decode MCP network plan: %w", err)
	}
	expectedServers := []string{"startup-research-mcp", "startup-research-mcp", "business-advisor-mcp", "business-advisor-mcp", "startup-research-mcp"}
	expectedTools := []string{"search_business_news", "summarize_business_news", "score_business_opportunities", "build_action_plan", "save_business_report"}
	if len(planned.Route) != len(expectedTools) {
		return domain.MultiServerResult{}, ErrInvalidPipelinePlan
	}
	for index := range expectedTools {
		if planned.Route[index].Server != expectedServers[index] || planned.Route[index].Tool != expectedTools[index] {
			return domain.MultiServerResult{}, ErrInvalidPipelinePlan
		}
	}

	route := make([]domain.PipelineStage, 0, 5)
	call := func(session *mcp.ClientSession, order int, server, tool, inputSummary string, args map[string]any, target any, outputSummary func() string) error {
		started := time.Now()
		result, callErr := session.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if callErr != nil {
			return callErr
		}
		if result.IsError {
			return errors.New("MCP tool returned an error")
		}
		if err := decodeStructured(result.StructuredContent, target); err != nil {
			return err
		}
		route = append(route, domain.PipelineStage{Order: order, Server: server, Tool: tool, InputSummary: inputSummary, OutputSummary: outputSummary(), DurationMS: time.Since(started).Milliseconds()})
		return nil
	}
	var searched struct {
		Stories []domain.BusinessStory `json:"stories"`
	}
	if err := call(researchSession, 1, expectedServers[0], expectedTools[0], request, map[string]any{"query": request, "limit": 5}, &searched, func() string { return fmt.Sprintf("%d stories", len(searched.Stories)) }); err != nil {
		return domain.MultiServerResult{}, err
	}
	var summarized struct {
		Brief domain.BusinessBrief `json:"brief"`
	}
	if err := call(researchSession, 2, expectedServers[1], expectedTools[1], fmt.Sprintf("%d stories", len(searched.Stories)), map[string]any{"query": request, "stories": searched.Stories}, &summarized, func() string { return fmt.Sprintf("%d signals", len(summarized.Brief.KeySignals)) }); err != nil {
		return domain.MultiServerResult{}, err
	}
	var scored struct {
		Scores domain.OpportunityScores `json:"scores"`
	}
	if err := call(advisorSession, 3, expectedServers[2], expectedTools[2], "brief + founder profile", map[string]any{"query": request, "brief": summarized.Brief, "profile": profile}, &scored, func() string { return fmt.Sprintf("%d ranked opportunities", len(scored.Scores.Items)) }); err != nil {
		return domain.MultiServerResult{}, err
	}
	if len(scored.Scores.Items) == 0 {
		return domain.MultiServerResult{}, errors.New("advisor returned no opportunities")
	}
	var plannedAction struct {
		Plan domain.FounderActionPlan `json:"plan"`
	}
	if err := call(advisorSession, 4, expectedServers[3], expectedTools[3], "top score + founder profile", map[string]any{"query": request, "selected": scored.Scores.Items[0], "profile": profile}, &plannedAction, func() string { return fmt.Sprintf("%d action steps", len(plannedAction.Plan.Steps)) }); err != nil {
		return domain.MultiServerResult{}, err
	}
	var saved struct {
		Report domain.RadarReport `json:"report"`
	}
	if err := call(researchSession, 5, expectedServers[4], expectedTools[4], "approved brief", map[string]any{"query": request, "brief": summarized.Brief, "sourceCount": len(searched.Stories)}, &saved, func() string { return "saved as " + saved.Report.ID }); err != nil {
		return domain.MultiServerResult{}, err
	}
	usage := addUsage(planResponse.Usage, summarized.Brief.Usage)
	usage = addUsage(usage, scored.Scores.Usage)
	usage = addUsage(usage, plannedAction.Plan.Usage)
	return domain.MultiServerResult{Request: request, Profile: profile, Route: route, Servers: servers, Stories: searched.Stories, Brief: summarized.Brief, Scores: scored.Scores, ActionPlan: plannedAction.Plan, SavedReport: saved.Report, Rationale: planned.Rationale, Usage: usage, CompletedAt: s.now().UTC()}, nil
}

func connectAndList(ctx context.Context, name, endpoint string) (*mcp.ClientSession, []*mcp.Tool, error) {
	client := mcp.NewClient(&mcp.Implementation{Name: name, Version: "v1.0.0"}, &mcp.ClientOptions{Capabilities: &mcp.ClientCapabilities{}})
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: endpoint}, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("connect MCP %s: %w", endpoint, err)
	}
	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		session.Close()
		return nil, nil, fmt.Errorf("list MCP tools: %w", err)
	}
	return session, listed.Tools, nil
}
