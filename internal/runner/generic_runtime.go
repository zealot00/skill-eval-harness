package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"skill-eval-harness/internal/dataset"
)

const defaultTimeout = 30 * time.Second

type GenericRuntime struct {
	config    *RuntimeConfig
	skillPath string
}

func NewGenericRuntime(skillPath string, cfg *RuntimeConfig) *GenericRuntime {
	return &GenericRuntime{
		config:    cfg,
		skillPath: skillPath,
	}
}

func (g *GenericRuntime) Execute(ctx context.Context, input map[string]any) (SkillResult, error) {
	timeout := defaultTimeout
	if g.config.Runtime.Execution.Timeout != "" {
		parsed, err := time.ParseDuration(g.config.Runtime.Execution.Timeout)
		if err == nil {
			timeout = parsed
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	workDir := g.resolveWorkdir()
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return g.errorResult(fmt.Sprintf("create workdir: %v", err), input), nil
	}

	if err := g.prepareInputs(input, workDir); err != nil {
		return g.errorResult(fmt.Sprintf("prepare inputs: %v", err), input), nil
	}

	var result SkillResult
	var err error

	switch g.config.Runtime.Type {
	case "python", "node", "go", "shell", "command", "typescript", "tsx", "bun", "deno", "rust":
		result, err = g.executeCommand(ctx, input, workDir)
	case "docker":
		result, err = g.executeDocker(ctx, input, workDir)
	case "http":
		result, err = g.executeHTTP(ctx, input)
	default:
		return g.errorResult(fmt.Sprintf("unsupported runtime type: %s", g.config.Runtime.Type), input), nil
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
	}

	return result, nil
}

func (g *GenericRuntime) resolveWorkdir() string {
	workdir := g.config.Runtime.Workdir
	if workdir == "" {
		workdir = "{skill_path}"
	}

	workdir = g.config.ResolvePath(workdir, g.skillPath, os.TempDir())

	if workdir == "{skill_path}" {
		return g.skillPath
	}

	return workdir
}

func (g *GenericRuntime) prepareInputs(input map[string]any, workDir string) error {
	if g.config.Runtime.Inputs == nil {
		return nil
	}

	for _, inputFile := range g.config.Runtime.Inputs.Files {
		srcPath := g.resolveInputPath(inputFile.Path, input)
		if srcPath == "" {
			if inputFile.Required {
				return fmt.Errorf("required input %q is missing", inputFile.Name)
			}
			continue
		}

		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			if inputFile.Required {
				return fmt.Errorf("required input file %q does not exist: %s", inputFile.Name, srcPath)
			}
			continue
		}

		dest := filepath.Join(workDir, inputFile.Dest)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return fmt.Errorf("create dest dir for %q: %w", inputFile.Name, err)
		}

		if err := copyFile(srcPath, dest); err != nil {
			return fmt.Errorf("copy input %q: %w", inputFile.Name, err)
		}
	}

	return nil
}

func (g *GenericRuntime) resolveInputPath(pathTemplate string, input map[string]any) string {
	path := pathTemplate

	for key, val := range input {
		placeholder := fmt.Sprintf("{input.%s}", key)
		path = strings.ReplaceAll(path, placeholder, fmt.Sprintf("%v", val))
	}

	path = g.config.ResolvePath(path, g.skillPath, os.TempDir())
	return path
}

func (g *GenericRuntime) executeCommand(ctx context.Context, input map[string]any, workDir string) (SkillResult, error) {
	cmd, err := g.buildCommand(input, workDir)
	if err != nil {
		return g.errorResult(fmt.Sprintf("build command: %v", err), input), nil
	}

	env := g.buildEnv(workDir)

	cmd.Dir = workDir
	cmd.Env = env

	start := time.Now()
	output, err := cmd.CombinedOutput()
	latency := time.Since(start).Milliseconds()

	result := SkillResult{
		LatencyMS:  latency,
		TokenUsage: estimateTokenUsage(input),
		Trajectory: Trajectory{
			Steps:     []string{"generic_runtime", "execute_command", "parse_output"},
			ToolCalls: []string{cmd.Path},
		},
	}

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("exec error: %v, output: %s", err, string(output))
		result.Output = g.parseOutput(string(output))
		return result, nil
	}

	result.Success = true
	result.Output = g.parseOutput(string(output))

	return result, nil
}

func (g *GenericRuntime) buildCommand(input map[string]any, workDir string) (*exec.Cmd, error) {
	template := g.config.Runtime.Command.Template

	script := g.findScript()
	binary := g.findBinary()
	template = strings.ReplaceAll(template, "{script}", script)
	template = strings.ReplaceAll(template, "{binary}", binary)
	template = strings.ReplaceAll(template, "{args}", g.buildArgs(input))

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	return exec.Command(shell, "-c", template), nil
}

