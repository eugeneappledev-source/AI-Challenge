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
	Agent        AgentProfile `json:"agent"`
	UserMessage  AgentMessage `json:"userMessage"`
	Reply        AgentMessage `json:"reply"`
	Model        string       `json:"model"`
	FinishReason string       `json:"finishReason"`
	Usage        Usage        `json:"usage"`
	Trace        []string     `json:"trace"`
}
