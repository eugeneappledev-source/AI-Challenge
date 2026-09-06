package domain

type ReasoningMethod string

const (
	ReasoningMethodDirect      ReasoningMethod = "direct"
	ReasoningMethodStepByStep  ReasoningMethod = "step_by_step"
	ReasoningMethodMetaPrompt  ReasoningMethod = "meta_prompt"
	ReasoningMethodExpertPanel ReasoningMethod = "expert_panel"
)

func (m ReasoningMethod) IsValid() bool {
	return m == ReasoningMethodDirect ||
		m == ReasoningMethodStepByStep ||
		m == ReasoningMethodMetaPrompt ||
		m == ReasoningMethodExpertPanel
}

type ModelRequest struct {
	SystemPrompt string
	UserPrompt   string
	JSON         bool
	MaxTokens    int
	Temperature  *float64
}

type ModelResponse struct {
	Content      string
	Model        string
	FinishReason string
	Usage        Usage
}

type ExpertSolution struct {
	Role   string `json:"role"`
	Answer string `json:"answer"`
}

type ReasoningAttempt struct {
	Method          ReasoningMethod  `json:"method"`
	Answer          string           `json:"answer"`
	GeneratedPrompt string           `json:"generatedPrompt,omitempty"`
	Experts         []ExpertSolution `json:"experts"`
	Model           string           `json:"model"`
	FinishReason    string           `json:"finishReason"`
	Usage           Usage            `json:"usage"`
}

type ReasoningScore struct {
	Method       ReasoningMethod `json:"method"`
	Correctness  int             `json:"correctness"`
	Clarity      int             `json:"clarity"`
	Verification int             `json:"verification"`
	Feedback     string          `json:"feedback"`
}

type ReasoningReview struct {
	Winner          ReasoningMethod  `json:"winner"`
	Verdict         string           `json:"verdict"`
	ReferenceAnswer string           `json:"referenceAnswer"`
	Differences     []string         `json:"differences"`
	Scores          []ReasoningScore `json:"scores"`
	Model           string           `json:"model"`
	Usage           Usage            `json:"usage"`
}
