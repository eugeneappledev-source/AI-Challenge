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
	ErrEmptyAgentMessage   = errors.New("agent message is required")
	ErrAgentMessageTooLong = errors.New("agent message is too long")
)

type Agent struct {
	profile         domain.AgentProfile
	client          ReasoningClient
	maxMessageRunes int
	now             func() time.Time
}

func NewAgent(profile domain.AgentProfile, client ReasoningClient, maxMessageRunes int) *Agent {
	return &Agent{profile: profile, client: client, maxMessageRunes: maxMessageRunes, now: time.Now}
}

func (a *Agent) Profile() domain.AgentProfile { return a.profile }

func (a *Agent) Respond(ctx context.Context, input string) (domain.AgentExchange, error) {
	normalized := strings.TrimSpace(input)
	if normalized == "" {
		return domain.AgentExchange{}, ErrEmptyAgentMessage
	}
	if utf8.RuneCountInString(normalized) > a.maxMessageRunes {
		return domain.AgentExchange{}, ErrAgentMessageTooLong
	}

	temperature := a.profile.Temperature
	response, err := a.client.Generate(ctx, domain.ModelRequest{
		Model:        a.profile.Model,
		SystemPrompt: a.profile.Instructions,
		UserPrompt:   normalized,
		Temperature:  &temperature,
		MaxTokens:    a.profile.MaxOutputTokens,
	})
	if err != nil {
		return domain.AgentExchange{}, err
	}
	now := a.now().UTC()
	usage := response.Usage
	return domain.AgentExchange{
		Agent:        a.profile,
		UserMessage:  domain.AgentMessage{ID: newID("usr", now), Role: "user", Content: normalized, CreatedAt: now},
		Reply:        domain.AgentMessage{ID: newID("asst", now), Role: "assistant", Content: response.Content, CreatedAt: now, Usage: &usage},
		Model:        response.Model,
		FinishReason: response.FinishReason,
		Usage:        response.Usage,
		Trace:        []string{"Agent accepted user input", "Agent applied its profile and instructions", "Agent called LLMProvider", "Agent returned the model response"},
	}, nil
}

func newID(prefix string, moment time.Time) string {
	return prefix + "_" + moment.Format("20060102T150405.000000000")
}
