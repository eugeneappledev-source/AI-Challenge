package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

var (
	ErrEmptyMessage              = errors.New("message is required")
	ErrMessageTooLong            = errors.New("message is too long")
	ErrInvalidMode               = errors.New("response mode is invalid")
	ErrInvalidControlledResponse = errors.New("controlled response does not match the contract")
)

type CompletionClient interface {
	Complete(ctx context.Context, request domain.CompletionRequest) (domain.ChatReply, error)
}

type ChatService struct {
	client          CompletionClient
	maxMessageRunes int
}

func NewChatService(client CompletionClient, maxMessageRunes int) *ChatService {
	return &ChatService{client: client, maxMessageRunes: maxMessageRunes}
}

func (s *ChatService) Send(ctx context.Context, message string, mode domain.ResponseMode) (domain.ChatReply, error) {
	normalized := strings.TrimSpace(message)
	if normalized == "" {
		return domain.ChatReply{}, ErrEmptyMessage
	}
	if utf8.RuneCountInString(normalized) > s.maxMessageRunes {
		return domain.ChatReply{}, ErrMessageTooLong
	}
	if mode == "" {
		mode = domain.ResponseModeUnrestricted
	}
	if !mode.IsValid() {
		return domain.ChatReply{}, ErrInvalidMode
	}
	reply, err := s.client.Complete(ctx, domain.CompletionRequest{
		Message: normalized,
		Mode:    mode,
	})
	if err != nil {
		return domain.ChatReply{}, err
	}
	if mode == domain.ResponseModeControlled {
		reply.Answer, err = validateAndNormalizeControlledAnswer(reply.Answer)
		if err != nil {
			return domain.ChatReply{}, fmt.Errorf("%w: %v", ErrInvalidControlledResponse, err)
		}
	}
	return reply, nil
}
