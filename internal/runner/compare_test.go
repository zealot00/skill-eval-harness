package runner

import "testing"

func TestCompareRuns(t *testing.T) {
	t.Parallel()

	current := RunResult{
		Metrics: &RunMetrics{
			SuccessRate: 0.8,
			P95Latency:  150,
			AvgTokens:   40,
		},
	}
	baseline := RunResult{
		Metrics: &RunMetrics{
			SuccessRate: 0.9,
			P95Latency:  100,
			AvgTokens:   30,
		},
	}

	got := CompareRuns(current, baseline)
	if got.SuccessRateDelta != -0.1 {
		t.Fatalf("SuccessRateDelta = %v, want -0.1", got.SuccessRateDelta)
	}

	if got.LatencyDelta != 50 {
		t.Fatalf("LatencyDelta = %d, want 50", got.LatencyDelta)
	}

	if got.TokenDelta != 10 {
		t.Fatalf("TokenDelta = %d, want 10", got.TokenDelta)
	}

	if !got.HasRegression() {
		t.Fatal("HasRegression() = false, want true")
	}
}
