package runner

import (
	"context"
	"testing"

	"skill-eval-harness/internal/dataset"
)

type matrixStubRuntime struct {
	success    bool
	tokenUsage int64
}

func (m matrixStubRuntime) Execute(ctx context.Context, input map[string]any) (SkillResult, error) {
	return SkillResult{
		Success:    m.success,
		TokenUsage: m.tokenUsage,
		Output:     map[string]any{"echo": input["prompt"]},
	}, nil
}

func TestBuildComparisonMatrix(t *testing.T) {
	registry := defaultRegistry
	originalOne, hadOne := registry["matrix-runtime-a"]
	originalTwo, hadTwo := registry["matrix-runtime-b"]
	registry.Register("matrix-runtime-a", matrixStubRuntime{success: true, tokenUsage: 10})
	registry.Register("matrix-runtime-b", matrixStubRuntime{success: true, tokenUsage: 20})
	t.Cleanup(func() {
		if hadOne {
			registry["matrix-runtime-a"] = originalOne
		} else {
			delete(registry, "matrix-runtime-a")
		}

		if hadTwo {
			registry["matrix-runtime-b"] = originalTwo
		} else {
			delete(registry, "matrix-runtime-b")
		}
	})

	cases := []dataset.EvaluationCase{
		{CaseID: "case-1", Skill: "demo-skill", Input: map[string]any{"prompt": "alpha"}},
		{CaseID: "case-2", Skill: "demo-skill", Input: map[string]any{"prompt": "beta"}},
	}

	got, err := BuildComparisonMatrix([]string{"matrix-runtime-a", "matrix-runtime-b"}, cases, RunOptions{
		DatasetVersion: "v1",
		DatasetHash:    "hash123",
		Seed:           42,
	})
	if err != nil {
		t.Fatalf("BuildComparisonMatrix() error = %v", err)
	}

	if got.DatasetVersion != "v1" || got.DatasetHash != "hash123" || got.CaseCount != 2 {
		t.Fatalf("matrix metadata = %#v, want version/hash/count", got)
	}

	if len(got.Runtimes) != 2 {
		t.Fatalf("len(Runtimes) = %d, want 2", len(got.Runtimes))
	}

	if got.Runtimes[0].Runtime != "matrix-runtime-a" || got.Runtimes[0].AvgTokens != 10 {
		t.Fatalf("Runtimes[0] = %#v, want runtime-a avg_tokens 10", got.Runtimes[0])
	}

	if got.Runtimes[1].Runtime != "matrix-runtime-b" || got.Runtimes[1].AvgTokens != 20 {
		t.Fatalf("Runtimes[1] = %#v, want runtime-b avg_tokens 20", got.Runtimes[1])
	}
}
