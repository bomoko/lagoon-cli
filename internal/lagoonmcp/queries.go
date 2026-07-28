package lagoonmcp

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
