package runner

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// MetricEvaluator evaluates a run and returns a metric value.
type MetricEvaluator interface {
	Evaluate(run RunResult) float64
}

// ScoreConfig defines weighted metric scoring.
type ScoreConfig struct {
	Weights map[string]float64 `json:"weights" yaml:"weights"`
}

// DefaultScoreConfig preserves the historical composite formula.
var DefaultScoreConfig = ScoreConfig{
	Weights: map[string]float64{
		"success_rate":          0.5,
		"cost_factor":           0.3,
		"classification_factor": 0.2,
	},
}

var metricEvaluators = map[string]MetricEvaluator{
	"success_rate":          SuccessRateEvaluator{},
	"latency":               LatencyEvaluator{},
	"cost_factor":           CostFactorEvaluator{},
	"classification_factor": ClassificationEvaluator{},
}

// RegisterMetricEvaluator adds or replaces a named metric evaluator.
func RegisterMetricEvaluator(name string, evaluator MetricEvaluator) {
	metricEvaluators[name] = evaluator
}

// LoadScoreConfig reads YAML scoring config from disk.
func LoadScoreConfig(path string) (ScoreConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ScoreConfig{}, fmt.Errorf("read score config %q: %w", path, err)
	}

	var cfg ScoreConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return ScoreConfig{}, fmt.Errorf("unmarshal score config %q: %w", path, err)
	}

	return cfg, nil
}

// CalculateScore evaluates a run using configured metric weights.
func CalculateScore(run RunResult, cfg ScoreConfig) float64 {
	if len(cfg.Weights) == 0 {
		cfg = DefaultScoreConfig
	}

	var score float64
	for name, weight := range cfg.Weights {
		evaluator, ok := metricEvaluators[name]
		if !ok {
			continue
		}

		score += evaluator.Evaluate(run) * weight
	}

	return roundFloat(score)
}

// SuccessRateEvaluator returns the run success rate metric.
type SuccessRateEvaluator struct{}

// Evaluate returns the success rate for the run.
func (SuccessRateEvaluator) Evaluate(run RunResult) float64 {
	if run.Metrics != nil {
		return run.Metrics.SuccessRate
	}

	if len(run.Results) == 0 {
		return 0
	}

	successCount := 0
	for _, result := range run.Results {
		if result.Success {
			successCount++
		}
	}

	return roundFloat(float64(successCount) / float64(len(run.Results)))
}

// LatencyEvaluator returns a latency-derived metric from p95 latency.
type LatencyEvaluator struct{}

// Evaluate returns a normalized latency score where lower p95 latency is better.
func (LatencyEvaluator) Evaluate(run RunResult) float64 {
	var p95Latency int64
	if run.Metrics != nil {
		p95Latency = run.Metrics.P95Latency
	} else {
		latencies := make([]int64, 0, len(run.Results))
		for _, result := range run.Results {
			latencies = append(latencies, result.LatencyMS)
		}
		p95Latency = percentile(latencies, 0.95)
	}

	return roundFloat(1 / (1 + float64(p95Latency)))
}

// CostFactorEvaluator returns the cost factor metric.
type CostFactorEvaluator struct{}

// Evaluate returns the cost factor where lower average token usage is better.
func (CostFactorEvaluator) Evaluate(run RunResult) float64 {
	if run.Metrics != nil {
		return run.Metrics.CostFactor
	}

	var avgTokens float64
	if len(run.Results) > 0 {
		var totalTokens int64
		for _, result := range run.Results {
			totalTokens += result.TokenUsage
		}
		avgTokens = float64(totalTokens) / float64(len(run.Results))
	}

	return roundFloat(1 / (1 + avgTokens))
}

// ClassificationEvaluator returns the aggregate classification factor metric.
type ClassificationEvaluator struct{}

// Evaluate returns the classification factor where more severe failures score lower.
func (ClassificationEvaluator) Evaluate(run RunResult) float64 {
	if run.Metrics != nil {
		return run.Metrics.ClassificationFactor
	}

	if len(run.Results) == 0 {
		return 0
	}

	var total float64
	for _, result := range run.Results {
		total += classificationWeight(result.Classification)
	}

	return roundFloat(total / float64(len(run.Results)))
}
