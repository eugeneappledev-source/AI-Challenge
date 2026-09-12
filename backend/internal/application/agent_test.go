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
