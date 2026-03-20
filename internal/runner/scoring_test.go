package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCalculateMetrics(t *testing.T) {
	t.Parallel()

	results := []CaseRunResult{
		{Success: true, TokenUsage: 10, LatencyMS: 100, Classification: ""},
		{Success: false, TokenUsage: 20, LatencyMS: 200, Classification: "semantic_failure"},
		{Success: true, TokenUsage: 30, LatencyMS: 300, Classification: ""},
		{Success: true, TokenUsage: 40, LatencyMS: 400, Classification: ""},
	}

	got := CalculateMetrics(results)
	if got == nil {
		t.Fatal("CalculateMetrics() = nil, want metrics")
	}

	if got.SuccessRate != 0.75 {
		t.Fatalf("SuccessRate = %v, want 0.75", got.SuccessRate)
	}

	if got.AvgTokens != 25 {
		t.Fatalf("AvgTokens = %v, want 25", got.AvgTokens)
	}

	if got.P95Latency != 400 {
		t.Fatalf("P95Latency = %d, want 400", got.P95Latency)
	}

	if got.CostFactor != 0.038462 {
		t.Fatalf("CostFactor = %v, want 0.038462", got.CostFactor)
	}

	if got.ClassificationFactor != 0.975 {
		t.Fatalf("ClassificationFactor = %v, want 0.975", got.ClassificationFactor)
	}

	if got.CostUSD != 0.0002 {
		t.Fatalf("CostUSD = %v, want 0.0002", got.CostUSD)
	}

	if got.StabilityVariance != 0 {
		t.Fatalf("StabilityVariance = %v, want 0", got.StabilityVariance)
	}

	if got.Score != 0.581539 {
		t.Fatalf("Score = %v, want 0.581539", got.Score)
	}
}

func TestCalculateMetricsEmpty(t *testing.T) {
	t.Parallel()

	got := CalculateMetrics(nil)
	if got == nil {
		t.Fatal("CalculateMetrics() = nil, want metrics")
	}

	if got.SuccessRate != 0 {
		t.Fatalf("SuccessRate = %v, want 0", got.SuccessRate)
	}

	if got.AvgTokens != 0 {
		t.Fatalf("AvgTokens = %v, want 0", got.AvgTokens)
	}

	if got.P95Latency != 0 {
		t.Fatalf("P95Latency = %d, want 0", got.P95Latency)
	}

	if got.CostFactor != 0 {
		t.Fatalf("CostFactor = %v, want 0", got.CostFactor)
	}

	if got.ClassificationFactor != 0 {
		t.Fatalf("ClassificationFactor = %v, want 0", got.ClassificationFactor)
	}

	if got.CostUSD != 0 {
		t.Fatalf("CostUSD = %v, want 0", got.CostUSD)
	}

	if got.StabilityVariance != 0 {
		t.Fatalf("StabilityVariance = %v, want 0", got.StabilityVariance)
	}

	if got.Score != 0 {
		t.Fatalf("Score = %v, want 0", got.Score)
	}
}

func TestCalculateScoreWithConfig(t *testing.T) {
	t.Parallel()

	run := RunResult{
		Results: []CaseRunResult{
			{Success: true, TokenUsage: 10, LatencyMS: 100, Classification: ""},
			{Success: false, TokenUsage: 20, LatencyMS: 200, Classification: "semantic_failure"},
			{Success: true, TokenUsage: 30, LatencyMS: 300, Classification: ""},
			{Success: true, TokenUsage: 40, LatencyMS: 400, Classification: ""},
		},
		Metrics: &RunMetrics{
			SuccessRate:          0.75,
			AvgTokens:            25,
			P95Latency:           400,
			CostFactor:           0.038462,
			ClassificationFactor: 0.975,
			CostUSD:              0.0002,
			StabilityVariance:    0,
		},
	}

	configPath := filepath.Join(t.TempDir(), "score.yaml")
	data := []byte("weights:\n  success_rate: 0.8\n  latency: 0.2\n")
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := LoadScoreConfig(configPath)
	if err != nil {
		t.Fatalf("LoadScoreConfig() error = %v", err)
	}

	got := CalculateScore(run, cfg)
	if got != 0.600499 {
		t.Fatalf("CalculateScore() = %v, want 0.600499", got)
	}
}

func TestCalculateScoreUsesClassification(t *testing.T) {
	t.Parallel()

	runWithoutFailures := RunResult{
		Metrics: &RunMetrics{
			SuccessRate:          0.75,
			CostFactor:           0.038462,
			ClassificationFactor: 1,
			StabilityVariance:    0,
		},
	}
	runWithRuntimeFailure := RunResult{
		Metrics: &RunMetrics{
			SuccessRate:          0.75,
			CostFactor:           0.038462,
			ClassificationFactor: 0.7,
			StabilityVariance:    0,
		},
	}

	base := CalculateScore(runWithoutFailures, DefaultScoreConfig)
	penalized := CalculateScore(runWithRuntimeFailure, DefaultScoreConfig)
	if penalized >= base {
		t.Fatalf("penalized score = %v, want less than base score %v", penalized, base)
	}
}
