package application

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type MCPConnectionService struct {
	endpoint string
}

func NewMCPConnectionService(endpoint string) *MCPConnectionService {
	return &MCPConnectionService{endpoint: endpoint}
}

func (s *MCPConnectionService) Inspect(ctx context.Context) (domain.MCPConnectionResult, error) {
	client := mcp.NewClient(&mcp.Implementation{Name: "startup-radar-client", Version: "v1.0.0"}, &mcp.ClientOptions{
		Capabilities: &mcp.ClientCapabilities{},
	})
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: s.endpoint}, nil)
	if err != nil {
		return domain.MCPConnectionResult{}, fmt.Errorf("connect MCP client: %w", err)
	}
	defer session.Close()

	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		return domain.MCPConnectionResult{}, fmt.Errorf("list MCP tools: %w", err)
	}

	result := domain.MCPConnectionResult{
		Connected: true,
		Transport: "Streamable HTTP",
		Endpoint:  s.endpoint,
		ClientName: "startup-radar-client",
		Tools:     make([]domain.MCPToolDescriptor, 0, len(listed.Tools)),
		Trace: []string{
			"Client opened a Streamable HTTP connection",
			"MCP lifecycle negotiation completed",
			"Client requested tools/list",
			"Server returned registered tool schemas",
		},
		CheckedAt: time.Now().UTC(),
	}
	if initialized := session.InitializeResult(); initialized != nil {
		result.ProtocolVersion = initialized.ProtocolVersion
		if initialized.ServerInfo != nil {
			result.ServerName = initialized.ServerInfo.Name
			result.ServerVersion = initialized.ServerInfo.Version
		}
	}
	for _, tool := range listed.Tools {
		schema := map[string]any{}
		if raw, marshalErr := json.Marshal(tool.InputSchema); marshalErr == nil {
			_ = json.Unmarshal(raw, &schema)
		}
		result.Tools = append(result.Tools, domain.MCPToolDescriptor{
			Name: tool.Name, Title: tool.Title, Description: tool.Description, InputSchema: schema,
		})
	}
	return result, nil
}
