package runner

import "testing"

func TestMetricEvaluators(t *testing.T) {
	t.Parallel()

	run := RunResult{
		Results: []CaseRunResult{
			{Success: true, LatencyMS: 100},
			{Success: false, LatencyMS: 200},
			{Success: true, LatencyMS: 300},
			{Success: true, LatencyMS: 400},
		},
		Metrics: &RunMetrics{
			SuccessRate: 0.75,
			P95Latency:  400,
		},
	}

	t.Run("success rate plugin", func(t *testing.T) {
		t.Parallel()

		got := SuccessRateEvaluator{}.Evaluate(run)
		if got != 0.75 {
			t.Fatalf("Evaluate() = %v, want 0.75", got)
		}
	})

	t.Run("latency plugin", func(t *testing.T) {
		t.Parallel()

		got := LatencyEvaluator{}.Evaluate(run)
		if got != 0.002494 {
			t.Fatalf("Evaluate() = %v, want 0.002494", got)
		}
	})
}
