package domain

import "time"

type AgentProfile struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Role            string  `json:"role"`
	Instructions    string  `json:"instructions"`
	Model           string  `json:"model"`
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

type AgentMessage struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	Usage     *Usage    `json:"usage,omitempty"`
}

type AgentExchange struct {
	Agent          AgentProfile `json:"agent"`
	ConversationID string       `json:"conversationId,omitempty"`
	HistoryCount   int          `json:"historyCount,omitempty"`
	UserMessage    AgentMessage `json:"userMessage"`
	Reply          AgentMessage `json:"reply"`
	Model          string       `json:"model"`
	FinishReason   string       `json:"finishReason"`
	Usage          Usage        `json:"usage"`
	Trace          []string     `json:"trace"`
}

type AgentConversation struct {
	ID        string         `json:"id"`
	AgentID   string         `json:"agentId"`
	Messages  []AgentMessage `json:"messages"`
	UpdatedAt *time.Time     `json:"updatedAt,omitempty"`
}

type MemoryLayer string

const (
	MemoryLayerAuto      MemoryLayer = "auto"
	MemoryLayerShortTerm MemoryLayer = "short_term"
	MemoryLayerWorking   MemoryLayer = "working"
	MemoryLayerLongTerm  MemoryLayer = "long_term"
)

type MemoryItem struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Source    string    `json:"source"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type MemoryRoute struct {
	RequestedLayer MemoryLayer `json:"requestedLayer"`
	SelectedLayer  MemoryLayer `json:"selectedLayer"`
	Key            string      `json:"key"`
	Value          string      `json:"value"`
	Reason         string      `json:"reason"`
	Automatic      bool        `json:"automatic"`
}

type LayeredMemoryState struct {
	SessionID string         `json:"sessionId"`
	TaskID    string         `json:"taskId"`
	UserID    string         `json:"userId"`
	ShortTerm []AgentMessage `json:"shortTerm"`
	Working   []MemoryItem   `json:"working"`
	LongTerm  []MemoryItem   `json:"longTerm"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

type LayeredMemoryExchange struct {
	Agent          AgentProfile       `json:"agent"`
	Message        string             `json:"message"`
	Answer         string             `json:"answer"`
	Model          string             `json:"model"`
	FinishReason   string             `json:"finishReason"`
	Usage          Usage              `json:"usage"`
	Route          MemoryRoute        `json:"route"`
	State          LayeredMemoryState `json:"state"`
	ContextPreview []string           `json:"contextPreview"`
	Trace          []string           `json:"trace"`
}

type UserProfile struct {
	ID             string    `json:"id"`
	UserID         string    `json:"userId"`
	Name           string    `json:"name"`
	Address        string    `json:"address"`
	Style          string    `json:"style"`
	ResponseFormat string    `json:"responseFormat"`
	Constraints    []string  `json:"constraints"`
	PipelineID     string    `json:"pipelineId"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type ProfileSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ProfileSkillRun struct {
	Skill  ProfileSkill `json:"skill"`
	Output string       `json:"output"`
	Usage  Usage        `json:"usage"`
}

type PersonalizedExchange struct {
	Profile      UserProfile        `json:"profile"`
	Skills       []ProfileSkillRun  `json:"skills"`
	Message      string             `json:"message"`
	Answer       string             `json:"answer"`
	Model        string             `json:"model"`
	FinishReason string             `json:"finishReason"`
	Usage        Usage              `json:"usage"`
	Route        MemoryRoute        `json:"route"`
	Memory       LayeredMemoryState `json:"memory"`
	Trace        []string           `json:"trace"`
}

type AgentTokenMetrics struct {
	ConversationID                string          `json:"conversationId"`
	Model                         string          `json:"model"`
	MessageCount                  int             `json:"messageCount"`
	CurrentMessageEstimatedTokens int             `json:"currentMessageEstimatedTokens"`
	LastContextPromptTokens       int             `json:"lastContextPromptTokens"`
	LastResponseTokens            int             `json:"lastResponseTokens"`
	HistoryEstimatedTokens        int             `json:"historyEstimatedTokens"`
	CumulativePromptTokens        int             `json:"cumulativePromptTokens"`
	CumulativeResponseTokens      int             `json:"cumulativeResponseTokens"`
	CumulativeTotalTokens         int             `json:"cumulativeTotalTokens"`
	EstimatedCostUSD              float64         `json:"estimatedCostUSD"`
	ContextWindowTokens           int             `json:"contextWindowTokens"`
	EstimatedRemainingTokens      int             `json:"estimatedRemainingTokens"`
	Scenarios                     []TokenScenario `json:"scenarios"`
}

type TokenScenario struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	MessageCount    int    `json:"messageCount"`
	EstimatedTokens int    `json:"estimatedTokens"`
	Accepted        bool   `json:"accepted"`
	Outcome         string `json:"outcome"`
}

type ConversationSummary struct {
	ConversationID  string    `json:"conversationId"`
	AgentID         string    `json:"agentId"`
	Content         string    `json:"content"`
	CoveredMessages int       `json:"coveredMessages"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type ContextState struct {
	ConversationID            string `json:"conversationId"`
	Summary                   string `json:"summary"`
	SummaryCoveredMessages    int    `json:"summaryCoveredMessages"`
	RecentMessages            int    `json:"recentMessages"`
	FullHistoryMessages       int    `json:"fullHistoryMessages"`
	FullEstimatedTokens       int    `json:"fullEstimatedTokens"`
	CompressedEstimatedTokens int    `json:"compressedEstimatedTokens"`
	EstimatedSavedTokens      int    `json:"estimatedSavedTokens"`
	CompressionActive         bool   `json:"compressionActive"`
}

type ContextAnswer struct {
	Mode   string `json:"mode"`
	Answer string `json:"answer"`
	Usage  Usage  `json:"usage"`
}

type ContextReview struct {
	QualityPreserved bool     `json:"qualityPreserved"`
	Verdict          string   `json:"verdict"`
	Differences      []string `json:"differences"`
	Recommendation   string   `json:"recommendation"`
}

type ContextComparison struct {
	Question          string        `json:"question"`
	Summary           string        `json:"summary"`
	Full              ContextAnswer `json:"full"`
	Compressed        ContextAnswer `json:"compressed"`
	PromptTokensSaved int           `json:"promptTokensSaved"`
	SavingsPercent    float64       `json:"savingsPercent"`
	Review            ContextReview `json:"review"`
}
