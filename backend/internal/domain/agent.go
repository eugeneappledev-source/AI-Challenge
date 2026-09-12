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
