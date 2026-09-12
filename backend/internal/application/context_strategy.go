package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

var (
	ErrInvalidContextStrategy = errors.New("invalid context strategy")
	ErrInvalidContextWindow   = errors.New("invalid context window size")
	ErrBranchNotCreated       = errors.New("branch is not created")
	ErrCheckpointRequired     = errors.New("checkpoint requires messages")
)

const (
	defaultStrategyWindowMessages = 6
	minStrategyWindowMessages     = 2
	maxStrategyWindowMessages     = 20
)

var strategyBranches = []struct {
	id    string
	title string
}{
	{id: "main", title: "Основная"},
	{id: "mvp", title: "MVP"},
	{id: "growth", title: "Growth"},
}

func (a *Agent) RespondWithStrategy(ctx context.Context, sessionID string, strategy domain.ContextStrategy, branchID string, windowSize int, input string) (domain.ContextStrategyExchange, error) {
	if err := a.validateStrategyRequest(sessionID, strategy); err != nil {
		return domain.ContextStrategyExchange{}, err
	}
	windowSize, err := normalizeStrategyWindowSize(windowSize)
	if err != nil {
		return domain.ContextStrategyExchange{}, err
	}
	branchID = normalizeBranchID(strategy, branchID)
	conversationID := strategyConversationID(sessionID, strategy, branchID)
	conversation, err := a.store.Load(ctx, conversationID, a.profile.ID)
	if err != nil {
		return domain.ContextStrategyExchange{}, err
	}
	if strategy != domain.ContextStrategyBranching {
		conversation, err = a.trimStrategyConversation(ctx, conversation, windowSize)
		if err != nil {
			return domain.ContextStrategyExchange{}, err
		}
	}
	if strategy == domain.ContextStrategyBranching && branchID != "main" && len(conversation.Messages) == 0 {
		return domain.ContextStrategyExchange{}, ErrBranchNotCreated
	}

	history := conversation.Messages
	operationUsage := domain.Usage{}
	if strategy == domain.ContextStrategyStickyFacts {
		facts, usage, factsErr := a.updateFacts(ctx, sessionID, input)
		if factsErr != nil {
			return domain.ContextStrategyExchange{}, factsErr
		}
		operationUsage = addUsage(operationUsage, usage)
		history = append([]domain.AgentMessage{{Role: "system", Content: factsContext(facts)}}, history...)
	}

	exchange, err := a.respond(ctx, conversationID, history, input)
	if err != nil {
		return domain.ContextStrategyExchange{}, err
	}
	operationUsage = addUsage(operationUsage, exchange.Usage)
	if strategy != domain.ContextStrategyBranching {
		if err := a.store.Trim(ctx, conversationID, a.profile.ID, windowSize); err != nil {
			return domain.ContextStrategyExchange{}, err
		}
	}
	state, err := a.StrategyState(ctx, sessionID, strategy, branchID, windowSize)
	if err != nil {
		return domain.ContextStrategyExchange{}, err
	}
	state.LastUsage = operationUsage
	return domain.ContextStrategyExchange{Strategy: strategy, BranchID: branchID, Exchange: exchange, State: state}, nil
}

func (a *Agent) StrategyState(ctx context.Context, sessionID string, strategy domain.ContextStrategy, branchID string, windowSize int) (domain.ContextStrategyState, error) {
	if err := a.validateStrategyRequest(sessionID, strategy); err != nil {
		return domain.ContextStrategyState{}, err
	}
	windowSize, err := normalizeStrategyWindowSize(windowSize)
	if err != nil {
		return domain.ContextStrategyState{}, err
	}
	branchID = normalizeBranchID(strategy, branchID)
	conversation, err := a.store.Load(ctx, strategyConversationID(sessionID, strategy, branchID), a.profile.ID)
	if err != nil {
		return domain.ContextStrategyState{}, err
	}
	if strategy != domain.ContextStrategyBranching {
		conversation, err = a.trimStrategyConversation(ctx, conversation, windowSize)
		if err != nil {
			return domain.ContextStrategyState{}, err
		}
	}
	state := domain.ContextStrategyState{
		SessionID: sessionID, Strategy: strategy, ActiveBranchID: branchID,
		WindowSize: windowSize, Messages: conversation.Messages,
		Facts: []domain.MemoryFact{}, Branches: []domain.StrategyBranch{},
	}
	if len(conversation.Messages) > 0 {
		last := conversation.Messages[len(conversation.Messages)-1]
		if last.Usage != nil {
			state.LastUsage = *last.Usage
		}
	}
	if strategy == domain.ContextStrategyStickyFacts {
		facts, loadErr := a.store.LoadFacts(ctx, sessionID, a.profile.ID)
		if loadErr != nil {
			return domain.ContextStrategyState{}, loadErr
		}
		state.Facts = sortedFacts(facts)
	}
	if strategy == domain.ContextStrategyBranching {
		branches, branchErr := a.loadBranches(ctx, sessionID)
		if branchErr != nil {
			return domain.ContextStrategyState{}, branchErr
		}
		state.Branches = branches
	}
	return state, nil
}

