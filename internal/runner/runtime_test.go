package runner

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"skill-eval-harness/internal/dataset"
)

func TestResultJSONMarshalStable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   any
		want string
	}{
		{
			name: "run result",
			in: RunResult{
				Success:    true,
				LatencyMS:  125,
				TokenUsage: 42,
				Output:     map[string]any{"status": "ok"},
				Error:      "",
				Results: []CaseRunResult{
					{
						Success:          true,
						LatencyMS:        25,
						TokenUsage:       7,
						Output:           map[string]any{"status": "ok"},
						Error:            "",
						Classification:   "",
						FailureClusterID: "",
						Trajectory: Trajectory{
							Steps:                   []string{"step-1"},
							ToolCalls:               []string{"tool-1"},
							ReasoningTokensEstimate: 11,
						},
					},
				},
				Metrics: &RunMetrics{
					SuccessRate:          1,
					AvgTokens:            42,
					P95Latency:           25,
					CostFactor:           0.023256,
					ClassificationFactor: 1,
					CostUSD:              0,
					StabilityVariance:    0,
					Score:                0.706977,
				},
				GitCommit:      "abc123",
				ModelName:      "demo-skill",
				Timestamp:      "2026-03-20T00:00:00Z",
				DatasetVersion: "v1",
				DatasetHash:    "hash123",
				RunID:          "run-1",
			},
			want: `{"success":true,"latency_ms":125,"token_usage":42,"output":{"status":"ok"},"error":"","results":[{"success":true,"latency_ms":25,"token_usage":7,"output":{"status":"ok"},"error":"","classification":"","trajectory":{"steps":["step-1"],"tool_calls":["tool-1"],"reasoning_tokens_estimate":11}}],"metrics":{"success_rate":1,"avg_tokens":42,"p95_latency":25,"cost_factor":0.023256,"classification_factor":1,"cost_usd":0,"stability_variance":0,"score":0.706977},"git_commit":"abc123","model_name":"demo-skill","timestamp":"2026-03-20T00:00:00Z","dataset_version":"v1","dataset_hash":"hash123","run_id":"run-1"}`,
		},
		{
			name: "case run result",
			in: CaseRunResult{
				Success:          false,
				LatencyMS:        30,
				TokenUsage:       9,
				Output:           map[string]any{"status": "failed"},
				Error:            "boom",
				Classification:   "runtime_error",
				FailureClusterID: "cluster-1",
				Trajectory: Trajectory{
					Steps:                   []string{"step-1"},
					ToolCalls:               []string{"tool-1"},
					ReasoningTokensEstimate: 5,
				},
			},
			want: `{"success":false,"latency_ms":30,"token_usage":9,"output":{"status":"failed"},"error":"boom","classification":"runtime_error","failure_cluster_id":"cluster-1","trajectory":{"steps":["step-1"],"tool_calls":["tool-1"],"reasoning_tokens_estimate":5}}`,
		},
		{
			name: "skill result",
			in: SkillResult{
				Success:    true,
				LatencyMS:  15,
				TokenUsage: 3,
				Output:     map[string]any{"status": "ok"},
				Error:      "",
				Trajectory: Trajectory{
					Steps:                   []string{"step-1"},
					ToolCalls:               []string{"tool-1"},
					ReasoningTokensEstimate: 2,
				},
			},
			want: `{"success":true,"latency_ms":15,"token_usage":3,"output":{"status":"ok"},"error":"","trajectory":{"steps":["step-1"],"tool_calls":["tool-1"],"reasoning_tokens_estimate":2}}`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(tt.in)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			if string(got) != tt.want {
				t.Fatalf("json = %s, want %s", string(got), tt.want)
			}
		})
	}
}

