package apiclient

import (
	"context"

	"skill-eval-harness/internal/dataset"
)

// Pagination captures standard list pagination metadata.
type Pagination struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	Total   int  `json:"total"`
	HasMore bool `json:"has_more"`
}

// ErrorResponse captures the API error payload.
type ErrorResponse struct {
	Error APIErrorPayload `json:"error"`
}

// APIErrorPayload captures code/message/details from API errors.
type APIErrorPayload struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Details []map[string]any `json:"details,omitempty"`
}

// AuthTokenRequest exchanges API key for JWT token.
type AuthTokenRequest struct {
	APIKey string `json:"api_key"`
}

// AuthTokenResponse is returned by POST /auth/token.
type AuthTokenResponse struct {
	Token    string `json:"token"`
	Role     string `json:"role,omitempty"`
	IssuedAt string `json:"issued_at,omitempty"`
}

// DatasetSummaryDTO describes dataset list rows.
type DatasetSummaryDTO struct {
	DatasetID string `json:"dataset_id"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Owner     string `json:"owner"`
	CaseCount int    `json:"case_count"`
	CreatedAt string `json:"created_at"`
}

// DatasetDetailDTO describes a full dataset record.
type DatasetDetailDTO struct {
	DatasetID   string           `json:"dataset_id"`
	Name        string           `json:"name"`
	Version     string           `json:"version"`
	Owner       string           `json:"owner"`
	Description string           `json:"description,omitempty"`
	Manifest    dataset.Manifest `json:"manifest"`
	CaseCount   int              `json:"case_count"`
	Checksum    string           `json:"checksum,omitempty"`
	CreatedAt   string           `json:"created_at"`
}

// DatasetVerifyDTO is returned by GET /datasets/:dataset_id/verify.
type DatasetVerifyDTO struct {
	Valid      bool   `json:"valid"`
	Checksum   string `json:"checksum"`
	Expected   string `json:"expected,omitempty"`
	Actual     string `json:"actual,omitempty"`
	CaseCount  int    `json:"case_count"`
	VerifiedAt string `json:"verified_at"`
}

// ListDatasetsQuery contains dataset list filters.
type ListDatasetsQuery struct {
	Tag    string
	Owner  string
	Name   string
	Limit  int
	Offset int
	Sort   string
}

// ListDatasetCasesQuery contains dataset case list filters.
type ListDatasetCasesQuery struct {
	Source string
	Status string
	Tag    string
	Limit  int
	Offset int
}

// ListRunsQuery contains run list filters.
type ListRunsQuery struct {
	Skill     string
	DatasetID string
	From      string
	To        string
	MinScore  *float64
	Limit     int
	Offset    int
}

// ListDatasetsResponse is returned by GET /datasets.
type ListDatasetsResponse struct {
	Datasets   []DatasetSummaryDTO `json:"datasets"`
	Pagination Pagination          `json:"pagination"`
}

// GetDatasetCasesResponse is returned by GET /datasets/:dataset_id/cases.
type GetDatasetCasesResponse struct {
	Manifest   dataset.Manifest         `json:"manifest"`
	Cases      []dataset.EvaluationCase `json:"cases"`
	Pagination Pagination               `json:"pagination"`
}

// CreateRunResponse is returned by POST /runs.
type CreateRunResponse struct {
	RunID       string  `json:"run_id"`
	Score       float64 `json:"score"`
	SuccessRate float64 `json:"success_rate"`
	CreatedAt   string  `json:"created_at"`
}

// ListRunsResponse is returned by GET /runs.
type ListRunsResponse struct {
	Runs       []RunResultDTO `json:"runs"`
	Pagination Pagination     `json:"pagination"`
}

// GateRunRequest is accepted by POST /runs/:run_id/gate.
type GateRunRequest struct {
	PolicyID string `json:"policy_id"`
}

// GateRuleDetail provides rule-level gate decision details.
type GateRuleDetail struct {
	Rule     string `json:"rule"`
	Required string `json:"required"`
	Actual   string `json:"actual"`
	Passed   bool   `json:"passed"`
}

// GateRunResponse is returned by POST /runs/:run_id/gate.
type GateRunResponse struct {
	Passed      bool             `json:"passed"`
	RunID       string           `json:"run_id"`
	PolicyID    string           `json:"policy_id"`
	Details     []GateRuleDetail `json:"details"`
	EvaluatedAt string           `json:"evaluated_at"`
}

// RunResultDTO carries run data through API.
type RunResultDTO struct {
	RunID          string          `json:"run_id"`
	DatasetID      string          `json:"dataset_id"`
	Skill          string          `json:"skill"`
	Success        bool            `json:"success"`
	LatencyMS      int64           `json:"latency_ms"`
	TokenUsage     int64           `json:"token_usage"`
	Error          string          `json:"error,omitempty"`
	Results        []CaseResultDTO `json:"results"`
	Metrics        RunMetricsDTO   `json:"metrics"`
	CreatedAt      string          `json:"created_at"`
	DatasetVersion string          `json:"dataset_version,omitempty"`
	DatasetHash    string          `json:"dataset_hash,omitempty"`
	ModelName      string          `json:"model_name,omitempty"`
	GitCommit      string          `json:"git_commit,omitempty"`
}

// CaseResultDTO describes each case result in a run.
type CaseResultDTO struct {
	CaseID            string         `json:"case_id"`
	Success           bool           `json:"success"`
	LatencyMS         int64          `json:"latency_ms"`
	TokenUsage        int64          `json:"token_usage"`
	Output            map[string]any `json:"output,omitempty"`
	Error             string         `json:"error,omitempty"`
	Classification    string         `json:"classification,omitempty"`
	FailureClusterID  string         `json:"failure_cluster_id,omitempty"`
	ReasoningEstimate int64          `json:"reasoning_tokens_estimate,omitempty"`
}

// RunMetricsDTO describes aggregate run metrics.
type RunMetricsDTO struct {
	SuccessRate          float64 `json:"success_rate"`
	AvgTokens            float64 `json:"avg_tokens"`
	P95Latency           int64   `json:"p95_latency"`
	CostFactor           float64 `json:"cost_factor,omitempty"`
	ClassificationFactor float64 `json:"classification_factor,omitempty"`
	CostUSD              float64 `json:"cost_usd,omitempty"`
	StabilityVariance    float64 `json:"stability_variance,omitempty"`
	Score                float64 `json:"score"`
}

// Client exposes the P0 API capability surface used by CLI.
type Client interface {
	ListDatasets(ctx context.Context, query ListDatasetsQuery) (ListDatasetsResponse, error)
	GetDataset(ctx context.Context, datasetID string) (DatasetDetailDTO, error)
	GetDatasetCases(ctx context.Context, datasetID string, query ListDatasetCasesQuery) (GetDatasetCasesResponse, error)
	VerifyDataset(ctx context.Context, datasetID string) (DatasetVerifyDTO, error)
	CreateRun(ctx context.Context, run RunResultDTO) (CreateRunResponse, error)
	GetRun(ctx context.Context, runID string) (RunResultDTO, error)
	ListRuns(ctx context.Context, query ListRunsQuery) (ListRunsResponse, error)
	GateRun(ctx context.Context, runID string, req GateRunRequest) (GateRunResponse, error)
}
