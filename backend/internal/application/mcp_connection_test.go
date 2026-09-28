package application_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/application"
	"github.com/eugeneappledev-source/AI-Challenge/backend/internal/infrastructure/mcpserver"
)

func TestMCPConnectionServiceInspectsServerAndTools(t *testing.T) {
	server := httptest.NewServer(mcpserver.StreamableHandler(mcpserver.NewResearchServer()))
	defer server.Close()

	result, err := application.NewMCPConnectionService(server.URL).Inspect(context.Background())
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if !result.Connected || result.ServerName != "startup-research-mcp" {
		t.Fatalf("unexpected connection result: %+v", result)
	}
	if result.ProtocolVersion == "" || len(result.Tools) != 1 || result.Tools[0].Name != "describe_business_radar" {
		t.Fatalf("unexpected MCP discovery: %+v", result)
	}
}
