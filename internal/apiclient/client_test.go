package apiclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"skill-eval-harness/internal/dataset"
)

func TestLocalClient_CreateRunAndGate(t *testing.T) {
	root := t.TempDir()
	mustWriteJSON(t, filepath.Join(root, "datasets.json"), map[string]any{
		"datasets": []map[string]any{{
			"dataset_id": "ds_1",
			"name":       "dataset-1",
			"version":    "v1",
			"owner":      "team-a",
			"case_count": 1,
			"manifest": map[string]any{
				"dataset_name": "dataset-1",
				"version":      "v1",
				"owner":        "team-a",
			},
			"checksum":   "",
			"created_at": "2026-03-01T00:00:00Z",
		}},
	})
	mustWriteJSON(t, filepath.Join(root, "dataset_cases", "ds_1.json"), map[string]any{
		"manifest": map[string]any{"dataset_name": "dataset-1", "version": "v1", "owner": "team-a"},
		"cases":    []dataset.EvaluationCase{{CaseID: "case-1", Skill: "demo", Input: map[string]any{"a": 1}, Expected: map[string]any{"ok": true}, Tags: []string{"golden"}}},
	})
	mustWriteJSON(t, filepath.Join(root, "policies.json"), map[string]any{
		"policies": []map[string]any{{
			"policy_id":        "pol_strict",
			"name":             "strict",
			"min_score":        0.8,
			"min_success_rate": 0.9,
			"max_p95_latency":  2000,
			"max_avg_tokens":   100,
		}},
	})

	client := NewLocalClient(root)
	created, err := client.CreateRun(context.Background(), RunResultDTO{
		DatasetID: "ds_1",
		Skill:     "demo-skill",
		Results: []CaseResultDTO{{
			CaseID:     "case-1",
			Success:    true,
			LatencyMS:  10,
			TokenUsage: 5,
		}},
		Metrics: RunMetricsDTO{SuccessRate: 1, AvgTokens: 5, P95Latency: 10, Score: 0.95},
	})
	if err != nil {
		t.Fatalf("CreateRun error = %v", err)
	}
	if created.RunID == "" {
		t.Fatal("CreateRun run_id is empty")
	}

	gated, err := client.GateRun(context.Background(), created.RunID, GateRunRequest{PolicyID: "pol_strict"})
	if err != nil {
		t.Fatalf("GateRun error = %v", err)
	}
	if !gated.Passed {
		t.Fatalf("GateRun passed = false, details = %#v", gated.Details)
	}

	runs, err := client.ListRuns(context.Background(), ListRunsQuery{DatasetID: "ds_1"})
	if err != nil {
		t.Fatalf("ListRuns error = %v", err)
	}
	if len(runs.Runs) != 1 {
		t.Fatalf("ListRuns len = %d, want 1", len(runs.Runs))
	}
}

func TestHybridClient_FallbackWhenRemoteUnavailable(t *testing.T) {
	root := t.TempDir()
	mustWriteJSON(t, filepath.Join(root, "datasets.json"), map[string]any{
		"datasets": []map[string]any{{
			"dataset_id": "ds_local",
			"name":       "local-dataset",
			"version":    "v1",
			"owner":      "team-local",
			"case_count": 0,
			"manifest": map[string]any{
				"dataset_name": "local-dataset",
				"version":      "v1",
				"owner":        "team-local",
			},
			"created_at": "2026-03-01T00:00:00Z",
		}},
	})

	client := NewHybridClient(Config{
		BaseURL:     "http://remote.invalid/v1",
		APIToken:    "token-1",
		MockDataDir: root,
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusServiceUnavailable,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"error":{"code":503,"message":"service unavailable"}}`)),
					Request:    req,
				}, nil
			}),
		},
	})

	resp, err := client.ListDatasets(context.Background(), ListDatasetsQuery{})
	if err != nil {
		t.Fatalf("ListDatasets error = %v", err)
	}
	if len(resp.Datasets) != 1 {
		t.Fatalf("datasets len = %d, want 1", len(resp.Datasets))
	}
	if resp.Datasets[0].DatasetID != "ds_local" {
		t.Fatalf("dataset_id = %s, want ds_local", resp.Datasets[0].DatasetID)
	}
}

func mustWriteJSON(t *testing.T, path string, v any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
