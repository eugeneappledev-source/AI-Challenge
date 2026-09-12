package domain

type ContextStrategy string

const (
	ContextStrategySlidingWindow ContextStrategy = "sliding_window"
	ContextStrategyStickyFacts   ContextStrategy = "sticky_facts"
	ContextStrategyBranching     ContextStrategy = "branching"
)

func (s ContextStrategy) IsValid() bool {
	return s == ContextStrategySlidingWindow || s == ContextStrategyStickyFacts || s == ContextStrategyBranching
}

type MemoryFact struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type StrategyBranch struct {
	ID                 string `json:"id"`
	Title              string `json:"title"`
	MessageCount       int    `json:"messageCount"`
	CheckpointMessages int    `json:"checkpointMessages"`
}

type ContextStrategyState struct {
	SessionID      string           `json:"sessionId"`
	Strategy       ContextStrategy  `json:"strategy"`
	ActiveBranchID string           `json:"activeBranchId,omitempty"`
	WindowSize     int              `json:"windowSize"`
	Messages       []AgentMessage   `json:"messages"`
	Facts          []MemoryFact     `json:"facts"`
	Branches       []StrategyBranch `json:"branches"`
	LastUsage      Usage            `json:"lastUsage"`
}

type ContextStrategyExchange struct {
	Strategy ContextStrategy      `json:"strategy"`
	BranchID string               `json:"branchId,omitempty"`
	Exchange AgentExchange        `json:"exchange"`
	State    ContextStrategyState `json:"state"`
}

type StrategyBranchAnswer struct {
	BranchID string `json:"branchId"`
	Title    string `json:"title"`
	Answer   string `json:"answer"`
	Usage    Usage  `json:"usage"`
}

type ContextStrategyResult struct {
	Strategy     ContextStrategy        `json:"strategy"`
	Answer       string                 `json:"answer"`
	Usage        Usage                  `json:"usage"`
	MessagesKept int                    `json:"messagesKept"`
	FactsKept    int                    `json:"factsKept"`
	Branches     []StrategyBranchAnswer `json:"branches"`
	Behavior     string                 `json:"behavior"`
}

type ContextStrategyScore struct {
	Strategy        ContextStrategy `json:"strategy"`
	Quality         int             `json:"quality"`
	Stability       int             `json:"stability"`
	TokenEfficiency int             `json:"tokenEfficiency"`
	Usability       int             `json:"usability"`
	Feedback        string          `json:"feedback"`
}

type ContextStrategyReview struct {
	Winner          ContextStrategy        `json:"winner"`
	Verdict         string                 `json:"verdict"`
	Differences     []string               `json:"differences"`
	Scores          []ContextStrategyScore `json:"scores"`
	Recommendations []string               `json:"recommendations"`
	Model           string                 `json:"model"`
	Usage           Usage                  `json:"usage"`
}

type ContextStrategyComparison struct {
	SessionID  string                  `json:"sessionId"`
	WindowSize int                     `json:"windowSize"`
	Scenario   []string                `json:"scenario"`
	Question   string                  `json:"question"`
	Results    []ContextStrategyResult `json:"results"`
	Review     ContextStrategyReview   `json:"review"`
}