func TestRunCases(t *testing.T) {
	t.Parallel()

	t.Run("sequentially executes every case", func(t *testing.T) {
		t.Parallel()

		rt := &mockSkillRuntime{
			results: []SkillResult{
				{Success: true, TokenUsage: 10, Output: map[string]any{"case": "one"}},
				{Success: true, TokenUsage: 20, Output: map[string]any{"case": "two"}},
				{Success: true, TokenUsage: 30, Output: map[string]any{"case": "three"}},
			},
		}
		cases := []dataset.EvaluationCase{
			{CaseID: "case-1", Skill: "seh", Input: map[string]any{"index": 1}},
			{CaseID: "case-2", Skill: "seh", Input: map[string]any{"index": 2}},
			{CaseID: "case-3", Skill: "seh", Input: map[string]any{"index": 3}},
		}

		got := RunCases("seh", cases, rt)

		if gotCount := rt.callCount; gotCount != 3 {
			t.Fatalf("call count = %d, want 3", gotCount)
		}

		if !got.Success {
			t.Fatalf("RunCases().Success = false, want true")
		}

		if gotCount := len(got.Results); gotCount != 3 {
			t.Fatalf("len(results) = %d, want 3", gotCount)
		}

		if got.TokenUsage != 60 {
			t.Fatalf("TokenUsage = %d, want 60", got.TokenUsage)
		}

		wantInputs := []map[string]any{
			{"index": 1},
			{"index": 2},
			{"index": 3},
		}
		if !reflect.DeepEqual(rt.inputs, wantInputs) {
			t.Fatalf("inputs = %#v, want %#v", rt.inputs, wantInputs)
		}
	})

	t.Run("captures runtime error", func(t *testing.T) {
		t.Parallel()

		rt := &mockSkillRuntime{
			results: []SkillResult{
				{Success: true, TokenUsage: 5, Output: map[string]any{"case": "one"}},
				{Success: false},
			},
			errs: []error{
				nil,
				errors.New("runtime failed"),
			},
		}
		cases := []dataset.EvaluationCase{
			{CaseID: "case-1", Skill: "seh", Input: map[string]any{"index": 1}},
			{CaseID: "case-2", Skill: "seh", Input: map[string]any{"index": 2}},
		}

		got := RunCases("seh", cases, rt)

		if got.Success {
			t.Fatal("RunCases().Success = true, want false")
		}

		if got.Error != "case-2: runtime failed" {
			t.Fatalf("Error = %q, want %q", got.Error, "case-2: runtime failed")
		}

		if got.Results[1].Error != "runtime failed" {
			t.Fatalf("results[1].Error = %q, want %q", got.Results[1].Error, "runtime failed")
		}

		if got.Results[1].Classification != "runtime_error" {
			t.Fatalf("results[1].Classification = %q, want %q", got.Results[1].Classification, "runtime_error")
		}

		if got.Results[1].FailureClusterID == "" {
			t.Fatal("results[1].FailureClusterID = empty, want cluster id")
		}

		if got.Metrics == nil {
			t.Fatal("Metrics = nil, want metrics")
		}
	})

	t.Run("clusters similar failure outputs", func(t *testing.T) {
		t.Parallel()

		rt := &mockSkillRuntime{
			results: []SkillResult{
				{Success: false, Output: map[string]any{"message": "timeout contacting upstream"}},
				{Success: false, Output: map[string]any{"message": "timeout contacting upstream service"}},
				{Success: false, Output: map[string]any{"message": "schema validation failed"}},
			},
		}
		cases := []dataset.EvaluationCase{
			{CaseID: "case-1", Skill: "seh", Input: map[string]any{"index": 1}},
			{CaseID: "case-2", Skill: "seh", Input: map[string]any{"index": 2}},
			{CaseID: "case-3", Skill: "seh", Input: map[string]any{"index": 3}},
		}

		got := RunCases("seh", cases, rt)
		if got.Results[0].FailureClusterID == "" {
			t.Fatal("results[0].FailureClusterID = empty, want cluster id")
		}
		if got.Results[0].FailureClusterID != got.Results[1].FailureClusterID {
			t.Fatalf("results[0].FailureClusterID = %q, results[1].FailureClusterID = %q, want same cluster", got.Results[0].FailureClusterID, got.Results[1].FailureClusterID)
		}
		if got.Results[2].FailureClusterID == got.Results[0].FailureClusterID {
			t.Fatalf("results[2].FailureClusterID = %q, want different cluster", got.Results[2].FailureClusterID)
		}
	})

	t.Run("case timeout cancels context", func(t *testing.T) {
		t.Parallel()

		rt := &timeoutRuntime{}
		cases := []dataset.EvaluationCase{
			{CaseID: "case-timeout", Skill: "seh", Input: map[string]any{"index": 1}},
		}

		got := RunCasesWithOptions("seh", cases, rt, RunOptions{
			Workers:     1,
			CaseTimeout: 20 * time.Millisecond,
		})

		if got.Success {
			t.Fatal("RunCasesWithOptions().Success = true, want false")
		}

		if len(got.Results) != 1 {
			t.Fatalf("len(results) = %d, want 1", len(got.Results))
		}

		if got.Results[0].Success {
			t.Fatal("results[0].Success = true, want false")
		}

		if !strings.Contains(got.Results[0].Error, context.DeadlineExceeded.Error()) {
			t.Fatalf("results[0].Error = %q, want deadline exceeded", got.Results[0].Error)
		}

		if got.Results[0].Classification != "timeout" {
			t.Fatalf("results[0].Classification = %q, want %q", got.Results[0].Classification, "timeout")
		}
	})

	t.Run("retries flaky runtime", func(t *testing.T) {
		t.Parallel()

		rt := &flakyRuntime{}
		cases := []dataset.EvaluationCase{
			{CaseID: "case-flaky", Skill: "seh", Input: map[string]any{"index": 1}},
		}

		got := RunCasesWithOptions("seh", cases, rt, RunOptions{
			Workers:    1,
			MaxRetries: 1,
		})

		if !got.Success {
			t.Fatal("RunCasesWithOptions().Success = false, want true")
		}

		if rt.calls != 2 {
			t.Fatalf("call count = %d, want 2", rt.calls)
		}

		if len(got.Results) != 1 {
			t.Fatalf("len(results) = %d, want 1", len(got.Results))
		}

		if !got.Results[0].Success {
			t.Fatal("results[0].Success = false, want true")
		}
	})

	t.Run("output hash mismatch triggers failure", func(t *testing.T) {
		t.Parallel()

		rt := &mockSkillRuntime{
			results: []SkillResult{
				{Success: true, Output: map[string]any{"status": "ok"}},
			},
		}
		cases := []dataset.EvaluationCase{
			{
				CaseID: "case-hash",
				Skill:  "seh",
				Input:  map[string]any{"index": 1},
				Expected: map[string]any{
					"output_hash": "deadbeef",
				},
			},
		}

		got := RunCases("seh", cases, rt)

		if got.Success {
			t.Fatal("RunCases().Success = true, want false")
		}

		if got.Results[0].Success {
			t.Fatal("results[0].Success = true, want false")
		}

		if got.Results[0].Classification != "semantic_failure" {
			t.Fatalf("results[0].Classification = %q, want %q", got.Results[0].Classification, "semantic_failure")
		}

		if !strings.Contains(got.Results[0].Error, "output_hash mismatch") {
			t.Fatalf("results[0].Error = %q, want output_hash mismatch", got.Results[0].Error)
		}
	})

	t.Run("auto injects runtime metadata", func(t *testing.T) {
		rt := &mockSkillRuntime{
			results: []SkillResult{
				{Success: true, TokenUsage: 1, Output: map[string]any{"case": "one"}},
			},
		}
		cases := []dataset.EvaluationCase{
			{CaseID: "case-1", Skill: "seh", Input: map[string]any{"index": 1}},
		}

		got := RunCases("seh", cases, rt)

		if got.GitCommit == "" {
			t.Fatal("GitCommit = empty, want auto-injected value")
		}

		if got.ModelName != "seh" {
			t.Fatalf("ModelName = %q, want %q", got.ModelName, "seh")
		}

		if _, err := time.Parse(time.RFC3339, got.Timestamp); err != nil {
			t.Fatalf("Timestamp = %q, want valid RFC3339: %v", got.Timestamp, err)
		}
	})
}

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "result.json")
	input := RunResult{
		Success:    true,
		LatencyMS:  125,
		TokenUsage: 42,
		Output:     map[string]any{"status": "ok"},
		Error:      "",
		Results: []CaseRunResult{
			{
				Success:          true,
				LatencyMS:        25,
				TokenUsage:       7,
				Output:           map[string]any{"status": "ok"},
				Error:            "",
				Classification:   "",
				FailureClusterID: "",
				Trajectory: Trajectory{
					Steps:                   []string{"step-1"},
					ToolCalls:               []string{"tool-1"},
					ReasoningTokensEstimate: 11,
				},
			},
		},
		Metrics: &RunMetrics{
			SuccessRate:          1,
			AvgTokens:            42,
			P95Latency:           25,
			CostFactor:           0.023256,
			ClassificationFactor: 1,
			CostUSD:              0,
			StabilityVariance:    0,
			Score:                0.706977,
		},
		GitCommit:      "abc123",
		ModelName:      "demo-skill",
		Timestamp:      "2026-03-20T00:00:00Z",
		DatasetVersion: "v1",
		DatasetHash:    "hash123",
		RunID:          "run-1",
	}

	if err := WriteJSON(path, input); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "write_json.golden"))
	if err != nil {
		t.Fatalf("ReadFile(golden) error = %v", err)
	}

	if string(got) != string(want) {
		t.Fatalf("written json = %s, want %s", string(got), string(want))
	}
}

