package application

import (
	"context"
	"errors"
	"testing"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

type completionClientStub struct {
	reply   domain.ChatReply
	err     error
	request domain.CompletionRequest
}

func (s *completionClientStub) Complete(_ context.Context, request domain.CompletionRequest) (domain.ChatReply, error) {
	s.request = request
	return s.reply, s.err
}

func TestChatServiceTrimsAndForwardsMessage(t *testing.T) {
	client := &completionClientStub{reply: domain.ChatReply{Answer: "answer"}}
	service := NewChatService(client, 100)

	reply, err := service.Send(context.Background(), "  hello  ", "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.request.Message != "hello" {
		t.Fatalf("expected normalized message, got %q", client.request.Message)
	}
	if client.request.Mode != domain.ResponseModeUnrestricted {
		t.Fatalf("expected default unrestricted mode, got %q", client.request.Mode)
	}
	if reply.Answer != "answer" {
		t.Fatalf("expected reply to be forwarded, got %q", reply.Answer)
	}
}

func TestChatServiceRejectsEmptyMessage(t *testing.T) {
	service := NewChatService(&completionClientStub{}, 100)

	_, err := service.Send(context.Background(), "   ", domain.ResponseModeUnrestricted)

	if !errors.Is(err, ErrEmptyMessage) {
		t.Fatalf("expected ErrEmptyMessage, got %v", err)
	}
}

func TestChatServiceRejectsLongUnicodeMessage(t *testing.T) {
	service := NewChatService(&completionClientStub{}, 3)

	_, err := service.Send(context.Background(), "привет", domain.ResponseModeUnrestricted)

	if !errors.Is(err, ErrMessageTooLong) {
		t.Fatalf("expected ErrMessageTooLong, got %v", err)
	}
}

func TestChatServiceForwardsControlledMode(t *testing.T) {
	client := &completionClientStub{reply: domain.ChatReply{Answer: "answer"}}
	service := NewChatService(client, 100)

	_, err := service.Send(context.Background(), "hello", domain.ResponseModeControlled)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.request.Mode != domain.ResponseModeControlled {
		t.Fatalf("expected controlled mode, got %q", client.request.Mode)
	}
}

func TestChatServiceRejectsInvalidMode(t *testing.T) {
	service := NewChatService(&completionClientStub{}, 100)

	_, err := service.Send(context.Background(), "hello", domain.ResponseMode("creative"))

	if !errors.Is(err, ErrInvalidMode) {
		t.Fatalf("expected ErrInvalidMode, got %v", err)
	}
}
