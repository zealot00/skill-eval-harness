package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RuntimeConfig defines how to execute a skill.
// This is loaded from .seh/runtime.yaml in the skill directory.
type RuntimeConfig struct {
	Runtime RuntimeSpec `yaml:"runtime" json:"runtime"`
}

// RuntimeSpec describes the execution environment for a skill.
type RuntimeSpec struct {
	// Name is the skill name (defaults to directory name).
	Name string `yaml:"name" json:"name"`

	// Type specifies the runtime type: python, node, go, shell, docker, http, command.
	Type string `yaml:"type" json:"type"`

	// Workdir sets the working directory for execution.
	// Supports placeholders: {skill_path}, {temp_dir}.
	Workdir string `yaml:"workdir" json:"workdir"`

	// Env defines environment variables.
	// Supports placeholders: {env.VAR_NAME}, {skill_path}.
	Env map[string]string `yaml:"env" json:"env"`

	// Command defines how to build the execution command.
	Command CommandSpec `yaml:"command" json:"command"`

	// Inputs declares input files that should be copied to workdir.
	Inputs *InputsSpec `yaml:"inputs,omitempty" json:"inputs,omitempty"`

	// Output defines how to parse stdout and collect output files.
	Output OutputSpec `yaml:"output" json:"output"`

	// Execution controls timeout and retry behavior.
	Execution ExecutionSpec `yaml:"execution,omitempty" json:"execution,omitempty"`
}

// CommandSpec defines command building.
type CommandSpec struct {
	// Template is the command template.
	// Placeholders: {script}, {args}, {binary}, {image}, {endpoint}.
	Template string `yaml:"template" json:"template"`

	// Args defines how input map is converted to arguments.
	Args ArgsSpec `yaml:"args,omitempty" json:"args,omitempty"`

	// For http type: headers to include.
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`

	// For http type: body template.
	Body string `yaml:"body,omitempty" json:"body,omitempty"`
}

// ArgsSpec defines how to convert input map to command arguments.
type ArgsSpec struct {
	// Format: key-value (--key value), positional, shell.
	Format string `yaml:"format" json:"format"`

	// Mapping defines how each input key maps to arguments.
	Mapping map[string]string `yaml:"mapping,omitempty" json:"mapping"`

	// PositionalOrder defines the order of positional arguments.
	PositionalOrder []string `yaml:"positional_order,omitempty" json:"positional_order"`
}

// InputsSpec declares input files to copy to workdir.
type InputsSpec struct {
	Files []InputFileSpec `yaml:"files,omitempty" json:"files"`
}

// InputFileSpec describes a single input file.
type InputFileSpec struct {
	// Name is the key in the input map.
	Name string `yaml:"name" json:"name"`

	// Path is the source path (can use {input.KEY} placeholder).
	Path string `yaml:"path" json:"path"`

	// Dest is the destination relative to workdir.
	Dest string `yaml:"dest" json:"dest"`

	// Required indicates if this file must exist.
	Required bool `yaml:"required" json:"required"`
}

// OutputSpec defines how to parse stdout and collect output files.
type OutputSpec struct {
	// Stdout specifies how to parse stdout.
	Stdout StdoutSpec `yaml:"stdout" json:"stdout"`

	// Files specifies expected output files.
	Files []OutputFileSpec `yaml:"files,omitempty" json:"files"`
}

// StdoutSpec defines stdout parsing rules.
type StdoutSpec struct {
	// ParseAs: text, json, regex.
	ParseAs string `yaml:"parse_as" json:"parse_as"`

	// Extract defines extraction rules.
	Extract *ExtractSpec `yaml:"extract,omitempty" json:"extract,omitempty"`
}

// ExtractSpec defines what to extract from stdout.
type ExtractSpec struct {
	// FilePattern extracts file paths using regex.
	FilePattern string `yaml:"file_pattern,omitempty" json:"file_pattern,omitempty"`

	// StatusPattern extracts status using regex.
	StatusPattern string `yaml:"status_pattern,omitempty" json:"status_pattern,omitempty"`

	// JSONPath extracts value from JSON output.
	JSONPath string `yaml:"json_path,omitempty" json:"json_path,omitempty"`
}

// OutputFileSpec describes an expected output file.
type OutputFileSpec struct {
	// Path is the file path pattern (supports globs).
	Path string `yaml:"path" json:"path"`

	// Name is the key to store in output.
	Name string `yaml:"name,omitempty" json:"name,omitempty"`

	// MinSize in bytes.
	MinSize int64 `yaml:"min_size,omitempty" json:"min_size,omitempty"`

	// MaxSize in bytes.
	MaxSize int64 `yaml:"max_size,omitempty" json:"max_size,omitempty"`
}

// ExecutionSpec defines execution controls.
type ExecutionSpec struct {
	// Timeout for each execution (e.g., "30s", "1m").
	Timeout string `yaml:"timeout,omitempty" json:"timeout,omitempty"`

	// Retries on failure.
	Retries int `yaml:"retries,omitempty" json:"retries,omitempty"`
}

// LoadRuntimeConfig loads runtime configuration from .seh/runtime.yaml.
func LoadRuntimeConfig(skillPath string) (*RuntimeConfig, error) {
	configPath := filepath.Join(skillPath, ".seh", "runtime.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read runtime config %q: %w", configPath, err)
	}

	var cfg RuntimeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse runtime config %q: %w", configPath, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate runtime config: %w", err)
	}

	return &cfg, nil
}

// Validate checks the configuration for required fields.
func (c *RuntimeConfig) Validate() error {
	if strings.TrimSpace(c.Runtime.Type) == "" {
		return fmt.Errorf("runtime.type is required")
	}

	validTypes := map[string]bool{
		"python":  true,
		"node":    true,
		"go":      true,
		"shell":   true,
		"docker":  true,
		"http":    true,
		"command": true,
	}

	if !validTypes[c.Runtime.Type] {
		return fmt.Errorf("runtime.type %q is not supported (valid: python, node, go, shell, docker, http, command)", c.Runtime.Type)
	}

	if strings.TrimSpace(c.Runtime.Command.Template) == "" {
		return fmt.Errorf("runtime.command.template is required")
	}

	return nil
}

// ResolvePath resolves placeholders in a path string.
func (c *RuntimeConfig) ResolvePath(path, skillPath, tempDir string) string {
	path = strings.ReplaceAll(path, "{skill_path}", skillPath)
	path = strings.ReplaceAll(path, "{temp_dir}", tempDir)
	return path
}

// ResolveEnv resolves environment variable placeholders.
func (c *RuntimeConfig) ResolveEnv(value, skillPath string) string {
	// Replace {env.VAR_NAME} with actual env value
	for {
		idx := strings.Index(value, "{env.")
		if idx == -1 {
			break
		}

		end := strings.Index(value[idx:], "}")
		if end == -1 {
			break
		}

		varName := value[idx+5 : idx+end]
		envValue := os.Getenv(varName)
		value = value[:idx] + envValue + value[idx+end+1:]
	}

	value = strings.ReplaceAll(value, "{skill_path}", skillPath)
	return value
}
