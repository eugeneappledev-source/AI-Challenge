package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

func TestInvariantGuardRejectsExplicitConflictBeforeLLM(t *testing.T) {
	client := &agentModelSequence{}
	store := &conversationStoreStub{}
	agent := invariantTestAgent(client, store)

	exchange, err := agent.RespondWithInvariants(context.Background(), "task-1", "user-1", "engineer", "Перепиши экран на UIKit и RxSwift")
	if err != nil {
		t.Fatalf("check invariants: %v", err)
	}
	if exchange.Verdict != domain.InvariantVerdictRejected || len(exchange.Conflicts) != 1 || exchange.Conflicts[0].InvariantID != "ios_stack" {
		t.Fatalf("unexpected hard rejection: %+v", exchange)
	}
	if exchange.Conflicts[0].DetectedBy != "hard_check" || len(client.requests) != 0 {
		t.Fatalf("hard conflict must block all LLM calls: %+v requests=%d", exchange.Conflicts[0], len(client.requests))
	}
}

func TestInvariantGuardRejectsSemanticConflictAndBlocksMainAgent(t *testing.T) {
	client := &agentModelSequence{replies: []domain.ModelResponse{{
		Content: `{"allowed":false,"assessments":[{"invariantId":"offline_first","verdict":"conflict","note":"Экран зависит от сети"}],"conflicts":[{"invariantId":"offline_first","reason":"Запрос делает сеть обязательной"}],"explanation":"Это нарушает offline-first.","safeAlternative":"Показывать локальные данные и синхронизировать позже."}`,
		Model:   "deepseek-flash", FinishReason: "stop", Usage: domain.Usage{TotalTokens: 80},
	}}}
	store := &conversationStoreStub{}
	agent := invariantTestAgent(client, store)

	exchange, err := agent.RespondWithInvariants(context.Background(), "task-1", "user-1", "engineer", "Показывай статистику только при наличии интернета, иначе ошибку")
	if err != nil {
		t.Fatalf("check semantic conflict: %v", err)
	}
	if exchange.Verdict != domain.InvariantVerdictRejected || len(exchange.Conflicts) != 1 || exchange.Conflicts[0].DetectedBy != "semantic_guard" {
		t.Fatalf("unexpected semantic rejection: %+v", exchange)
	}
	if len(client.requests) != 1 || exchange.Answer != "" {
		t.Fatalf("semantic rejection must not call the main agent: calls=%d answer=%q", len(client.requests), exchange.Answer)
	}
}

func TestInvariantGuardAllowsCompatibleRequestAndAttachesRules(t *testing.T) {
	client := &agentModelSequence{replies: []domain.ModelResponse{
		{
			Content: `{"allowed":true,"assessments":[{"invariantId":"architecture_boundaries","verdict":"compliant","note":"Границы сохранены"},{"invariantId":"ios_stack","verdict":"compliant","note":"Стек сохранён"},{"invariantId":"offline_first","verdict":"compliant","note":"Offline-first сохранён"},{"invariantId":"accessibility","verdict":"compliant","note":"VoiceOver учтён"}],"conflicts":[],"explanation":"Запрос совместим.","safeAlternative":""}`,
			Model:   "deepseek-flash", FinishReason: "stop", Usage: domain.Usage{TotalTokens: 70},
		},
		{Content: "Совместимое решение", Model: "deepseek-flash", FinishReason: "stop", Usage: domain.Usage{TotalTokens: 100}},
	}}
	store := &conversationStoreStub{}
	agent := invariantTestAgent(client, store)

	exchange, err := agent.RespondWithInvariants(context.Background(), "task-1", "user-1", "engineer", "Предложи структуру модулей SwiftUI")
	if err != nil {
		t.Fatalf("allow request: %v", err)
	}
	if exchange.Verdict != domain.InvariantVerdictAllowed || exchange.Answer != "Совместимое решение" || len(client.requests) != 2 {
		t.Fatalf("unexpected allowed exchange: %+v calls=%d", exchange, len(client.requests))
	}
	mainPrompt := modelMessageText(client.requests[1].Messages)
	if !strings.Contains(mainPrompt, "ОБЯЗАТЕЛЬНЫЕ ИНВАРИАНТЫ") || !strings.Contains(mainPrompt, "Offline-first") || !strings.Contains(mainPrompt, "VoiceOver") {
		t.Fatalf("main agent must receive invariants: %s", mainPrompt)
	}
	if exchange.Usage.TotalTokens != 170 {
		t.Fatalf("expected combined guard and answer usage, got %+v", exchange.Usage)
	}
}

func invariantTestAgent(client ReasoningClient, store ConversationStore) *Agent {
	agent := NewAgent(domain.AgentProfile{
		ID: "mentor", Name: "Compass", Instructions: "Помогай.", Model: "deepseek-flash", Temperature: 0.3, MaxOutputTokens: 1000,
	}, client, 4000).WithMemory(store)
	agent.now = func() time.Time { return time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC) }
	return agent
}
