package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
)

var ErrInvalidTaskPhase = errors.New("invalid task phase")

var controlledTransitionRules = []domain.TaskTransitionRule{
	{From: domain.TaskPhasePlanning, To: domain.TaskPhaseExecution, Direction: "forward", Requirement: "Сохранён planning-артефакт"},
	{From: domain.TaskPhaseExecution, To: domain.TaskPhasePlanning, Direction: "rollback", Requirement: "Указана причина возврата к плану"},
	{From: domain.TaskPhaseExecution, To: domain.TaskPhaseValidation, Direction: "forward", Requirement: "Сохранён execution-артефакт"},
	{From: domain.TaskPhaseValidation, To: domain.TaskPhaseExecution, Direction: "rollback", Requirement: "Указана причина доработки"},
	{From: domain.TaskPhaseValidation, To: domain.TaskPhaseDone, Direction: "forward", Requirement: "Сохранён validation-артефакт"},
	{From: domain.TaskPhaseDone, To: domain.TaskPhaseValidation, Direction: "rollback", Requirement: "Указана причина повторной проверки"},
}

func (a *Agent) TaskLifecycleGraph() domain.TaskLifecycleGraph {
	return domain.TaskLifecycleGraph{
		States: []domain.TaskPhase{domain.TaskPhasePlanning, domain.TaskPhaseExecution, domain.TaskPhaseValidation, domain.TaskPhaseDone},
		Rules:  append([]domain.TaskTransitionRule(nil), controlledTransitionRules...),
	}
}

func (a *Agent) TransitionTask(ctx context.Context, taskID string, target domain.TaskPhase, reason string) (domain.ControlledTransitionExchange, error) {
	state, err := a.TaskState(ctx, taskID)
	if err != nil {
		return domain.ControlledTransitionExchange{}, err
	}
	if !isKnownTaskPhase(target) {
		return domain.ControlledTransitionExchange{}, ErrInvalidTaskPhase
	}
	reason = strings.TrimSpace(reason)
	from := state.Phase
	allowedTargets := taskAllowedTargets(from)

	if code, rejection := rejectTaskTransition(state, target, reason); rejection != "" {
		now := a.now().UTC()
		state.Attempts = append(state.Attempts, domain.TaskTransitionAttempt{
			From: from, To: target, Allowed: false, Code: code, Reason: rejection, CreatedAt: now,
		})
		state.Revision++
		state.UpdatedAt = now
		if err := a.store.SaveTaskState(ctx, a.profile.ID, state); err != nil {
			return domain.ControlledTransitionExchange{}, err
		}
		return domain.ControlledTransitionExchange{
			Allowed: false, Code: code, Reason: rejection,
			RequestedFrom: from, RequestedTo: target, AllowedTargets: allowedTargets, State: state,
			Trace: []string{
				"Transition request entered the single gateway",
				"Backend checked the explicit transition graph and preconditions",
				"State change and LLM call were blocked",
				"Rejected attempt was persisted for audit",
			},
		}, nil
	}

	exchange, err := a.executeTaskTransition(ctx, state, target, "controlled_transition", reason)
	if err != nil {
		return domain.ControlledTransitionExchange{}, err
	}
	return domain.ControlledTransitionExchange{
		Allowed: true, Code: "transition_applied", Reason: transitionSuccessReason(from, target),
		RequestedFrom: from, RequestedTo: target, AllowedTargets: taskAllowedTargets(target),
		State: exchange.State, Answer: exchange.Answer, Model: exchange.Model,
		FinishReason: exchange.FinishReason, Usage: exchange.Usage,
		Trace: []string{
			"Transition request entered the single gateway",
			"Backend approved the edge and verified its precondition",
			"Agent received only the approved target state",
			"Artifact, transition and audit attempt were persisted atomically",
		},
	}, nil
}

