package lagoonmcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func longruntest(lagoonMCPServer *LagoonMCPServer) {

	lagoonMCPServer.Server.AddTool(
		mcp.NewTool("longruntest",
			mcp.WithDescription("run this tool when the user requests you to run a long run test"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			for i := 1; i <= 10; i++ {
				err := lagoonMCPServer.Server.SendLogMessageToClient(ctx, mcp.NewLoggingMessageNotification(mcp.LoggingLevelAlert, "notification/progress", fmt.Sprintf("In iteration %v of 10", i)))
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("marshal error: %v", err)), nil
				}
			}

			return mcp.NewToolResultText("All done!"), nil
		},
	)

}

func init() {
	toolRegistrations = append(toolRegistrations, longruntest)
}
