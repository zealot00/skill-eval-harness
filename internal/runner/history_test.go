package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPersistListLoadRun(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("SEH_HISTORY_DIR", filepath.Join(tempDir, ".history"))

	run := RunResult{
		RunID:          "run-1",
		DatasetVersion: "v1",
		DatasetHash:    "abc123",
	}

	if err := PersistRun(run); err != nil {
		t.Fatalf("PersistRun() error = %v", err)
	}

	runIDs, err := ListRuns()
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}

	if len(runIDs) != 1 || runIDs[0] != "run-1" {
		t.Fatalf("ListRuns() = %#v, want [run-1]", runIDs)
	}

	got, err := LoadRun("run-1")
	if err != nil {
		t.Fatalf("LoadRun() error = %v", err)
	}

	if got.RunID != "run-1" {
		t.Fatalf("RunID = %q, want %q", got.RunID, "run-1")
	}

	if got.DatasetHash != "abc123" {
		t.Fatalf("DatasetHash = %q, want %q", got.DatasetHash, "abc123")
	}

	if _, err := os.Stat(filepath.Join(tempDir, ".history", "run-1.json")); err != nil {
		t.Fatalf("history file missing: %v", err)
	}
}

func TestPromoteBaseline(t *testing.T) {
	tempDir := t.TempDir()
	historyDir := filepath.Join(tempDir, ".history")
	t.Setenv("SEH_HISTORY_DIR", historyDir)

	run := RunResult{
		RunID:          "run-1",
		DatasetVersion: "v1",
		DatasetHash:    "abc123",
	}
	if err := PersistRun(run); err != nil {
		t.Fatalf("PersistRun() error = %v", err)
	}

	if err := PromoteBaseline("run-1"); err != nil {
		t.Fatalf("PromoteBaseline() error = %v", err)
	}

	got, err := LoadBaselineRunID()
	if err != nil {
		t.Fatalf("LoadBaselineRunID() error = %v", err)
	}

	if got != "run-1" {
		t.Fatalf("LoadBaselineRunID() = %q, want %q", got, "run-1")
	}

	data, err := os.ReadFile(filepath.Join(historyDir, "baseline"))
	if err != nil {
		t.Fatalf("ReadFile(baseline) error = %v", err)
	}

	if string(data) != "run-1\n" {
		t.Fatalf("baseline pointer = %q, want %q", string(data), "run-1\n")
	}
}

func TestIngestRun(t *testing.T) {
	tempDir := t.TempDir()
	historyDir := filepath.Join(tempDir, ".history")
	t.Setenv("SEH_HISTORY_DIR", historyDir)

	runPath := filepath.Join(tempDir, "remote.json")
	input := RunResult{
		RunID:          "remote-run",
		DatasetVersion: "v1",
		DatasetHash:    "hash123",
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := os.WriteFile(runPath, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := IngestRun(runPath)
	if err != nil {
		t.Fatalf("IngestRun() error = %v", err)
	}

	if got.RunID != "remote-run" {
		t.Fatalf("RunID = %q, want %q", got.RunID, "remote-run")
	}

	runIDs, err := ListRuns()
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}

	if len(runIDs) != 1 || runIDs[0] != "remote-run" {
		t.Fatalf("ListRuns() = %#v, want [remote-run]", runIDs)
	}
}

func TestComputeStabilityVariance(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("SEH_HISTORY_DIR", filepath.Join(tempDir, ".history"))

	historicalRuns := []RunResult{
		{
			RunID:       "run-1",
			DatasetHash: "hash123",
			Output:      map[string]any{"skill": "demo-skill"},
			Metrics:     &RunMetrics{Score: 0.8},
		},
		{
			RunID:       "run-2",
			DatasetHash: "hash123",
			Output:      map[string]any{"skill": "demo-skill"},
			Metrics:     &RunMetrics{Score: 0.6},
		},
	}
	for _, run := range historicalRuns {
		if err := PersistRun(run); err != nil {
			t.Fatalf("PersistRun() error = %v", err)
		}
	}

	current := RunResult{
		RunID:       "run-3",
		DatasetHash: "hash123",
		Output:      map[string]any{"skill": "demo-skill"},
		Metrics:     &RunMetrics{Score: 1.0},
	}

	got, err := ComputeStabilityVariance(current)
	if err != nil {
		t.Fatalf("ComputeStabilityVariance() error = %v", err)
	}

	if got != 0.026667 {
		t.Fatalf("ComputeStabilityVariance() = %v, want 0.026667", got)
	}
}
