package runner

import (
	"fmt"
	"sort"
)

// RoutingSimulationReport captures simulated routing outcomes across policies.
type RoutingSimulationReport struct {
	TotalRequests int                   `json:"total_requests"`
	Policies      []RoutingPolicyReport `json:"policies"`
}

// RoutingPolicyReport captures a single routing policy summary.
type RoutingPolicyReport struct {
	Name           string         `json:"name"`
	SelectedSkill  string         `json:"selected_skill,omitempty"`
	RequestsRouted int            `json:"requests_routed"`
	CompositeScore float64        `json:"composite_score"`
	CostUSD        float64        `json:"cost_usd"`
	Allocations    map[string]int `json:"allocations,omitempty"`
}

type skillSimulationStats struct {
	Skill          string
	CompositeScore float64
	CostUSD        float64
	Runs           int
}

// SimulateRoutingPolicies builds a report for round_robin, best_score, and cost_aware policies.
func SimulateRoutingPolicies() (RoutingSimulationReport, error) {
	stats, totalRequests, err := loadSimulationStats()
	if err != nil {
		return RoutingSimulationReport{}, err
	}

	report := RoutingSimulationReport{
		TotalRequests: totalRequests,
		Policies: []RoutingPolicyReport{
			simulateRoundRobin(stats, totalRequests),
			simulateBestScore(stats, totalRequests),
			simulateCostAware(stats, totalRequests),
		},
	}

	return report, nil
}

func loadSimulationStats() ([]skillSimulationStats, int, error) {
	runIDs, err := ListRuns()
	if err != nil {
		return nil, 0, err
	}

	type aggregate struct {
		totalScore float64
		totalCost  float64
		runs       int
	}

	aggregates := make(map[string]aggregate)
	totalRequests := 0
	for _, runID := range runIDs {
		run, err := LoadRun(runID)
		if err != nil {
			return nil, 0, fmt.Errorf("load run %q: %w", runID, err)
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
		current.totalCost += metrics.CostUSD
		current.runs++
		aggregates[skill] = current
		totalRequests++
	}

	stats := make([]skillSimulationStats, 0, len(aggregates))
	for skill, aggregate := range aggregates {
		stats = append(stats, skillSimulationStats{
			Skill:          skill,
			CompositeScore: roundFloat(aggregate.totalScore / float64(aggregate.runs)),
			CostUSD:        roundFloat(aggregate.totalCost / float64(aggregate.runs)),
			Runs:           aggregate.runs,
		})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Skill < stats[j].Skill
	})

	return stats, totalRequests, nil
}

func simulateRoundRobin(stats []skillSimulationStats, totalRequests int) RoutingPolicyReport {
	report := RoutingPolicyReport{
		Name:           "round_robin",
		RequestsRouted: totalRequests,
		Allocations:    make(map[string]int, len(stats)),
	}
	if len(stats) == 0 || totalRequests == 0 {
		return report
	}

	var totalScore float64
	var totalCost float64
	for i := 0; i < totalRequests; i++ {
		selected := stats[i%len(stats)]
		report.Allocations[selected.Skill]++
		totalScore += selected.CompositeScore
		totalCost += selected.CostUSD
	}

	report.CompositeScore = roundFloat(totalScore / float64(totalRequests))
	report.CostUSD = roundFloat(totalCost)
	return report
}

func simulateBestScore(stats []skillSimulationStats, totalRequests int) RoutingPolicyReport {
	report := RoutingPolicyReport{
		Name:           "best_score",
		RequestsRouted: totalRequests,
		Allocations:    make(map[string]int),
	}
	if len(stats) == 0 || totalRequests == 0 {
		return report
	}

	best := stats[0]
	for _, stat := range stats[1:] {
		if stat.CompositeScore > best.CompositeScore || (stat.CompositeScore == best.CompositeScore && stat.Skill < best.Skill) {
			best = stat
		}
	}

	report.SelectedSkill = best.Skill
	report.Allocations[best.Skill] = totalRequests
	report.CompositeScore = best.CompositeScore
	report.CostUSD = roundFloat(best.CostUSD * float64(totalRequests))
	return report
}

func simulateCostAware(stats []skillSimulationStats, totalRequests int) RoutingPolicyReport {
	report := RoutingPolicyReport{
		Name:           "cost_aware",
		RequestsRouted: totalRequests,
		Allocations:    make(map[string]int),
	}
	if len(stats) == 0 || totalRequests == 0 {
		return report
	}

	best := stats[0]
	bestValue := scorePerDollar(best)
	for _, stat := range stats[1:] {
		value := scorePerDollar(stat)
		if value > bestValue || (value == bestValue && stat.Skill < best.Skill) {
			best = stat
			bestValue = value
		}
	}

	report.SelectedSkill = best.Skill
	report.Allocations[best.Skill] = totalRequests
	report.CompositeScore = best.CompositeScore
	report.CostUSD = roundFloat(best.CostUSD * float64(totalRequests))
	return report
}

func scorePerDollar(stat skillSimulationStats) float64 {
	if stat.CostUSD == 0 {
		return stat.CompositeScore
	}

	return roundFloat(stat.CompositeScore / stat.CostUSD)
}
