package domain

import "time"

type TaskPhase string

const (
	TaskPhasePlanning   TaskPhase = "planning"
	TaskPhaseExecution  TaskPhase = "execution"
	TaskPhaseValidation TaskPhase = "validation"
	TaskPhaseDone       TaskPhase = "done"
)

type TaskStatus string

const (
	TaskStatusActive TaskStatus = "active"
	TaskStatusPaused TaskStatus = "paused"
)

type TaskAction string

const (
	TaskActionAdvance TaskAction = "advance"
	TaskActionPause   TaskAction = "pause"
	TaskActionResume  TaskAction = "resume"
)

type TaskArtifact struct {
	Phase     TaskPhase `json:"phase"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type TaskTransition struct {
	Action    string    `json:"action"`
	From      TaskPhase `json:"from,omitempty"`
	To        TaskPhase `json:"to"`
	Summary   string    `json:"summary"`
	CreatedAt time.Time `json:"createdAt"`
}

type TaskTransitionAttempt struct {
	From      TaskPhase `json:"from"`
	To        TaskPhase `json:"to"`
	Allowed   bool      `json:"allowed"`
	Code      string    `json:"code"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"createdAt"`
}

type TaskTransitionRule struct {
	From        TaskPhase `json:"from"`
	To          TaskPhase `json:"to"`
	Direction   string    `json:"direction"`
	Requirement string    `json:"requirement"`
}

type TaskLifecycleGraph struct {
	States []TaskPhase          `json:"states"`
	Rules  []TaskTransitionRule `json:"rules"`
}

type TaskState struct {
	ID                   string                  `json:"id"`
	UserID               string                  `json:"userId"`
	ProfileID            string                  `json:"profileId"`
	Goal                 string                  `json:"goal"`
	Phase                TaskPhase               `json:"phase"`
	Status               TaskStatus              `json:"status"`
	CurrentStep          string                  `json:"currentStep"`
	ExpectedAction       string                  `json:"expectedAction"`
	ResumeExpectedAction string                  `json:"resumeExpectedAction,omitempty"`
	Artifacts            []TaskArtifact          `json:"artifacts"`
	Transitions          []TaskTransition        `json:"transitions"`
	Attempts             []TaskTransitionAttempt `json:"attempts"`
	Revision             int                     `json:"revision"`
	CreatedAt            time.Time               `json:"createdAt"`
	UpdatedAt            time.Time               `json:"updatedAt"`
}

type TaskExchange struct {
	State        TaskState `json:"state"`
	Answer       string    `json:"answer"`
	Model        string    `json:"model,omitempty"`
	FinishReason string    `json:"finishReason,omitempty"`
	Usage        Usage     `json:"usage"`
	Trace        []string  `json:"trace"`
}

type ControlledTransitionExchange struct {
	Allowed        bool        `json:"allowed"`
	Code           string      `json:"code"`
	Reason         string      `json:"reason"`
	RequestedFrom  TaskPhase   `json:"requestedFrom"`
	RequestedTo    TaskPhase   `json:"requestedTo"`
	AllowedTargets []TaskPhase `json:"allowedTargets"`
	State          TaskState   `json:"state"`
	Answer         string      `json:"answer,omitempty"`
	Model          string      `json:"model,omitempty"`
	FinishReason   string      `json:"finishReason,omitempty"`
	Usage          Usage       `json:"usage"`
	Trace          []string    `json:"trace"`
}
