package runner

import (
	"fmt"
	"sort"
)

// ParetoFrontierReport captures non-dominated historical runs by cost and score.
type ParetoFrontierReport struct {
	Points []ParetoFrontierPoint `json:"points"`
}

// ParetoFrontierPoint is one point on the cost-vs-score frontier.
type ParetoFrontierPoint struct {
	RunID          string  `json:"run_id"`
	Skill          string  `json:"skill"`
	CostUSD        float64 `json:"cost_usd"`
	CompositeScore float64 `json:"composite_score"`
}

// BuildParetoFrontier computes the non-dominated set of historical runs.
func BuildParetoFrontier() (ParetoFrontierReport, error) {
	runIDs, err := ListRuns()
	if err != nil {
		return ParetoFrontierReport{}, err
	}

	points := make([]ParetoFrontierPoint, 0, len(runIDs))
	for _, runID := range runIDs {
		run, err := LoadRun(runID)
		if err != nil {
			return ParetoFrontierReport{}, fmt.Errorf("load run %q: %w", runID, err)
		}

		metrics := run.Metrics
		if metrics == nil {
			metrics = CalculateMetrics(run.Results)
		}

		skill, _ := run.Output["skill"].(string)
		points = append(points, ParetoFrontierPoint{
			RunID:          run.RunID,
			Skill:          skill,
			CostUSD:        metrics.CostUSD,
			CompositeScore: metrics.Score,
		})
	}

	frontier := make([]ParetoFrontierPoint, 0, len(points))
	for _, point := range points {
		dominated := false
		for _, other := range points {
			if other.RunID == point.RunID {
				continue
			}

			betterOrEqualCost := other.CostUSD <= point.CostUSD
			betterOrEqualScore := other.CompositeScore >= point.CompositeScore
			strictlyBetter := other.CostUSD < point.CostUSD || other.CompositeScore > point.CompositeScore
			if betterOrEqualCost && betterOrEqualScore && strictlyBetter {
				dominated = true
				break
			}
		}

		if !dominated {
			frontier = append(frontier, point)
		}
	}

	sort.Slice(frontier, func(i, j int) bool {
		if frontier[i].CostUSD == frontier[j].CostUSD {
			if frontier[i].CompositeScore == frontier[j].CompositeScore {
				return frontier[i].RunID < frontier[j].RunID
			}
			return frontier[i].CompositeScore > frontier[j].CompositeScore
		}
		return frontier[i].CostUSD < frontier[j].CostUSD
	})

	return ParetoFrontierReport{Points: frontier}, nil
}
