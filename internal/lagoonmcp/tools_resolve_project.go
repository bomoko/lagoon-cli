package lagoonmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/mark3labs/mcp-go/mcp"
)

// projectByGitURLQuery fetches a project by its git remote URL.
// machinery has no wrapper for this query so we use ProcessRaw.
const projectByGitURLQuery = `query ($gitUrl: String!) {
  projectByGitUrl(gitUrl: $gitUrl) {
    id
    name
    gitUrl
    productionEnvironment
    developmentEnvironmentsLimit
    environments {
      id
      name
      environmentType
      route
    }
  }
}`

// resolvedProject is a minimal projection of the Lagoon Project type
// containing the fields returned by projectByGitURLQuery.
type resolvedProject struct {
	ID                           uint   `json:"id"`
	Name                         string `json:"name"`
	GitURL                       string `json:"gitUrl"`
	ProductionEnvironment        string `json:"productionEnvironment"`
	DevelopmentEnvironmentsLimit int    `json:"developmentEnvironmentsLimit"`
	Environments                 []struct {
		ID              uint   `json:"id"`
		Name            string `json:"name"`
		EnvironmentType string `json:"environmentType"`
		Route           string `json:"route"`
	} `json:"environments"`
}

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

				urls, err := gitRemoteURLs(path)
				if err != nil {
					return mcp.NewToolResultError(
						fmt.Sprintf("could not read git remotes from %q: %v", path, err),
					), nil
				}
				if len(urls) == 0 {
					return mcp.NewToolResultText("no git remotes found in the given directory"), nil
				}

				lc := s.NewLagoonClient()

				// Query Lagoon for each remote URL, deduplicate by project ID.
				seen := make(map[uint]bool)
				var projects []resolvedProject

				for _, url := range urls {
					raw, err := lc.ProcessRaw(ctx, projectByGitURLQuery, map[string]interface{}{
						"gitUrl": url,
					})
					if err != nil {
						// This remote is not known to Lagoon — skip silently.
						continue
					}

					b, err := json.Marshal(raw)
					if err != nil {
						continue
					}

					var resp struct {
						ProjectByGitURL *resolvedProject `json:"projectByGitUrl"`
					}
					if err := json.Unmarshal(b, &resp); err != nil || resp.ProjectByGitURL == nil {
						continue
					}

					p := resp.ProjectByGitURL
					if !seen[p.ID] {
						seen[p.ID] = true
						projects = append(projects, *p)
					}
				}

				if len(projects) == 0 {
					return mcp.NewToolResultText(
						"no Lagoon projects found matching the git remotes in this directory",
					), nil
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

// gitRemoteURLs opens the git repository at path and returns the deduplicated
// set of remote URLs configured for it. Uses go-git so no git binary is required.
func gitRemoteURLs(path string) ([]string, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("opening git repository at %q: %w", path, err)
	}

	remotes, err := repo.Remotes()
	if err != nil {
		return nil, fmt.Errorf("listing remotes: %w", err)
	}

	seen := make(map[string]bool)
	var urls []string
	for _, remote := range remotes {
		for _, url := range remote.Config().URLs {
			if !seen[url] {
				seen[url] = true
				urls = append(urls, url)
			}
		}
	}
	return urls, nil
}
