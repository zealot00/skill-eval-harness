package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteHTMLReport(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "report.html")
	run := RunResult{
		Success:   true,
		GitCommit: "abc123",
		ModelName: "demo-skill",
		Timestamp: "2026-03-20T00:00:00Z",
		Seed:      42,
		Results: []CaseRunResult{
			{
				Success:        true,
				LatencyMS:      12,
				TokenUsage:     7,
				Classification: "",
				Trajectory: Trajectory{
					Steps:     []string{"load", "execute"},
					ToolCalls: []string{"demo-runtime"},
				},
			},
		},
		Metrics: &RunMetrics{
			Score:             0.7,
			SuccessRate:       1,
			AvgTokens:         7,
			P95Latency:        12,
			CostUSD:           0.000014,
			StabilityVariance: 0.012345,
		},
	}

	if err := WriteHTMLReport(path, run); err != nil {
		t.Fatalf("WriteHTMLReport() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	html := string(data)
	if !strings.Contains(html, "<!doctype html>") {
		t.Fatalf("html output missing doctype: %s", html)
	}

	if !strings.Contains(html, "Skill Evaluation Report") {
		t.Fatalf("html output missing title text: %s", html)
	}

	if !strings.Contains(html, "$0.000014") {
		t.Fatalf("html output missing cost: %s", html)
	}

	if !strings.Contains(html, "0.012345") {
		t.Fatalf("html output missing stability variance: %s", html)
	}
}
