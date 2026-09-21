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
	ErrTaskScopeRequired = errors.New("task scope is required")
	ErrTaskNotFound      = errors.New("task not found")
	ErrInvalidTaskAction = errors.New("invalid task action")
	ErrTaskAlreadyPaused = errors.New("task is already paused")
	ErrTaskNotPaused     = errors.New("task is not paused")
	ErrTaskDone          = errors.New("task is done")
)

func (a *Agent) CreateTask(ctx context.Context, taskID, userID, profileID, goal string) (domain.TaskExchange, error) {
	taskID, userID, profileID, goal = strings.TrimSpace(taskID), strings.TrimSpace(userID), strings.TrimSpace(profileID), strings.TrimSpace(goal)
	if taskID == "" || userID == "" || profileID == "" {
		return domain.TaskExchange{}, ErrTaskScopeRequired
	}
	if goal == "" {
		return domain.TaskExchange{}, ErrEmptyAgentMessage
	}
	if utf8.RuneCountInString(goal) > a.maxMessageRunes {
		return domain.TaskExchange{}, ErrAgentMessageTooLong
	}
	profile, err := a.taskProfile(ctx, userID, profileID)
	if err != nil {
		return domain.TaskExchange{}, err
	}

	now := a.now().UTC()
	state := domain.TaskState{
		ID: taskID, UserID: userID, ProfileID: profileID, Goal: goal,
		Phase: domain.TaskPhasePlanning, Status: domain.TaskStatusActive,
		CurrentStep:    "Сформировать план и критерии успеха",
		ExpectedAction: "Проверьте план и перейдите к выполнению",
		Artifacts:      []domain.TaskArtifact{}, Transitions: []domain.TaskTransition{},
		Revision: 1, CreatedAt: now, UpdatedAt: now,
	}
	state.Transitions = append(state.Transitions, domain.TaskTransition{
		Action: "create", To: state.Phase, Summary: "Задача создана из исходного запроса", CreatedAt: now,
	})
	response, err := a.runTaskPhase(ctx, profile, state, "Составь план текущего этапа.")
	if err != nil {
		return domain.TaskExchange{}, err
	}
	state.Artifacts = append(state.Artifacts, domain.TaskArtifact{Phase: state.Phase, Title: taskArtifactTitle(state.Phase), Content: response.Content, CreatedAt: now})
	if err := a.store.SaveTaskState(ctx, a.profile.ID, state); err != nil {
		return domain.TaskExchange{}, err
	}
	return taskExchange(state, response, []string{
		"Task was created from the initial user request",
		"State machine entered planning",
		"Profile and memory were attached to the phase request",
		"Planning artifact and task state were persisted in SQLite",
	}), nil
}

func (a *Agent) TaskState(ctx context.Context, taskID string) (domain.TaskState, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return domain.TaskState{}, ErrTaskScopeRequired
	}
	if a.store == nil {
		return domain.TaskState{}, errors.New("agent memory is not configured")
	}
	state, found, err := a.store.LoadTaskState(ctx, taskID, a.profile.ID)
	if err != nil {
		return domain.TaskState{}, err
	}
	if !found {
		return domain.TaskState{}, ErrTaskNotFound
	}
	return state, nil
}

func (a *Agent) ActOnTask(ctx context.Context, taskID string, action domain.TaskAction) (domain.TaskExchange, error) {
	state, err := a.TaskState(ctx, taskID)
	if err != nil {
		return domain.TaskExchange{}, err
	}
	switch action {
	case domain.TaskActionPause:
		return a.pauseTask(ctx, state)
	case domain.TaskActionResume:
		return a.resumeTask(ctx, state)
	case domain.TaskActionAdvance:
		return a.advanceTask(ctx, state)
	default:
		return domain.TaskExchange{}, ErrInvalidTaskAction
	}
}

func (a *Agent) DeleteTask(ctx context.Context, taskID string) error {
	if strings.TrimSpace(taskID) == "" {
		return ErrTaskScopeRequired
	}
	if a.store == nil {
		return errors.New("agent memory is not configured")
	}
	return a.store.DeleteTaskState(ctx, strings.TrimSpace(taskID), a.profile.ID)
}

