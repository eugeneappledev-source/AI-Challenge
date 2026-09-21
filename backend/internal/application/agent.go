package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
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
	Trim(ctx context.Context, conversationID, agentID string, keep int) error
	LoadSummary(ctx context.Context, conversationID, agentID string) (domain.ConversationSummary, error)
	SaveSummary(ctx context.Context, summary domain.ConversationSummary) error
	LoadFacts(ctx context.Context, sessionID, agentID string) (map[string]string, error)
	SaveFacts(ctx context.Context, sessionID, agentID string, facts map[string]string, updatedAt time.Time) error
	ClearFacts(ctx context.Context, sessionID, agentID string) error
	LoadWorkingMemory(ctx context.Context, taskID, agentID string) ([]domain.MemoryItem, error)
	SaveWorkingMemory(ctx context.Context, taskID, agentID string, item domain.MemoryItem) error
	ClearWorkingMemory(ctx context.Context, taskID, agentID string) error
	LoadLongTermMemory(ctx context.Context, userID, agentID string) ([]domain.MemoryItem, error)
	SaveLongTermMemory(ctx context.Context, userID, agentID string, item domain.MemoryItem) error
	ClearLongTermMemory(ctx context.Context, userID, agentID string) error
	LoadProfiles(ctx context.Context, userID, agentID string) ([]domain.UserProfile, error)
	SaveProfile(ctx context.Context, agentID string, profile domain.UserProfile) error
}

type Agent struct {
	profile         domain.AgentProfile
	client          ReasoningClient
	maxMessageRunes int
	now             func() time.Time
	store           ConversationStore
	idCounter       atomic.Uint64
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

const (
	compressionBatchMessages = 10
	recentMessagesToKeep     = 6
)

func (a *Agent) RespondWithCompression(ctx context.Context, conversationID, input string) (domain.AgentExchange, error) {
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
	summary, err := a.ensureSummary(ctx, conversation)
	if err != nil {
		return domain.AgentExchange{}, err
	}
	compact := compactHistory(conversation.Messages, summary)
	exchange, err := a.respond(ctx, conversationID, compact, input)
	if err == nil {
		exchange.HistoryCount = len(conversation.Messages) + 2
	}
	return exchange, err
}

func (a *Agent) ContextState(ctx context.Context, conversationID string) (domain.ContextState, error) {
	conversation, err := a.History(ctx, conversationID)
	if err != nil {
		return domain.ContextState{}, err
	}
	summary, err := a.ensureSummary(ctx, conversation)
	if err != nil {
		return domain.ContextState{}, err
	}
	compact := compactHistory(conversation.Messages, summary)
	state := domain.ContextState{
		ConversationID: conversationID, Summary: summary.Content, SummaryCoveredMessages: summary.CoveredMessages,
		RecentMessages: len(conversation.Messages) - summary.CoveredMessages, FullHistoryMessages: len(conversation.Messages),
	}
	for _, message := range conversation.Messages {
		state.FullEstimatedTokens += estimateTokens(message.Content)
	}
	for _, message := range compact {
		state.CompressedEstimatedTokens += estimateTokens(message.Content)
	}
	state.EstimatedSavedTokens = state.FullEstimatedTokens - state.CompressedEstimatedTokens
	state.CompressionActive = summary.CoveredMessages > 0
	return state, nil
}

func (a *Agent) CompareContexts(ctx context.Context, conversationID, question string) (domain.ContextComparison, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return domain.ContextComparison{}, ErrEmptyAgentMessage
	}
	conversation, err := a.History(ctx, conversationID)
	if err != nil {
		return domain.ContextComparison{}, err
	}
	summary, err := a.ensureSummary(ctx, conversation)
	if err != nil {
		return domain.ContextComparison{}, err
	}
	fullMessages := a.modelMessages(conversation.Messages, question)
	compressedMessages := a.modelMessages(compactHistory(conversation.Messages, summary), question)
	temperature := a.profile.Temperature
	fullResponse, err := a.client.Generate(ctx, domain.ModelRequest{Model: a.profile.Model, Messages: fullMessages, Temperature: &temperature, MaxTokens: a.profile.MaxOutputTokens})
	if err != nil {
		return domain.ContextComparison{}, err
	}
	compressedResponse, err := a.client.Generate(ctx, domain.ModelRequest{Model: a.profile.Model, Messages: compressedMessages, Temperature: &temperature, MaxTokens: a.profile.MaxOutputTokens})
	if err != nil {
		return domain.ContextComparison{}, err
	}
	review, err := a.reviewContexts(ctx, question, fullResponse.Content, compressedResponse.Content)
	if err != nil {
		return domain.ContextComparison{}, err
	}
	saved := fullResponse.Usage.PromptTokens - compressedResponse.Usage.PromptTokens
	percent := 0.0
	if fullResponse.Usage.PromptTokens > 0 {
		percent = float64(saved) / float64(fullResponse.Usage.PromptTokens) * 100
	}
	return domain.ContextComparison{
		Question: question, Summary: summary.Content,
		Full:              domain.ContextAnswer{Mode: "full", Answer: fullResponse.Content, Usage: fullResponse.Usage},
		Compressed:        domain.ContextAnswer{Mode: "compressed", Answer: compressedResponse.Content, Usage: compressedResponse.Usage},
		PromptTokensSaved: saved, SavingsPercent: percent, Review: review,
	}, nil
}

