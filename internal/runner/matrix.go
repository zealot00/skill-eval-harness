package runner

import (
	"fmt"

	"skill-eval-harness/internal/dataset"
)

// RuntimeComparisonMatrix captures the same dataset executed against multiple runtimes.
type RuntimeComparisonMatrix struct {
	DatasetVersion string             `json:"dataset_version"`
	DatasetHash    string             `json:"dataset_hash"`
	CaseCount      int                `json:"case_count"`
	Runtimes       []RuntimeMatrixRow `json:"runtimes"`
}

// RuntimeMatrixRow captures per-runtime summary metrics for the matrix output.
type RuntimeMatrixRow struct {
	Runtime        string  `json:"runtime"`
	Success        bool    `json:"success"`
	SuccessRate    float64 `json:"success_rate"`
	P95Latency     int64   `json:"p95_latency"`
	AvgTokens      float64 `json:"avg_tokens"`
	CompositeScore float64 `json:"composite_score"`
	RunID          string  `json:"run_id"`
	Error          string  `json:"error"`
}

// BuildComparisonMatrix runs the same cases against multiple registered runtimes.
func BuildComparisonMatrix(runtimeNames []string, cases []dataset.EvaluationCase, opts RunOptions) (RuntimeComparisonMatrix, error) {
	matrix := RuntimeComparisonMatrix{
		DatasetVersion: opts.DatasetVersion,
		DatasetHash:    opts.DatasetHash,
		CaseCount:      len(cases),
		Runtimes:       make([]RuntimeMatrixRow, 0, len(runtimeNames)),
	}

	for _, runtimeName := range runtimeNames {
		rt, err := ResolveRuntime(runtimeName)
		if err != nil {
			return RuntimeComparisonMatrix{}, fmt.Errorf("resolve runtime %q: %w", runtimeName, err)
		}

		run := RunCasesWithOptions(runtimeName, cases, rt, opts)
		row := RuntimeMatrixRow{
			Runtime: runtimeName,
			Success: run.Success,
			RunID:   run.RunID,
			Error:   run.Error,
		}
		if run.Metrics != nil {
			row.SuccessRate = run.Metrics.SuccessRate
			row.P95Latency = run.Metrics.P95Latency
			row.AvgTokens = run.Metrics.AvgTokens
			row.CompositeScore = run.Metrics.Score
		}

		matrix.Runtimes = append(matrix.Runtimes, row)
	}

	return matrix, nil
}