func (a *Agent) CreateStrategyBranches(ctx context.Context, sessionID string) (domain.ContextStrategyState, error) {
	if strings.TrimSpace(sessionID) == "" {
		return domain.ContextStrategyState{}, ErrConversationIDRequired
	}
	mainID := strategyConversationID(sessionID, domain.ContextStrategyBranching, "main")
	mainConversation, err := a.store.Load(ctx, mainID, a.profile.ID)
	if err != nil {
		return domain.ContextStrategyState{}, err
	}
	if len(mainConversation.Messages) == 0 {
		return domain.ContextStrategyState{}, ErrCheckpointRequired
	}
	for _, branchID := range []string{"mvp", "growth"} {
		targetID := strategyConversationID(sessionID, domain.ContextStrategyBranching, branchID)
		if err := a.store.Clear(ctx, targetID, a.profile.ID); err != nil {
			return domain.ContextStrategyState{}, err
		}
		cloned := cloneMessages(mainConversation.Messages, targetID, a.now().UTC())
		if err := a.store.Append(ctx, targetID, a.profile.ID, cloned...); err != nil {
			return domain.ContextStrategyState{}, err
		}
	}
	return a.StrategyState(ctx, sessionID, domain.ContextStrategyBranching, "mvp", defaultStrategyWindowMessages)
}

func (a *Agent) ClearContextStrategies(ctx context.Context, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return ErrConversationIDRequired
	}
	conversationIDs := []string{
		strategyConversationID(sessionID, domain.ContextStrategySlidingWindow, "main"),
		strategyConversationID(sessionID, domain.ContextStrategyStickyFacts, "main"),
		strategyConversationID(sessionID, domain.ContextStrategyBranching, "main"),
		strategyConversationID(sessionID, domain.ContextStrategyBranching, "mvp"),
		strategyConversationID(sessionID, domain.ContextStrategyBranching, "growth"),
	}
	for _, conversationID := range conversationIDs {
		if err := a.store.Clear(ctx, conversationID, a.profile.ID); err != nil {
			return err
		}
	}
	return a.store.ClearFacts(ctx, sessionID, a.profile.ID)
}

