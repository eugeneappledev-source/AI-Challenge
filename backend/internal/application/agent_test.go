package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

type agentModelRecorder struct {
	request domain.ModelRequest
	reply   domain.ModelResponse
}

type agentModelSequence struct {
	requests []domain.ModelRequest
	replies  []domain.ModelResponse
}

func (s *agentModelSequence) Generate(_ context.Context, request domain.ModelRequest) (domain.ModelResponse, error) {
	s.requests = append(s.requests, request)
	reply := s.replies[0]
	s.replies = s.replies[1:]
	return reply, nil
}

type conversationStoreStub struct {
	conversation domain.AgentConversation
	appended     []domain.AgentMessage
	summary      domain.ConversationSummary
	facts        map[string]string
	working      []domain.MemoryItem
	longTerm     []domain.MemoryItem
}

func (s *conversationStoreStub) LoadSummary(_ context.Context, conversationID, agentID string) (domain.ConversationSummary, error) {
	result := s.summary
	result.ConversationID, result.AgentID = conversationID, agentID
	return result, nil
}

func (s *conversationStoreStub) SaveSummary(_ context.Context, summary domain.ConversationSummary) error {
	s.summary = summary
	return nil
}

func (s *conversationStoreStub) Load(_ context.Context, conversationID, agentID string) (domain.AgentConversation, error) {
	result := s.conversation
	result.ID, result.AgentID = conversationID, agentID
	return result, nil
}

func (s *conversationStoreStub) Append(_ context.Context, _, _ string, messages ...domain.AgentMessage) error {
	s.appended = append(s.appended, messages...)
	s.conversation.Messages = append(s.conversation.Messages, messages...)
	return nil
}

func (s *conversationStoreStub) Clear(_ context.Context, _, _ string) error {
	s.conversation.Messages = nil
	return nil
}

func (s *conversationStoreStub) Trim(_ context.Context, _, _ string, keep int) error {
	if keep < len(s.conversation.Messages) {
		s.conversation.Messages = append([]domain.AgentMessage(nil), s.conversation.Messages[len(s.conversation.Messages)-keep:]...)
	}
	return nil
}

func (s *conversationStoreStub) LoadFacts(_ context.Context, _, _ string) (map[string]string, error) {
	result := map[string]string{}
	for key, value := range s.facts {
		result[key] = value
	}
	return result, nil
}

func (s *conversationStoreStub) SaveFacts(_ context.Context, _, _ string, facts map[string]string, _ time.Time) error {
	s.facts = facts
	return nil
}

func (s *conversationStoreStub) ClearFacts(_ context.Context, _, _ string) error {
	s.facts = nil
	return nil
}

func (s *conversationStoreStub) LoadWorkingMemory(_ context.Context, _, _ string) ([]domain.MemoryItem, error) {
	return append([]domain.MemoryItem(nil), s.working...), nil
}

func (s *conversationStoreStub) SaveWorkingMemory(_ context.Context, _, _ string, item domain.MemoryItem) error {
	s.working = upsertMemoryItem(s.working, item)
	return nil
}

func (s *conversationStoreStub) ClearWorkingMemory(_ context.Context, _, _ string) error {
	s.working = nil
	return nil
}

func (s *conversationStoreStub) LoadLongTermMemory(_ context.Context, _, _ string) ([]domain.MemoryItem, error) {
	return append([]domain.MemoryItem(nil), s.longTerm...), nil
}

func (s *conversationStoreStub) SaveLongTermMemory(_ context.Context, _, _ string, item domain.MemoryItem) error {
	s.longTerm = upsertMemoryItem(s.longTerm, item)
	return nil
}

func (s *conversationStoreStub) ClearLongTermMemory(_ context.Context, _, _ string) error {
	s.longTerm = nil
	return nil
}

