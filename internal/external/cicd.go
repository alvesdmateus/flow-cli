package external

import (
	"context"
	"fmt"
	"time"
)

// CICDProvider represents a CI/CD provider.
type CICDProvider string

const (
	CICDGitHubActions CICDProvider = "github"
	CICDGitLab        CICDProvider = "gitlab"
	CICDCircleCI      CICDProvider = "circleci"
)

// CICDClient provides CI/CD operations.
type CICDClient struct {
	provider CICDProvider
	client   *APIClient
	owner    string
	repo     string
}

// CICDConfig configures the CI/CD client.
type CICDConfig struct {
	Provider CICDProvider
	Token    string
	Owner    string
	Repo     string
}

// WorkflowRun represents a workflow run.
type WorkflowRun struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	Conclusion   string    `json:"conclusion"`
	Branch       string    `json:"branch"`
	Event        string    `json:"event"`
	Actor        string    `json:"actor"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	URL          string    `json:"url"`
	WorkflowID   int64     `json:"workflow_id"`
	WorkflowName string    `json:"workflow_name"`
	CommitSHA    string    `json:"commit_sha"`
	CommitMsg    string    `json:"commit_message"`
}

// WorkflowJob represents a job in a workflow.
type WorkflowJob struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Conclusion  string    `json:"conclusion"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Steps       []JobStep `json:"steps"`
}

// JobStep represents a step in a job.
type JobStep struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Conclusion  string    `json:"conclusion"`
	Number      int       `json:"number"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

// Workflow represents a workflow definition.
type Workflow struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	URL       string    `json:"url"`
}

// Artifact represents a build artifact.
type Artifact struct {
	ID                 int64     `json:"id"`
	Name               string    `json:"name"`
	SizeInBytes        int64     `json:"size_in_bytes"`
	URL                string    `json:"url"`
	ArchiveDownloadURL string    `json:"archive_download_url"`
	Expired            bool      `json:"expired"`
	CreatedAt          time.Time `json:"created_at"`
	ExpiresAt          time.Time `json:"expires_at"`
}

// NewCICDClient creates a new CI/CD client.
func NewCICDClient(config CICDConfig) (*CICDClient, error) {
	var baseURL string
	var authType string

	switch config.Provider {
	case CICDGitHubActions:
		baseURL = "https://api.github.com"
		authType = "bearer"
	case CICDGitLab:
		baseURL = "https://gitlab.com/api/v4"
		authType = "bearer"
	case CICDCircleCI:
		baseURL = "https://circleci.com/api/v2"
		authType = "bearer"
	default:
		return nil, fmt.Errorf("unsupported CI/CD provider: %s", config.Provider)
	}

	client := NewAPIClient(baseURL, authType, config.Token)

	// Add GitHub-specific headers
	if config.Provider == CICDGitHubActions {
		client.SetHeader("Accept", "application/vnd.github.v3+json")
		client.SetHeader("X-GitHub-Api-Version", "2022-11-28")
	}

	return &CICDClient{
		provider: config.Provider,
		client:   client,
		owner:    config.Owner,
		repo:     config.Repo,
	}, nil
}

// ListWorkflows lists all workflows in the repository.
func (cc *CICDClient) ListWorkflows(ctx context.Context) ([]Workflow, error) {
	switch cc.provider {
	case CICDGitHubActions:
		return cc.githubListWorkflows(ctx)
	default:
		return nil, fmt.Errorf("list workflows not supported for: %s", cc.provider)
	}
}

func (cc *CICDClient) githubListWorkflows(ctx context.Context) ([]Workflow, error) {
	path := fmt.Sprintf("/repos/%s/%s/actions/workflows", cc.owner, cc.repo)

	var result struct {
		Workflows []struct {
			ID        int64  `json:"id"`
			Name      string `json:"name"`
			Path      string `json:"path"`
			State     string `json:"state"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
			HTMLURL   string `json:"html_url"`
		} `json:"workflows"`
	}

	if err := cc.client.GetJSON(ctx, path, nil, &result); err != nil {
		return nil, err
	}

	var workflows []Workflow
	for _, w := range result.Workflows {
		createdAt, _ := time.Parse(time.RFC3339, w.CreatedAt)
		updatedAt, _ := time.Parse(time.RFC3339, w.UpdatedAt)

		workflows = append(workflows, Workflow{
			ID:        w.ID,
			Name:      w.Name,
			Path:      w.Path,
			State:     w.State,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			URL:       w.HTMLURL,
		})
	}

	return workflows, nil
}

