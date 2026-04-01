package apiclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"skill-eval-harness/internal/dataset"
)

// LocalClient serves API DTOs from local JSON files for offline fallback mode.
type LocalClient struct {
	rootDir string
	nowFunc func() time.Time
}

// NewLocalClient creates the local fallback client backed by .mock-seh data files.
func NewLocalClient(rootDir string) *LocalClient {
	dir := strings.TrimSpace(rootDir)
	if dir == "" {
		dir = ".mock-seh"
	}
	return &LocalClient{rootDir: dir, nowFunc: func() time.Time { return time.Now().UTC() }}
}

func (c *LocalClient) ListDatasets(_ context.Context, query ListDatasetsQuery) (ListDatasetsResponse, error) {
	items, err := c.readDatasets()
	if err != nil {
		return ListDatasetsResponse{}, err
	}

	filtered := make([]DatasetDetailDTO, 0, len(items))
	for _, item := range items {
		if query.Owner != "" && item.Owner != query.Owner {
			continue
		}
		if query.Name != "" && !strings.Contains(strings.ToLower(item.Name), strings.ToLower(query.Name)) {
			continue
		}
		if query.Tag != "" {
			cases, err := c.readDatasetCases(item.DatasetID)
			if err != nil {
				continue
			}
			if !datasetHasTag(cases.Cases, query.Tag) {
				continue
			}
		}
		filtered = append(filtered, item)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt > filtered[j].CreatedAt
	})

	start, end := clampPage(len(filtered), query.Limit, query.Offset, 20)
	subset := filtered[start:end]

	summaries := make([]DatasetSummaryDTO, 0, len(subset))
	for _, item := range subset {
		summaries = append(summaries, DatasetSummaryDTO{
			DatasetID: item.DatasetID,
			Name:      item.Name,
			Version:   item.Version,
			Owner:     item.Owner,
			CaseCount: item.CaseCount,
			CreatedAt: item.CreatedAt,
		})
	}

	return ListDatasetsResponse{Datasets: summaries, Pagination: Pagination{
		Limit:   end - start,
		Offset:  start,
		Total:   len(filtered),
		HasMore: end < len(filtered),
	}}, nil
}

func (c *LocalClient) GetDataset(_ context.Context, datasetID string) (DatasetDetailDTO, error) {
	items, err := c.readDatasets()
	if err != nil {
		return DatasetDetailDTO{}, err
	}
	for _, item := range items {
		if item.DatasetID == datasetID {
			if item.CaseCount == 0 {
				cases, err := c.readDatasetCases(datasetID)
				if err == nil {
					item.CaseCount = len(cases.Cases)
				}
			}
			return item, nil
		}
	}
	return DatasetDetailDTO{}, &APIError{StatusCode: 404, Payload: APIErrorPayload{Code: 404, Message: "dataset not found"}}
}

func (c *LocalClient) GetDatasetCases(_ context.Context, datasetID string, query ListDatasetCasesQuery) (GetDatasetCasesResponse, error) {
	payload, err := c.readDatasetCases(datasetID)
	if err != nil {
		return GetDatasetCasesResponse{}, err
	}

	filtered := make([]dataset.EvaluationCase, 0, len(payload.Cases))
	for _, item := range payload.Cases {
		if query.Tag != "" && !contains(item.Tags, query.Tag) {
			continue
		}
		filtered = append(filtered, item)
	}

	start, end := clampPage(len(filtered), query.Limit, query.Offset, 100)
	return GetDatasetCasesResponse{
		Manifest: payload.Manifest,
		Cases:    filtered[start:end],
		Pagination: Pagination{
			Limit:   end - start,
			Offset:  start,
			Total:   len(filtered),
			HasMore: end < len(filtered),
		},
	}, nil
}

func (c *LocalClient) VerifyDataset(_ context.Context, datasetID string) (DatasetVerifyDTO, error) {
	datasetItem, err := c.GetDataset(context.Background(), datasetID)
	if err != nil {
		return DatasetVerifyDTO{}, err
	}

	cases, err := c.readDatasetCases(datasetID)
	if err != nil {
		return DatasetVerifyDTO{}, err
	}

	actual := computeDatasetChecksum(cases.Manifest, cases.Cases)
	valid := datasetItem.Checksum == "" || datasetItem.Checksum == actual
	result := DatasetVerifyDTO{
		Valid:      valid,
		Checksum:   actual,
		CaseCount:  len(cases.Cases),
		VerifiedAt: c.nowFunc().Format(time.RFC3339),
	}
	if !valid {
		result.Expected = datasetItem.Checksum
		result.Actual = actual
	}
	return result, nil
}

