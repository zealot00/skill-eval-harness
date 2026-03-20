package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSimulateRoutingPolicies(t *testing.T) {
	historyDir := filepath.Join(t.TempDir(), ".history")
	t.Setenv("SEH_HISTORY_DIR", historyDir)

	runs := []RunResult{
		{
			RunID:  "run-1",
			Output: map[string]any{"skill": "skill-a"},
			Metrics: &RunMetrics{
				Score:   0.9,
				CostUSD: 0.3,
			},
		},
		{
			RunID:  "run-2",
			Output: map[string]any{"skill": "skill-b"},
			Metrics: &RunMetrics{
				Score:   0.7,
				CostUSD: 0.1,
			},
		},
		{
			RunID:  "run-3",
			Output: map[string]any{"skill": "skill-a"},
			Metrics: &RunMetrics{
				Score:   0.8,
				CostUSD: 0.2,
			},
		},
	}

	for _, run := range runs {
		if err := PersistRun(run); err != nil {
			t.Fatalf("PersistRun() error = %v", err)
		}
	}

	got, err := SimulateRoutingPolicies()
	if err != nil {
		t.Fatalf("SimulateRoutingPolicies() error = %v", err)
	}

	if got.TotalRequests != 3 {
		t.Fatalf("TotalRequests = %d, want 3", got.TotalRequests)
	}

	if len(got.Policies) != 3 {
		t.Fatalf("len(Policies) = %d, want 3", len(got.Policies))
	}

	if got.Policies[0].Name != "round_robin" {
		t.Fatalf("Policies[0].Name = %q, want round_robin", got.Policies[0].Name)
	}

	if got.Policies[1].Name != "best_score" || got.Policies[1].SelectedSkill != "skill-a" {
		t.Fatalf("Policies[1] = %#v, want best_score on skill-a", got.Policies[1])
	}

	if got.Policies[2].Name != "cost_aware" || got.Policies[2].SelectedSkill != "skill-b" {
		t.Fatalf("Policies[2] = %#v, want cost_aware on skill-b", got.Policies[2])
	}
}

func TestSimulationReportJSONGenerated(t *testing.T) {
	t.Parallel()

	report := RoutingSimulationReport{
		TotalRequests: 1,
		Policies: []RoutingPolicyReport{
			{Name: "round_robin", RequestsRouted: 1, CompositeScore: 0.8, CostUSD: 0.1},
		},
	}

	path := filepath.Join(t.TempDir(), "simulation.json")
	if err := WriteJSON(path, report); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var got RoutingSimulationReport
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got.TotalRequests != 1 || len(got.Policies) != 1 {
		t.Fatalf("decoded report = %#v, want generated report", got)
	}
}