// ListRuns lists workflow runs.
func (cc *CICDClient) ListRuns(ctx context.Context, workflowID int64, branch string, limit int) ([]WorkflowRun, error) {
	switch cc.provider {
	case CICDGitHubActions:
		return cc.githubListRuns(ctx, workflowID, branch, limit)
	default:
		return nil, fmt.Errorf("list runs not supported for: %s", cc.provider)
	}
}

func (cc *CICDClient) githubListRuns(ctx context.Context, workflowID int64, branch string, limit int) ([]WorkflowRun, error) {
	var path string
	if workflowID > 0 {
		path = fmt.Sprintf("/repos/%s/%s/actions/workflows/%d/runs", cc.owner, cc.repo, workflowID)
	} else {
		path = fmt.Sprintf("/repos/%s/%s/actions/runs", cc.owner, cc.repo)
	}

	params := map[string]string{}
	if branch != "" {
		params["branch"] = branch
	}
	if limit > 0 {
		params["per_page"] = fmt.Sprintf("%d", limit)
	}

	var result struct {
		WorkflowRuns []struct {
			ID           int64  `json:"id"`
			Name         string `json:"name"`
			Status       string `json:"status"`
			Conclusion   string `json:"conclusion"`
			HeadBranch   string `json:"head_branch"`
			Event        string `json:"event"`
			Actor        struct {
				Login string `json:"login"`
			} `json:"actor"`
			CreatedAt   string `json:"created_at"`
			UpdatedAt   string `json:"updated_at"`
			HTMLURL     string `json:"html_url"`
			WorkflowID  int64  `json:"workflow_id"`
			HeadCommit  struct {
				ID      string `json:"id"`
				Message string `json:"message"`
			} `json:"head_commit"`
		} `json:"workflow_runs"`
	}

	if err := cc.client.GetJSON(ctx, path, params, &result); err != nil {
		return nil, err
	}

	var runs []WorkflowRun
	for _, r := range result.WorkflowRuns {
		createdAt, _ := time.Parse(time.RFC3339, r.CreatedAt)
		updatedAt, _ := time.Parse(time.RFC3339, r.UpdatedAt)

		runs = append(runs, WorkflowRun{
			ID:         r.ID,
			Name:       r.Name,
			Status:     r.Status,
			Conclusion: r.Conclusion,
			Branch:     r.HeadBranch,
			Event:      r.Event,
			Actor:      r.Actor.Login,
			CreatedAt:  createdAt,
			UpdatedAt:  updatedAt,
			URL:        r.HTMLURL,
			WorkflowID: r.WorkflowID,
			CommitSHA:  r.HeadCommit.ID,
			CommitMsg:  r.HeadCommit.Message,
		})
	}

	return runs, nil
}

// GetRun gets a specific workflow run.
func (cc *CICDClient) GetRun(ctx context.Context, runID int64) (*WorkflowRun, error) {
	switch cc.provider {
	case CICDGitHubActions:
		return cc.githubGetRun(ctx, runID)
	default:
		return nil, fmt.Errorf("get run not supported for: %s", cc.provider)
	}
}

func (cc *CICDClient) githubGetRun(ctx context.Context, runID int64) (*WorkflowRun, error) {
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d", cc.owner, cc.repo, runID)

	var result struct {
		ID           int64  `json:"id"`
		Name         string `json:"name"`
		Status       string `json:"status"`
		Conclusion   string `json:"conclusion"`
		HeadBranch   string `json:"head_branch"`
		Event        string `json:"event"`
		Actor        struct {
			Login string `json:"login"`
		} `json:"actor"`
		CreatedAt   string `json:"created_at"`
		UpdatedAt   string `json:"updated_at"`
		HTMLURL     string `json:"html_url"`
		WorkflowID  int64  `json:"workflow_id"`
		HeadCommit  struct {
			ID      string `json:"id"`
			Message string `json:"message"`
		} `json:"head_commit"`
	}

	if err := cc.client.GetJSON(ctx, path, nil, &result); err != nil {
		return nil, err
	}

	createdAt, _ := time.Parse(time.RFC3339, result.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, result.UpdatedAt)

	return &WorkflowRun{
		ID:         result.ID,
		Name:       result.Name,
		Status:     result.Status,
		Conclusion: result.Conclusion,
		Branch:     result.HeadBranch,
		Event:      result.Event,
		Actor:      result.Actor.Login,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
		URL:        result.HTMLURL,
		WorkflowID: result.WorkflowID,
		CommitSHA:  result.HeadCommit.ID,
		CommitMsg:  result.HeadCommit.Message,
	}, nil
}

