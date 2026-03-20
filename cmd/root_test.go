package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"skill-eval-harness/internal/runner"
)

type cliMatrixRuntime struct {
	tokenUsage int64
}

func (c cliMatrixRuntime) Execute(ctx context.Context, input map[string]any) (runner.SkillResult, error) {
	return runner.SkillResult{
		Success:    true,
		TokenUsage: c.tokenUsage,
		Output:     map[string]any{"prompt": input["prompt"]},
	}, nil
}

func TestRootCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "no args shows help",
			args:     nil,
			contains: []string{"Usage:", "seh", "help"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := newRootCmd()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			output := buf.String()
			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Fatalf("output %q does not contain %q", output, want)
				}
			}
		})
	}
}

func TestRunCommand(t *testing.T) {
	t.Parallel()

	casesDir := t.TempDir()
	writeCLIFile(t, filepath.Join(casesDir, "manifest.yaml"), `
dataset_name: sample-dataset
version: v1
owner: qa
description: sample
`)
	writeCLIFile(t, filepath.Join(casesDir, "one.yaml"), `
case_id: case-1
skill: demo-skill
input:
  prompt: alpha
`)
	writeCLIFile(t, filepath.Join(casesDir, "nested", "two.yaml"), `
case_id: case-2
skill: demo-skill
input:
  prompt: beta
`)

	outPath := filepath.Join(t.TempDir(), "result.json")
	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"run",
		"--skill", "demo-skill",
		"--cases", casesDir,
		"--out", outPath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var got runner.RunResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !got.Success {
		t.Fatalf("RunResult.Success = false, want true")
	}

	if gotCount := len(got.Results); gotCount != 2 {
		t.Fatalf("len(results) = %d, want 2", gotCount)
	}

	for i, result := range got.Results {
		if !result.Success {
			t.Fatalf("results[%d].Success = false, want true", i)
		}
	}

	if got.DatasetVersion != "v1" {
		t.Fatalf("DatasetVersion = %q, want %q", got.DatasetVersion, "v1")
	}

	if got.DatasetHash == "" {
		t.Fatal("DatasetHash = empty, want dataset hash")
	}
}