func (a *Agent) CompareContextStrategies(ctx context.Context, sessionID string, windowSize int) (domain.ContextStrategyComparison, error) {
	if strings.TrimSpace(sessionID) == "" {
		return domain.ContextStrategyComparison{}, ErrConversationIDRequired
	}
	windowSize, err := normalizeStrategyWindowSize(windowSize)
	if err != nil {
		return domain.ContextStrategyComparison{}, err
	}
	if err := a.ClearContextStrategies(ctx, sessionID); err != nil {
		return domain.ContextStrategyComparison{}, err
	}
	common := []string{
		"Продукт называется PulsePlan: это iOS-приложение для формирования привычек. Зафиксируй требование одним коротким предложением.",
		"Целевая платформа — iOS 17, регистрация для первого запуска не нужна. Зафиксируй требование одним коротким предложением.",
		"Приложение должно работать offline-first и хранить личные данные только на устройстве. Зафиксируй требование одним коротким предложением.",
		"MVP делает команда из двух человек за шесть недель. Зафиксируй требование одним коротким предложением.",
	}
	mvp := []string{
		"В MVP нужны список привычек, локальные напоминания и недельная статистика. Зафиксируй требование одним коротким предложением.",
		"Интерфейс — спокойная синяя палитра, обязательна поддержка VoiceOver. Зафиксируй требование одним коротким предложением.",
	}
	growth := []string{
		"В Growth-ветке добавь облачную синхронизацию и совместные челленджи. Зафиксируй требование одним коротким предложением.",
		"В Growth-ветке также нужны виджеты и AI-рекомендации. Зафиксируй требование одним коротким предложением.",
	}
	scenario := append(append(append([]string{}, common...), mvp...), "Checkpoint: альтернативная Growth-ветка заменяет два последних требования MVP.")
	question := "Составь короткое финальное ТЗ: название, платформа, ограничения, сроки, функции и UX-требования. Не придумывай отсутствующие данные."

	usageByStrategy := map[domain.ContextStrategy]domain.Usage{}
	for _, strategy := range []domain.ContextStrategy{domain.ContextStrategySlidingWindow, domain.ContextStrategyStickyFacts} {
		for _, prompt := range append(append([]string{}, common...), mvp...) {
			exchange, err := a.RespondWithStrategy(ctx, sessionID, strategy, "", windowSize, prompt)
			if err != nil {
				return domain.ContextStrategyComparison{}, err
			}
			usageByStrategy[strategy] = addUsage(usageByStrategy[strategy], exchange.State.LastUsage)
		}
	}
	for _, prompt := range common {
		exchange, err := a.RespondWithStrategy(ctx, sessionID, domain.ContextStrategyBranching, "main", windowSize, prompt)
		if err != nil {
			return domain.ContextStrategyComparison{}, err
		}
		usageByStrategy[domain.ContextStrategyBranching] = addUsage(usageByStrategy[domain.ContextStrategyBranching], exchange.State.LastUsage)
	}
	if _, err := a.CreateStrategyBranches(ctx, sessionID); err != nil {
		return domain.ContextStrategyComparison{}, err
	}
	for _, item := range []struct {
		branch  string
		prompts []string
	}{{branch: "mvp", prompts: mvp}, {branch: "growth", prompts: growth}} {
		for _, prompt := range item.prompts {
			exchange, err := a.RespondWithStrategy(ctx, sessionID, domain.ContextStrategyBranching, item.branch, windowSize, prompt)
			if err != nil {
				return domain.ContextStrategyComparison{}, err
			}
			usageByStrategy[domain.ContextStrategyBranching] = addUsage(usageByStrategy[domain.ContextStrategyBranching], exchange.State.LastUsage)
		}
	}

	slidingResult, err := a.generateStrategyResult(ctx, sessionID, domain.ContextStrategySlidingWindow, question)
	if err != nil {
		return domain.ContextStrategyComparison{}, err
	}
	slidingResult.Usage = addUsage(usageByStrategy[domain.ContextStrategySlidingWindow], slidingResult.Usage)
	factsResult, err := a.generateStrategyResult(ctx, sessionID, domain.ContextStrategyStickyFacts, question)
	if err != nil {
		return domain.ContextStrategyComparison{}, err
	}
	factsResult.Usage = addUsage(usageByStrategy[domain.ContextStrategyStickyFacts], factsResult.Usage)
	branchingResult, err := a.generateStrategyResult(ctx, sessionID, domain.ContextStrategyBranching, question)
	if err != nil {
		return domain.ContextStrategyComparison{}, err
	}
	branchingResult.Usage = addUsage(usageByStrategy[domain.ContextStrategyBranching], branchingResult.Usage)
	results := []domain.ContextStrategyResult{slidingResult, factsResult, branchingResult}
	review, err := a.reviewStrategies(ctx, question, results)
	if err != nil {
		return domain.ContextStrategyComparison{}, err
	}
	return domain.ContextStrategyComparison{SessionID: sessionID, WindowSize: windowSize, Scenario: scenario, Question: question, Results: results, Review: review}, nil
}

