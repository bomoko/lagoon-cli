package lagoonmcp

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

type RouteCheckResult struct {
	Route      string `json:"route"`
	CertValid  bool   `json:"certValid"`
	CertError  string `json:"certError,omitempty"`
	HTTPStatus int    `json:"httpStatus,omitempty"`
	HTTPError  string `json:"httpError,omitempty"`
}

func mcpToolVerifyRoute(lagoonMCPServer *LagoonMCPServer) {
	lagoonMCPServer.Server.AddTool(
		mcp.NewTool("verify_route",
			mcp.WithDescription("Verify the TLS certificate for the primary route of a Lagoon project environment"),
			mcp.WithString("project",
				mcp.Required(),
				mcp.Description("Name of the Lagoon project"),
			),
			mcp.WithString("environment",
				mcp.Required(),
				mcp.Description("Name of the environment"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectName, err := req.RequireString("project")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			environmentName, err := req.RequireString("environment")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			lc := lagoonMCPServer.NewLagoonClient()
			project, err := getProjectByName(ctx, lc, projectName)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get project %q: %v", projectName, err)), nil
			}

			var route string
			for _, env := range project.Environments {
				if env.Name == environmentName {
					route = env.Route
					break
				}
			}

			if route == "" {
				return mcp.NewToolResultError(fmt.Sprintf("environment %q in project %q has no primary route", environmentName, projectName)), nil
			}

			result, err := CheckRoute(ctx, route)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("route %q could not be checked: %v", route, err)), nil
			}

			out, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("marshal error: %v", err)), nil
			}
			return mcp.NewToolResultText(string(out)), nil
		},
	)
}

// CheckRoute returns an error only for parse failures; check results are in RouteCheckResult.
func CheckRoute(ctx context.Context, route string) (RouteCheckResult, error) {
	result := RouteCheckResult{Route: route}

	u, err := url.Parse(route)
	if err != nil {
		return result, err
	}

	var port int64
	if u.Port() != "" {
		port, err = strconv.ParseInt(u.Port(), 10, 0)
		if err != nil {
			return result, err
		}
	}

	if err := assertCertGood(ctx, u.Hostname(), int(port)); err != nil {
		result.CertError = err.Error()
	} else {
		result.CertValid = true
	}

	status, err := assertRouteOK(ctx, route)
	if err != nil {
		result.HTTPError = err.Error()
	} else {
		result.HTTPStatus = status
	}

	return result, nil
}

func assertRouteOK(ctx context.Context, route string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, route, nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("route %q returned status %d", route, resp.StatusCode)
	}
	return resp.StatusCode, nil
}

func assertCertGood(ctx context.Context, serverName string, port int) error {
	if port == 0 {
		port = 443
	}
	d := tls.Dialer{Config: &tls.Config{ServerName: serverName}}
	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", serverName, port))
	if err != nil {
		return err
	}
	defer conn.Close()

	state := conn.(*tls.Conn).ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return fmt.Errorf("server %s:%d has no peer certificates", serverName, port)
	}

	return nil
}

func init() {
	toolRegistrations = append(toolRegistrations, mcpToolVerifyRoute)
}
