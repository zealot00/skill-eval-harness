package runner

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"

	"skill-eval-harness/internal/dataset"
)

// RunResult captures the aggregate outcome of a run across multiple cases.
type RunResult struct {
	Success        bool            `json:"success"`
	LatencyMS      int64           `json:"latency_ms"`
	TokenUsage     int64           `json:"token_usage"`
	Output         map[string]any  `json:"output"`
	Error          string          `json:"error"`
	Results        []CaseRunResult `json:"results,omitempty"`
	Metrics        *RunMetrics     `json:"metrics,omitempty"`
	GitCommit      string          `json:"git_commit"`
	ModelName      string          `json:"model_name"`
	Timestamp      string          `json:"timestamp"`
	DatasetVersion string          `json:"dataset_version"`
	DatasetHash    string          `json:"dataset_hash"`
	RunID          string          `json:"run_id"`
	Seed           int64           `json:"seed,omitempty"`
}

// CaseRunResult captures the outcome of running a single evaluation case.
type CaseRunResult struct {
	Success          bool           `json:"success"`
	LatencyMS        int64          `json:"latency_ms"`
	TokenUsage       int64          `json:"token_usage"`
	Output           map[string]any `json:"output"`
	Error            string         `json:"error"`
	Classification   string         `json:"classification"`
	FailureClusterID string         `json:"failure_cluster_id,omitempty"`
	Trajectory       Trajectory     `json:"trajectory"`
}

// SkillResult captures the structured output produced by a skill runtime.
type SkillResult struct {
	Success    bool           `json:"success"`
	LatencyMS  int64          `json:"latency_ms"`
	TokenUsage int64          `json:"token_usage"`
	Output     map[string]any `json:"output"`
	Error      string         `json:"error"`
	Trajectory Trajectory     `json:"trajectory"`
}

// Trajectory captures execution trace details for a case.
type Trajectory struct {
	Steps                   []string `json:"steps"`
	ToolCalls               []string `json:"tool_calls"`
	ReasoningTokensEstimate int64    `json:"reasoning_tokens_estimate"`
}

// RunMetrics captures aggregate scoring information for a run.
type RunMetrics struct {
	SuccessRate          float64 `json:"success_rate"`
	AvgTokens            float64 `json:"avg_tokens"`
	P95Latency           int64   `json:"p95_latency"`
	CostFactor           float64 `json:"cost_factor"`
	ClassificationFactor float64 `json:"classification_factor"`
	CostUSD              float64 `json:"cost_usd"`
	StabilityVariance    float64 `json:"stability_variance"`
	Score                float64 `json:"score"`
}

// SkillRuntime executes a skill against structured input.
type SkillRuntime interface {
	Execute(ctx context.Context, input map[string]any) (SkillResult, error)
}

// RunOptions configures case execution behavior.
type RunOptions struct {
	Workers        int
	CaseTimeout    time.Duration
	MaxRetries     int
	DatasetVersion string
	DatasetHash    string
	Seed           int64
}

var nowFunc = func() time.Time {
	return time.Now().UTC()
}

const usdPerToken = 0.000002

// RunCases executes evaluation cases sequentially for a given skill runtime.
func RunCases(skill string, cases []dataset.EvaluationCase, rt SkillRuntime) RunResult {
	return RunCasesWithOptions(skill, cases, rt, RunOptions{Workers: 1})
}

// RunCasesWithOptions executes evaluation cases with the configured worker count.
func RunCasesWithOptions(skill string, cases []dataset.EvaluationCase, rt SkillRuntime, opts RunOptions) RunResult {
	workers := opts.Workers
	if workers < 1 {
		workers = 1
	}

	results := make([]CaseRunResult, len(cases))
	type workItem struct {
		index int
		caze  dataset.EvaluationCase
	}

	jobs := make(chan workItem)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for job := range jobs {
				start := time.Now()
				skillResult, err := executeWithRetry(job.caze, rt, opts)
				latencyMS := time.Since(start).Milliseconds()
				if opts.Seed != 0 {
					latencyMS = 0
				}

				caseResult := CaseRunResult{
					Success:        err == nil && skillResult.Success,
					LatencyMS:      latencyMS,
					TokenUsage:     skillResult.TokenUsage,
					Output:         skillResult.Output,
					Error:          skillResult.Error,
					Classification: classifyCaseResult(err, skillResult),
					Trajectory:     skillResult.Trajectory,
				}

				if err != nil {
					caseResult.Error = err.Error()
				}

				if hashErr := validateOutputHash(job.caze, caseResult.Output); hashErr != nil {
					caseResult.Success = false
					caseResult.Error = hashErr.Error()
					caseResult.Classification = "semantic_failure"
				}

				results[job.index] = caseResult
			}
		}()
	}

	for index, evaluationCase := range cases {
		jobs <- workItem{index: index, caze: evaluationCase}
	}
	close(jobs)
	wg.Wait()

	return finalizeRunResult(skill, cases, results, opts)
}

func executeWithRetry(evaluationCase dataset.EvaluationCase, rt SkillRuntime, opts RunOptions) (SkillResult, error) {
	attempts := opts.MaxRetries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastResult SkillResult
	var lastErr error

	for attempt := 0; attempt < attempts; attempt++ {
		ctx := context.Background()
		cancel := func() {}
		if opts.CaseTimeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, opts.CaseTimeout)
		}

		result, err := rt.Execute(ctx, evaluationCase.Input)
		cancel()

		lastResult = result
		lastErr = err
		if err == nil && result.Success {
			return result, nil
		}
	}

	return lastResult, lastErr
}