func (a *Agent) ensureSummary(ctx context.Context, conversation domain.AgentConversation) (domain.ConversationSummary, error) {
	summary, err := a.store.LoadSummary(ctx, conversation.ID, a.profile.ID)
	if err != nil {
		return domain.ConversationSummary{}, err
	}
	cutoff := len(conversation.Messages) - recentMessagesToKeep
	if cutoff < 0 {
		cutoff = 0
	}
	firstCompressionPending := summary.CoveredMessages == 0 && cutoff >= compressionBatchMessages
	batchUpdatePending := summary.CoveredMessages > 0 && cutoff-summary.CoveredMessages >= compressionBatchMessages
	if !firstCompressionPending && !batchUpdatePending {
		return summary, nil
	}
	chunk := conversation.Messages[summary.CoveredMessages:cutoff]
	var builder strings.Builder
	if summary.Content != "" {
		builder.WriteString("Предыдущая сводка:\n" + summary.Content + "\n\n")
	}
	builder.WriteString("Новые сообщения для сжатия:\n")
	for _, message := range chunk {
		builder.WriteString(message.Role + ": " + message.Content + "\n")
	}
	temperature := 0.1
	response, err := a.client.Generate(ctx, domain.ModelRequest{
		Model:        a.profile.Model,
		SystemPrompt: "Сожми историю диалога в компактную фактологическую сводку не длиннее 120 слов. Сохрани имена, предпочтения, решения, ограничения и незавершённые задачи. Удали повторы и служебные ответы. Не добавляй новых фактов.",
		UserPrompt:   builder.String(), Temperature: &temperature, MaxTokens: 300,
	})
	if err != nil {
		return domain.ConversationSummary{}, err
	}
	summary = domain.ConversationSummary{
		ConversationID: conversation.ID, AgentID: a.profile.ID, Content: strings.TrimSpace(response.Content),
		CoveredMessages: cutoff, UpdatedAt: a.now().UTC(),
	}
	if err := a.store.SaveSummary(ctx, summary); err != nil {
		return domain.ConversationSummary{}, err
	}
	return summary, nil
}

func compactHistory(messages []domain.AgentMessage, summary domain.ConversationSummary) []domain.AgentMessage {
	result := make([]domain.AgentMessage, 0, len(messages)-summary.CoveredMessages+1)
	if summary.Content != "" {
		result = append(result, domain.AgentMessage{Role: "system", Content: "Сводка предыдущего разговора:\n" + summary.Content})
	}
	if summary.CoveredMessages < len(messages) {
		result = append(result, messages[summary.CoveredMessages:]...)
	}
	return result
}

func (a *Agent) modelMessages(history []domain.AgentMessage, question string) []domain.ModelMessage {
	result := []domain.ModelMessage{{Role: "system", Content: a.profile.Instructions}}
	for _, message := range history {
		result = append(result, domain.ModelMessage{Role: message.Role, Content: message.Content})
	}
	return append(result, domain.ModelMessage{Role: "user", Content: question})
}

