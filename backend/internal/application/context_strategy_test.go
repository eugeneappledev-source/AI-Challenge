package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
	sqlitestore "github.com/eugeneappledev-source/AI-Challenge/backend/internal/infrastructure/sqlite"
)

func newStrategyTestAgent(t *testing.T, replies []domain.ModelResponse) (*Agent, *agentModelSequence, *sqlitestore.ConversationStore) {
	t.Helper()
	store, err := sqlitestore.Open(t.TempDir() + "/agent.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	client := &agentModelSequence{replies: replies}
	agent := NewAgent(domain.AgentProfile{
		ID: "mentor", Instructions: "Помогай", Model: "deepseek-flash", Temperature: 0.3, MaxOutputTokens: 500,
	}, client, 4000).WithMemory(store)
	agent.now = func() time.Time { return time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC) }
	return agent, client, store
}

func TestSlidingWindowPhysicallyKeepsOnlyLastMessages(t *testing.T) {
	const windowSize = 4
	replies := make([]domain.ModelResponse, 5)
	for index := range replies {
		replies[index] = domain.ModelResponse{Content: "ack", Usage: domain.Usage{PromptTokens: 10, CompletionTokens: 2, TotalTokens: 12}}
	}
	agent, client, _ := newStrategyTestAgent(t, replies)

	for _, message := range []string{"one", "two", "three", "four", "five"} {
		if _, err := agent.RespondWithStrategy(context.Background(), "s1", domain.ContextStrategySlidingWindow, "", windowSize, message); err != nil {
			t.Fatalf("respond: %v", err)
		}
	}
	state, err := agent.StrategyState(context.Background(), "s1", domain.ContextStrategySlidingWindow, "", windowSize)
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if len(state.Messages) != windowSize || state.WindowSize != windowSize || state.Messages[0].Content != "four" {
		t.Fatalf("expected physical last-%d window, got %+v", windowSize, state.Messages)
	}
	for _, request := range client.requests {
		if len(request.Messages) > windowSize+2 {
			t.Fatalf("request exceeded sliding window: %+v", request.Messages)
		}
	}
}

func TestSlidingWindowCanBeChangedForExperiment(t *testing.T) {
	agent, _, _ := newStrategyTestAgent(t, []domain.ModelResponse{
		{Content: "ack"}, {Content: "ack"}, {Content: "ack"},
	})
	ctx := context.Background()
	for _, message := range []string{"one", "two", "three"} {
		if _, err := agent.RespondWithStrategy(ctx, "variable", domain.ContextStrategySlidingWindow, "", 6, message); err != nil {
			t.Fatalf("respond: %v", err)
		}
	}
	state, err := agent.StrategyState(ctx, "variable", domain.ContextStrategySlidingWindow, "", 2)
	if err != nil {
		t.Fatalf("apply smaller window: %v", err)
	}
	if state.WindowSize != 2 || len(state.Messages) != 2 || state.Messages[0].Content != "three" {
		t.Fatalf("window selection was not applied: %+v", state)
	}
	if _, err := agent.StrategyState(ctx, "variable", domain.ContextStrategySlidingWindow, "", 1); !errors.Is(err, ErrInvalidContextWindow) {
		t.Fatalf("expected invalid window error, got %v", err)
	}
}

func TestStickyFactsUpdatesKVAndInjectsItIntoContext(t *testing.T) {
	agent, client, _ := newStrategyTestAgent(t, []domain.ModelResponse{
		{Content: `{"goal":"ship MVP"}`, Usage: domain.Usage{PromptTokens: 8, CompletionTokens: 4, TotalTokens: 12}},
		{Content: "Принято", Usage: domain.Usage{PromptTokens: 12, CompletionTokens: 2, TotalTokens: 14}},
		{Content: `{"goal":"ship MVP","platform":"iOS 17"}`, Usage: domain.Usage{PromptTokens: 14, CompletionTokens: 6, TotalTokens: 20}},
		{Content: "Запомнил", Usage: domain.Usage{PromptTokens: 20, CompletionTokens: 2, TotalTokens: 22}},
	})

	if _, err := agent.RespondWithStrategy(context.Background(), "s2", domain.ContextStrategyStickyFacts, "", 6, "Цель — ship MVP"); err != nil {
		t.Fatalf("first response: %v", err)
	}
	exchange, err := agent.RespondWithStrategy(context.Background(), "s2", domain.ContextStrategyStickyFacts, "", 6, "Платформа iOS 17")
	if err != nil {
		t.Fatalf("second response: %v", err)
	}
	if len(exchange.State.Facts) != 2 || exchange.State.LastUsage.TotalTokens != 42 {
		t.Fatalf("facts or operation usage not preserved: %+v", exchange.State)
	}
	lastRequest := client.requests[len(client.requests)-1]
	if len(lastRequest.Messages) < 2 || !strings.Contains(lastRequest.Messages[1].Content, `"goal":"ship MVP"`) {
		t.Fatalf("facts must be a separate model context block: %+v", lastRequest.Messages)
	}
}

func TestBranchCheckpointProducesIndependentHistories(t *testing.T) {
	agent, _, _ := newStrategyTestAgent(t, []domain.ModelResponse{
		{Content: "Общая база"}, {Content: "Только MVP"}, {Content: "Только Growth"},
	})
	ctx := context.Background()
	if _, err := agent.RespondWithStrategy(ctx, "s3", domain.ContextStrategyBranching, "main", 6, "Общее требование"); err != nil {
		t.Fatalf("main response: %v", err)
	}
	if _, err := agent.CreateStrategyBranches(ctx, "s3"); err != nil {
		t.Fatalf("create branches: %v", err)
	}
	if _, err := agent.RespondWithStrategy(ctx, "s3", domain.ContextStrategyBranching, "mvp", 6, "Упростить MVP"); err != nil {
		t.Fatalf("mvp response: %v", err)
	}
	if _, err := agent.RespondWithStrategy(ctx, "s3", domain.ContextStrategyBranching, "growth", 6, "Добавить Growth"); err != nil {
		t.Fatalf("growth response: %v", err)
	}
	mvpState, _ := agent.StrategyState(ctx, "s3", domain.ContextStrategyBranching, "mvp", 6)
	growthState, _ := agent.StrategyState(ctx, "s3", domain.ContextStrategyBranching, "growth", 6)
	if len(mvpState.Messages) != 4 || len(growthState.Messages) != 4 {
		t.Fatalf("unexpected branch sizes: mvp=%d growth=%d", len(mvpState.Messages), len(growthState.Messages))
	}
	if strings.Contains(growthState.Messages[2].Content, "MVP") || strings.Contains(mvpState.Messages[2].Content, "Growth") {
		t.Fatalf("branch histories leaked: mvp=%+v growth=%+v", mvpState.Messages, growthState.Messages)
	}
	if len(mvpState.Branches) != 3 || mvpState.Branches[1].CheckpointMessages != 2 {
		t.Fatalf("checkpoint metadata missing: %+v", mvpState.Branches)
	}
}