func upsertMemoryItem(items []domain.MemoryItem, item domain.MemoryItem) []domain.MemoryItem {
	for index := range items {
		if items[index].Key == item.Key {
			items[index] = item
			return items
		}
	}
	return append(items, item)
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

func TestAgentTokenMetricsSeparateEstimatesFromProviderUsage(t *testing.T) {
	store := &conversationStoreStub{conversation: domain.AgentConversation{Messages: []domain.AgentMessage{
		{ID: "u1", Role: "user", Content: "Короткий вопрос"},
		{ID: "a1", Role: "assistant", Content: "Короткий ответ", Usage: &domain.Usage{
			PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120,
			PromptCacheHitTokens: 60, PromptCacheMissTokens: 40,
		}},
	}}}
	agent := NewAgent(domain.AgentProfile{ID: "mentor", Model: "deepseek-flash"}, &agentModelRecorder{}, 4000).WithMemory(store)

	metrics, err := agent.TokenMetrics(context.Background(), "c1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metrics.CurrentMessageEstimatedTokens == 0 || metrics.HistoryEstimatedTokens == 0 {
		t.Fatalf("expected explicit local estimates, got %+v", metrics)
	}
	if metrics.LastContextPromptTokens != 100 || metrics.LastResponseTokens != 20 || metrics.CumulativeTotalTokens != 120 {
		t.Fatalf("expected exact provider usage, got %+v", metrics)
	}
	if metrics.EstimatedCostUSD <= 0 || len(metrics.Scenarios) != 3 || metrics.Scenarios[2].Accepted {
		t.Fatalf("expected cost and blocked overflow scenario, got %+v", metrics)
	}
}

func TestCompressedAgentSummarizesOldBatchAndKeepsRecentMessages(t *testing.T) {
	history := make([]domain.AgentMessage, 16)
	for index := range history {
		role := "user"
		if index%2 == 1 {
			role = "assistant"
		}
		history[index] = domain.AgentMessage{ID: newID(role, time.Unix(int64(index), 0)), Role: role, Content: role + " message"}
	}
	store := &conversationStoreStub{conversation: domain.AgentConversation{Messages: history}}
	client := &agentModelSequence{replies: []domain.ModelResponse{
		{Content: "Пользователь изучает AI-агентов."},
		{Content: "Продолжаем с учётом сводки.", Usage: domain.Usage{PromptTokens: 80, CompletionTokens: 10, TotalTokens: 90}},
	}}
	agent := NewAgent(domain.AgentProfile{ID: "mentor", Model: "deepseek-flash", Instructions: "Помогай"}, client, 4000).WithMemory(store)

	exchange, err := agent.RespondWithCompression(context.Background(), "c1", "Что дальше?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.summary.CoveredMessages != 10 || store.summary.Content == "" {
		t.Fatalf("expected first 10 messages to be summarized, got %+v", store.summary)
	}
	if len(client.requests) != 2 || len(client.requests[1].Messages) != 9 {
		t.Fatalf("expected system + summary + 6 recent + question, got %+v", client.requests)
	}
	if client.requests[1].Messages[1].Role != "system" || !strings.Contains(client.requests[1].Messages[1].Content, "Сводка") {
		t.Fatalf("summary must be a separate context message: %+v", client.requests[1].Messages)
	}
	if exchange.HistoryCount != 18 {
		t.Fatalf("full archive count must remain visible, got %d", exchange.HistoryCount)
	}
}

func TestContextComparisonUsesSameQuestionAndIndependentReviewer(t *testing.T) {
	history := make([]domain.AgentMessage, 16)
	for index := range history {
		history[index] = domain.AgentMessage{Role: "user", Content: "fact"}
	}
	store := &conversationStoreStub{
		conversation: domain.AgentConversation{Messages: history},
		summary:      domain.ConversationSummary{Content: "Краткие факты", CoveredMessages: 10},
	}
	client := &agentModelSequence{replies: []domain.ModelResponse{
		{Content: "Полный ответ", Usage: domain.Usage{PromptTokens: 200, CompletionTokens: 20, TotalTokens: 220}},
		{Content: "Сжатый ответ", Usage: domain.Usage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120}},
		{Content: `{"qualityPreserved":true,"verdict":"Смысл сохранён","differences":[],"recommendation":"Использовать сжатие"}`},
	}}
	agent := NewAgent(domain.AgentProfile{ID: "mentor", Model: "deepseek-flash", Instructions: "Помогай"}, client, 4000).WithMemory(store)

	comparison, err := agent.CompareContexts(context.Background(), "c1", "Что мы решили?")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(client.requests) != 3 || client.requests[0].Messages[len(client.requests[0].Messages)-1].Content != "Что мы решили?" || client.requests[1].Messages[len(client.requests[1].Messages)-1].Content != "Что мы решили?" {
		t.Fatalf("both modes must receive the same question: %+v", client.requests)
	}
	if comparison.PromptTokensSaved != 100 || comparison.SavingsPercent != 50 || !comparison.Review.QualityPreserved {
		t.Fatalf("unexpected comparison: %+v", comparison)
	}
}

func TestLayeredMemoryRoutesTaskFactAndBuildsThreeContextLayers(t *testing.T) {
	client := &agentModelSequence{replies: []domain.ModelResponse{
		{Content: `{"layer":"working","key":"offline_mode","value":"Приложение должно работать офлайн","reason":"Ограничение текущей задачи"}`},
		{Content: "Учту offline-first архитектуру.", Model: "deepseek-flash", FinishReason: "stop", Usage: domain.Usage{TotalTokens: 42}},
	}}
	store := &conversationStoreStub{longTerm: []domain.MemoryItem{{Key: "role", Value: "iOS-разработчик"}}}
	agent := NewAgent(domain.AgentProfile{ID: "mentor", Model: "deepseek-flash", Instructions: "Помогай", MaxOutputTokens: 1000}, client, 4000).WithMemory(store)

	exchange, err := agent.RespondWithLayeredMemory(context.Background(), "s1", "pulse-plan", "u1", domain.MemoryLayerAuto, "Приложение должно работать офлайн")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exchange.Route.SelectedLayer != domain.MemoryLayerWorking || len(store.working) != 1 {
		t.Fatalf("expected task fact in working memory, got route=%+v state=%+v", exchange.Route, store.working)
	}
	if len(client.requests) != 2 || len(client.requests[1].Messages) < 4 {
		t.Fatalf("expected routing call and assembled generation context, got %+v", client.requests)
	}
	if !strings.Contains(client.requests[1].Messages[1].Content, "iOS-разработчик") || !strings.Contains(client.requests[1].Messages[2].Content, "offline") {
		t.Fatalf("expected long-term and working blocks, got %+v", client.requests[1].Messages)
	}
	if len(exchange.State.ShortTerm) != 2 || exchange.Answer == "" {
		t.Fatalf("expected persisted short-term exchange, got %+v", exchange)
	}
}
