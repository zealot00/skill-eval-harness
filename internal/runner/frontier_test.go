package runner

import (
	"path/filepath"
	"testing"
)

func TestBuildParetoFrontier(t *testing.T) {
	historyDir := filepath.Join(t.TempDir(), ".history")
	t.Setenv("SEH_HISTORY_DIR", historyDir)

	runs := []RunResult{
		{
			RunID:  "run-a",
			Output: map[string]any{"skill": "skill-a"},
			Metrics: &RunMetrics{
				CostUSD: 0.1,
				Score:   0.8,
			},
		},
		{
			RunID:  "run-b",
			Output: map[string]any{"skill": "skill-b"},
			Metrics: &RunMetrics{
				CostUSD: 0.2,
				Score:   0.7,
			},
		},
		{
			RunID:  "run-c",
			Output: map[string]any{"skill": "skill-c"},
			Metrics: &RunMetrics{
				CostUSD: 0.3,
				Score:   0.95,
			},
		},
	}

	for _, run := range runs {
		if err := PersistRun(run); err != nil {
			t.Fatalf("PersistRun() error = %v", err)
		}
	}

	got, err := BuildParetoFrontier()
	if err != nil {
		t.Fatalf("BuildParetoFrontier() error = %v", err)
	}

	if len(got.Points) != 2 {
		t.Fatalf("len(Points) = %d, want 2", len(got.Points))
	}

	if got.Points[0].RunID != "run-a" || got.Points[1].RunID != "run-c" {
		t.Fatalf("Points = %#v, want run-a and run-c", got.Points)
	}
}