type mockSkillRuntime struct {
	callCount int
	inputs    []map[string]any
	results   []SkillResult
	errs      []error
}

func (m *mockSkillRuntime) Execute(_ context.Context, input map[string]any) (SkillResult, error) {
	m.callCount++
	m.inputs = append(m.inputs, input)

	resultIndex := m.callCount - 1
	if resultIndex >= len(m.results) {
		return SkillResult{}, nil
	}

	var err error
	if resultIndex < len(m.errs) {
		err = m.errs[resultIndex]
	}

	return m.results[resultIndex], err
}

type timeoutRuntime struct{}

func (t *timeoutRuntime) Execute(ctx context.Context, input map[string]any) (SkillResult, error) {
	<-ctx.Done()
	return SkillResult{
		Success: false,
		Output:  input,
	}, ctx.Err()
}

type flakyRuntime struct {
	calls int
}

func (f *flakyRuntime) Execute(ctx context.Context, input map[string]any) (SkillResult, error) {
	f.calls++
	if f.calls == 1 {
		return SkillResult{
			Success: false,
			Output:  input,
			Error:   "temporary failure",
		}, errors.New("temporary failure")
	}

	return SkillResult{
		Success:    true,
		TokenUsage: int64(len(input)),
		Output:     input,
	}, nil
}