func (a *Agent) executeTaskTransition(ctx context.Context, state domain.TaskState, target domain.TaskPhase, action, reason string) (domain.TaskExchange, error) {
	from := state.Phase
	if code, rejection := rejectTaskTransition(state, target, reason); rejection != "" {
		return domain.TaskExchange{}, fmt.Errorf("%w: %s (%s)", ErrInvalidTaskAction, rejection, code)
	}

	state.Phase = target
	state.CurrentStep, state.ExpectedAction = taskPhaseCopy(target)
	profile, err := a.taskProfile(ctx, state.UserID, state.ProfileID)
	if err != nil {
		return domain.TaskExchange{}, err
	}
	response, err := a.runTaskPhase(ctx, profile, state, controlledPhaseInstruction(from, target, reason))
	if err != nil {
		return domain.TaskExchange{}, err
	}

	now := a.now().UTC()
	state.Revision++
	state.UpdatedAt = now
	state.Artifacts = append(state.Artifacts, domain.TaskArtifact{Phase: target, Title: taskArtifactTitle(target), Content: response.Content, CreatedAt: now})
	state.Transitions = append(state.Transitions, domain.TaskTransition{
		Action: action, From: from, To: target, Summary: transitionSuccessReason(from, target), CreatedAt: now,
	})
	state.Attempts = append(state.Attempts, domain.TaskTransitionAttempt{
		From: from, To: target, Allowed: true, Code: "transition_applied", Reason: fallbackText(reason, "Переход разрешён графом"), CreatedAt: now,
	})
	if err := a.store.SaveTaskState(ctx, a.profile.ID, state); err != nil {
		return domain.TaskExchange{}, err
	}
	return taskExchange(state, response, []string{
		"Transition passed through the explicit graph gateway",
		"Agent received the approved target state and previous artifacts",
		"Phase artifact, transition and audit attempt were persisted",
	}), nil
}

func rejectTaskTransition(state domain.TaskState, target domain.TaskPhase, reason string) (string, string) {
	if state.Status == domain.TaskStatusPaused {
		return "task_paused", "Задача на паузе: сначала возобновите её. Этап сохранён и не изменён."
	}
	if state.Phase == target {
		return "same_state", "Задача уже находится в состоянии " + string(target) + "."
	}
	rule, ok := taskTransitionRule(state.Phase, target)
	if !ok {
		return "transition_not_allowed", fmt.Sprintf("Переход %s → %s отсутствует в разрешённом графе. Доступно: %s.", state.Phase, target, phaseList(taskAllowedTargets(state.Phase)))
	}
	if rule.Direction == "rollback" && strings.TrimSpace(reason) == "" {
		return "rollback_reason_required", "Для безопасного отката укажите причину и ожидаемую доработку."
	}
	if rule.Direction == "forward" && !hasTaskArtifact(state.Artifacts, state.Phase) {
		return "phase_artifact_required", fmt.Sprintf("Нельзя перейти дальше: отсутствует обязательный артефакт этапа %s.", state.Phase)
	}
	return "", ""
}

func taskTransitionRule(from, to domain.TaskPhase) (domain.TaskTransitionRule, bool) {
	for _, rule := range controlledTransitionRules {
		if rule.From == from && rule.To == to {
			return rule, true
		}
	}
	return domain.TaskTransitionRule{}, false
}

func taskAllowedTargets(from domain.TaskPhase) []domain.TaskPhase {
	targets := []domain.TaskPhase{}
	for _, rule := range controlledTransitionRules {
		if rule.From == from {
			targets = append(targets, rule.To)
		}
	}
	return targets
}

func hasTaskArtifact(artifacts []domain.TaskArtifact, phase domain.TaskPhase) bool {
	for _, artifact := range artifacts {
		if artifact.Phase == phase && strings.TrimSpace(artifact.Content) != "" {
			return true
		}
	}
	return false
}

func isKnownTaskPhase(phase domain.TaskPhase) bool {
	switch phase {
	case domain.TaskPhasePlanning, domain.TaskPhaseExecution, domain.TaskPhaseValidation, domain.TaskPhaseDone:
		return true
	default:
		return false
	}
}

func controlledPhaseInstruction(from, target domain.TaskPhase, reason string) string {
	if rule, ok := taskTransitionRule(from, target); ok && rule.Direction == "rollback" {
		return fmt.Sprintf("Задача контролируемо возвращена из %s в %s. Причина: %s. Подготовь конкретный план доработки для повторного прохождения этапа; не переходи дальше самостоятельно.", from, target, reason)
	}
	return taskPhaseInstruction(target)
}

func transitionSuccessReason(from, to domain.TaskPhase) string {
	rule, _ := taskTransitionRule(from, to)
	if rule.Direction == "rollback" {
		return fmt.Sprintf("Контролируемый откат %s → %s выполнен по разрешённому ребру графа", from, to)
	}
	return fmt.Sprintf("Переход %s → %s выполнен после проверки графа и предусловий", from, to)
}

func phaseList(phases []domain.TaskPhase) string {
	if len(phases) == 0 {
		return "нет разрешённых переходов"
	}
	parts := make([]string, len(phases))
	for index, phase := range phases {
		parts[index] = string(phase)
	}
	return strings.Join(parts, ", ")
}
