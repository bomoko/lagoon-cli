package lagoonmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

type siteStatusResponse struct {
	Project          string                `json:"project"`
	Environment      string                `json:"environment"`
	Route            string                `json:"route"`
	RouteStatus      *routeStatus          `json:"routeStatus"`
	LatestDeployment *deploymentDetails    `json:"latestDeployment,omitempty"`
	Environments     []resolvedEnvironment `json:"environments"`
}

func init() {
	toolRegistrations = append(toolRegistrations, func(s *LagoonMCPServer) {
		s.Server.AddTool(
			mcp.NewTool("get_site_status",
				mcp.WithDescription("Get the status of a Lagoon site including route health and latest deployment"),
				mcp.WithString("project",
					mcp.Description("Lagoon project name. If provided, path is not required."),
				),
				mcp.WithString("path",
					mcp.Description("Absolute path to the local git repository. Required if project is not provided."),
				),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				projectName := req.GetString("project", "")
				path := req.GetString("path", "")

				if projectName == "" && path == "" {
					return mcp.NewToolResultError("either 'project' or 'path' must be provided"), nil
				}

				var project *resolvedProject
				lc := s.NewLagoonClient()

				if projectName != "" {
					p, err := getProjectByName(ctx, lc, projectName)
					if err != nil {
						return mcp.NewToolResultError(fmt.Sprintf("failed to get project %q: %v", projectName, err)), nil
					}
					project = p
					s.setProjectResource(*p, "")
				} else {
					projects, err := getProjectsByPath(ctx, lc, path)
					if err != nil {
						return mcp.NewToolResultError(err.Error()), nil
					}
					if len(projects) == 0 {
						return mcp.NewToolResultError("no projects found"), nil
					}
					project = &projects[0]
					s.setProjectResource(*project, path)
				}

				// Only checking the prod route - at least as a first pass
				var prodEnv *resolvedEnvironment
				for i := range project.Environments {
					if project.Environments[i].Name == project.ProductionEnvironment {
						prodEnv = &project.Environments[i]
						break
					}
				}

				statusResp := siteStatusResponse{
					Project:      project.Name,
					Environments: project.Environments,
				}

				if prodEnv == nil {
					statusResp.Environment = project.ProductionEnvironment
					statusResp.RouteStatus = &routeStatus{
						Error: "production environment not found",
					}
				} else {
					statusResp.Environment = prodEnv.Name
					statusResp.Route = prodEnv.Route

					if prodEnv.Route != "" {
						statusResp.RouteStatus = pingRoute(prodEnv.Route)
					} else {
						statusResp.RouteStatus = &routeStatus{
							Error: "no route set for prod environment",
						}
					}

					deployment, err := getLatestDeployment(ctx, lc, project.ID, prodEnv.Name)
					if err == nil && deployment != nil {
						statusResp.LatestDeployment = deployment
					} else if err != nil {
						statusResp.LatestDeployment = &deploymentDetails{
							Error: err.Error(),
						}
					}
				}

				out, err := json.MarshalIndent(statusResp, "", "  ")
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("marshal error: %v", err)), nil
				}
				return mcp.NewToolResultText(string(out)), nil
			},
		)
	})
}

func pingRoute(url string) *routeStatus {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Head(url)
	if err != nil {
		return &routeStatus{Error: err.Error()}
	}
	defer resp.Body.Close()

	return &routeStatus{
		StatusCode:    resp.StatusCode,
		XLagoonHeader: resp.Header.Get("x-lagoon"),
	}
}
