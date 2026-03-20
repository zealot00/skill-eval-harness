package runner

import (
	"path/filepath"
	"testing"
)

func TestBuildLeaderboard(t *testing.T) {
	historyDir := filepath.Join(t.TempDir(), ".history")
	t.Setenv("SEH_HISTORY_DIR", historyDir)

	runs := []RunResult{
		{
			RunID:  "run-1",
			Output: map[string]any{"skill": "skill-b"},
			Metrics: &RunMetrics{
				Score: 0.8,
			},
		},
		{
			RunID:  "run-2",
			Output: map[string]any{"skill": "skill-a"},
			Metrics: &RunMetrics{
				Score: 0.9,
			},
		},
		{
			RunID:  "run-3",
			Output: map[string]any{"skill": "skill-b"},
			Metrics: &RunMetrics{
				Score: 0.6,
			},
		},
	}

	for _, run := range runs {
		if err := PersistRun(run); err != nil {
			t.Fatalf("PersistRun() error = %v", err)
		}
	}

	got, err := BuildLeaderboard()
	if err != nil {
		t.Fatalf("BuildLeaderboard() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(BuildLeaderboard()) = %d, want 2", len(got))
	}

	if got[0].Skill != "skill-a" || got[0].CompositeScore != 0.9 || got[0].Runs != 1 {
		t.Fatalf("got[0] = %#v, want skill-a score 0.9 runs 1", got[0])
	}

	if got[1].Skill != "skill-b" || got[1].CompositeScore != 0.7 || got[1].Runs != 2 {
		t.Fatalf("got[1] = %#v, want skill-b score 0.7 runs 2", got[1])
	}
}
