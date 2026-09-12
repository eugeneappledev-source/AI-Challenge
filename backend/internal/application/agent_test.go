package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

type agentModelRecorder struct {
	request domain.ModelRequest
	reply   domain.ModelResponse
}

type conversationStoreStub struct {
	conversation domain.AgentConversation
	appended     []domain.AgentMessage
}

func (s *conversationStoreStub) Load(_ context.Context, conversationID, agentID string) (domain.AgentConversation, error) {
	result := s.conversation
	result.ID, result.AgentID = conversationID, agentID
	return result, nil
}

func (s *conversationStoreStub) Append(_ context.Context, _, _ string, messages ...domain.AgentMessage) error {
	s.appended = append(s.appended, messages...)
	return nil
}

func (s *conversationStoreStub) Clear(_ context.Context, _, _ string) error {
	s.conversation.Messages = nil
	return nil
}

func (r *agentModelRecorder) Generate(_ context.Context, request domain.ModelRequest) (domain.ModelResponse, error) {
	r.request = request
	return r.reply, nil
}

func TestAgentOwnsLLMConfigurationAndReturnsExchange(t *testing.T) {
	client := &agentModelRecorder{reply: domain.ModelResponse{
		Content: "Ответ агента", Model: "deepseek-flash", FinishReason: "stop",
		Usage: domain.Usage{PromptTokens: 12, CompletionTokens: 4, TotalTokens: 16},
	}}
	profile := domain.AgentProfile{
		ID: "mentor", Name: "Compass", Role: "AI-наставник",
		Instructions: "Помогай пользователю.", Model: "deepseek-flash",
		Temperature: 0.3, MaxOutputTokens: 1000,
	}
	agent := NewAgent(profile, client, 4000)
	agent.now = func() time.Time { return time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC) }

	exchange, err := agent.Respond(context.Background(), "  Объясни AI-агентов  ")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.request.Model != profile.Model || client.request.SystemPrompt != profile.Instructions {
		t.Fatalf("agent configuration was not forwarded: %+v", client.request)
	}
	if client.request.Temperature == nil || *client.request.Temperature != profile.Temperature || client.request.MaxTokens != profile.MaxOutputTokens {
		t.Fatalf("agent generation settings were not forwarded: %+v", client.request)
	}
	if client.request.UserPrompt != "Объясни AI-агентов" {
		t.Fatalf("expected trimmed user input, got %q", client.request.UserPrompt)
	}
	if exchange.Agent != profile || exchange.UserMessage.Content != "Объясни AI-агентов" || exchange.Reply.Content != "Ответ агента" {
		t.Fatalf("unexpected exchange: %+v", exchange)
	}
	if len(exchange.Trace) != 4 || exchange.Usage.TotalTokens != 16 {
		t.Fatalf("expected observable agent trace and usage, got %+v", exchange)
	}
}

func TestAgentValidatesInputBeforeCallingModel(t *testing.T) {
	client := &agentModelRecorder{}
	agent := NewAgent(domain.AgentProfile{}, client, 3)

	if _, err := agent.Respond(context.Background(), "  "); !errors.Is(err, ErrEmptyAgentMessage) {
		t.Fatalf("expected empty message error, got %v", err)
	}
	if _, err := agent.Respond(context.Background(), "четыре"); !errors.Is(err, ErrAgentMessageTooLong) {
		t.Fatalf("expected long message error, got %v", err)
	}
	if client.request.UserPrompt != "" {
		t.Fatal("model must not be called for invalid input")
	}
}

func TestAgentRestoresHistoryBeforeCallingModelAndPersistsExchange(t *testing.T) {
	client := &agentModelRecorder{reply: domain.ModelResponse{Content: "Тебя зовут Женя", Model: "deepseek-flash"}}
	store := &conversationStoreStub{conversation: domain.AgentConversation{Messages: []domain.AgentMessage{
		{ID: "u0", Role: "user", Content: "Меня зовут Женя"},
		{ID: "a0", Role: "assistant", Content: "Запомнил"},
	}}}
	agent := NewAgent(domain.AgentProfile{ID: "mentor", Instructions: "Помни контекст", Model: "deepseek-flash"}, client, 4000).WithMemory(store)

	exchange, err := agent.RespondInConversation(context.Background(), "conversation-1", "Как меня зовут?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(client.request.Messages) != 4 || client.request.Messages[1].Content != "Меня зовут Женя" || client.request.Messages[3].Content != "Как меня зовут?" {
		t.Fatalf("expected system + restored history + new input, got %+v", client.request.Messages)
	}
	if len(store.appended) != 2 || store.appended[0].Role != "user" || store.appended[1].Role != "assistant" {
		t.Fatalf("expected atomic exchange persistence, got %+v", store.appended)
	}
	if exchange.ConversationID != "conversation-1" || exchange.HistoryCount != 4 {
		t.Fatalf("unexpected conversation metadata: %+v", exchange)
	}
}