func finalizeRunResult(skill string, cases []dataset.EvaluationCase, caseResults []CaseRunResult, opts RunOptions) RunResult {
	aggregatedResults := make([]CaseRunResult, 0, len(cases))
	runResult := RunResult{
		Success:        true,
		GitCommit:      resolveGitCommit(),
		ModelName:      resolveModelName(skill),
		Timestamp:      resolveTimestamp(opts.Seed),
		DatasetVersion: opts.DatasetVersion,
		DatasetHash:    opts.DatasetHash,
		RunID:          resolveRunID(opts.Seed),
		Seed:           opts.Seed,
		Output: map[string]any{
			"skill":      skill,
			"case_count": len(cases),
		},
	}

	var errorMessages []string
	for index, caseResult := range caseResults {
		evaluationCase := cases[index]

		runResult.LatencyMS += caseResult.LatencyMS
		runResult.TokenUsage += caseResult.TokenUsage
		aggregatedResults = append(aggregatedResults, caseResult)

		if !caseResult.Success {
			runResult.Success = false

			if caseResult.Error != "" {
				errorMessages = append(errorMessages, fmt.Sprintf("%s: %s", evaluationCase.CaseID, caseResult.Error))
			}
		}
	}

	if len(errorMessages) > 0 {
		runResult.Error = strings.Join(errorMessages, "; ")
	}

	runResult.Results = assignFailureClusters(aggregatedResults)
	runResult.Metrics = CalculateMetrics(aggregatedResults)
	return runResult
}

// CalculateMetrics derives scoring metrics from per-case run results.
func CalculateMetrics(results []CaseRunResult) *RunMetrics {
	if len(results) == 0 {
		return &RunMetrics{}
	}

	successCount := 0
	var totalTokens int64
	var classificationFactorTotal float64
	latencies := make([]int64, 0, len(results))

	for _, result := range results {
		if result.Success {
			successCount++
		}

		totalTokens += result.TokenUsage
		classificationFactorTotal += classificationWeight(result.Classification)
		latencies = append(latencies, result.LatencyMS)
	}

	slices.Sort(latencies)

	successRate := float64(successCount) / float64(len(results))
	avgTokens := float64(totalTokens) / float64(len(results))
	costFactor := 1 / (1 + avgTokens)
	metrics := &RunMetrics{
		SuccessRate:          roundFloat(successRate),
		AvgTokens:            roundFloat(avgTokens),
		P95Latency:           percentile(latencies, 0.95),
		CostFactor:           roundFloat(costFactor),
		ClassificationFactor: roundFloat(classificationFactorTotal / float64(len(results))),
		CostUSD:              roundFloat(float64(totalTokens) * usdPerToken),
	}

	run := RunResult{
		Results: results,
		Metrics: metrics,
	}
	metrics.Score = CalculateScore(run, DefaultScoreConfig)

	return metrics
}

func percentile(sortedValues []int64, p float64) int64 {
	if len(sortedValues) == 0 {
		return 0
	}

	index := int(math.Ceil(p*float64(len(sortedValues)))) - 1
	if index < 0 {
		index = 0
	}

	if index >= len(sortedValues) {
		index = len(sortedValues) - 1
	}

	return sortedValues[index]
}

func roundFloat(v float64) float64 {
	return math.Round(v*1_000_000) / 1_000_000
}

func classifyCaseResult(err error, result SkillResult) string {
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
			return "timeout"
		}

		message := strings.ToLower(err.Error())
		if strings.Contains(message, "validation") || strings.Contains(message, "invalid") {
			return "validation_error"
		}

		return "runtime_error"
	}

	if !result.Success {
		return "semantic_failure"
	}

	return ""
}

func classificationWeight(classification string) float64 {
	switch classification {
	case "":
		return 1
	case "semantic_failure":
		return 0.9
	case "validation_error":
		return 0.8
	case "runtime_error":
		return 0.7
	case "timeout":
		return 0.6
	default:
		return 0.7
	}
}

func validateOutputHash(evaluationCase dataset.EvaluationCase, output map[string]any) error {
	expectedHash, ok := evaluationCase.Expected["output_hash"]
	if !ok {
		return nil
	}

	expectedHashString, ok := expectedHash.(string)
	if !ok || strings.TrimSpace(expectedHashString) == "" {
		return fmt.Errorf("expected.output_hash must be a non-empty string")
	}

	actualHash, err := hashOutput(output)
	if err != nil {
		return fmt.Errorf("hash output: %w", err)
	}

	if actualHash != expectedHashString {
		return fmt.Errorf("output_hash mismatch: got %s want %s", actualHash, expectedHashString)
	}

	return nil
}

func hashOutput(output map[string]any) (string, error) {
	data, err := json.Marshal(output)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum), nil
}

func resolveGitCommit() string {
	if gitCommit := strings.TrimSpace(os.Getenv("SEH_GIT_COMMIT")); gitCommit != "" {
		return gitCommit
	}

	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	gitCommit := strings.TrimSpace(string(output))
	if gitCommit == "" {
		return "unknown"
	}

	return gitCommit
}

func resolveModelName(skill string) string {
	if modelName := strings.TrimSpace(os.Getenv("SEH_MODEL_NAME")); modelName != "" {
		return modelName
	}

	if strings.TrimSpace(skill) == "" {
		return "unknown"
	}

	return skill
}

func resolveTimestamp(seed int64) string {
	if seed != 0 {
		return time.Unix(seed, 0).UTC().Format(time.RFC3339)
	}

	return nowFunc().Format(time.RFC3339)
}

func resolveRunID(seed int64) string {
	if seed != 0 {
		return fmt.Sprintf("seed-%d", seed)
	}

	return nowFunc().Format("20060102T150405Z")
}
