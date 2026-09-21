package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

var (
	ErrInvalidMemoryLayer  = errors.New("invalid memory layer")
	ErrMemoryScopeRequired = errors.New("memory scope is required")
)

const layeredShortTermLimit = 8

var memoryKeySanitizer = regexp.MustCompile(`[^a-z0-9_]+`)

type memoryRoutePayload struct {
	Layer  domain.MemoryLayer `json:"layer"`
	Key    string             `json:"key"`
	Value  string             `json:"value"`
	Reason string             `json:"reason"`
}

func (a *Agent) RespondWithLayeredMemory(ctx context.Context, sessionID, taskID, userID string, requested domain.MemoryLayer, input string) (domain.LayeredMemoryExchange, error) {
	normalized, err := a.validateLayeredMemoryInput(sessionID, taskID, userID, requested, input)
	if err != nil {
		return domain.LayeredMemoryExchange{}, err
	}
	if a.store == nil {
		return domain.LayeredMemoryExchange{}, errors.New("agent memory is not configured")
	}

	route, err := a.routeMemory(ctx, requested, normalized)
	if err != nil {
		return domain.LayeredMemoryExchange{}, err
	}
	now := a.now().UTC()
	item := domain.MemoryItem{Key: route.Key, Value: route.Value, Source: normalized, UpdatedAt: now}
	switch route.SelectedLayer {
	case domain.MemoryLayerWorking:
		if err := a.store.SaveWorkingMemory(ctx, taskID, a.profile.ID, item); err != nil {
			return domain.LayeredMemoryExchange{}, err
		}
	case domain.MemoryLayerLongTerm:
		if err := a.store.SaveLongTermMemory(ctx, userID, a.profile.ID, item); err != nil {
			return domain.LayeredMemoryExchange{}, err
		}
	}

	state, err := a.LayeredMemoryState(ctx, sessionID, taskID, userID)
	if err != nil {
		return domain.LayeredMemoryExchange{}, err
	}
	messages, preview := a.layeredMemoryMessages(state, normalized)
	temperature := a.profile.Temperature
	response, err := a.client.Generate(ctx, domain.ModelRequest{
		Model: a.profile.Model, Messages: messages, Temperature: &temperature, MaxTokens: a.profile.MaxOutputTokens,
	})
	if err != nil {
		return domain.LayeredMemoryExchange{}, err
	}

	conversationID := layeredConversationID(sessionID)
	userMessage := domain.AgentMessage{ID: a.nextMessageID("usr", now), Role: "user", Content: normalized, CreatedAt: now}
	usage := response.Usage
	reply := domain.AgentMessage{ID: a.nextMessageID("asst", now.Add(1)), Role: "assistant", Content: response.Content, CreatedAt: now, Usage: &usage}
	if err := a.store.Append(ctx, conversationID, a.profile.ID, userMessage, reply); err != nil {
		return domain.LayeredMemoryExchange{}, err
	}
	state, err = a.LayeredMemoryState(ctx, sessionID, taskID, userID)
	if err != nil {
		return domain.LayeredMemoryExchange{}, err
	}

	return domain.LayeredMemoryExchange{
		Agent: a.profile, Message: normalized, Answer: response.Content, Model: response.Model,
		FinishReason: response.FinishReason, Usage: response.Usage, Route: route, State: state,
		ContextPreview: preview,
		Trace: []string{
			"Agent classified the message by memory lifetime",
			"Agent stored the message in the selected memory layer",
			"Context adapter assembled long-term, working and short-term memory",
			"Agent called LLMProvider and persisted the exchange",
		},
	}, nil
}

func (a *Agent) LayeredMemoryState(ctx context.Context, sessionID, taskID, userID string) (domain.LayeredMemoryState, error) {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(taskID) == "" || strings.TrimSpace(userID) == "" {
		return domain.LayeredMemoryState{}, ErrMemoryScopeRequired
	}
	if a.store == nil {
		return domain.LayeredMemoryState{}, errors.New("agent memory is not configured")
	}
	conversation, err := a.store.Load(ctx, layeredConversationID(sessionID), a.profile.ID)
	if err != nil {
		return domain.LayeredMemoryState{}, err
	}
	working, err := a.store.LoadWorkingMemory(ctx, taskID, a.profile.ID)
	if err != nil {
		return domain.LayeredMemoryState{}, err
	}
	longTerm, err := a.store.LoadLongTermMemory(ctx, userID, a.profile.ID)
	if err != nil {
		return domain.LayeredMemoryState{}, err
	}
	shortTerm := conversation.Messages
	if len(shortTerm) > layeredShortTermLimit {
		shortTerm = shortTerm[len(shortTerm)-layeredShortTermLimit:]
	}
	return domain.LayeredMemoryState{
		SessionID: sessionID, TaskID: taskID, UserID: userID, ShortTerm: shortTerm,
		Working: working, LongTerm: longTerm, UpdatedAt: a.now().UTC(),
	}, nil
}

func (a *Agent) ClearLayeredMemory(ctx context.Context, sessionID, taskID, userID string) error {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(taskID) == "" || strings.TrimSpace(userID) == "" {
		return ErrMemoryScopeRequired
	}
	if a.store == nil {
		return errors.New("agent memory is not configured")
	}
	if err := a.store.Clear(ctx, layeredConversationID(sessionID), a.profile.ID); err != nil {
		return err
	}
	if err := a.store.ClearWorkingMemory(ctx, taskID, a.profile.ID); err != nil {
		return err
	}
	return a.store.ClearLongTermMemory(ctx, userID, a.profile.ID)
}

