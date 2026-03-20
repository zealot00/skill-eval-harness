package runner

// RunComparison captures delta metrics between a current run and a baseline run.
type RunComparison struct {
	SuccessRateDelta float64 `json:"success_rate_delta"`
	LatencyDelta     int64   `json:"latency_delta"`
	TokenDelta       int64   `json:"token_delta"`
}

// CompareRuns computes delta metrics as current minus baseline.
func CompareRuns(current RunResult, baseline RunResult) RunComparison {
	currentMetrics := current.Metrics
	if currentMetrics == nil {
		currentMetrics = CalculateMetrics(current.Results)
	}

	baselineMetrics := baseline.Metrics
	if baselineMetrics == nil {
		baselineMetrics = CalculateMetrics(baseline.Results)
	}

	return RunComparison{
		SuccessRateDelta: roundFloat(currentMetrics.SuccessRate - baselineMetrics.SuccessRate),
		LatencyDelta:     currentMetrics.P95Latency - baselineMetrics.P95Latency,
		TokenDelta:       int64(currentMetrics.AvgTokens - baselineMetrics.AvgTokens),
	}
}

// HasRegression reports whether the comparison indicates a regression.
func (c RunComparison) HasRegression() bool {
	return c.SuccessRateDelta < 0 || c.LatencyDelta > 0 || c.TokenDelta > 0
}
