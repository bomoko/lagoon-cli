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
//
// All tools registered via toolRegistrations (see tools_*.go) are added to the
// server before it is returned.
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

	ls := &LagoonMCPServer{
		Server:          s,
		NewLagoonClient: newClient,
	}

	for _, register := range toolRegistrations {
		register(ls)
	}

	return ls, nil
}

// LagoonMCPRegistrator is a function that registers one or more tools (or
// resources/prompts) with a LagoonMCPServer. Each tools_*.go file in this
// package should declare an init() that appends a LagoonMCPRegistrator to
// toolRegistrations — that is the only change required to add new tools.
type LagoonMCPRegistrator func(*LagoonMCPServer)

// toolRegistrations holds all tool registrators collected via init() functions
// in this package. Add a new tool by creating a tools_<domain>.go file and
// appending to this slice in its init(). No other file needs to change.
var toolRegistrations []LagoonMCPRegistrator
