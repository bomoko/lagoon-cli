package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/uselagoon/machinery/api/lagoon"
	lclient "github.com/uselagoon/machinery/api/lagoon/client"
)

type LagoonMCPServer struct {
	Server          *server.MCPServer
	NewLagoonClient func() *lclient.Client
}

func NewLagoonMCPServer(currentLagoon, token string) (*LagoonMCPServer, error) {
	var lagoonMcpServer LagoonMCPServer
	// newClient creates a fresh lagoon API client.
	// We create one per tool call so that token refreshes are picked up.
	lagoonMcpServer.NewLagoonClient = func() *lclient.Client {
		return lclient.New(
			lagoonCLIConfig.Lagoons[currentLagoon].GraphQL,
			lagoonCLIVersion,
			lagoonCLIConfig.Lagoons[currentLagoon].Version,
			&token,
			false,
		)
	}

	s := server.NewMCPServer(
		"Lagoon CLI MCP Server",
		lagoonCLIVersion,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)
	lagoonMcpServer.Server = s

	return &lagoonMcpServer, nil
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start a Model Context Protocol (MCP) server for Lagoon",
	Long: `Start a Model Context Protocol (MCP) server for Lagoon.

The server communicates over standard input/output (stdio), making it compatible
with MCP clients such as Claude Desktop, VS Code Copilot, and other LLM tooling.

Configure your MCP client to run:

  lagoon mcp

Example Claude Desktop config (~/.config/claude/claude_desktop_config.json):

  {
    "mcpServers": {
      "lagoon": {
        "command": "lagoon",
        "args": ["mcp"]
      }
    }
  }`,
	PreRunE: func(_ *cobra.Command, _ []string) error {
		return validateTokenE(lagoonCLIConfig.Current)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		current := lagoonCLIConfig.Current
		token := lagoonCLIConfig.Lagoons[current].Token

		lagoonMCPServer, err := NewLagoonMCPServer(current, token)
		if err != nil {
			log.Fatal(err.Error())
		}

		// ------------------------------------------------------------------ //
		// Tool: whoami
		// ------------------------------------------------------------------ //
		lagoonMCPServer.Server.AddTool(
			mcp.NewTool("whoami",
				mcp.WithDescription("Return information about the currently authenticated Lagoon user"),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				user, err := lagoon.Me(ctx, lagoonMCPServer.NewLagoonClient())
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				out, err := json.MarshalIndent(user, "", "  ")
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("marshal error: %v", err)), nil
				}
				return mcp.NewToolResultText(string(out)), nil
			},
		)

		// ------------------------------------------------------------------ //
		// Tool: list_projects
		// ------------------------------------------------------------------ //
		lagoonMCPServer.Server.AddTool(
			mcp.NewTool("list_projects",
				mcp.WithDescription("List all Lagoon projects the current user has access to"),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				projects, err := lagoon.ListAllProjects(ctx, lagoonMCPServer.NewLagoonClient())
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				out, err := json.MarshalIndent(projects, "", "  ")
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("marshal error: %v", err)), nil
				}
				return mcp.NewToolResultText(string(out)), nil
			},
		)

		// ------------------------------------------------------------------ //
		// Tool: list_environments
		// ------------------------------------------------------------------ //
		lagoonMCPServer.Server.AddTool(
			mcp.NewTool("list_environments",
				mcp.WithDescription("List all environments for a given Lagoon project"),
				mcp.WithString("project",
					mcp.Required(),
					mcp.Description("Name of the Lagoon project"),
				),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				projectName, err := req.RequireString("project")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				envs, err := lagoon.GetEnvironmentsByProjectName(ctx, projectName, lagoonMCPServer.NewLagoonClient())
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				out, err := json.MarshalIndent(envs, "", "  ")
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("marshal error: %v", err)), nil
				}
				return mcp.NewToolResultText(string(out)), nil
			},
		)

		// ------------------------------------------------------------------ //
		// Start the STDIO server — all MCP traffic flows over stdin/stdout.
		// Nothing else should write to stdout once ServeStdio is called.
		// ------------------------------------------------------------------ //
		return server.ServeStdio(lagoonMCPServer.Server)
	},
}

func init() {
	// mcpCmd is registered in root.go's init() via rootCmd.AddCommand(mcpCmd).
}