func (a *Agent) validateLayeredMemoryInput(sessionID, taskID, userID string, requested domain.MemoryLayer, input string) (string, error) {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(taskID) == "" || strings.TrimSpace(userID) == "" {
		return "", ErrMemoryScopeRequired
	}
	if requested == "" {
		requested = domain.MemoryLayerAuto
	}
	if !isMemoryLayer(requested, true) {
		return "", ErrInvalidMemoryLayer
	}
	normalized := strings.TrimSpace(input)
	if normalized == "" {
		return "", ErrEmptyAgentMessage
	}
	if utf8.RuneCountInString(normalized) > a.maxMessageRunes {
		return "", ErrAgentMessageTooLong
	}
	return normalized, nil
}

func (a *Agent) routeMemory(ctx context.Context, requested domain.MemoryLayer, input string) (domain.MemoryRoute, error) {
	if requested == "" {
		requested = domain.MemoryLayerAuto
	}
	temperature := 0.0
	requestBody, _ := json.Marshal(map[string]string{"requestedLayer": string(requested), "message": input})
	response, err := a.client.Generate(ctx, domain.ModelRequest{
		Model: a.profile.Model, JSON: true, Temperature: &temperature, MaxTokens: 220,
		SystemPrompt: `Ты маршрутизатор памяти AI-агента. Верни только JSON {"layer":"short_term|working|long_term","key":"snake_case","value":"краткий факт","reason":"краткое объяснение"}.
short_term — текущий вопрос или временная реплика, нужная только в этой беседе.
working — требования, решения, ограничения и цели текущей задачи или проекта.
long_term — устойчивые сведения и предпочтения пользователя, полезные в будущих задачах.
Если requestedLayer не auto, строго используй указанный слой. Не сохраняй секреты, пароли и токены; для них используй short_term и value "Не сохранено: чувствительные данные".`,
		UserPrompt: string(requestBody),
	})
	if err != nil {
		return domain.MemoryRoute{}, err
	}
	var payload memoryRoutePayload
	if err := json.Unmarshal([]byte(response.Content), &payload); err != nil {
		return fallbackMemoryRoute(requested, input, "Маршрутизатор вернул невалидный JSON; применён безопасный fallback."), nil
	}
	if requested != domain.MemoryLayerAuto {
		payload.Layer = requested
	}
	if !isMemoryLayer(payload.Layer, false) {
		payload.Layer = domain.MemoryLayerShortTerm
	}
	key := sanitizeMemoryKey(payload.Key)
	if key == "" {
		key = defaultMemoryKey(payload.Layer)
	}
	value := strings.TrimSpace(payload.Value)
	if value == "" {
		value = input
	}
	reason := strings.TrimSpace(payload.Reason)
	if reason == "" {
		reason = "Слой выбран по сроку жизни информации."
	}
	return domain.MemoryRoute{RequestedLayer: requested, SelectedLayer: payload.Layer, Key: key, Value: value, Reason: reason, Automatic: requested == domain.MemoryLayerAuto}, nil
}

func (a *Agent) layeredMemoryMessages(state domain.LayeredMemoryState, input string) ([]domain.ModelMessage, []string) {
	longBlock := formatMemoryBlock("Долговременная память пользователя", state.LongTerm)
	workingBlock := formatMemoryBlock("Рабочая память текущей задачи", state.Working)
	preview := []string{longBlock, workingBlock, fmt.Sprintf("Краткосрочная память: %d последних сообщений", len(state.ShortTerm))}
	messages := []domain.ModelMessage{
		{Role: "system", Content: a.profile.Instructions},
		{Role: "system", Content: longBlock},
		{Role: "system", Content: workingBlock},
	}
	for _, message := range state.ShortTerm {
		messages = append(messages, domain.ModelMessage{Role: message.Role, Content: message.Content})
	}
	messages = append(messages, domain.ModelMessage{Role: "user", Content: input})
	return messages, preview
}

func formatMemoryBlock(title string, items []domain.MemoryItem) string {
	if len(items) == 0 {
		return title + ": пусто. Не выдумывай отсутствующие сведения."
	}
	var builder strings.Builder
	builder.WriteString(title + ":\n")
	for _, item := range items {
		builder.WriteString("- " + item.Key + ": " + item.Value + "\n")
	}
	return strings.TrimSpace(builder.String())
}

func layeredConversationID(sessionID string) string { return "layered:" + strings.TrimSpace(sessionID) }

func isMemoryLayer(layer domain.MemoryLayer, allowAuto bool) bool {
	return layer == domain.MemoryLayerShortTerm || layer == domain.MemoryLayerWorking || layer == domain.MemoryLayerLongTerm || (allowAuto && layer == domain.MemoryLayerAuto)
}

func sanitizeMemoryKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = memoryKeySanitizer.ReplaceAllString(value, "_")
	return strings.Trim(value, "_")
}

func defaultMemoryKey(layer domain.MemoryLayer) string {
	switch layer {
	case domain.MemoryLayerWorking:
		return "task_context"
	case domain.MemoryLayerLongTerm:
		return "user_preference"
	default:
		return "current_intent"
	}
}

func fallbackMemoryRoute(requested domain.MemoryLayer, input, reason string) domain.MemoryRoute {
	selected := requested
	if selected == "" || selected == domain.MemoryLayerAuto {
		selected = domain.MemoryLayerShortTerm
	}
	return domain.MemoryRoute{RequestedLayer: requested, SelectedLayer: selected, Key: defaultMemoryKey(selected), Value: input, Reason: reason, Automatic: requested == domain.MemoryLayerAuto}
}
