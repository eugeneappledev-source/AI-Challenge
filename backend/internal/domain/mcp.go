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

type ScheduledDigest struct {
	ID        string                 `json:"id"`
	Trigger   string                 `json:"trigger"`
	Query     string                 `json:"query"`
	Result    BusinessResearchResult `json:"result"`
	CreatedAt time.Time              `json:"createdAt"`
}

type RadarSchedule struct {
	Cron       string     `json:"cron"`
	Timezone   string     `json:"timezone"`
	Enabled    bool       `json:"enabled"`
	NextRunAt  time.Time  `json:"nextRunAt"`
	LastRunAt  *time.Time `json:"lastRunAt,omitempty"`
	LastStatus string     `json:"lastStatus"`
}

type DigestDashboard struct {
	Schedule RadarSchedule     `json:"schedule"`
	Digests  []ScheduledDigest `json:"digests"`
}

type BusinessBrief struct {
	Summary      string   `json:"summary"`
	KeySignals   []string `json:"keySignals"`
	Risks        []string `json:"risks"`
	NextQuestion string   `json:"nextQuestion"`
	Model        string   `json:"model"`
	Usage        Usage    `json:"usage"`
}

type RadarReport struct {
	ID          string        `json:"id"`
	Query       string        `json:"query"`
	Brief       BusinessBrief `json:"brief"`
	SourceCount int           `json:"sourceCount"`
	CreatedAt   time.Time     `json:"createdAt"`
}

type PipelinePlan struct {
	Tools     []string `json:"tools"`
	Rationale string   `json:"rationale"`
}

type PipelineStage struct {
	Order         int    `json:"order"`
	Server        string `json:"server"`
	Tool          string `json:"tool"`
	InputSummary  string `json:"inputSummary"`
	OutputSummary string `json:"outputSummary"`
	DurationMS    int64  `json:"durationMs"`
}

type BusinessPipelineResult struct {
	Request        string              `json:"request"`
	Plan           PipelinePlan        `json:"plan"`
	Stages         []PipelineStage     `json:"stages"`
	Stories        []BusinessStory     `json:"stories"`
	Brief          BusinessBrief       `json:"brief"`
	SavedReport    RadarReport         `json:"savedReport"`
	AvailableTools []MCPToolDescriptor `json:"availableTools"`
	Usage          Usage               `json:"usage"`
	CompletedAt    time.Time           `json:"completedAt"`
}

type FounderProfile struct {
	BudgetUSD    int      `json:"budgetUsd"`
	HoursPerWeek int      `json:"hoursPerWeek"`
	Skills       []string `json:"skills"`
	RiskLevel    string   `json:"riskLevel"`
}

type ScoredOpportunity struct {
	Title     string `json:"title"`
	Evidence  string `json:"evidence"`
	FitScore  int    `json:"fitScore"`
	Effort    string `json:"effort"`
	Rationale string `json:"rationale"`
	Risk      string `json:"risk"`
}

type OpportunityScores struct {
	Items []ScoredOpportunity `json:"items"`
	Model string              `json:"model"`
	Usage Usage               `json:"usage"`
}

type FounderActionPlan struct {
	SelectedTitle  string   `json:"selectedTitle"`
	Goal           string   `json:"goal"`
	Steps          []string `json:"steps"`
	StopConditions []string `json:"stopConditions"`
	BudgetNote     string   `json:"budgetNote"`
	Model          string   `json:"model"`
	Usage          Usage    `json:"usage"`
}

type MCPServerSnapshot struct {
	Name  string              `json:"name"`
	Tools []MCPToolDescriptor `json:"tools"`
}

type MultiServerResult struct {
	Request     string              `json:"request"`
	Profile     FounderProfile      `json:"profile"`
	Route       []PipelineStage     `json:"route"`
	Servers     []MCPServerSnapshot `json:"servers"`
	Stories     []BusinessStory     `json:"stories"`
	Brief       BusinessBrief       `json:"brief"`
	Scores      OpportunityScores   `json:"scores"`
	ActionPlan  FounderActionPlan   `json:"actionPlan"`
	SavedReport RadarReport         `json:"savedReport"`
	Rationale   string              `json:"rationale"`
	Usage       Usage               `json:"usage"`
	CompletedAt time.Time           `json:"completedAt"`
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
