package lagoonmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/uselagoon/machinery/api/lagoon"
)

func init() {
	toolRegistrations = append(toolRegistrations, func(s *LagoonMCPServer) {
		s.Server.AddTool(
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
				envs, err := lagoon.GetEnvironmentsByProjectName(ctx, projectName, s.NewLagoonClient())
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
	})
}
