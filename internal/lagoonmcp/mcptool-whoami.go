package lagoonmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/uselagoon/machinery/api/lagoon"
)

func mcpToolWhoAmI(lagoonMCPServer *LagoonMCPServer) {

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

}

func init() {
	lMCPRegistratorRegistry = append(lMCPRegistratorRegistry, mcpToolWhoAmI)
}
