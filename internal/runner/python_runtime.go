package runner

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// PythonRuntime executes Python skill scripts
type PythonRuntime struct {
	skillPath string
	skillName string
}

// NewPythonRuntime creates a runtime for a Python-based skill
func NewPythonRuntime(skillPath, skillName string) *PythonRuntime {
	return &PythonRuntime{
		skillPath: skillPath,
		skillName: skillName,
	}
}

// Execute runs the Python skill script with the given input
func (p *PythonRuntime) Execute(ctx context.Context, input map[string]any) (SkillResult, error) {
	// Build command: python3 scripts/generate.py <doc_type> --project X --system Y --category N --output Z
	args := p.buildArgs(input)

	cmd := exec.CommandContext(ctx, "python3", args...)
	cmd.Dir = p.skillPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return SkillResult{
			Success:    false,
			Error:      fmt.Sprintf("exec error: %v, output: %s", err, string(output)),
			TokenUsage: estimateTokenUsage(input),
			Output:     parseOutput(string(output)),
			Trajectory: Trajectory{
				Steps:     []string{"python_runtime", "execute_script", "parse_output"},
				ToolCalls: []string{cmd.Path},
			},
		}, nil
	}

	return SkillResult{
		Success:    true,
		TokenUsage: estimateTokenUsage(input),
		Output:     parseOutput(string(output)),
		Trajectory: Trajectory{
			Steps:     []string{"python_runtime", "execute_script", "parse_output"},
			ToolCalls: []string{cmd.Path},
		},
	}, nil
}

func (p *PythonRuntime) buildArgs(input map[string]any) []string {
	var args []string

	// Script path
	scriptPath := filepath.Join(p.skillPath, "scripts", "generate.py")
	args = append(args, scriptPath)

	// Doc type
	if docType, ok := input["doc_type"].(string); ok && docType != "" {
		args = append(args, docType)
	} else if docType, ok := input["docType"].(string); ok && docType != "" {
		args = append(args, docType)
	} else {
		args = append(args, "vp")
	}

	// Project
	if project, ok := input["project"].(string); ok && project != "" {
		args = append(args, "--project", project)
	}

	// System
	if system, ok := input["system"].(string); ok && system != "" {
		args = append(args, "--system", system)
	}

	// Category
	if category, ok := input["category"].(float64); ok {
		args = append(args, "--category", fmt.Sprintf("%.0f", category))
	} else if category, ok := input["category"].(string); ok && category != "" {
		args = append(args, "--category", category)
	}

	// Output
	if output, ok := input["output"].(string); ok && output != "" {
		args = append(args, "--output", output)
	} else {
		args = append(args, "--output", "./output")
	}

	// Bilingual
	if bilingual, ok := input["bilingual"].(bool); ok {
		args = append(args, "--bilingual", fmt.Sprintf("%t", bilingual))
	}

	// Language
	if language, ok := input["language"].(string); ok && language != "" {
		args = append(args, "--language", language)
	}

	// Verbose
	if verbose, ok := input["verbose"].(bool); ok && verbose {
		args = append(args, "--verbose")
	}

	return args
}

func parseOutput(output string) map[string]any {
	result := map[string]any{
		"stdout": output,
	}

	// Try to extract generated files from output
	lines := strings.Split(output, "\n")
	var files []string
	for _, line := range lines {
		if strings.Contains(line, "Generated:") || strings.Contains(line, "生成:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				f := strings.TrimSpace(parts[len(parts)-1])
				if f != "" {
					files = append(files, f)
				}
			}
		}
	}

	if len(files) > 0 {
		result["generated_files"] = files
	}

	return result
}

// RegisterPythonRuntime registers a Python skill with the harness
func RegisterPythonRuntime(name, skillPath string) {
	RegisterRuntime(name, NewPythonRuntime(skillPath, name))
}