func (g *GenericRuntime) findScript() string {
	cfg := g.config.Runtime

	switch cfg.Type {
	case "python":
		return g.findMainScript("main.py", "generate.py")
	case "node":
		return g.findMainScript("main.js", "index.js")
	case "go":
		return g.findBinary()
	case "shell":
		return g.findMainScript("main.sh", "run.sh")
	case "typescript", "tsx":
		return g.findMainScript("main.tsx", "main.ts", "index.tsx", "index.ts", "generate.tsx", "generate.ts")
	case "bun":
		return g.findMainScript("main.ts", "index.ts", "run.ts")
	case "deno":
		return g.findMainScript("main.ts", "mod.ts", "index.ts")
	}

	return ""
}

func (g *GenericRuntime) findMainScript(fallbacks ...string) string {
	if len(fallbacks) == 0 {
		return ""
	}

	scriptsDir := filepath.Join(g.skillPath, "scripts")

	for _, name := range fallbacks {
		p := filepath.Join(scriptsDir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	for _, name := range fallbacks {
		p := filepath.Join(g.skillPath, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

func (g *GenericRuntime) findBinary() string {
	possible := []string{
		filepath.Join(g.skillPath, "bin", "main"),
		filepath.Join(g.skillPath, "main"),
		filepath.Join(g.skillPath, "cmd"),
	}

	for _, p := range possible {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}

	return filepath.Join(g.skillPath, "main")
}

func (g *GenericRuntime) buildArgs(input map[string]any) string {
	format := g.config.Runtime.Command.Args.Format
	if format == "" {
		format = "key-value"
	}

	mapping := g.config.Runtime.Command.Args.Mapping
	order := g.config.Runtime.Command.Args.PositionalOrder

	var args []string

	if format == "positional" && len(order) > 0 {
		for _, key := range order {
			if val, ok := input[key]; ok {
				args = append(args, fmt.Sprintf("%v", val))
			}
		}
	} else if format == "shell" {
		argStr := g.config.Runtime.Command.Args.Mapping["_"]
		if argStr != "" {
			for key, val := range input {
				argStr = strings.ReplaceAll(argStr, fmt.Sprintf("{%s}", key), fmt.Sprintf("%v", val))
			}
			return argStr
		}
	} else {
		for key, val := range input {
			strVal := fmt.Sprintf("%v", val)
			if mapping != nil {
				if mapped, ok := mapping[key]; ok {
					if mapped == "positional" {
						args = append(args, quoteIfNeeded(strVal))
					} else if strings.Contains(mapped, "{value}") {
						args = append(args, strings.ReplaceAll(mapped, "{value}", quoteIfNeeded(strVal)))
					}
				} else {
					args = append(args, "--"+key, quoteIfNeeded(strVal))
				}
			} else {
				args = append(args, "--"+key, quoteIfNeeded(strVal))
			}
		}
	}

	return strings.Join(args, " ")
}

func quoteIfNeeded(s string) string {
	if strings.Contains(s, " ") || strings.Contains(s, "'") || strings.Contains(s, "\"") {
		return fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "'\\''"))
	}
	return s
}

func (g *GenericRuntime) buildEnv(workDir string) []string {
	env := os.Environ()

	for key, val := range g.config.Runtime.Env {
		val = g.config.ResolveEnv(val, g.skillPath)
		env = append(env, fmt.Sprintf("%s=%s", key, val))
	}

	return env
}

func (g *GenericRuntime) executeDocker(ctx context.Context, input map[string]any, workDir string) (SkillResult, error) {
	template := g.config.Runtime.Command.Template
	image := g.config.Runtime.Command.Args.Mapping["image"]
	if image == "" {
		image = "python:3.12"
	}

	args := g.buildArgs(input)
	cmdStr := strings.ReplaceAll(template, "{image}", image)
	cmdStr = strings.ReplaceAll(cmdStr, "{args}", args)
	cmdStr = strings.ReplaceAll(cmdStr, "{workdir}", workDir)

	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", cmdStr)
	cmd.Dir = workDir
	cmd.Env = g.buildEnv(workDir)

	start := time.Now()
	output, err := cmd.CombinedOutput()
	latency := time.Since(start).Milliseconds()

	result := SkillResult{
		LatencyMS:  latency,
		TokenUsage: estimateTokenUsage(input),
		Trajectory: Trajectory{
			Steps:     []string{"generic_runtime", "docker_run", "parse_output"},
			ToolCalls: []string{"docker"},
		},
	}

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("docker error: %v, output: %s", err, string(output))
		result.Output = g.parseOutput(string(output))
		return result, nil
	}

	result.Success = true
	result.Output = g.parseOutput(string(output))

	return result, nil
}

func (g *GenericRuntime) executeHTTP(ctx context.Context, input map[string]any) (SkillResult, error) {
	endpoint := g.config.Runtime.Command.Template
	endpoint = g.resolveTemplate(endpoint, input)

	headers := make(map[string]string)
	for key, val := range g.config.Runtime.Command.Headers {
		headers[key] = g.resolveTemplate(val, input)
	}

	var body io.Reader
	if g.config.Runtime.Command.Body != "" {
		bodyStr := g.resolveTemplate(g.config.Runtime.Command.Body, input)
		body = strings.NewReader(bodyStr)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, body)
	if err != nil {
		return g.errorResult(fmt.Sprintf("create request: %v", err), input), nil
	}

	for key, val := range headers {
		req.Header.Set(key, val)
	}

	client := &http.Client{Timeout: defaultTimeout}
	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()

	result := SkillResult{
		LatencyMS:  latency,
		TokenUsage: estimateTokenUsage(input),
		Trajectory: Trajectory{
			Steps:     []string{"generic_runtime", "http_request"},
			ToolCalls: []string{endpoint},
		},
	}

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("http error: %v", err)
		return result, nil
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	result.Output = map[string]any{
		"status_code": resp.StatusCode,
		"body":        string(bodyBytes),
		"headers":     resp.Header,
	}

	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300

	return result, nil
}

func (g *GenericRuntime) resolveTemplate(template string, input map[string]any) string {
	result := template

	for key, val := range input {
		result = strings.ReplaceAll(result, fmt.Sprintf("{%s}", key), fmt.Sprintf("%v", val))
	}

	result = g.config.ResolveEnv(result, g.skillPath)
	result = g.config.ResolvePath(result, g.skillPath, os.TempDir())

	return result
}

func (g *GenericRuntime) parseOutput(output string) map[string]any {
	result := map[string]any{
		"stdout": output,
	}

	parseAs := g.config.Runtime.Output.Stdout.ParseAs
	if parseAs == "" {
		parseAs = "text"
	}

	if parseAs == "json" {
		var jsonData any
		if err := json.Unmarshal([]byte(output), &jsonData); err == nil {
			result["parsed"] = jsonData
		}
	}

	extract := g.config.Runtime.Output.Stdout.Extract
	if extract != nil {
		if extract.FilePattern != "" {
			files := g.extractWithRegex(output, extract.FilePattern)
			if len(files) > 0 {
				result["generated_files"] = files
			}
		}

		if extract.StatusPattern != "" {
			if matches := g.extractWithRegex(output, extract.StatusPattern); len(matches) > 0 {
				result["status"] = matches[0]
			}
		}
	}

	if _, ok := result["generated_files"]; !ok {
		files := g.extractDefaultFiles(output)
		if len(files) > 0 {
			result["generated_files"] = files
		}
	}

	return result
}

func (g *GenericRuntime) extractWithRegex(text, pattern string) []string {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(text, -1)

	var results []string
	for _, match := range matches {
		if len(match) > 1 {
			results = append(results, strings.TrimSpace(match[1]))
		}
	}

	return results
}

func (g *GenericRuntime) extractDefaultFiles(output string) []string {
	var files []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "generated:") ||
			strings.Contains(lower, "created:") ||
			strings.Contains(lower, "output:") ||
			strings.Contains(lower, "saved to:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) >= 2 {
				file := strings.TrimSpace(parts[len(parts)-1])
				file = strings.Trim(file, " `\"")
				if file != "" {
					files = append(files, file)
				}
			}
		}
	}

	return files
}

func (g *GenericRuntime) errorResult(errMsg string, input map[string]any) SkillResult {
	return SkillResult{
		Success:    false,
		Error:      errMsg,
		TokenUsage: estimateTokenUsage(input),
		Output:     map[string]any{},
		Trajectory: Trajectory{
			Steps: []string{"generic_runtime", "error"},
		},
	}
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

func RegisterGenericRuntime(skillPath string, cfg *RuntimeConfig) error {
	name := cfg.Runtime.Name
	if name == "" {
		name = filepath.Base(skillPath)
	}

	runtime := NewGenericRuntime(skillPath, cfg)
	RegisterRuntime(name, runtime)

	return nil
}

func estimateTokenUsage(input map[string]any) int64 {
	data, _ := json.Marshal(input)
	return int64(len(data) / 4)
}

var _ SkillRuntime = (*GenericRuntime)(nil)

func (g *GenericRuntime) RunCasesWithCases(skill string, cases []dataset.EvaluationCase, opts RunOptions) RunResult {
	return RunCasesWithOptions(skill, cases, g, opts)
}