// GetJobs gets jobs for a workflow run.
func (cc *CICDClient) GetJobs(ctx context.Context, runID int64) ([]WorkflowJob, error) {
	switch cc.provider {
	case CICDGitHubActions:
		return cc.githubGetJobs(ctx, runID)
	default:
		return nil, fmt.Errorf("get jobs not supported for: %s", cc.provider)
	}
}

func (cc *CICDClient) githubGetJobs(ctx context.Context, runID int64) ([]WorkflowJob, error) {
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/jobs", cc.owner, cc.repo, runID)

	var result struct {
		Jobs []struct {
			ID          int64  `json:"id"`
			Name        string `json:"name"`
			Status      string `json:"status"`
			Conclusion  string `json:"conclusion"`
			StartedAt   string `json:"started_at"`
			CompletedAt string `json:"completed_at"`
			Steps       []struct {
				Name        string `json:"name"`
				Status      string `json:"status"`
				Conclusion  string `json:"conclusion"`
				Number      int    `json:"number"`
				StartedAt   string `json:"started_at"`
				CompletedAt string `json:"completed_at"`
			} `json:"steps"`
		} `json:"jobs"`
	}

	if err := cc.client.GetJSON(ctx, path, nil, &result); err != nil {
		return nil, err
	}

	var jobs []WorkflowJob
	for _, j := range result.Jobs {
		startedAt, _ := time.Parse(time.RFC3339, j.StartedAt)
		completedAt, _ := time.Parse(time.RFC3339, j.CompletedAt)

		var steps []JobStep
		for _, s := range j.Steps {
			stepStartedAt, _ := time.Parse(time.RFC3339, s.StartedAt)
			stepCompletedAt, _ := time.Parse(time.RFC3339, s.CompletedAt)

			steps = append(steps, JobStep{
				Name:        s.Name,
				Status:      s.Status,
				Conclusion:  s.Conclusion,
				Number:      s.Number,
				StartedAt:   stepStartedAt,
				CompletedAt: stepCompletedAt,
			})
		}

		jobs = append(jobs, WorkflowJob{
			ID:          j.ID,
			Name:        j.Name,
			Status:      j.Status,
			Conclusion:  j.Conclusion,
			StartedAt:   startedAt,
			CompletedAt: completedAt,
			Steps:       steps,
		})
	}

	return jobs, nil
}

// GetLogs retrieves logs for a job.
func (cc *CICDClient) GetLogs(ctx context.Context, jobID int64) (string, error) {
	switch cc.provider {
	case CICDGitHubActions:
		return cc.githubGetLogs(ctx, jobID)
	default:
		return "", fmt.Errorf("get logs not supported for: %s", cc.provider)
	}
}

func (cc *CICDClient) githubGetLogs(ctx context.Context, jobID int64) (string, error) {
	path := fmt.Sprintf("/repos/%s/%s/actions/jobs/%d/logs", cc.owner, cc.repo, jobID)

	resp, err := cc.client.GET(ctx, path, nil)
	if err != nil {
		return "", err
	}

	return resp.Body, nil
}

// TriggerWorkflow manually triggers a workflow.
func (cc *CICDClient) TriggerWorkflow(ctx context.Context, workflowID int64, ref string, inputs map[string]string) error {
	switch cc.provider {
	case CICDGitHubActions:
		return cc.githubTriggerWorkflow(ctx, workflowID, ref, inputs)
	default:
		return fmt.Errorf("trigger workflow not supported for: %s", cc.provider)
	}
}

func (cc *CICDClient) githubTriggerWorkflow(ctx context.Context, workflowID int64, ref string, inputs map[string]string) error {
	path := fmt.Sprintf("/repos/%s/%s/actions/workflows/%d/dispatches", cc.owner, cc.repo, workflowID)

	body := map[string]interface{}{
		"ref": ref,
	}
	if len(inputs) > 0 {
		body["inputs"] = inputs
	}

	return cc.client.PostJSON(ctx, path, body, nil)
}

