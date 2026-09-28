package mcpserver

import (
	"context"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type OpportunityAdvisor interface {
	Score(context.Context, string, domain.BusinessBrief, domain.FounderProfile) (domain.OpportunityScores, error)
	BuildPlan(context.Context, string, domain.ScoredOpportunity, domain.FounderProfile) (domain.FounderActionPlan, error)
}

type ScoreOpportunitiesInput struct {
	Query   string                `json:"query" jsonschema:"original business research task"`
	Brief   domain.BusinessBrief  `json:"brief" jsonschema:"brief from startup-research-mcp"`
	Profile domain.FounderProfile `json:"profile" jsonschema:"founder resources and constraints"`
}
type ScoreOpportunitiesOutput struct {
	Scores domain.OpportunityScores `json:"scores" jsonschema:"ranked opportunities"`
}
type BuildActionPlanInput struct {
	Query    string                   `json:"query" jsonschema:"original business research task"`
	Selected domain.ScoredOpportunity `json:"selected" jsonschema:"top scored opportunity"`
	Profile  domain.FounderProfile    `json:"profile" jsonschema:"founder resources and constraints"`
}
type BuildActionPlanOutput struct {
	Plan domain.FounderActionPlan `json:"plan" jsonschema:"bounded validation plan"`
}

func NewAdvisorServer(advisor OpportunityAdvisor) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "business-advisor-mcp", Version: "v1.0.0"}, &mcp.ServerOptions{Instructions: "Tools for scoring source-grounded opportunities and preparing bounded validation plans."})
	mcp.AddTool(server, &mcp.Tool{Name: "score_business_opportunities", Title: "Score opportunities for a founder", Description: "Use a brief produced by startup-research-mcp and a founder profile to rank realistic opportunities. Call before build_action_plan.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}, func(ctx context.Context, _ *mcp.CallToolRequest, input ScoreOpportunitiesInput) (*mcp.CallToolResult, ScoreOpportunitiesOutput, error) {
		scores, err := advisor.Score(ctx, input.Query, input.Brief, input.Profile)
		return nil, ScoreOpportunitiesOutput{Scores: scores}, err
	})
	mcp.AddTool(server, &mcp.Tool{Name: "build_action_plan", Title: "Build a small validation plan", Description: "Build a resource-aware action plan for the selected scored opportunity. Requires output from score_business_opportunities.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}, func(ctx context.Context, _ *mcp.CallToolRequest, input BuildActionPlanInput) (*mcp.CallToolResult, BuildActionPlanOutput, error) {
		plan, err := advisor.BuildPlan(ctx, input.Query, input.Selected, input.Profile)
		return nil, BuildActionPlanOutput{Plan: plan}, err
	})
	return server
}
