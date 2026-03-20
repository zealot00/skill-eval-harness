package runner

import (
	"context"
	"testing"
)

func TestRegisterRuntime(t *testing.T) {
	t.Parallel()

	demoRuntime, err := ResolveRuntime("demo-skill")
	if err != nil {
		t.Fatalf("ResolveRuntime(demo-skill) error = %v", err)
	}

	if demoRuntime == nil {
		t.Fatal("ResolveRuntime(demo-skill) = nil, want runtime")
	}

	registry := RuntimeRegistry{}
	mock := mockRegisteredRuntime{}
	registry.Register("mock-skill", mock)

	got, err := registry.Resolve("mock-skill")
	if err != nil {
		t.Fatalf("Resolve(mock-skill) error = %v", err)
	}

	if got == nil {
		t.Fatal("Resolve(mock-skill) = nil, want runtime")
	}
}

type mockRegisteredRuntime struct{}

func (m mockRegisteredRuntime) Execute(ctx context.Context, input map[string]any) (SkillResult, error) {
	return SkillResult{
		Success: true,
		Output:  input,
	}, nil
}
