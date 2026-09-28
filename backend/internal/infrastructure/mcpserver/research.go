package mcpserver

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CapabilitiesInput struct {
	Topic string `json:"topic,omitempty" jsonschema:"optional business topic to describe"`
}

type CapabilitiesOutput struct {
	Server  string   `json:"server" jsonschema:"MCP server name"`
	Purpose string   `json:"purpose" jsonschema:"what this server is responsible for"`
	Sources []string `json:"sources" jsonschema:"trusted external sources used by the server"`
}

func NewResearchServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "startup-research-mcp", Version: "v1.0.0"}, &mcp.ServerOptions{
		Instructions: "Trusted read-only tools for the Startup & Business Radar.",
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "describe_business_radar",
		Title:       "Business Radar capabilities",
		Description: "Describe this trusted MCP server and the business research capabilities it exposes.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}, func(_ context.Context, _ *mcp.CallToolRequest, input CapabilitiesInput) (*mcp.CallToolResult, CapabilitiesOutput, error) {
		purpose := "Inspect startup and business research capabilities"
		if input.Topic != "" {
			purpose += " for " + input.Topic
		}
		return nil, CapabilitiesOutput{
			Server: "startup-research-mcp", Purpose: purpose,
			Sources: []string{"Hacker News official API"},
		}, nil
	})
	return server
}

func StreamableHandler(server *mcp.Server) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{
		JSONResponse: true,
		Stateless:    true,
	})
}
