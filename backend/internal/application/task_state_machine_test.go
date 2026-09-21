package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

func TestTaskStateMachinePausesPersistsAndResumesWithoutOriginalPrompt(t *testing.T) {
	client := &agentModelSequence{replies: []domain.ModelResponse{
		{Content: "План готов", Model: "deepseek-flash", FinishReason: "stop", Usage: domain.Usage{TotalTokens: 20}},
		{Content: "Продолжаю с планирования", Model: "deepseek-flash", FinishReason: "stop", Usage: domain.Usage{TotalTokens: 15}},
		{Content: "Архитектура готова", Model: "deepseek-flash", FinishReason: "stop", Usage: domain.Usage{TotalTokens: 24}},
	}}
	store := &conversationStoreStub{}
	agent := NewAgent(domain.AgentProfile{
		ID: "mentor", Name: "Compass", Instructions: "Помогай.", Model: "deepseek-flash", Temperature: 0.3, MaxOutputTokens: 1000,
	}, client, 4000).WithMemory(store)
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	agent.now = func() time.Time { now = now.Add(time.Second); return now }

	created, err := agent.CreateTask(context.Background(), "task-1", "user-1", "engineer", "Подготовить архитектуру PulsePlan")
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if created.State.Phase != domain.TaskPhasePlanning || created.State.CurrentStep == "" || created.State.ExpectedAction == "" {
		t.Fatalf("unexpected planning state: %+v", created.State)
	}
	paused, err := agent.ActOnTask(context.Background(), "task-1", domain.TaskActionPause)
	if err != nil {
		t.Fatalf("pause task: %v", err)
	}
	if paused.State.Status != domain.TaskStatusPaused || paused.State.Phase != domain.TaskPhasePlanning {
		t.Fatalf("pause must preserve phase: %+v", paused.State)
	}

	// A new Agent instance emulates an application restart; it only receives the persisted store.
	restarted := NewAgent(agent.profile, client, 4000).WithMemory(store)
	restarted.now = agent.now
	resumed, err := restarted.ActOnTask(context.Background(), "task-1", domain.TaskActionResume)
	if err != nil {
		t.Fatalf("resume task: %v", err)
	}
	if resumed.State.Status != domain.TaskStatusActive || resumed.State.Phase != domain.TaskPhasePlanning {
		t.Fatalf("resume must restore the same phase: %+v", resumed.State)
	}
	lastRequest := client.requests[len(client.requests)-1]
	joined := modelMessageText(lastRequest.Messages)
	if !strings.Contains(joined, "Подготовить архитектуру PulsePlan") || !strings.Contains(joined, "План готов") {
		t.Fatalf("resume request must restore goal and artifacts, got: %s", joined)
	}

	advanced, err := restarted.ActOnTask(context.Background(), "task-1", domain.TaskActionAdvance)
	if err != nil {
		t.Fatalf("advance task: %v", err)
	}
	if advanced.State.Phase != domain.TaskPhaseExecution || len(advanced.State.Artifacts) != 2 {
		t.Fatalf("unexpected execution state: %+v", advanced.State)
	}
}

func TestTaskStateMachineRejectsAdvanceWhilePaused(t *testing.T) {
	store := &conversationStoreStub{taskState: &domain.TaskState{
		ID: "task-1", Phase: domain.TaskPhaseExecution, Status: domain.TaskStatusPaused,
	}}
	agent := NewAgent(domain.AgentProfile{ID: "mentor"}, &agentModelRecorder{}, 4000).WithMemory(store)
	_, err := agent.ActOnTask(context.Background(), "task-1", domain.TaskActionAdvance)
	if err != ErrTaskAlreadyPaused {
		t.Fatalf("expected paused error, got %v", err)
	}
}

func modelMessageText(messages []domain.ModelMessage) string {
	var builder strings.Builder
	for _, message := range messages {
		builder.WriteString(message.Content)
		builder.WriteByte('\n')
	}
	return builder.String()
}
