package lagoonmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-git/go-git/v5"
	lclient "github.com/uselagoon/machinery/api/lagoon/client"
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

const projectByNameQuery = `query ($name: String!) {
  projectByName(name: $name) {
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

const deploymentsForEnvironmentQuery = `query ($project: Int!, $environment: String!, $limit: Int) {
  environmentByName(project: $project, name: $environment) {
    deployments(limit: $limit) {
      id
      name
      status
      created
      started
      completed
      remoteId
    }
  }
}`

// resolvedProject is a minimal projection of the Lagoon Project type
// containing the fields returned by projectByGitURLQuery.
type resolvedProject struct {
	ID                           uint                  `json:"id"`
	Name                         string                `json:"name"`
	GitURL                       string                `json:"gitUrl"`
	ProductionEnvironment        string                `json:"productionEnvironment"`
	DevelopmentEnvironmentsLimit int                   `json:"developmentEnvironmentsLimit"`
	Environments                 []resolvedEnvironment `json:"environments"`
}

type resolvedEnvironment struct {
	ID              uint   `json:"id"`
	Name            string `json:"name"`
	EnvironmentType string `json:"environmentType"`
	Route           string `json:"route"`
}

type deploymentDetails struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Created   string `json:"created"`
	Started   string `json:"started,omitempty"`
	Completed string `json:"completed,omitempty"`
	Error     string `json:"error,omitempty"`
}

type routeStatus struct {
	StatusCode    int    `json:"statusCode,omitempty"`
	XLagoonHeader string `json:"xLagoonHeader,omitempty"`
	Error         string `json:"error,omitempty"`
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

func getProjectsByPath(ctx context.Context, lc *lclient.Client, path string) ([]resolvedProject, error) {
	urls, err := gitRemoteURLs(path)
	if err != nil {
		return nil, fmt.Errorf("could not read git remotes from %q: %w", path, err)
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("no git remotes found in %q", path)
	}

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

	return projects, nil
}

func getProjectByName(ctx context.Context, lc *lclient.Client, name string) (*resolvedProject, error) {
	raw, err := lc.ProcessRaw(ctx, projectByNameQuery, map[string]interface{}{
		"name": name,
	})
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}

	var resp struct {
		ProjectByName *resolvedProject `json:"projectByName"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}

	if resp.ProjectByName == nil {
		return nil, fmt.Errorf("project %q not found", name)
	}

	return resp.ProjectByName, nil
}
