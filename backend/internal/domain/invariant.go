package domain

import "time"

type InvariantCategory string

const (
	InvariantCategoryArchitecture InvariantCategory = "architecture"
	InvariantCategoryDecision     InvariantCategory = "technical_decision"
	InvariantCategoryStack        InvariantCategory = "stack"
	InvariantCategoryBusiness     InvariantCategory = "business_rule"
)

type InvariantProtection string

const (
	InvariantProtectionHard     InvariantProtection = "hard_check"
	InvariantProtectionSemantic InvariantProtection = "semantic_guard"
)

type Invariant struct {
	ID             string              `json:"id"`
	TaskID         string              `json:"taskId"`
	Category       InvariantCategory   `json:"category"`
	Title          string              `json:"title"`
	Rule           string              `json:"rule"`
	Rationale      string              `json:"rationale"`
	Protection     InvariantProtection `json:"protection"`
	ForbiddenTerms []string            `json:"forbiddenTerms"`
	CreatedAt      time.Time           `json:"createdAt"`
}

type InvariantVerdict string

const (
	InvariantVerdictAllowed  InvariantVerdict = "allowed"
	InvariantVerdictRejected InvariantVerdict = "rejected"
)

type InvariantAssessment struct {
	InvariantID string `json:"invariantId"`
	Verdict     string `json:"verdict"`
	Note        string `json:"note"`
}

type InvariantConflict struct {
	InvariantID string `json:"invariantId"`
	Title       string `json:"title"`
	Reason      string `json:"reason"`
	DetectedBy  string `json:"detectedBy"`
}

type InvariantExchange struct {
	Request         string                `json:"request"`
	Verdict         InvariantVerdict      `json:"verdict"`
	Explanation     string                `json:"explanation"`
	SafeAlternative string                `json:"safeAlternative,omitempty"`
	Answer          string                `json:"answer,omitempty"`
	Invariants      []Invariant           `json:"invariants"`
	Assessments     []InvariantAssessment `json:"assessments"`
	Conflicts       []InvariantConflict   `json:"conflicts"`
	TaskState       *TaskState            `json:"taskState,omitempty"`
	Model           string                `json:"model,omitempty"`
	FinishReason    string                `json:"finishReason,omitempty"`
	Usage           Usage                 `json:"usage"`
	Trace           []string              `json:"trace"`
}
