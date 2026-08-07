package lagoonmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func sanitizeBuildLog(s string) string {
	s = strings.TrimPrefix(s, "\ufeff")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

var doubleHashRegex = regexp.MustCompile(`(?m)(^#{10,}\n)(^#{10,}\n)`)

func parseBuildLog(buildLog string) (string, error) {
	buildLog = sanitizeBuildLog(buildLog)
	fmt.Println("buildlog", buildLog)
	matches := doubleHashRegex.FindAllStringSubmatchIndex(buildLog, -1)
	fmt.Println("matches", matches)
	if len(matches) == 0 {
		return "", fmt.Errorf("unable to parse build failure")
	}

	// the last step is typically the fauilure point
	failureStep := matches[len(matches)-1]
	fmt.Println("*******failureStep", failureStep)
	// last[4], last[5] are the start/end indices of capture group 2 -
	// the second hash line, i.e. the start of the failing step's header.
	failureLog := strings.TrimSpace(buildLog[failureStep[4]:])
	if failureLog == "" {
		return "", fmt.Errorf("no content found after the last step boundary")
	}

	return failureLog, nil
}

func getDeploymentStatus(lagoonMCPServer *LagoonMCPServer) {
	lagoonMCPServer.Server.AddTool(
		mcp.NewTool("get_deployment_status",
			mcp.WithDescription("Get the status of a Lagoon deployment - defaults to the latest deployment"),
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

			environment := req.GetString("environment", "")

			lc := lagoonMCPServer.NewLagoonClient()
			project, err := getProjectByName(ctx, lc, projectName)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get project %q: %v", project, err)), nil
			}

			env, err := resolveEnvforProject(project, environment)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get environment %q: %v", environment, err)), nil
			}

			deployment, err := getLatestDeployment(ctx, lc, project.ID, env.Name)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get latest deployment for environment %q: %v", env.Name, err)), nil
			}
			if deployment == nil {
				return mcp.NewToolResultText(fmt.Sprintf("no deployments found for environment %q", env.Name)), nil
			}

			deploymentError := deployment.Status == "failed"

			if !deploymentError {
				dep := *deployment
				// stripping out the potentially huge buildlog for a successful deployment
				dep.BuildLog = ""
				out, err := json.MarshalIndent(dep, "", "  ")
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("marshal error: %v", err)), nil
				}
				return mcp.NewToolResultText(string(out)), nil
			}
			buildLog := deployment.BuildLog

			failureStep, err := parseBuildLog(buildLog)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to parse build log: %v", err)), nil
			}

			// prompt := "Analyse the failed build log, advise the most likely cause and suggest potential steps to resolve it."

			// attempt to constrain the sample repsonse so we don't get slop back
			prompt := "You are a Lagoon developer assistant analysing a failed build log excerpt. " +
				"Base your analysis ONLY on evidence present in the log; never invent errors. " +
				"If the log contains a clear, identifiable failure, advise the most likely cause and suggest potential steps to resolve it. " +
				"If the cause is obscure/ambiguous or cannot be determined with reasonable confidence, do NOT guess. " +
				"In that case reply with exactly this sentence and nothing else: " +
				"\"NO_ROOT_CAUSE: I couldn't determine a root cause from this build log — please contact your Lagoon administrator.\""

			sampleReq := mcp.CreateMessageRequest{
				CreateMessageParams: mcp.CreateMessageParams{
					SystemPrompt: prompt,
					Messages: []mcp.SamplingMessage{
						{
							Role: mcp.RoleUser,
							Content: mcp.TextContent{
								Type: "text",
								Text: failureStep,
							},
						},
					},
					MaxTokens:   1000,
					Temperature: 0,
				},
			}

			res, err := lagoonMCPServer.Server.RequestSampling(ctx, sampleReq)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to request sampling: %v", err)), nil
			}

			fmt.Println("res", res)
			sampleResp, ok := res.Content.(mcp.TextContent)
			if !ok {
				return mcp.NewToolResultError("sampling response did not contain text content"), nil
			}
			fmt.Println("sampleResp", sampleResp)

			sampleAnalysis := sampleResp.Text
			if strings.HasPrefix(strings.TrimSpace(sampleResp.Text), "NO_ROOT_CAUSE:") {
				sampleAnalysis = "I couldn't determine a root cause from the build log. Please contact your Lagoon administrator for further investigation."
			}

			// stripping out the potentially huge buildlog - we'll include the parsed failed step
			dep := *deployment
			dep.BuildLog = ""

			result := struct {
				Deployment     deploymentDetails `json:"deployment"`
				FailureLog     string            `json:"failureLog"`
				SampleAnalysis string            `json:"sampleAnalysis"`
			}{
				Deployment:     dep,
				FailureLog:     failureStep,
				SampleAnalysis: sampleAnalysis,
			}

			out, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("marshal error: %v", err)), nil
			}

			return mcp.NewToolResultText(string(out)), nil
		},
	)
}

func init() {
	toolRegistrations = append(toolRegistrations, getDeploymentStatus)
}