func (a *Agent) pauseTask(ctx context.Context, state domain.TaskState) (domain.TaskExchange, error) {
	if state.Phase == domain.TaskPhaseDone {
		return domain.TaskExchange{}, ErrTaskDone
	}
	if state.Status == domain.TaskStatusPaused {
		return domain.TaskExchange{}, ErrTaskAlreadyPaused
	}
	now := a.now().UTC()
	state.ResumeExpectedAction = state.ExpectedAction
	state.Status = domain.TaskStatusPaused
	state.ExpectedAction = "Возобновите задачу — повторно объяснять контекст не нужно"
	state.Revision++
	state.UpdatedAt = now
	state.Transitions = append(state.Transitions, domain.TaskTransition{
		Action: "pause", From: state.Phase, To: state.Phase, Summary: "Работа приостановлена; этап и шаг сохранены", CreatedAt: now,
	})
	if err := a.store.SaveTaskState(ctx, a.profile.ID, state); err != nil {
		return domain.TaskExchange{}, err
	}
	return domain.TaskExchange{
		State:  state,
		Answer: fmt.Sprintf("Задача поставлена на паузу на этапе %s. Текущий шаг «%s» сохранён в SQLite.", state.Phase, state.CurrentStep),
		Trace:  []string{"Pause command was handled without an LLM call", "Phase, current step and expected action were persisted"},
	}, nil
}

func (a *Agent) resumeTask(ctx context.Context, state domain.TaskState) (domain.TaskExchange, error) {
	if state.Status != domain.TaskStatusPaused {
		return domain.TaskExchange{}, ErrTaskNotPaused
	}
	profile, err := a.taskProfile(ctx, state.UserID, state.ProfileID)
	if err != nil {
		return domain.TaskExchange{}, err
	}
	response, err := a.runTaskPhase(ctx, profile, state, "Возобнови работу с сохранённого шага. Коротко назови восстановленный контекст и продолжи, не проси пользователя повторять задачу.")
	if err != nil {
		return domain.TaskExchange{}, err
	}
	now := a.now().UTC()
	state.Status = domain.TaskStatusActive
	if state.ResumeExpectedAction != "" {
		state.ExpectedAction = state.ResumeExpectedAction
	}
	state.ResumeExpectedAction = ""
	state.Revision++
	state.UpdatedAt = now
	state.Transitions = append(state.Transitions, domain.TaskTransition{
		Action: "resume", From: state.Phase, To: state.Phase, Summary: "Работа продолжена из сохранённого состояния без повторного описания", CreatedAt: now,
	})
	if err := a.store.SaveTaskState(ctx, a.profile.ID, state); err != nil {
		return domain.TaskExchange{}, err
	}
	return taskExchange(state, response, []string{
		"Task state was loaded from SQLite",
		"Goal, current phase, step and previous artifacts were restored",
		"Agent continued without receiving the original explanation again",
	}), nil
}

func (a *Agent) advanceTask(ctx context.Context, state domain.TaskState) (domain.TaskExchange, error) {
	if state.Status == domain.TaskStatusPaused {
		return domain.TaskExchange{}, ErrTaskAlreadyPaused
	}
	if state.Phase == domain.TaskPhaseDone {
		return domain.TaskExchange{}, ErrTaskDone
	}
	next := nextTaskPhase(state.Phase)
	previous := state.Phase
	state.Phase = next
	state.CurrentStep, state.ExpectedAction = taskPhaseCopy(next)
	profile, err := a.taskProfile(ctx, state.UserID, state.ProfileID)
	if err != nil {
		return domain.TaskExchange{}, err
	}
	response, err := a.runTaskPhase(ctx, profile, state, taskPhaseInstruction(next))
	if err != nil {
		return domain.TaskExchange{}, err
	}
	now := a.now().UTC()
	state.Revision++
	state.UpdatedAt = now
	state.Artifacts = append(state.Artifacts, domain.TaskArtifact{Phase: next, Title: taskArtifactTitle(next), Content: response.Content, CreatedAt: now})
	state.Transitions = append(state.Transitions, domain.TaskTransition{
		Action: "advance", From: previous, To: next, Summary: "Завершён предыдущий этап и сохранён результат нового этапа", CreatedAt: now,
	})
	if err := a.store.SaveTaskState(ctx, a.profile.ID, state); err != nil {
		return domain.TaskExchange{}, err
	}
	return taskExchange(state, response, []string{
		"Backend selected the next state deterministically",
		"Agent received the formal task state and previous artifacts",
		"Phase artifact and updated state were persisted",
	}), nil
}

