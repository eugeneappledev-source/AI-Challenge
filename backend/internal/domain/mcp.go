package domain

import "time"

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
