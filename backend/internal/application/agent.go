package application

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

var (
	ErrEmptyAgentMessage      = errors.New("agent message is required")
	ErrAgentMessageTooLong    = errors.New("agent message is too long")
	ErrConversationIDRequired = errors.New("conversation id is required")
)

type ConversationStore interface {
	Load(ctx context.Context, conversationID, agentID string) (domain.AgentConversation, error)
	Append(ctx context.Context, conversationID, agentID string, messages ...domain.AgentMessage) error
	Clear(ctx context.Context, conversationID, agentID string) error
}

type Agent struct {
	profile         domain.AgentProfile
	client          ReasoningClient
	maxMessageRunes int
	now             func() time.Time
	store           ConversationStore
}

func (a *Agent) WithMemory(store ConversationStore) *Agent {
	a.store = store
	return a
}

func NewAgent(profile domain.AgentProfile, client ReasoningClient, maxMessageRunes int) *Agent {
	return &Agent{profile: profile, client: client, maxMessageRunes: maxMessageRunes, now: time.Now}
}

func (a *Agent) Profile() domain.AgentProfile { return a.profile }

func (a *Agent) Respond(ctx context.Context, input string) (domain.AgentExchange, error) {
	return a.respond(ctx, "", nil, input)
}

func (a *Agent) RespondInConversation(ctx context.Context, conversationID, input string) (domain.AgentExchange, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return domain.AgentExchange{}, ErrConversationIDRequired
	}
	if a.store == nil {
		return domain.AgentExchange{}, errors.New("agent memory is not configured")
	}
	conversation, err := a.store.Load(ctx, conversationID, a.profile.ID)
	if err != nil {
		return domain.AgentExchange{}, err
	}
	return a.respond(ctx, conversationID, conversation.Messages, input)
}

func (a *Agent) History(ctx context.Context, conversationID string) (domain.AgentConversation, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return domain.AgentConversation{}, ErrConversationIDRequired
	}
	if a.store == nil {
		return domain.AgentConversation{}, errors.New("agent memory is not configured")
	}
	return a.store.Load(ctx, conversationID, a.profile.ID)
}

func (a *Agent) ClearHistory(ctx context.Context, conversationID string) error {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return ErrConversationIDRequired
	}
	if a.store == nil {
		return errors.New("agent memory is not configured")
	}
	return a.store.Clear(ctx, conversationID, a.profile.ID)
}

func (a *Agent) respond(ctx context.Context, conversationID string, history []domain.AgentMessage, input string) (domain.AgentExchange, error) {
	normalized := strings.TrimSpace(input)
	if normalized == "" {
		return domain.AgentExchange{}, ErrEmptyAgentMessage
	}
	if utf8.RuneCountInString(normalized) > a.maxMessageRunes {
		return domain.AgentExchange{}, ErrAgentMessageTooLong
	}

	temperature := a.profile.Temperature
	now := a.now().UTC()
	userMessage := domain.AgentMessage{ID: newID("usr", now), Role: "user", Content: normalized, CreatedAt: now}
	messages := make([]domain.ModelMessage, 0, len(history)+2)
	if conversationID != "" {
		messages = append(messages, domain.ModelMessage{Role: "system", Content: a.profile.Instructions})
		for _, message := range history {
			messages = append(messages, domain.ModelMessage{Role: message.Role, Content: message.Content})
		}
		messages = append(messages, domain.ModelMessage{Role: "user", Content: normalized})
	}
	response, err := a.client.Generate(ctx, domain.ModelRequest{
		Model:        a.profile.Model,
		SystemPrompt: a.profile.Instructions,
		UserPrompt:   normalized,
		Temperature:  &temperature,
		MaxTokens:    a.profile.MaxOutputTokens,
		Messages:     messages,
	})
	if err != nil {
		return domain.AgentExchange{}, err
	}
	usage := response.Usage
	reply := domain.AgentMessage{ID: newID("asst", now.Add(time.Nanosecond)), Role: "assistant", Content: response.Content, CreatedAt: now, Usage: &usage}
	if conversationID != "" {
		if err := a.store.Append(ctx, conversationID, a.profile.ID, userMessage, reply); err != nil {
			return domain.AgentExchange{}, err
		}
	}
	return domain.AgentExchange{
		Agent:          a.profile,
		ConversationID: conversationID,
		HistoryCount:   len(history) + 2,
		UserMessage:    userMessage,
		Reply:          reply,
		Model:          response.Model,
		FinishReason:   response.FinishReason,
		Usage:          response.Usage,
		Trace:          []string{"Agent accepted user input", "Agent applied its profile and instructions", "Agent called LLMProvider", "Agent returned the model response"},
	}, nil
}

func newID(prefix string, moment time.Time) string {
	return prefix + "_" + moment.Format("20060102T150405.000000000")
}