func TestRunScorePipeline(t *testing.T) {
	t.Parallel()

	casesDir := t.TempDir()
	writeCLIFile(t, filepath.Join(casesDir, "manifest.yaml"), `
dataset_name: sample-dataset
version: v1
owner: qa
description: sample
`)
	writeCLIFile(t, filepath.Join(casesDir, "one.yaml"), `
case_id: case-1
skill: demo-skill
input:
  prompt: alpha
`)
	writeCLIFile(t, filepath.Join(casesDir, "two.yaml"), `
case_id: case-2
skill: demo-skill
input:
  prompt: beta
`)

	runPath := filepath.Join(t.TempDir(), "run.json")
	scorePath := filepath.Join(t.TempDir(), "score.json")

	runCmd := newRootCmd()
	runCmd.SetArgs([]string{
		"run",
		"--skill", "demo-skill",
		"--cases", casesDir,
		"--out", runPath,
	})
	if err := runCmd.Execute(); err != nil {
		t.Fatalf("run Execute() error = %v", err)
	}

	scoreCmd := newRootCmd()
	scoreCmd.SetArgs([]string{
		"score",
		"--run", runPath,
		"--out", scorePath,
	})
	if err := scoreCmd.Execute(); err != nil {
		t.Fatalf("score Execute() error = %v", err)
	}

	data, err := os.ReadFile(scorePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var got runner.RunResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got.Metrics == nil {
		t.Fatal("Metrics = nil, want metrics")
	}

	if got.Metrics.SuccessRate != 1 {
		t.Fatalf("SuccessRate = %v, want 1", got.Metrics.SuccessRate)
	}

	if got.Metrics.Score <= 0 {
		t.Fatalf("Score = %v, want > 0", got.Metrics.Score)
	}
}

func TestRunCommandTagFilter(t *testing.T) {
	t.Parallel()

	casesDir := t.TempDir()
	writeCLIFile(t, filepath.Join(casesDir, "manifest.yaml"), `
dataset_name: sample-dataset
version: v1
owner: qa
description: sample
`)
	writeCLIFile(t, filepath.Join(casesDir, "regression.yaml"), `
case_id: case-1
skill: demo-skill
input:
  prompt: alpha
tags:
  - regression
`)
	writeCLIFile(t, filepath.Join(casesDir, "smoke.yaml"), `
case_id: case-2
skill: demo-skill
input:
  prompt: beta
tags:
  - smoke
`)

	outPath := filepath.Join(t.TempDir(), "result.json")
	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"run",
		"--skill", "demo-skill",
		"--cases", casesDir,
		"--out", outPath,
		"--tag", "regression",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var got runner.RunResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if gotCount := len(got.Results); gotCount != 1 {
		t.Fatalf("len(results) = %d, want 1", gotCount)
	}

	if got.Output["case_count"] != float64(1) {
		t.Fatalf("case_count = %#v, want 1", got.Output["case_count"])
	}
}

func TestRunCommandSeedDeterministic(t *testing.T) {
	t.Parallel()

	casesDir := t.TempDir()
	writeCLIFile(t, filepath.Join(casesDir, "manifest.yaml"), `
dataset_name: sample-dataset
version: v1
owner: qa
description: sample
`)
	writeCLIFile(t, filepath.Join(casesDir, "one.yaml"), `
case_id: case-1
skill: demo-skill
input:
  prompt: alpha
`)

	outPathOne := filepath.Join(t.TempDir(), "result-one.json")
	outPathTwo := filepath.Join(t.TempDir(), "result-two.json")

	runOnce := func(outPath string) {
		t.Helper()

		cmd := newRootCmd()
		cmd.SetArgs([]string{
			"run",
			"--skill", "demo-skill",
			"--cases", casesDir,
			"--out", outPath,
			"--seed", "42",
		})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	}

	runOnce(outPathOne)
	runOnce(outPathTwo)

	dataOne, err := os.ReadFile(outPathOne)
	if err != nil {
		t.Fatalf("ReadFile(result-one) error = %v", err)
	}

	dataTwo, err := os.ReadFile(outPathTwo)
	if err != nil {
		t.Fatalf("ReadFile(result-two) error = %v", err)
	}

	if string(dataOne) != string(dataTwo) {
		t.Fatalf("seeded run outputs differ:\n%s\n!=\n%s", string(dataOne), string(dataTwo))
	}

	var got runner.RunResult
	if err := json.Unmarshal(dataOne, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got.Seed != 42 {
		t.Fatalf("Seed = %d, want 42", got.Seed)
	}
}

func TestRunCommandSample(t *testing.T) {
	t.Parallel()

	casesDir := t.TempDir()
	writeCLIFile(t, filepath.Join(casesDir, "manifest.yaml"), `
dataset_name: sample-dataset
version: v1
owner: qa
description: sample
`)
	for i := 0; i < 10; i++ {
		writeCLIFile(t, filepath.Join(casesDir, fmt.Sprintf("case-%d.yaml", i)), fmt.Sprintf(`
case_id: case-%d
skill: demo-skill
input:
  prompt: sample-%d
`, i, i))
	}

	outPath := filepath.Join(t.TempDir(), "result.json")
	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"run",
		"--skill", "demo-skill",
		"--cases", casesDir,
		"--out", outPath,
		"--sample", "0.2",
		"--seed", "42",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var got runner.RunResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(got.Results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(got.Results))
	}
}

func TestReportCommand(t *testing.T) {
	t.Parallel()

	casesDir := t.TempDir()
	writeCLIFile(t, filepath.Join(casesDir, "manifest.yaml"), `
dataset_name: sample-dataset
version: v1
owner: qa
description: sample
`)
	writeCLIFile(t, filepath.Join(casesDir, "one.yaml"), `
case_id: case-1
skill: demo-skill
input:
  prompt: alpha
`)

	runPath := filepath.Join(t.TempDir(), "run.json")
	htmlPath := filepath.Join(t.TempDir(), "report.html")

	runCmd := newRootCmd()
	runCmd.SetArgs([]string{
		"run",
		"--skill", "demo-skill",
		"--cases", casesDir,
		"--out", runPath,
		"--seed", "42",
	})
	if err := runCmd.Execute(); err != nil {
		t.Fatalf("run Execute() error = %v", err)
	}

	reportCmd := newRootCmd()
	reportCmd.SetArgs([]string{
		"report",
		"--in", runPath,
		"--out", htmlPath,
	})
	if err := reportCmd.Execute(); err != nil {
		t.Fatalf("report Execute() error = %v", err)
	}

	data, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	html := string(data)
	if !strings.Contains(html, "<!doctype html>") {
		t.Fatalf("html = %q, want doctype", html)
	}

	if !strings.Contains(html, "Skill Evaluation Report") {
		t.Fatalf("html = %q, want report title", html)
	}
}

func TestHistoryListCommand(t *testing.T) {
	historyDir := filepath.Join(t.TempDir(), ".history")
	t.Setenv("SEH_HISTORY_DIR", historyDir)

	casesDir := t.TempDir()
	writeCLIFile(t, filepath.Join(casesDir, "manifest.yaml"), `
dataset_name: sample-dataset
version: v1
owner: qa
description: sample
`)
	writeCLIFile(t, filepath.Join(casesDir, "one.yaml"), `
case_id: case-1
skill: demo-skill
input:
  prompt: alpha
`)

	outPath := filepath.Join(t.TempDir(), "run.json")
	runCmd := newRootCmd()
	runCmd.SetArgs([]string{
		"run",
		"--skill", "demo-skill",
		"--cases", casesDir,
		"--out", outPath,
		"--seed", "42",
	})
	if err := runCmd.Execute(); err != nil {
		t.Fatalf("run Execute() error = %v", err)
	}

	historyCmd := newRootCmd()
	var buf bytes.Buffer
	historyCmd.SetOut(&buf)
	historyCmd.SetArgs([]string{"history", "list"})
	if err := historyCmd.Execute(); err != nil {
		t.Fatalf("history Execute() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "seed-42") {
		t.Fatalf("history output = %q, want run id", output)
	}
}

func TestCompareCommand(t *testing.T) {
	t.Parallel()

	currentPath := filepath.Join(t.TempDir(), "current.json")
	baselinePath := filepath.Join(t.TempDir(), "baseline.json")

	current := runner.RunResult{
		Metrics: &runner.RunMetrics{
			SuccessRate: 0.8,
			P95Latency:  150,
			AvgTokens:   40,
		},
	}
	baseline := runner.RunResult{
		Metrics: &runner.RunMetrics{
			SuccessRate: 0.9,
			P95Latency:  100,
			AvgTokens:   30,
		},
	}

	writeCLIJSON(t, currentPath, current)
	writeCLIJSON(t, baselinePath, baseline)

	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{
		"compare",
		"--run", currentPath,
		"--baseline", baselinePath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"Regression summary",
		"success_rate_delta: -0.100000",
		"latency_delta: 50",
		"token_delta: 10",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output %q does not contain %q", output, want)
		}
	}
}

func TestCompareCommandFailOnRegression(t *testing.T) {
	t.Parallel()

	currentPath := filepath.Join(t.TempDir(), "current.json")
	baselinePath := filepath.Join(t.TempDir(), "baseline.json")

	current := runner.RunResult{
		Metrics: &runner.RunMetrics{
			SuccessRate: 0.8,
			P95Latency:  150,
			AvgTokens:   40,
		},
	}
	baseline := runner.RunResult{
		Metrics: &runner.RunMetrics{
			SuccessRate: 0.9,
			P95Latency:  100,
			AvgTokens:   30,
		},
	}

	writeCLIJSON(t, currentPath, current)
	writeCLIJSON(t, baselinePath, baseline)

	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"compare",
		"--run", currentPath,
		"--baseline", baselinePath,
		"--fail-on-regression",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want regression failure")
	}

	if !strings.Contains(err.Error(), "regression detected") {
		t.Fatalf("error = %q, want regression detected", err.Error())
	}
}

