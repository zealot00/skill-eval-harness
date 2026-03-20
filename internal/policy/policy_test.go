package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"skill-eval-harness/internal/runner"

	"gopkg.in/yaml.v3"
)

func TestPolicyYAMLParse(t *testing.T) {
	t.Parallel()

	data := []byte(`
min_score: 0.8
min_success_rate: 0.95
max_p95_latency: 1500
max_avg_tokens: 250
`)

	var got Policy
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if got.MinScore != 0.8 {
		t.Fatalf("MinScore = %v, want 0.8", got.MinScore)
	}

	if got.MinSuccessRate != 0.95 {
		t.Fatalf("MinSuccessRate = %v, want 0.95", got.MinSuccessRate)
	}

	if got.MaxP95Latency != 1500 {
		t.Fatalf("MaxP95Latency = %d, want 1500", got.MaxP95Latency)
	}

	if got.MaxAvgTokens != 250 {
		t.Fatalf("MaxAvgTokens = %v, want 250", got.MaxAvgTokens)
	}
}

func TestLoad(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "policy.yaml")
	data := []byte("min_score: 0.8\nmin_success_rate: 0.95\nmax_p95_latency: 1500\nmax_avg_tokens: 250\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.MinScore != 0.8 || got.MinSuccessRate != 0.95 || got.MaxP95Latency != 1500 || got.MaxAvgTokens != 250 {
		t.Fatalf("Load() = %#v, want parsed policy values", got)
	}
}

func TestEvaluate(t *testing.T) {
	t.Parallel()

	baseReport := runner.RunResult{
		Metrics: &runner.RunMetrics{
			Score:       0.9,
			SuccessRate: 0.95,
			P95Latency:  1000,
			AvgTokens:   200,
		},
	}
	basePolicy := Policy{
		MinScore:       0.8,
		MinSuccessRate: 0.9,
		MaxP95Latency:  1500,
		MaxAvgTokens:   250,
	}

	tests := []struct {
		name        string
		report      runner.RunResult
		policy      Policy
		wantErrPart string
	}{
		{
			name:   "accepts compliant report",
			report: baseReport,
			policy: basePolicy,
		},
		{
			name: "rejects low score",
			report: runner.RunResult{
				Metrics: &runner.RunMetrics{
					Score:       0.7,
					SuccessRate: 0.95,
					P95Latency:  1000,
					AvgTokens:   200,
				},
			},
			policy:      basePolicy,
			wantErrPart: "score",
		},
		{
			name: "rejects low success rate",
			report: runner.RunResult{
				Metrics: &runner.RunMetrics{
					Score:       0.9,
					SuccessRate: 0.85,
					P95Latency:  1000,
					AvgTokens:   200,
				},
			},
			policy:      basePolicy,
			wantErrPart: "success_rate",
		},
		{
			name: "rejects high p95 latency",
			report: runner.RunResult{
				Metrics: &runner.RunMetrics{
					Score:       0.9,
					SuccessRate: 0.95,
					P95Latency:  1600,
					AvgTokens:   200,
				},
			},
			policy:      basePolicy,
			wantErrPart: "p95_latency",
		},
		{
			name: "rejects high avg tokens",
			report: runner.RunResult{
				Metrics: &runner.RunMetrics{
					Score:       0.9,
					SuccessRate: 0.95,
					P95Latency:  1000,
					AvgTokens:   300,
				},
			},
			policy:      basePolicy,
			wantErrPart: "avg_tokens",
		},
		{
			name:        "rejects missing metrics",
			report:      runner.RunResult{},
			policy:      basePolicy,
			wantErrPart: "metrics",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Evaluate(tt.report, tt.policy)
			if tt.wantErrPart == "" {
				if err != nil {
					t.Fatalf("Evaluate() error = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("Evaluate() error = nil, want error containing %q", tt.wantErrPart)
			}

			if !strings.Contains(err.Error(), tt.wantErrPart) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErrPart)
			}
		})
	}
}