func (a *Agent) generateStrategyResult(ctx context.Context, sessionID string, strategy domain.ContextStrategy, question string) (domain.ContextStrategyResult, error) {
	temperature := a.profile.Temperature
	result := domain.ContextStrategyResult{Strategy: strategy, Branches: []domain.StrategyBranchAnswer{}}
	if strategy == domain.ContextStrategyBranching {
		for _, branch := range strategyBranches[1:] {
			conversation, err := a.store.Load(ctx, strategyConversationID(sessionID, strategy, branch.id), a.profile.ID)
			if err != nil {
				return result, err
			}
			response, err := a.client.Generate(ctx, domain.ModelRequest{Model: a.profile.Model, Messages: a.modelMessages(conversation.Messages, question), Temperature: &temperature, MaxTokens: a.profile.MaxOutputTokens})
			if err != nil {
				return result, err
			}
			result.Branches = append(result.Branches, domain.StrategyBranchAnswer{BranchID: branch.id, Title: branch.title, Answer: response.Content, Usage: response.Usage})
			result.Usage = addUsage(result.Usage, response.Usage)
			result.MessagesKept += len(conversation.Messages)
		}
		result.Answer = "Две независимые версии ТЗ сохранены в ветках MVP и Growth."
		result.Behavior = "Checkpoint сохраняет общее решение, а ветки позволяют безопасно исследовать два взаимоисключающих продолжения."
		return result, nil
	}

	conversation, err := a.store.Load(ctx, strategyConversationID(sessionID, strategy, "main"), a.profile.ID)
	if err != nil {
		return result, err
	}
	history := conversation.Messages
	if strategy == domain.ContextStrategyStickyFacts {
		facts, loadErr := a.store.LoadFacts(ctx, sessionID, a.profile.ID)
		if loadErr != nil {
			return result, loadErr
		}
		history = append([]domain.AgentMessage{{Role: "system", Content: factsContext(facts)}}, history...)
		result.FactsKept = len(facts)
		result.Behavior = "Важные требования переживают выпадение старых сообщений, но обновление facts требует отдельного LLM-вызова."
	} else {
		result.Behavior = "Минимальный и дешёвый контекст, но ранние требования безвозвратно выпадают из окна."
	}
	response, err := a.client.Generate(ctx, domain.ModelRequest{Model: a.profile.Model, Messages: a.modelMessages(history, question), Temperature: &temperature, MaxTokens: a.profile.MaxOutputTokens})
	if err != nil {
		return result, err
	}
	result.Answer, result.Usage, result.MessagesKept = response.Content, response.Usage, len(conversation.Messages)
	return result, nil
}

func (a *Agent) updateFacts(ctx context.Context, sessionID, message string) (map[string]string, domain.Usage, error) {
	facts, err := a.store.LoadFacts(ctx, sessionID, a.profile.ID)
	if err != nil {
		return nil, domain.Usage{}, err
	}
	payload, _ := json.Marshal(map[string]any{"currentFacts": facts, "newUserMessage": strings.TrimSpace(message)})
	temperature := 0.0
	response, err := a.client.Generate(ctx, domain.ModelRequest{
		Model: a.profile.Model, JSON: true, Temperature: &temperature, MaxTokens: 350,
		SystemPrompt: `Ты модуль Sticky Facts. Верни только плоский JSON-объект string:string со ВСЕМИ актуальными важными фактами: goal, constraints, preferences, decisions, agreements и конкретные требования. Сохрани прежние актуальные факты, обнови противоречащие новым сообщением, не записывай приветствия и служебные фразы. Ключи — короткий snake_case.`,
		UserPrompt:   string(payload),
	})
	if err != nil {
		return nil, domain.Usage{}, err
	}
	updated := map[string]string{}
	if err := json.Unmarshal([]byte(response.Content), &updated); err != nil {
		return nil, domain.Usage{}, fmt.Errorf("decode sticky facts: %w", err)
	}
	if err := a.store.SaveFacts(ctx, sessionID, a.profile.ID, updated, a.now().UTC()); err != nil {
		return nil, domain.Usage{}, err
	}
	return updated, response.Usage, nil
}

func (a *Agent) reviewStrategies(ctx context.Context, question string, results []domain.ContextStrategyResult) (domain.ContextStrategyReview, error) {
	payload, _ := json.Marshal(map[string]any{"question": question, "results": results})
	temperature := 0.0
	response, err := a.client.Generate(ctx, domain.ModelRequest{
		Model: a.profile.Model, JSON: true, Temperature: &temperature, MaxTokens: 900,
		SystemPrompt: `Ты независимый рецензент стратегий контекста. Оцени sliding_window, sticky_facts и branching по quality, stability, tokenEfficiency, usability от 1 до 10. Учитывай фактический usage и потерю ранних требований. Верни только JSON: {"winner":"sliding_window|sticky_facts|branching","verdict":"...","differences":["..."],"scores":[{"strategy":"...","quality":1,"stability":1,"tokenEfficiency":1,"usability":1,"feedback":"..."}],"recommendations":["..."]}.`,
		UserPrompt:   string(payload),
	})
	if err != nil {
		return domain.ContextStrategyReview{}, err
	}
	var review domain.ContextStrategyReview
	if err := json.Unmarshal([]byte(response.Content), &review); err != nil {
		return domain.ContextStrategyReview{}, fmt.Errorf("decode context strategy review: %w", err)
	}
	review.Model, review.Usage = response.Model, response.Usage
	return review, nil
}

