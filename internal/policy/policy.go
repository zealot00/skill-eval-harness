package policy

import (
	"fmt"
	"os"

	"skill-eval-harness/internal/runner"

	"gopkg.in/yaml.v3"
)

// Policy defines thresholds for accepting a scored run.
type Policy struct {
	MinScore       float64 `json:"min_score" yaml:"min_score"`
	MinSuccessRate float64 `json:"min_success_rate" yaml:"min_success_rate"`
	MaxP95Latency  int64   `json:"max_p95_latency" yaml:"max_p95_latency"`
	MaxAvgTokens   float64 `json:"max_avg_tokens" yaml:"max_avg_tokens"`
}

// Load reads a policy definition from a YAML file.
func Load(path string) (Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, fmt.Errorf("read policy %q: %w", path, err)
	}

	var policy Policy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return Policy{}, fmt.Errorf("unmarshal policy %q: %w", path, err)
	}

	return policy, nil
}

// Evaluate checks whether a scored run satisfies the configured policy.
func Evaluate(report runner.RunResult, policy Policy) error {
	if report.Metrics == nil {
		return fmt.Errorf("report metrics are required")
	}

	if report.Metrics.Score < policy.MinScore {
		return fmt.Errorf("score %.6f is below minimum %.6f", report.Metrics.Score, policy.MinScore)
	}

	if report.Metrics.SuccessRate < policy.MinSuccessRate {
		return fmt.Errorf("success_rate %.6f is below minimum %.6f", report.Metrics.SuccessRate, policy.MinSuccessRate)
	}

	if report.Metrics.P95Latency > policy.MaxP95Latency {
		return fmt.Errorf("p95_latency %d exceeds maximum %d", report.Metrics.P95Latency, policy.MaxP95Latency)
	}

	if report.Metrics.AvgTokens > policy.MaxAvgTokens {
		return fmt.Errorf("avg_tokens %.6f exceeds maximum %.6f", report.Metrics.AvgTokens, policy.MaxAvgTokens)
	}

	return nil
}
