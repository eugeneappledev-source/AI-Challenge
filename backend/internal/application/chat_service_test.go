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
	message string
}

func (s *completionClientStub) Complete(_ context.Context, message string) (domain.ChatReply, error) {
	s.message = message
	return s.reply, s.err
}

func TestChatServiceTrimsAndForwardsMessage(t *testing.T) {
	client := &completionClientStub{reply: domain.ChatReply{Answer: "answer"}}
	service := NewChatService(client, 100)

	reply, err := service.Send(context.Background(), "  hello  ")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.message != "hello" {
		t.Fatalf("expected normalized message, got %q", client.message)
	}
	if reply.Answer != "answer" {
		t.Fatalf("expected reply to be forwarded, got %q", reply.Answer)
	}
}

func TestChatServiceRejectsEmptyMessage(t *testing.T) {
	service := NewChatService(&completionClientStub{}, 100)

	_, err := service.Send(context.Background(), "   ")

	if !errors.Is(err, ErrEmptyMessage) {
		t.Fatalf("expected ErrEmptyMessage, got %v", err)
	}
}

func TestChatServiceRejectsLongUnicodeMessage(t *testing.T) {
	service := NewChatService(&completionClientStub{}, 3)

	_, err := service.Send(context.Background(), "привет")

	if !errors.Is(err, ErrMessageTooLong) {
		t.Fatalf("expected ErrMessageTooLong, got %v", err)
	}
}