func (a *Agent) runTaskPhase(ctx context.Context, profile domain.UserProfile, state domain.TaskState, instruction string) (domain.ModelResponse, error) {
	memory, err := a.LayeredMemoryState(ctx, state.UserID+":day13", state.ID, state.UserID)
	if err != nil {
		return domain.ModelResponse{}, err
	}
	var artifacts strings.Builder
	if len(state.Artifacts) == 0 {
		artifacts.WriteString("Артефактов предыдущих этапов пока нет.")
	} else {
		for _, artifact := range state.Artifacts {
			artifacts.WriteString(fmt.Sprintf("\n[%s — %s]\n%s\n", artifact.Phase, artifact.Title, artifact.Content))
		}
	}
	stateBlock := fmt.Sprintf(`Формальное состояние задачи:
task_id: %s
goal: %s
phase: %s
status: %s
current_step: %s
expected_action: %s
revision: %d`, state.ID, state.Goal, state.Phase, state.Status, state.CurrentStep, state.ExpectedAction, state.Revision)
	temperature := 0.2
	return a.client.Generate(ctx, domain.ModelRequest{
		Model: a.profile.Model, Temperature: &temperature, MaxTokens: a.profile.MaxOutputTokens,
		Messages: []domain.ModelMessage{
			{Role: "system", Content: a.profile.Instructions},
			{Role: "system", Content: profileInstruction(profile)},
			{Role: "system", Content: stateBlock},
			{Role: "system", Content: formatMemoryBlock("Долговременная память пользователя", memory.LongTerm)},
			{Role: "system", Content: formatMemoryBlock("Рабочая память задачи", memory.Working)},
			{Role: "system", Content: "Артефакты предыдущих этапов:" + artifacts.String()},
			{Role: "user", Content: instruction},
		},
	})
}

func (a *Agent) taskProfile(ctx context.Context, userID, profileID string) (domain.UserProfile, error) {
	if a.store == nil {
		return domain.UserProfile{}, errors.New("agent memory is not configured")
	}
	profiles, err := a.Profiles(ctx, userID)
	if err != nil {
		return domain.UserProfile{}, err
	}
	profile, ok := findProfile(profiles, profileID)
	if !ok {
		return domain.UserProfile{}, ErrProfileNotFound
	}
	return profile, nil
}

func nextTaskPhase(phase domain.TaskPhase) domain.TaskPhase {
	switch phase {
	case domain.TaskPhasePlanning:
		return domain.TaskPhaseExecution
	case domain.TaskPhaseExecution:
		return domain.TaskPhaseValidation
	default:
		return domain.TaskPhaseDone
	}
}

func taskPhaseCopy(phase domain.TaskPhase) (string, string) {
	switch phase {
	case domain.TaskPhasePlanning:
		return "Сформировать план и критерии успеха", "Проверьте план и перейдите к выполнению"
	case domain.TaskPhaseExecution:
		return "Подготовить архитектурное решение и шаги реализации", "Проверьте реализацию и перейдите к валидации"
	case domain.TaskPhaseValidation:
		return "Проверить решение против цели и критериев успеха", "Подтвердите результат и завершите задачу"
	default:
		return "Зафиксировать итог задачи", "Задача завершена"
	}
}

func taskPhaseInstruction(phase domain.TaskPhase) string {
	switch phase {
	case domain.TaskPhaseExecution:
		return "Выполни утверждённый план: подготовь конкретное архитектурное решение и последовательность реализации. Не возвращайся к сбору требований."
	case domain.TaskPhaseValidation:
		return "Проверь результат execution против исходной цели и критериев плана. Дай чек-лист: что выполнено, риски и что проверить перед приёмкой."
	default:
		return "Собери короткий финальный итог по артефактам всех этапов: результат, принятые решения и следующий практический шаг."
	}
}

func taskArtifactTitle(phase domain.TaskPhase) string {
	switch phase {
	case domain.TaskPhasePlanning:
		return "План и критерии успеха"
	case domain.TaskPhaseExecution:
		return "Архитектурное решение"
	case domain.TaskPhaseValidation:
		return "Результат валидации"
	default:
		return "Финальный итог"
	}
}

func taskExchange(state domain.TaskState, response domain.ModelResponse, trace []string) domain.TaskExchange {
	return domain.TaskExchange{State: state, Answer: response.Content, Model: response.Model, FinishReason: response.FinishReason, Usage: response.Usage, Trace: trace}
}
