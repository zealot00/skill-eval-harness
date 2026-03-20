package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const historyDir = ".history"
const baselineFileName = "baseline"

func resolveHistoryDir() string {
	if dir := strings.TrimSpace(os.Getenv("SEH_HISTORY_DIR")); dir != "" {
		return dir
	}

	return historyDir
}

func baselinePath() string {
	return filepath.Join(resolveHistoryDir(), baselineFileName)
}

// PersistRun stores a run result under .history using its run ID.
func PersistRun(run RunResult) error {
	if strings.TrimSpace(run.RunID) == "" {
		return fmt.Errorf("run_id is required")
	}

	dir := resolveHistoryDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create history dir: %w", err)
	}

	path := filepath.Join(dir, run.RunID+".json")
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run history: %w", err)
	}

	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write history %q: %w", path, err)
	}

	return nil
}

// ListRuns returns run IDs found in .history.
func ListRuns() ([]string, error) {
	entries, err := os.ReadDir(resolveHistoryDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read history dir: %w", err)
	}

	runIDs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}

		runIDs = append(runIDs, strings.TrimSuffix(name, ".json"))
	}

	sort.Strings(runIDs)
	return runIDs, nil
}

// LoadRun reads a stored run by ID from .history.
func LoadRun(runID string) (RunResult, error) {
	path := filepath.Join(resolveHistoryDir(), runID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return RunResult{}, fmt.Errorf("read history %q: %w", path, err)
	}

	var run RunResult
	if err := json.Unmarshal(data, &run); err != nil {
		return RunResult{}, fmt.Errorf("unmarshal history %q: %w", path, err)
	}

	return run, nil
}

// PromoteBaseline updates the baseline pointer to the given persisted run ID.
func PromoteBaseline(runID string) error {
	if strings.TrimSpace(runID) == "" {
		return fmt.Errorf("run_id is required")
	}

	if _, err := LoadRun(runID); err != nil {
		return fmt.Errorf("load baseline run %q: %w", runID, err)
	}

	if err := os.MkdirAll(resolveHistoryDir(), 0o755); err != nil {
		return fmt.Errorf("create history dir: %w", err)
	}

	if err := os.WriteFile(baselinePath(), []byte(runID+"\n"), 0o644); err != nil {
		return fmt.Errorf("write baseline pointer: %w", err)
	}

	return nil
}

// LoadBaselineRunID returns the currently promoted baseline run ID.
func LoadBaselineRunID() (string, error) {
	data, err := os.ReadFile(baselinePath())
	if err != nil {
		return "", fmt.Errorf("read baseline pointer: %w", err)
	}

	runID := strings.TrimSpace(string(data))
	if runID == "" {
		return "", fmt.Errorf("baseline pointer is empty")
	}

	return runID, nil
}

// IngestRun reads a run JSON file and merges it into history by run ID.
func IngestRun(path string) (RunResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RunResult{}, fmt.Errorf("read run %q: %w", path, err)
	}

	var run RunResult
	if err := json.Unmarshal(data, &run); err != nil {
		return RunResult{}, fmt.Errorf("decode run %q: %w", path, err)
	}

	if err := PersistRun(run); err != nil {
		return RunResult{}, err
	}

	return run, nil
}

// ComputeStabilityVariance computes score variance across matching historical runs plus the current run.
func ComputeStabilityVariance(current RunResult) (float64, error) {
	runIDs, err := ListRuns()
	if err != nil {
		return 0, err
	}

	currentSkill, _ := current.Output["skill"].(string)
	scores := []float64{metricScore(current)}
	for _, runID := range runIDs {
		run, err := LoadRun(runID)
		if err != nil {
			continue
		}

		if run.RunID == current.RunID {
			continue
		}

		skill, _ := run.Output["skill"].(string)
		if skill != currentSkill {
			continue
		}
		if current.DatasetHash != "" && run.DatasetHash != "" && run.DatasetHash != current.DatasetHash {
			continue
		}

		scores = append(scores, metricScore(run))
	}

	if len(scores) <= 1 {
		return 0, nil
	}

	var mean float64
	for _, score := range scores {
		mean += score
	}
	mean /= float64(len(scores))

	var variance float64
	for _, score := range scores {
		delta := score - mean
		variance += delta * delta
	}

	return roundFloat(variance / float64(len(scores))), nil
}

func metricScore(run RunResult) float64 {
	metrics := run.Metrics
	if metrics == nil {
		metrics = CalculateMetrics(run.Results)
	}
	return metrics.Score
}
