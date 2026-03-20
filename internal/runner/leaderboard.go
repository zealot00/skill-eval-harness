package runner

import (
	"fmt"
	"sort"
)

// SkillLeaderboardEntry captures aggregated historical scoring for a skill.
type SkillLeaderboardEntry struct {
	Skill          string  `json:"skill"`
	CompositeScore float64 `json:"composite_score"`
	Runs           int     `json:"runs"`
}

// BuildLeaderboard aggregates historical run scores and ranks skills by composite score.
func BuildLeaderboard() ([]SkillLeaderboardEntry, error) {
	runIDs, err := ListRuns()
	if err != nil {
		return nil, err
	}

	type aggregate struct {
		totalScore float64
		runs       int
	}

	aggregates := make(map[string]aggregate)
	for _, runID := range runIDs {
		run, err := LoadRun(runID)
		if err != nil {
			return nil, fmt.Errorf("load run %q: %w", runID, err)
		}

		skill, ok := run.Output["skill"].(string)
		if !ok || skill == "" {
			continue
		}

		metrics := run.Metrics
		if metrics == nil {
			metrics = CalculateMetrics(run.Results)
		}

		current := aggregates[skill]
		current.totalScore += metrics.Score
		current.runs++
		aggregates[skill] = current
	}

	entries := make([]SkillLeaderboardEntry, 0, len(aggregates))
	for skill, aggregate := range aggregates {
		entries = append(entries, SkillLeaderboardEntry{
			Skill:          skill,
			CompositeScore: roundFloat(aggregate.totalScore / float64(aggregate.runs)),
			Runs:           aggregate.runs,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].CompositeScore == entries[j].CompositeScore {
			return entries[i].Skill < entries[j].Skill
		}

		return entries[i].CompositeScore > entries[j].CompositeScore
	})

	return entries, nil
}
