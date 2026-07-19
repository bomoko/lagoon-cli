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
			mcp.NewTool("list_projects",
				mcp.WithDescription("List all Lagoon projects the current user has access to"),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				projects, err := lagoon.ListAllProjects(ctx, s.NewLagoonClient())
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
	})
}
