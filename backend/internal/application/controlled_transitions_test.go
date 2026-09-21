package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

func TestControlledTransitionRejectsSkippedStageWithoutLLM(t *testing.T) {
	client := &agentModelSequence{}
	store := &conversationStoreStub{taskState: controlledTaskState(domain.TaskPhasePlanning)}
	agent := controlledTransitionTestAgent(client, store)

	exchange, err := agent.TransitionTask(context.Background(), "task-1", domain.TaskPhaseValidation, "Давай сразу проверять")
	if err != nil {
		t.Fatalf("reject skipped stage: %v", err)
	}
	if exchange.Allowed || exchange.Code != "transition_not_allowed" || exchange.State.Phase != domain.TaskPhasePlanning {
		t.Fatalf("unexpected rejection: %+v", exchange)
	}
	if len(client.requests) != 0 || exchange.Usage.TotalTokens != 0 {
		t.Fatalf("rejected transition must not call LLM: calls=%d usage=%+v", len(client.requests), exchange.Usage)
	}
	if len(exchange.State.Attempts) != 1 || exchange.State.Attempts[0].Allowed {
		t.Fatalf("rejected attempt must be persisted: %+v", exchange.State.Attempts)
	}
}

func TestControlledTransitionAppliesAllowedEdgeAndCreatesArtifact(t *testing.T) {
	client := &agentModelSequence{replies: []domain.ModelResponse{{
		Content: "Архитектурное решение готово", Model: "deepseek-flash", FinishReason: "stop", Usage: domain.Usage{TotalTokens: 90},
	}}}
	store := &conversationStoreStub{taskState: controlledTaskState(domain.TaskPhasePlanning)}
	agent := controlledTransitionTestAgent(client, store)

	exchange, err := agent.TransitionTask(context.Background(), "task-1", domain.TaskPhaseExecution, "План утверждён")
	if err != nil {
		t.Fatalf("apply allowed transition: %v", err)
	}
	if !exchange.Allowed || exchange.State.Phase != domain.TaskPhaseExecution || exchange.Answer == "" {
		t.Fatalf("unexpected allowed transition: %+v", exchange)
	}
	if len(client.requests) != 1 || len(exchange.State.Artifacts) != 2 || !exchange.State.Attempts[len(exchange.State.Attempts)-1].Allowed {
		t.Fatalf("expected one LLM call, artifact and audit: calls=%d state=%+v", len(client.requests), exchange.State)
	}
	joined := modelMessageText(client.requests[0].Messages)
	if !strings.Contains(joined, "phase: execution") || !strings.Contains(joined, "План готов") {
		t.Fatalf("agent must receive only approved target and previous artifact: %s", joined)
	}
}

func TestControlledTransitionSupportsRollbackAndBlocksPausedTask(t *testing.T) {
	client := &agentModelSequence{replies: []domain.ModelResponse{{Content: "План доработки", Model: "deepseek-flash", FinishReason: "stop"}}}
	state := controlledTaskState(domain.TaskPhaseValidation)
	state.Status = domain.TaskStatusPaused
	state.Artifacts = append(state.Artifacts,
		domain.TaskArtifact{Phase: domain.TaskPhaseExecution, Content: "Реализация готова"},
		domain.TaskArtifact{Phase: domain.TaskPhaseValidation, Content: "Найден риск"},
	)
	store := &conversationStoreStub{taskState: state}
	agent := controlledTransitionTestAgent(client, store)

	blocked, err := agent.TransitionTask(context.Background(), "task-1", domain.TaskPhaseExecution, "Исправить найденный риск")
	if err != nil {
		t.Fatalf("block paused transition: %v", err)
	}
	if blocked.Allowed || blocked.Code != "task_paused" || blocked.State.Phase != domain.TaskPhaseValidation || len(client.requests) != 0 {
		t.Fatalf("unexpected paused rejection: %+v calls=%d", blocked, len(client.requests))
	}

	store.taskState.Status = domain.TaskStatusActive
	rolledBack, err := agent.TransitionTask(context.Background(), "task-1", domain.TaskPhaseExecution, "Исправить найденный риск")
	if err != nil {
		t.Fatalf("rollback task: %v", err)
	}
	if !rolledBack.Allowed || rolledBack.State.Phase != domain.TaskPhaseExecution || !strings.Contains(rolledBack.Reason, "откат") {
		t.Fatalf("unexpected rollback: %+v", rolledBack)
	}
}

func controlledTaskState(phase domain.TaskPhase) *domain.TaskState {
	return &domain.TaskState{
		ID: "task-1", UserID: "user-1", ProfileID: "engineer", Goal: "Подготовить экран статистики",
		Phase: phase, Status: domain.TaskStatusActive, CurrentStep: "Текущий шаг", ExpectedAction: "Следующее действие",
		Artifacts:   []domain.TaskArtifact{{Phase: domain.TaskPhasePlanning, Content: "План готов"}},
		Transitions: []domain.TaskTransition{}, Attempts: []domain.TaskTransitionAttempt{}, Revision: 1,
	}
}

func controlledTransitionTestAgent(client ReasoningClient, store ConversationStore) *Agent {
	agent := NewAgent(domain.AgentProfile{
		ID: "mentor", Name: "Compass", Instructions: "Помогай.", Model: "deepseek-flash", Temperature: 0.3, MaxOutputTokens: 1000,
	}, client, 4000).WithMemory(store)
	agent.now = func() time.Time { return time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC) }
	return agent
}
