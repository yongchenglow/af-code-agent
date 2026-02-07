package webhook

// PRContext contains pull request context information
type PRContext struct {
	Owner          string
	Repo           string
	Number         int
	Branch         string
	BaseBranch     string
	Author         string
	Title          string
	Files          []string
	InstallationID int64
	HeadSHA        string
}

// WorkflowResult represents the result of a workflow execution
type WorkflowResult struct {
	Success bool
	Message string
	Error   error
}

// WebhookEvent represents a GitHub webhook event
type WebhookEvent struct {
	Type        string
	Action      string
	PullRequest *PullRequestPayload
	CheckSuite  *CheckSuitePayload
	WorkflowRun *WorkflowRunPayload
	Installation *InstallationPayload
}

// PullRequestPayload contains PR webhook data
type PullRequestPayload struct {
	Number int    `json:"number"`
	State  string `json:"state"`
	Title  string `json:"title"`
	Head   struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
}

// CheckSuitePayload contains check suite webhook data
type CheckSuitePayload struct {
	Conclusion string `json:"conclusion"`
	Status     string `json:"status"`
	HeadSHA    string `json:"head_sha"`
}

// WorkflowRunPayload contains workflow run webhook data
type WorkflowRunPayload struct {
	Conclusion string `json:"conclusion"`
	Status     string `json:"status"`
	HeadSHA    string `json:"head_sha"`
}

// InstallationPayload contains installation data
type InstallationPayload struct {
	ID int64 `json:"id"`
}

// RepositoryPayload contains repository data
type RepositoryPayload struct {
	Name  string `json:"name"`
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
}
