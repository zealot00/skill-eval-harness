package runner

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"skill-eval-harness/internal/dataset"
)

func BenchmarkRunCasesSequential(b *testing.B) {
	benchmarkRunCases(b, 1)
}

func BenchmarkRunCasesParallel(b *testing.B) {
	benchmarkRunCases(b, 4)
}

func benchmarkRunCases(b *testing.B, workers int) {
	cases := makeBenchmarkCases(12)
	rt := &sleepRuntime{delay: 5 * time.Millisecond}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RunCasesWithOptions("bench", cases, rt, RunOptions{Workers: workers})
	}
}

func makeBenchmarkCases(count int) []dataset.EvaluationCase {
	cases := make([]dataset.EvaluationCase, 0, count)
	for i := 0; i < count; i++ {
		cases = append(cases, dataset.EvaluationCase{
			CaseID: "case",
			Skill:  "bench",
			Input:  map[string]any{"index": i},
		})
	}

	return cases
}

type sleepRuntime struct {
	delay time.Duration
}

func (s *sleepRuntime) Execute(_ context.Context, input map[string]any) (SkillResult, error) {
	time.Sleep(s.delay)

	return SkillResult{
		Success:    true,
		TokenUsage: int64(len(input)),
		Output:     input,
	}, nil
}

func TestRunCasesWithOptionsParallel(t *testing.T) {
	t.Parallel()

	rt := &concurrentRuntime{delay: 20 * time.Millisecond}
	cases := []dataset.EvaluationCase{
		{CaseID: "case-1", Skill: "seh", Input: map[string]any{"index": 1}},
		{CaseID: "case-2", Skill: "seh", Input: map[string]any{"index": 2}},
		{CaseID: "case-3", Skill: "seh", Input: map[string]any{"index": 3}},
		{CaseID: "case-4", Skill: "seh", Input: map[string]any{"index": 4}},
	}

	got := RunCasesWithOptions("seh", cases, rt, RunOptions{Workers: 4})

	if rt.callCount.Load() != int64(len(cases)) {
		t.Fatalf("call count = %d, want %d", rt.callCount.Load(), len(cases))
	}

	if rt.maxConcurrent.Load() < 2 {
		t.Fatalf("max concurrent = %d, want at least 2", rt.maxConcurrent.Load())
	}

	if len(got.Results) != len(cases) {
		t.Fatalf("len(results) = %d, want %d", len(got.Results), len(cases))
	}
}

type concurrentRuntime struct {
	delay         time.Duration
	callCount     atomic.Int64
	inFlight      atomic.Int64
	maxConcurrent atomic.Int64
}

func (c *concurrentRuntime) Execute(_ context.Context, input map[string]any) (SkillResult, error) {
	c.callCount.Add(1)
	current := c.inFlight.Add(1)
	defer c.inFlight.Add(-1)

	for {
		maxSeen := c.maxConcurrent.Load()
		if current <= maxSeen {
			break
		}
		if c.maxConcurrent.CompareAndSwap(maxSeen, current) {
			break
		}
	}

	time.Sleep(c.delay)

	return SkillResult{
		Success:    true,
		TokenUsage: int64(len(input)),
		Output:     input,
	}, nil
}
