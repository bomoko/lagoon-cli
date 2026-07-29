package lagoonmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func init() {
	toolRegistrations = append(toolRegistrations, func(s *LagoonMCPServer) {
		s.Server.AddTool(
			mcp.NewTool("resolve_project",
				mcp.WithDescription(
					"Resolve Lagoon project(s) for a local repository. "+
						"Reads all git remotes from the given directory and looks up "+
						"matching projects in Lagoon by git URL.",
				),
				mcp.WithString("path",
					mcp.Required(),
					mcp.Description("Absolute path to the local git repository to inspect"),
				),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				path, err := req.RequireString("path")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}

				lc := s.NewLagoonClient()
				projects, err := getProjectsByPath(ctx, lc, path)
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}

				if len(projects) == 0 {
					return mcp.NewToolResultText("no Lagoon projects found matching git remotes in this directory"), nil
				}

				s.setProjectResource(projects[0], path)

				out, err := json.MarshalIndent(projects, "", "  ")
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("marshal error: %v", err)), nil
				}
				return mcp.NewToolResultText(string(out)), nil
			},
		)
	})
}