// CancelRun cancels a workflow run.
func (cc *CICDClient) CancelRun(ctx context.Context, runID int64) error {
	switch cc.provider {
	case CICDGitHubActions:
		return cc.githubCancelRun(ctx, runID)
	default:
		return fmt.Errorf("cancel run not supported for: %s", cc.provider)
	}
}

func (cc *CICDClient) githubCancelRun(ctx context.Context, runID int64) error {
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/cancel", cc.owner, cc.repo, runID)
	return cc.client.PostJSON(ctx, path, nil, nil)
}

// RerunWorkflow re-runs a workflow.
func (cc *CICDClient) RerunWorkflow(ctx context.Context, runID int64) error {
	switch cc.provider {
	case CICDGitHubActions:
		return cc.githubRerunWorkflow(ctx, runID)
	default:
		return fmt.Errorf("rerun workflow not supported for: %s", cc.provider)
	}
}

func (cc *CICDClient) githubRerunWorkflow(ctx context.Context, runID int64) error {
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/rerun", cc.owner, cc.repo, runID)
	return cc.client.PostJSON(ctx, path, nil, nil)
}

// ListArtifacts lists artifacts for a workflow run.
func (cc *CICDClient) ListArtifacts(ctx context.Context, runID int64) ([]Artifact, error) {
	switch cc.provider {
	case CICDGitHubActions:
		return cc.githubListArtifacts(ctx, runID)
	default:
		return nil, fmt.Errorf("list artifacts not supported for: %s", cc.provider)
	}
}

func (cc *CICDClient) githubListArtifacts(ctx context.Context, runID int64) ([]Artifact, error) {
	path := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/artifacts", cc.owner, cc.repo, runID)

	var result struct {
		Artifacts []struct {
			ID                 int64  `json:"id"`
			Name               string `json:"name"`
			SizeInBytes        int64  `json:"size_in_bytes"`
			URL                string `json:"url"`
			ArchiveDownloadURL string `json:"archive_download_url"`
			Expired            bool   `json:"expired"`
			CreatedAt          string `json:"created_at"`
			ExpiresAt          string `json:"expires_at"`
		} `json:"artifacts"`
	}

	if err := cc.client.GetJSON(ctx, path, nil, &result); err != nil {
		return nil, err
	}

	var artifacts []Artifact
	for _, a := range result.Artifacts {
		createdAt, _ := time.Parse(time.RFC3339, a.CreatedAt)
		expiresAt, _ := time.Parse(time.RFC3339, a.ExpiresAt)

		artifacts = append(artifacts, Artifact{
			ID:                 a.ID,
			Name:               a.Name,
			SizeInBytes:        a.SizeInBytes,
			URL:                a.URL,
			ArchiveDownloadURL: a.ArchiveDownloadURL,
			Expired:            a.Expired,
			CreatedAt:          createdAt,
			ExpiresAt:          expiresAt,
		})
	}

	return artifacts, nil
}

// GetLatestRunStatus gets the status of the latest run for a workflow.
func (cc *CICDClient) GetLatestRunStatus(ctx context.Context, workflowID int64, branch string) (*WorkflowRun, error) {
	runs, err := cc.ListRuns(ctx, workflowID, branch, 1)
	if err != nil {
		return nil, err
	}

	if len(runs) == 0 {
		return nil, fmt.Errorf("no runs found for workflow")
	}

	return &runs[0], nil
}

// WaitForRun waits for a workflow run to complete.
func (cc *CICDClient) WaitForRun(ctx context.Context, runID int64, pollInterval time.Duration) (*WorkflowRun, error) {
	if pollInterval == 0 {
		pollInterval = 10 * time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			run, err := cc.GetRun(ctx, runID)
			if err != nil {
				return nil, err
			}

			if run.Status == "completed" {
				return run, nil
			}
		}
	}
}

// RunStatus represents the overall status of a run.
type RunStatus struct {
	IsRunning   bool
	IsSuccess   bool
	IsFailed    bool
	IsCancelled bool
}

// GetStatus returns the status interpretation of a workflow run.
func (wr *WorkflowRun) GetStatus() RunStatus {
	return RunStatus{
		IsRunning:   wr.Status == "in_progress" || wr.Status == "queued",
		IsSuccess:   wr.Conclusion == "success",
		IsFailed:    wr.Conclusion == "failure",
		IsCancelled: wr.Conclusion == "cancelled",
	}
}