func (a *Agent) loadBranches(ctx context.Context, sessionID string) ([]domain.StrategyBranch, error) {
	loaded := make([]domain.AgentConversation, 0, len(strategyBranches))
	for _, branch := range strategyBranches {
		conversation, err := a.store.Load(ctx, strategyConversationID(sessionID, domain.ContextStrategyBranching, branch.id), a.profile.ID)
		if err != nil {
			return nil, err
		}
		loaded = append(loaded, conversation)
	}
	checkpoint := commonMessagePrefix(loaded[1].Messages, loaded[2].Messages)
	result := make([]domain.StrategyBranch, 0, len(strategyBranches))
	for index, branch := range strategyBranches {
		if index > 0 && len(loaded[index].Messages) == 0 {
			continue
		}
		branchCheckpoint := 0
		if index > 0 {
			branchCheckpoint = checkpoint
		}
		result = append(result, domain.StrategyBranch{ID: branch.id, Title: branch.title, MessageCount: len(loaded[index].Messages), CheckpointMessages: branchCheckpoint})
	}
	return result, nil
}

func (a *Agent) validateStrategyRequest(sessionID string, strategy domain.ContextStrategy) error {
	if strings.TrimSpace(sessionID) == "" {
		return ErrConversationIDRequired
	}
	if !strategy.IsValid() {
		return ErrInvalidContextStrategy
	}
	if a.store == nil {
		return errors.New("agent memory is not configured")
	}
	return nil
}

func normalizeStrategyWindowSize(windowSize int) (int, error) {
	if windowSize == 0 {
		return defaultStrategyWindowMessages, nil
	}
	if windowSize < minStrategyWindowMessages || windowSize > maxStrategyWindowMessages {
		return 0, ErrInvalidContextWindow
	}
	return windowSize, nil
}

func (a *Agent) trimStrategyConversation(ctx context.Context, conversation domain.AgentConversation, windowSize int) (domain.AgentConversation, error) {
	if len(conversation.Messages) <= windowSize {
		return conversation, nil
	}
	if err := a.store.Trim(ctx, conversation.ID, a.profile.ID, windowSize); err != nil {
		return domain.AgentConversation{}, err
	}
	conversation.Messages = append([]domain.AgentMessage(nil), conversation.Messages[len(conversation.Messages)-windowSize:]...)
	return conversation, nil
}

func normalizeBranchID(strategy domain.ContextStrategy, branchID string) string {
	if strategy != domain.ContextStrategyBranching || strings.TrimSpace(branchID) == "" {
		return "main"
	}
	return strings.TrimSpace(branchID)
}

func strategyConversationID(sessionID string, strategy domain.ContextStrategy, branchID string) string {
	return "day10:" + strings.TrimSpace(sessionID) + ":" + string(strategy) + ":" + branchID
}

func factsContext(facts map[string]string) string {
	if len(facts) == 0 {
		return "Sticky facts: {}"
	}
	raw, _ := json.Marshal(facts)
	return "Sticky facts (считай их проверенной памятью пользователя):\n" + string(raw)
}

func sortedFacts(facts map[string]string) []domain.MemoryFact {
	keys := make([]string, 0, len(facts))
	for key := range facts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]domain.MemoryFact, 0, len(keys))
	for _, key := range keys {
		result = append(result, domain.MemoryFact{Key: key, Value: facts[key]})
	}
	return result
}

func cloneMessages(messages []domain.AgentMessage, targetID string, moment time.Time) []domain.AgentMessage {
	result := make([]domain.AgentMessage, 0, len(messages))
	for index, message := range messages {
		copy := message
		copy.ID = fmt.Sprintf("branch_%s_%03d_%d", targetID, index, moment.UnixNano())
		copy.CreatedAt = moment.Add(time.Duration(index) * time.Nanosecond)
		result = append(result, copy)
	}
	return result
}

func commonMessagePrefix(left, right []domain.AgentMessage) int {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for index := 0; index < limit; index++ {
		if left[index].Role != right[index].Role || left[index].Content != right[index].Content {
			return index
		}
	}
	return limit
}
