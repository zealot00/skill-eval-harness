package runner

import (
	"strings"
	"testing"
)

func TestComputeDriftReport(t *testing.T) {
	t.Parallel()

	current := RunResult{
		Results: []CaseRunResult{
			{Output: map[string]any{"answer": "hello world"}},
			{Output: map[string]any{"answer": "cat dog"}},
		},
	}
	baseline := RunResult{
		Results: []CaseRunResult{
			{Output: map[string]any{"answer": "hello world"}},
			{Output: map[string]any{"answer": "apple banana"}},
		},
	}

	got := ComputeDriftReport(current, baseline, 0.95)
	if len(got.Cases) != 2 {
		t.Fatalf("len(Cases) = %d, want 2", len(got.Cases))
	}

	if got.Cases[0].Similarity != 1 {
		t.Fatalf("Cases[0].Similarity = %v, want 1", got.Cases[0].Similarity)
	}

	if !got.Cases[1].Drift {
		t.Fatal("Cases[1].Drift = false, want true")
	}

	if !got.DriftDetected {
		t.Fatal("DriftDetected = false, want true")
	}
}

func TestFormatDriftSummary(t *testing.T) {
	t.Parallel()

	report := DriftReport{
		AverageSimilarity: 0.5,
		DriftDetected:     true,
		Cases: []CaseDriftResult{
			{CaseIndex: 1, Drift: true},
		},
	}

	got := FormatDriftSummary(report)
	for _, want := range []string{
		"Drift report",
		"average_similarity: 0.500000",
		"drift_detected: true",
		"drift_cases: 1",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary %q does not contain %q", got, want)
		}
	}
}