func (c *LocalClient) CreateRun(_ context.Context, run RunResultDTO) (CreateRunResponse, error) {
	if strings.TrimSpace(run.DatasetID) == "" {
		return CreateRunResponse{}, &APIError{StatusCode: 422, Payload: APIErrorPayload{Code: 422, Message: "dataset_id is required"}}
	}
	if len(run.Results) == 0 {
		return CreateRunResponse{}, &APIError{StatusCode: 422, Payload: APIErrorPayload{Code: 422, Message: "results is required"}}
	}
	if _, err := c.GetDataset(context.Background(), run.DatasetID); err != nil {
		return CreateRunResponse{}, &APIError{StatusCode: 422, Payload: APIErrorPayload{Code: 422, Message: "dataset not found"}}
	}

	runs, err := c.readRuns()
	if err != nil {
		return CreateRunResponse{}, err
	}

	if strings.TrimSpace(run.RunID) == "" {
		run.RunID = fmt.Sprintf("run_%d", c.nowFunc().UnixNano())
	}
	if strings.TrimSpace(run.CreatedAt) == "" {
		run.CreatedAt = c.nowFunc().Format(time.RFC3339)
	}

	runs = append(runs, run)
	if err := c.writeRuns(runs); err != nil {
		return CreateRunResponse{}, err
	}

	return CreateRunResponse{RunID: run.RunID, Score: run.Metrics.Score, SuccessRate: run.Metrics.SuccessRate, CreatedAt: run.CreatedAt}, nil
}

func (c *LocalClient) GetRun(_ context.Context, runID string) (RunResultDTO, error) {
	runs, err := c.readRuns()
	if err != nil {
		return RunResultDTO{}, err
	}
	for _, item := range runs {
		if item.RunID == runID {
			return item, nil
		}
	}
	return RunResultDTO{}, &APIError{StatusCode: 404, Payload: APIErrorPayload{Code: 404, Message: "run not found"}}
}

func (c *LocalClient) ListRuns(_ context.Context, query ListRunsQuery) (ListRunsResponse, error) {
	runs, err := c.readRuns()
	if err != nil {
		return ListRunsResponse{}, err
	}

	filtered := make([]RunResultDTO, 0, len(runs))
	for _, item := range runs {
		if query.Skill != "" && item.Skill != query.Skill {
			continue
		}
		if query.DatasetID != "" && item.DatasetID != query.DatasetID {
			continue
		}
		if query.MinScore != nil && item.Metrics.Score < *query.MinScore {
			continue
		}
		if query.From != "" && item.CreatedAt < query.From {
			continue
		}
		if query.To != "" && item.CreatedAt > query.To {
			continue
		}
		filtered = append(filtered, item)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt > filtered[j].CreatedAt
	})

	start, end := clampPage(len(filtered), query.Limit, query.Offset, 20)
	return ListRunsResponse{Runs: filtered[start:end], Pagination: Pagination{
		Limit:   end - start,
		Offset:  start,
		Total:   len(filtered),
		HasMore: end < len(filtered),
	}}, nil
}

func (c *LocalClient) GateRun(_ context.Context, runID string, req GateRunRequest) (GateRunResponse, error) {
	run, err := c.GetRun(context.Background(), runID)
	if err != nil {
		return GateRunResponse{}, err
	}

	policy, err := c.readPolicy(req.PolicyID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return GateRunResponse{}, &APIError{StatusCode: 404, Payload: APIErrorPayload{Code: 404, Message: "policy not found"}}
		}
		return GateRunResponse{}, err
	}

	details := []GateRuleDetail{
		{
			Rule:     "min_score",
			Required: strconv.FormatFloat(policy.MinScore, 'f', -1, 64),
			Actual:   strconv.FormatFloat(run.Metrics.Score, 'f', -1, 64),
			Passed:   run.Metrics.Score >= policy.MinScore,
		},
		{
			Rule:     "min_success_rate",
			Required: strconv.FormatFloat(policy.MinSuccessRate, 'f', -1, 64),
			Actual:   strconv.FormatFloat(run.Metrics.SuccessRate, 'f', -1, 64),
			Passed:   run.Metrics.SuccessRate >= policy.MinSuccessRate,
		},
		{
			Rule:     "max_p95_latency",
			Required: strconv.FormatInt(policy.MaxP95Latency, 10),
			Actual:   strconv.FormatInt(run.Metrics.P95Latency, 10),
			Passed:   run.Metrics.P95Latency <= policy.MaxP95Latency,
		},
		{
			Rule:     "max_avg_tokens",
			Required: strconv.FormatFloat(policy.MaxAvgTokens, 'f', -1, 64),
			Actual:   strconv.FormatFloat(run.Metrics.AvgTokens, 'f', -1, 64),
			Passed:   run.Metrics.AvgTokens <= policy.MaxAvgTokens,
		},
	}

	passed := true
	for _, d := range details {
		if !d.Passed {
			passed = false
			break
		}
	}

	return GateRunResponse{
		Passed:      passed,
		RunID:       runID,
		PolicyID:    req.PolicyID,
		Details:     details,
		EvaluatedAt: c.nowFunc().Format(time.RFC3339),
	}, nil
}

