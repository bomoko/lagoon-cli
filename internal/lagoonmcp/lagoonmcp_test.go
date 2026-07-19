package lagoonmcp

import (
	"context"
	"testing"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uselagoon/lagoon-cli/internal/config"
)

// TestRegistrationMechanism verifies that the toolRegistrations registry works
// end-to-end: a registrator appended before NewLagoonMCPServer is called must
// result in a reachable, executable tool.
func TestRegistrationMechanism(t *testing.T) {
	toolRegistrations = append(toolRegistrations, func(s *LagoonMCPServer) {
		s.Server.AddTool(
			mcp.NewTool("test_ping", mcp.WithDescription("test tool")),
			func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return mcp.NewToolResultText("pong"), nil
			},
		)
	})

	ls, err := NewLagoonMCPServer(config.Context{
		GraphQL: "http://fake.lagoon.invalid/graphql",
		Token:   "fake-token",
		Version: "1.0.0",
	}, "test")
	require.NoError(t, err)

	c, err := mcpclient.NewInProcessClient(ls.Server)
	require.NoError(t, err)
	require.NoError(t, c.Start(context.Background()))

	var initReq mcp.InitializeRequest
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	_, err = c.Initialize(context.Background(), initReq)
	require.NoError(t, err)

	result, err := c.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "test_ping"},
	})
	require.NoError(t, err)
	require.False(t, result.IsError, "expected tool to execute without error")

	require.Len(t, result.Content, 1)
	text, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok, "expected TextContent")
	assert.Equal(t, "pong", text.Text)
}
