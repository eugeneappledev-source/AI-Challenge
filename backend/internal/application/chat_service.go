package application

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

var (
	ErrEmptyMessage   = errors.New("message is required")
	ErrMessageTooLong = errors.New("message is too long")
)

type CompletionClient interface {
	Complete(ctx context.Context, message string) (domain.ChatReply, error)
}

type ChatService struct {
	client          CompletionClient
	maxMessageRunes int
}

func NewChatService(client CompletionClient, maxMessageRunes int) *ChatService {
	return &ChatService{client: client, maxMessageRunes: maxMessageRunes}
}

func (s *ChatService) Send(ctx context.Context, message string) (domain.ChatReply, error) {
	normalized := strings.TrimSpace(message)
	if normalized == "" {
		return domain.ChatReply{}, ErrEmptyMessage
	}
	if utf8.RuneCountInString(normalized) > s.maxMessageRunes {
		return domain.ChatReply{}, ErrMessageTooLong
	}
	return s.client.Complete(ctx, normalized)
}
