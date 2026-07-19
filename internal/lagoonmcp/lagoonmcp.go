// Package lagoonmcp provides the MCP server implementation for lagoon-cli.
package lagoonmcp

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/uselagoon/lagoon-cli/internal/config"
	lclient "github.com/uselagoon/machinery/api/lagoon/client"
)

// LagoonMCPServer wraps an MCP server together with a factory for creating
// authenticated Lagoon API clients.
type LagoonMCPServer struct {
	Server          *server.MCPServer
	NewLagoonClient func() *lclient.Client
}

// NewLagoonMCPServer constructs a LagoonMCPServer from the given Lagoon context
// and CLI version string. The context supplies the GraphQL endpoint, API version,
// and auth token; cliVersion is injected at build time via ldflags.
func NewLagoonMCPServer(lagoonCtx config.Context, cliVersion string) (*LagoonMCPServer, error) {
	token := lagoonCtx.Token

	newClient := func() *lclient.Client {
		return lclient.New(
			lagoonCtx.GraphQL,
			cliVersion,
			lagoonCtx.Version,
			&token,
			false,
		)
	}

	s := server.NewMCPServer(
		"Lagoon CLI MCP Server",
		cliVersion,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	return &LagoonMCPServer{
		Server:          s,
		NewLagoonClient: newClient,
	}, nil
}