type datasetCasesFile struct {
	Manifest dataset.Manifest         `json:"manifest"`
	Cases    []dataset.EvaluationCase `json:"cases"`
}

type datasetsEnvelope struct {
	Datasets []DatasetDetailDTO `json:"datasets"`
}

type runsEnvelope struct {
	Runs []RunResultDTO `json:"runs"`
}

type localPolicy struct {
	PolicyID       string  `json:"policy_id"`
	Name           string  `json:"name"`
	MinScore       float64 `json:"min_score"`
	MinSuccessRate float64 `json:"min_success_rate"`
	MaxP95Latency  int64   `json:"max_p95_latency"`
	MaxAvgTokens   float64 `json:"max_avg_tokens"`
}

type policyEnvelope struct {
	Policies []localPolicy `json:"policies"`
}

func (c *LocalClient) readDatasets() ([]DatasetDetailDTO, error) {
	path := filepath.Join(c.rootDir, "datasets.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []DatasetDetailDTO{}, nil
		}
		return nil, fmt.Errorf("read datasets: %w", err)
	}

	var envelope datasetsEnvelope
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Datasets != nil {
		return envelope.Datasets, nil
	}

	var plain []DatasetDetailDTO
	if err := json.Unmarshal(data, &plain); err != nil {
		return nil, fmt.Errorf("decode datasets: %w", err)
	}
	return plain, nil
}

func (c *LocalClient) readDatasetCases(datasetID string) (datasetCasesFile, error) {
	path := filepath.Join(c.rootDir, "dataset_cases", datasetID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return datasetCasesFile{}, &APIError{StatusCode: 404, Payload: APIErrorPayload{Code: 404, Message: "dataset cases not found"}}
		}
		return datasetCasesFile{}, fmt.Errorf("read dataset cases: %w", err)
	}

	var out datasetCasesFile
	if err := json.Unmarshal(data, &out); err != nil {
		return datasetCasesFile{}, fmt.Errorf("decode dataset cases: %w", err)
	}
	return out, nil
}

func (c *LocalClient) readRuns() ([]RunResultDTO, error) {
	path := filepath.Join(c.rootDir, "runs.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []RunResultDTO{}, nil
		}
		return nil, fmt.Errorf("read runs: %w", err)
	}

	var envelope runsEnvelope
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Runs != nil {
		return envelope.Runs, nil
	}

	var plain []RunResultDTO
	if err := json.Unmarshal(data, &plain); err != nil {
		return nil, fmt.Errorf("decode runs: %w", err)
	}
	return plain, nil
}

func (c *LocalClient) writeRuns(runs []RunResultDTO) error {
	if err := os.MkdirAll(c.rootDir, 0o755); err != nil {
		return fmt.Errorf("create mock data dir: %w", err)
	}
	path := filepath.Join(c.rootDir, "runs.json")
	payload, err := json.MarshalIndent(runsEnvelope{Runs: runs}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode runs: %w", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write runs: %w", err)
	}
	return nil
}

func (c *LocalClient) readPolicy(policyID string) (localPolicy, error) {
	path := filepath.Join(c.rootDir, "policies.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return localPolicy{}, ErrNotFound
		}
		return localPolicy{}, fmt.Errorf("read policies: %w", err)
	}

	var envelope policyEnvelope
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Policies != nil {
		for _, item := range envelope.Policies {
			if item.PolicyID == policyID {
				return item, nil
			}
		}
		return localPolicy{}, ErrNotFound
	}

	var plain []localPolicy
	if err := json.Unmarshal(data, &plain); err != nil {
		return localPolicy{}, fmt.Errorf("decode policies: %w", err)
	}
	for _, item := range plain {
		if item.PolicyID == policyID {
			return item, nil
		}
	}
	return localPolicy{}, ErrNotFound
}

func clampPage(total, limit, offset, defaultLimit int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return offset, end
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func datasetHasTag(cases []dataset.EvaluationCase, tag string) bool {
	for _, item := range cases {
		if contains(item.Tags, tag) {
			return true
		}
	}
	return false
}

func computeDatasetChecksum(manifest dataset.Manifest, cases []dataset.EvaluationCase) string {
	payload := struct {
		Manifest dataset.Manifest         `json:"manifest"`
		Cases    []dataset.EvaluationCase `json:"cases"`
	}{Manifest: manifest, Cases: cases}
	data, _ := json.Marshal(payload)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
