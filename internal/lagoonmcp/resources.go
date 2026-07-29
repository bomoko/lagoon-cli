package lagoonmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

const (
	lagoonPrefix        = "lagoon://"
	projectResourcePath = "project/"
)

func (s *LagoonMCPServer) setProjectResource(project resolvedProject, fromPath string) {
	projectURI := fmt.Sprintf("%s%s%s", lagoonPrefix, projectResourcePath, project.Name)
	pathsURI := fmt.Sprintf("%s%s%s/paths", lagoonPrefix, projectResourcePath, project.Name)

	s.Server.AddResource(
		mcp.Resource{
			URI:         projectURI,
			Name:        project.Name,
			Description: fmt.Sprintf("Lagoon project: %s (production: %s)", project.Name, project.ProductionEnvironment),
			MIMEType:    "application/json",
		},
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			data, err := json.MarshalIndent(project, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("project marshal failed: %w", err)
			}
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(data),
				},
			}, nil
		},
	)

	paths := []string{}
	if fromPath != "" {
		paths = []string{fromPath}
	}

	s.Server.AddResource(
		mcp.Resource{
			URI:         pathsURI,
			Name:        fmt.Sprintf("%s/paths", project.Name),
			Description: fmt.Sprintf("Local paths for Lagoon project %s", project.Name),
			MIMEType:    "application/json",
		},
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			data, err := json.MarshalIndent(paths, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("path marshal failed: %w", err)
			}
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(data),
				},
			}, nil
		},
	)
}
