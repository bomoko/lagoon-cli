package cmd

import (
	"log"

	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/uselagoon/lagoon-cli/internal/lagoonmcp"
)

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

		lagoonMCPServer, err := lagoonmcp.NewLagoonMCPServer(lagoonCLIConfig.Lagoons[current], lagoonCLIVersion)
		if err != nil {
			log.Fatal(err.Error())
		}

		// All tools are registered automatically via init() functions in
		// internal/lagoonmcp/tools_*.go — nothing to wire here.

		httpAddr, err := cmd.Flags().GetString("http")
		if err != nil {
			return err
		}

		// For local dev/testing - use 'npx @modelcontextprotocol/inspector@latest' once server is running
		if httpAddr != "" {
			log.Printf("Starting on http://localhost%s/mcp", httpAddr)
			return server.NewStreamableHTTPServer(lagoonMCPServer.Server).Start(httpAddr)
		}

		// Start the STDIO server — all MCP traffic flows over stdin/stdout.
		// Nothing else should write to stdout once ServeStdio is called.
		return server.ServeStdio(lagoonMCPServer.Server)
	},
}

func init() {
	// mcpCmd is registered in root.go's init() via rootCmd.AddCommand(mcpCmd).
	mcpCmd.Flags().String("http", "", "Serve HTTP on the provided address instead of stdio")
}
