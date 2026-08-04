package lagoonmcp

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

type RouteCheckResult struct {
	Project           string   `json:"project"`
	Environment       string   `json:"environment"`
	Route             string   `json:"route"`
	CertValid         bool     `json:"certValid"`
	CertExpiry        string   `json:"certExpiry,omitempty"`
	CertDaysRemaining int      `json:"certDaysRemaining,omitempty"`
	CertError         string   `json:"certError,omitempty"`
	HTTPStatus        int      `json:"httpStatus,omitempty"`
	HTTPError         string   `json:"httpError,omitempty"`
	RedirectChain     []string `json:"redirectChain,omitempty"`
	BodySnippet       string   `json:"bodySnippet,omitempty"`
}

func mcpToolVerifyRoute(lagoonMCPServer *LagoonMCPServer) {
	lagoonMCPServer.Server.AddTool(
		mcp.NewTool("verify_route",
			mcp.WithDescription("Verify the TLS certificate and HTTP response for the primary route of a Lagoon project environment"),
			mcp.WithString("project",
				mcp.Required(),
				mcp.Description("Name of the Lagoon project"),
			),
			mcp.WithString("environment",
				mcp.Description("Name of the environment; defaults to the production environment"),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectName, err := req.RequireString("project")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			environmentName := req.GetString("environment", "")

			lc := lagoonMCPServer.NewLagoonClient()
			project, err := getProjectByName(ctx, lc, projectName)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get project %q: %v", projectName, err)), nil
			}

			var foundEnv *resolvedEnvironment
			for i := range project.Environments {
				if environmentName != "" {
					if project.Environments[i].Name == environmentName {
						foundEnv = &project.Environments[i]
						break
					}
				} else {
					if project.Environments[i].EnvironmentType == "production" {
						foundEnv = &project.Environments[i]
						break
					}
				}

			}
			if foundEnv == nil {
				if environmentName == "" {
					return mcp.NewToolResultError(fmt.Sprintf("no production environment found in project %q", projectName)), nil
				}
				return mcp.NewToolResultError(fmt.Sprintf("environment %q not found in project %q", environmentName, projectName)), nil
			}
			if foundEnv.Route == "" {
				return mcp.NewToolResultError(fmt.Sprintf("environment %q in project %q has no primary route set", environmentName, projectName)), nil
			}
			route := foundEnv.Route

			result, err := CheckRoute(ctx, route)
			result.Project = projectName
			result.Environment = foundEnv.Name
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

	// port 0 is handled by assertCertGood (defaults to 443)
	var port int
	if p := u.Port(); p != "" {
		if _, err := fmt.Sscanf(p, "%d", &port); err != nil {
			return result, fmt.Errorf("invalid port %q: %w", p, err)
		}
	}

	expiry, certErr := assertCertGood(ctx, u.Hostname(), port)
	if certErr != nil {
		result.CertError = certErr.Error()
	} else {
		result.CertValid = true
		result.CertExpiry = expiry.UTC().Format(time.RFC3339)
		result.CertDaysRemaining = int(time.Until(expiry).Hours() / 24)
	}

	status, redirectChain, bodySnippet, httpErr := assertRouteOK(ctx, route)
	result.HTTPStatus = status
	result.RedirectChain = redirectChain
	result.BodySnippet = bodySnippet
	if httpErr != nil {
		result.HTTPError = httpErr.Error()
	}

	return result, nil
}

// assertRouteOK issues a GET request, tracking any redirect hops and capturing
// a snippet of the final response body. redirectChain is only populated when
// there is at least one redirect; it includes the original URL and all hops.
func assertRouteOK(ctx context.Context, route string) (status int, redirectChain []string, bodySnippet string, err error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// On the first redirect, seed the chain with the original request URL.
			if len(via) == 1 {
				redirectChain = append(redirectChain, via[0].URL.String())
			}
			redirectChain = append(redirectChain, req.URL.String())
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, route, nil)
	if err != nil {
		return 0, redirectChain, "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, redirectChain, "", err
	}
	defer resp.Body.Close()

	// Read up to 512 bytes of the response body for diagnostic context.
	buf := make([]byte, 512)
	n, _ := io.ReadFull(resp.Body, buf)
	bodySnippet = string(buf[:n])

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, redirectChain, bodySnippet, fmt.Errorf("route %q returned status %d", route, resp.StatusCode)
	}
	return resp.StatusCode, redirectChain, bodySnippet, nil
}

// assertCertGood verifies the TLS certificate for the given host and returns
// the expiry time of the leaf certificate.
func assertCertGood(ctx context.Context, serverName string, port int) (time.Time, error) {
	if port == 0 {
		port = 443
	}
	d := tls.Dialer{Config: &tls.Config{ServerName: serverName}}
	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", serverName, port))
	if err != nil {
		return time.Time{}, err
	}
	defer conn.Close()

	state := conn.(*tls.Conn).ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return time.Time{}, fmt.Errorf("server %s:%d has no peer certificates", serverName, port)
	}

	return state.PeerCertificates[0].NotAfter, nil
}

func init() {
	toolRegistrations = append(toolRegistrations, mcpToolVerifyRoute)
}