func TestDriftCommand(t *testing.T) {
	t.Parallel()

	currentPath := filepath.Join(t.TempDir(), "current.json")
	baselinePath := filepath.Join(t.TempDir(), "baseline.json")
	outPath := filepath.Join(t.TempDir(), "drift.json")

	current := runner.RunResult{
		Results: []runner.CaseRunResult{
			{Output: map[string]any{"answer": "hello world"}},
			{Output: map[string]any{"answer": "cat dog"}},
		},
	}
	baseline := runner.RunResult{
		Results: []runner.CaseRunResult{
			{Output: map[string]any{"answer": "hello world"}},
			{Output: map[string]any{"answer": "apple banana"}},
		},
	}

	writeCLIJSON(t, currentPath, current)
	writeCLIJSON(t, baselinePath, baseline)

	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{
		"drift",
		"--run", currentPath,
		"--baseline", baselinePath,
		"--out", outPath,
		"--threshold", "0.95",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var got runner.DriftReport
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !got.DriftDetected {
		t.Fatal("DriftDetected = false, want true")
	}

	if len(got.Cases) != 2 {
		t.Fatalf("len(Cases) = %d, want 2", len(got.Cases))
	}

	for _, want := range []string{"Drift report", "drift_detected: true"} {
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("output %q does not contain %q", buf.String(), want)
		}
	}
}

