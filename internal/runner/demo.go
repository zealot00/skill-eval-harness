package runner

import "context"

// DemoRuntime is a simple runtime used by the CLI for local evaluation flows.
type DemoRuntime struct {
	skill string
}

// NewDemoRuntime creates a demo runtime for the provided skill name.
func NewDemoRuntime(skill string) *DemoRuntime {
	return &DemoRuntime{skill: skill}
}

// Execute returns a deterministic successful result for the provided input.
func (d *DemoRuntime) Execute(_ context.Context, input map[string]any) (SkillResult, error) {
	output := map[string]any{
		"skill": d.skill,
		"input": input,
	}

	return SkillResult{
		Success:    true,
		TokenUsage: int64(len(input)),
		Output:     output,
		Trajectory: Trajectory{
			Steps:                   []string{"load_input", "execute_demo_runtime", "produce_output"},
			ToolCalls:               []string{"demo-runtime"},
			ReasoningTokensEstimate: 16,
		},
	}, nil
}