func (a *Agent) reviewContexts(ctx context.Context, question, full, compressed string) (domain.ContextReview, error) {
	payload, _ := json.Marshal(map[string]string{"question": question, "fullAnswer": full, "compressedAnswer": compressed})
	temperature := 0.0
	response, err := a.client.Generate(ctx, domain.ModelRequest{
		Model: a.profile.Model, JSON: true, Temperature: &temperature, MaxTokens: 500,
		SystemPrompt: `Ты независимый рецензент управления контекстом. Сравни два ответа вслепую. Верни только JSON: {"qualityPreserved":boolean,"verdict":"короткий вывод","differences":["..."],"recommendation":"..."}.`,
		UserPrompt:   string(payload),
	})
	if err != nil {
		return domain.ContextReview{}, err
	}
	var review domain.ContextReview
	if err := json.Unmarshal([]byte(response.Content), &review); err != nil {
		return domain.ContextReview{}, fmt.Errorf("decode context review: %w", err)
	}
	return review, nil
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

const agentContextWindowTokens = 1_000_000

func (a *Agent) TokenMetrics(ctx context.Context, conversationID string) (domain.AgentTokenMetrics, error) {
	conversation, err := a.History(ctx, conversationID)
	if err != nil {
		return domain.AgentTokenMetrics{}, err
	}
	metrics := domain.AgentTokenMetrics{
		ConversationID: conversationID, Model: a.profile.Model, MessageCount: len(conversation.Messages),
		ContextWindowTokens: agentContextWindowTokens,
	}
	for _, message := range conversation.Messages {
		metrics.HistoryEstimatedTokens += estimateTokens(message.Content)
		if message.Role == "user" {
			metrics.CurrentMessageEstimatedTokens = estimateTokens(message.Content)
		}
		if message.Usage != nil {
			usage := message.Usage
			metrics.LastContextPromptTokens = usage.PromptTokens
			metrics.LastResponseTokens = usage.CompletionTokens
			metrics.CumulativePromptTokens += usage.PromptTokens
			metrics.CumulativeResponseTokens += usage.CompletionTokens
			metrics.CumulativeTotalTokens += usage.TotalTokens
			cacheMiss := usage.PromptCacheMissTokens
			if cacheMiss == 0 && usage.PromptCacheHitTokens == 0 {
				cacheMiss = usage.PromptTokens
			}
			metrics.EstimatedCostUSD += float64(usage.PromptCacheHitTokens)*0.006/1_000_000 + float64(cacheMiss)*0.30/1_000_000 + float64(usage.CompletionTokens)*1.20/1_000_000
		}
	}
	metrics.EstimatedRemainingTokens = agentContextWindowTokens - metrics.HistoryEstimatedTokens
	if metrics.EstimatedRemainingTokens < 0 {
		metrics.EstimatedRemainingTokens = 0
	}
	metrics.Scenarios = []domain.TokenScenario{
		{ID: "short", Title: "Короткий диалог", MessageCount: 2, EstimatedTokens: 120, Accepted: true, Outcome: "Контекст мал: запрос быстрый и дешёвый."},
		{ID: "long", Title: "Длинный диалог", MessageCount: 40, EstimatedTokens: 12_000, Accepted: true, Outcome: "Каждый следующий запрос повторно оплачивает растущую историю."},
		{ID: "overflow", Title: "Переполнение", MessageCount: 3200, EstimatedTokens: agentContextWindowTokens + 1, Accepted: false, Outcome: "Заблокировано до API: context_length_exceeded."},
	}
	return metrics, nil
}

func estimateTokens(value string) int {
	runes := utf8.RuneCountInString(value)
	if runes == 0 {
		return 0
	}
	return (runes + 2) / 3
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
	userMessage := domain.AgentMessage{ID: a.nextMessageID("usr", now), Role: "user", Content: normalized, CreatedAt: now}
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
	reply := domain.AgentMessage{ID: a.nextMessageID("asst", now.Add(time.Nanosecond)), Role: "assistant", Content: response.Content, CreatedAt: now, Usage: &usage}
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

func (a *Agent) nextMessageID(prefix string, moment time.Time) string {
	return fmt.Sprintf("%s_%d", newID(prefix, moment), a.idCounter.Add(1))
}