func TestBaselinePromoteCommand(t *testing.T) {
	historyDir := filepath.Join(t.TempDir(), ".history")
	t.Setenv("SEH_HISTORY_DIR", historyDir)

	run := runner.RunResult{
		RunID:          "run-123",
		DatasetVersion: "v1",
		DatasetHash:    "hash123",
	}
	if err := runner.PersistRun(run); err != nil {
		t.Fatalf("PersistRun() error = %v", err)
	}

	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{
		"baseline",
		"promote",
		"--run", "run-123",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	baselineRunID, err := runner.LoadBaselineRunID()
	if err != nil {
		t.Fatalf("LoadBaselineRunID() error = %v", err)
	}

	if baselineRunID != "run-123" {
		t.Fatalf("LoadBaselineRunID() = %q, want %q", baselineRunID, "run-123")
	}

	if !strings.Contains(buf.String(), "promoted baseline: run-123") {
		t.Fatalf("output = %q, want promoted baseline summary", buf.String())
	}
}

func TestSimulateCommand(t *testing.T) {
	historyDir := filepath.Join(t.TempDir(), ".history")
	t.Setenv("SEH_HISTORY_DIR", historyDir)

	runs := []runner.RunResult{
		{
			RunID:  "run-1",
			Output: map[string]any{"skill": "skill-a"},
			Metrics: &runner.RunMetrics{
				Score:   0.9,
				CostUSD: 0.3,
			},
		},
		{
			RunID:  "run-2",
			Output: map[string]any{"skill": "skill-b"},
			Metrics: &runner.RunMetrics{
				Score:   0.7,
				CostUSD: 0.1,
			},
		},
	}

	for _, run := range runs {
		if err := runner.PersistRun(run); err != nil {
			t.Fatalf("PersistRun() error = %v", err)
		}
	}

	outPath := filepath.Join(t.TempDir(), "simulation.json")
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{
		"simulate",
		"--out", outPath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var got runner.RoutingSimulationReport
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(got.Policies) != 3 {
		t.Fatalf("len(Policies) = %d, want 3", len(got.Policies))
	}

	for _, want := range []string{"round_robin", "best_score", "cost_aware"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("report %q does not contain %q", string(data), want)
		}
	}

	if !strings.Contains(buf.String(), "simulation report generated") {
		t.Fatalf("output = %q, want simulation summary", buf.String())
	}
}

func TestMatrixCommand(t *testing.T) {
	runner.RegisterRuntime("matrix-cli-a", cliMatrixRuntime{tokenUsage: 10})
	runner.RegisterRuntime("matrix-cli-b", cliMatrixRuntime{tokenUsage: 20})

	casesDir := t.TempDir()
	writeCLIFile(t, filepath.Join(casesDir, "manifest.yaml"), `
dataset_name: sample-dataset
version: v1
owner: qa
description: sample
`)
	writeCLIFile(t, filepath.Join(casesDir, "one.yaml"), `
case_id: case-1
skill: demo-skill
input:
  prompt: alpha
`)

	outPath := filepath.Join(t.TempDir(), "matrix.json")
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{
		"matrix",
		"--runtimes", "matrix-cli-a,matrix-cli-b",
		"--cases", casesDir,
		"--out", outPath,
		"--seed", "42",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var got runner.RuntimeComparisonMatrix
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(got.Runtimes) != 2 {
		t.Fatalf("len(Runtimes) = %d, want 2", len(got.Runtimes))
	}

	for _, want := range []string{"matrix-cli-a", "matrix-cli-b"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("matrix %q does not contain %q", string(data), want)
		}
	}

	if !strings.Contains(buf.String(), "comparison matrix written") {
		t.Fatalf("output = %q, want matrix summary", buf.String())
	}
}

func TestFrontierCommand(t *testing.T) {
	historyDir := filepath.Join(t.TempDir(), ".history")
	t.Setenv("SEH_HISTORY_DIR", historyDir)

	runs := []runner.RunResult{
		{
			RunID:  "run-a",
			Output: map[string]any{"skill": "skill-a"},
			Metrics: &runner.RunMetrics{
				CostUSD: 0.1,
				Score:   0.8,
			},
		},
		{
			RunID:  "run-b",
			Output: map[string]any{"skill": "skill-b"},
			Metrics: &runner.RunMetrics{
				CostUSD: 0.2,
				Score:   0.7,
			},
		},
		{
			RunID:  "run-c",
			Output: map[string]any{"skill": "skill-c"},
			Metrics: &runner.RunMetrics{
				CostUSD: 0.3,
				Score:   0.95,
			},
		},
	}

	for _, run := range runs {
		if err := runner.PersistRun(run); err != nil {
			t.Fatalf("PersistRun() error = %v", err)
		}
	}

	outPath := filepath.Join(t.TempDir(), "frontier.json")
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"frontier", "--out", outPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var got runner.ParetoFrontierReport
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(got.Points) != 2 {
		t.Fatalf("len(Points) = %d, want 2", len(got.Points))
	}

	for _, want := range []string{"run-a", "run-c"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("frontier %q does not contain %q", string(data), want)
		}
	}
}

