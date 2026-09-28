package domain

import "time"

type BusinessStory struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	URL           string    `json:"url"`
	DiscussionURL string    `json:"discussionUrl"`
	Source        string    `json:"source"`
	Author        string    `json:"author"`
	Score         int       `json:"score"`
	PublishedAt   time.Time `json:"publishedAt"`
}

type MCPToolSelection struct {
	Server    string         `json:"server"`
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
	Rationale string         `json:"rationale"`
}

type BusinessOpportunity struct {
	Title     string `json:"title"`
	WhyNow    string `json:"whyNow"`
	FirstStep string `json:"firstStep"`
	Risk      string `json:"risk"`
}

type BusinessAdvice struct {
	Summary       string                `json:"summary"`
	Opportunities []BusinessOpportunity `json:"opportunities"`
	Caveat        string                `json:"caveat"`
}

type BusinessResearchResult struct {
	Request        string              `json:"request"`
	Selection      MCPToolSelection    `json:"selection"`
	Stories        []BusinessStory     `json:"stories"`
	Advice         BusinessAdvice      `json:"advice"`
	Model          string              `json:"model"`
	FinishReason   string              `json:"finishReason"`
	Usage          Usage               `json:"usage"`
	AvailableTools []MCPToolDescriptor `json:"availableTools"`
	Trace          []string            `json:"trace"`
	CompletedAt    time.Time           `json:"completedAt"`
}

type MCPToolDescriptor struct {
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type MCPConnectionResult struct {
	Connected       bool                `json:"connected"`
	Transport       string              `json:"transport"`
	Endpoint        string              `json:"endpoint"`
	ClientName      string              `json:"clientName"`
	ServerName      string              `json:"serverName"`
	ServerVersion   string              `json:"serverVersion"`
	ProtocolVersion string              `json:"protocolVersion"`
	Tools           []MCPToolDescriptor `json:"tools"`
	Trace           []string            `json:"trace"`
	CheckedAt       time.Time           `json:"checkedAt"`
}