func TestIngestCommand(t *testing.T) {
	historyDir := filepath.Join(t.TempDir(), ".history")
	t.Setenv("SEH_HISTORY_DIR", historyDir)

	runPath := filepath.Join(t.TempDir(), "remote.json")
	input := runner.RunResult{
		RunID:          "remote-run",
		DatasetVersion: "v1",
		DatasetHash:    "hash123",
	}
	writeCLIJSON(t, runPath, input)

	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"ingest", "--run", runPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	runIDs, err := runner.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}

	if len(runIDs) != 1 || runIDs[0] != "remote-run" {
		t.Fatalf("ListRuns() = %#v, want [remote-run]", runIDs)
	}

	if !strings.Contains(buf.String(), "ingested run: remote-run") {
		t.Fatalf("output = %q, want ingest summary", buf.String())
	}
}

func TestLeaderboardCommand(t *testing.T) {
	historyDir := filepath.Join(t.TempDir(), ".history")
	t.Setenv("SEH_HISTORY_DIR", historyDir)

	runs := []runner.RunResult{
		{
			RunID:  "run-1",
			Output: map[string]any{"skill": "skill-b"},
			Metrics: &runner.RunMetrics{
				Score: 0.8,
			},
		},
		{
			RunID:  "run-2",
			Output: map[string]any{"skill": "skill-a"},
			Metrics: &runner.RunMetrics{
				Score: 0.9,
			},
		},
		{
			RunID:  "run-3",
			Output: map[string]any{"skill": "skill-b"},
			Metrics: &runner.RunMetrics{
				Score: 0.6,
			},
		},
	}

	for _, run := range runs {
		if err := runner.PersistRun(run); err != nil {
			t.Fatalf("PersistRun() error = %v", err)
		}
	}

	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"leaderboard"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"1. skill-a composite_score=0.900000 runs=1",
		"2. skill-b composite_score=0.700000 runs=2",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output %q does not contain %q", output, want)
		}
	}
}

func TestRunCommandStrict(t *testing.T) {
	t.Parallel()

	casesDir := t.TempDir()
	writeCLIFile(t, filepath.Join(casesDir, "manifest.yaml"), `
dataset_name: sample-dataset
version: v1
owner: qa
description: sample
`)
	writeCLIFile(t, filepath.Join(casesDir, "one.yaml"), `
case_id: case-1
skill: demo-skill
input:
  prompt: alpha
expected:
  output_hash: deadbeef
`)

	outPath := filepath.Join(t.TempDir(), "result.json")
	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"run",
		"--skill", "demo-skill",
		"--cases", casesDir,
		"--out", outPath,
		"--strict",
		"--seed", "42",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want strict failure")
	}

	if !strings.Contains(err.Error(), "strict mode") {
		t.Fatalf("error = %q, want strict mode failure", err.Error())
	}

	data, readErr := os.ReadFile(outPath)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v, want written output", readErr)
	}

	var got runner.RunResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got.Success {
		t.Fatal("RunResult.Success = true, want false")
	}
}

func writeCLIFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}

	if err := os.WriteFile(path, []byte(strings.TrimSpace(contents)), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func writeCLIJSON(t *testing.T, path string, v any) {
	t.Helper()

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
