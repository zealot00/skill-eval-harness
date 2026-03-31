package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestTypeScriptRuntime_Execute(t *testing.T) {
	t.Parallel()

	skillPath := t.TempDir()
	scriptsDir := filepath.Join(skillPath, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatalf("create scripts dir: %v", err)
	}

	tsScript := filepath.Join(scriptsDir, "generate.ts")
	if err := os.WriteFile(tsScript, []byte(`#!/usr/bin/env ts-node
console.log(JSON.stringify({ status: "ok", received: process.argv[2] }));
`), 0755); err != nil {
		t.Fatalf("write ts script: %v", err)
	}

	cfg := &RuntimeConfig{
		Runtime: RuntimeSpec{
			Type: "typescript",
			Command: CommandSpec{
				Template: "ts-node {script} {args}",
			},
		},
	}

	rt := NewGenericRuntime(skillPath, cfg)
	result, err := rt.Execute(context.Background(), map[string]any{"name": "test"})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Fatalf("Execute().Success = false, want true; error = %s", result.Error)
	}

	if result.Output == nil {
		t.Fatal("Execute().Output = nil, want map")
	}
}

func TestTypeScriptRuntime_FindScript(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		files      []string
		wantScript string
	}{
		{
			name:       "prefers main.ts in scripts/",
			files:      []string{"scripts/main.ts"},
			wantScript: "scripts/main.ts",
		},
		{
			name:       "falls back to index.ts",
			files:      []string{"scripts/index.ts"},
			wantScript: "scripts/index.ts",
		},
		{
			name:       "finds tsx files",
			files:      []string{"scripts/generate.tsx"},
			wantScript: "scripts/generate.tsx",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			skillPath := t.TempDir()
			for _, f := range tt.files {
				fullPath := filepath.Join(skillPath, f)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatalf("create dir: %v", err)
				}
				if err := os.WriteFile(fullPath, []byte("// test"), 0644); err != nil {
					t.Fatalf("write file: %v", err)
				}
			}

			cfg := &RuntimeConfig{
				Runtime: RuntimeSpec{
					Type: "typescript",
					Command: CommandSpec{
						Template: "ts-node {script} {args}",
					},
				},
			}

			rt := NewGenericRuntime(skillPath, cfg)
			script := rt.findScript()

			wantPath := filepath.Join(skillPath, tt.wantScript)
			if script != wantPath {
				t.Errorf("findScript() = %s, want %s", script, wantPath)
			}
		})
	}
}

func TestBunRuntime_Execute(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("bun"); err != nil {
		t.Skip("bun not installed")
	}

	skillPath := t.TempDir()
	scriptsDir := filepath.Join(skillPath, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatalf("create scripts dir: %v", err)
	}

	bunScript := filepath.Join(scriptsDir, "generate.ts")
	if err := os.WriteFile(bunScript, []byte(`#!/usr/bin/env bun
console.log(JSON.stringify({ status: "ok", bun: true }));
`), 0755); err != nil {
		t.Fatalf("write bun script: %v", err)
	}

	cfg := &RuntimeConfig{
		Runtime: RuntimeSpec{
			Type: "bun",
			Command: CommandSpec{
				Template: "bun {script} {args}",
			},
		},
	}

	rt := NewGenericRuntime(skillPath, cfg)
	result, err := rt.Execute(context.Background(), map[string]any{"test": true})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Fatalf("Execute().Success = false, want true; error = %s", result.Error)
	}
}

func TestBunRuntime_FindScript(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		files      []string
		wantScript string
	}{
		{
			name:       "prefers main.ts in scripts/",
			files:      []string{"scripts/main.ts"},
			wantScript: "scripts/main.ts",
		},
		{
			name:       "finds index.ts",
			files:      []string{"scripts/index.ts"},
			wantScript: "scripts/index.ts",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			skillPath := t.TempDir()
			for _, f := range tt.files {
				fullPath := filepath.Join(skillPath, f)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatalf("create dir: %v", err)
				}
				if err := os.WriteFile(fullPath, []byte("// test"), 0644); err != nil {
					t.Fatalf("write file: %v", err)
				}
			}

			cfg := &RuntimeConfig{
				Runtime: RuntimeSpec{
					Type: "bun",
					Command: CommandSpec{
						Template: "bun {script} {args}",
					},
				},
			}

			rt := NewGenericRuntime(skillPath, cfg)
			script := rt.findScript()

			wantPath := filepath.Join(skillPath, tt.wantScript)
			if script != wantPath {
				t.Errorf("findScript() = %s, want %s", script, wantPath)
			}
		})
	}
}

func TestRustRuntime_Execute(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo not installed")
	}

	skillPath := t.TempDir()
	binDir := filepath.Join(skillPath, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("create bin dir: %v", err)
	}

	binary := filepath.Join(binDir, "main")
	if err := os.WriteFile(binary, []byte("#!/bin/bash\necho '{\"status\":\"ok\"}'"), 0755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	cfg := &RuntimeConfig{
		Runtime: RuntimeSpec{
			Type: "rust",
			Command: CommandSpec{
				Template: "{binary} {args}",
			},
		},
	}

	rt := NewGenericRuntime(skillPath, cfg)
	result, err := rt.Execute(context.Background(), map[string]any{"test": true})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Fatalf("Execute().Success = false, want true; error = %s", result.Error)
	}
}

func TestRustRuntime_FindBinary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		files      []string
		wantBinary string
	}{
		{
			name:       "finds binary in bin/ directory",
			files:      []string{"bin/main"},
			wantBinary: "bin/main",
		},
		{
			name:       "finds main binary in root",
			files:      []string{"main"},
			wantBinary: "main",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			skillPath := t.TempDir()
			for _, f := range tt.files {
				fullPath := filepath.Join(skillPath, f)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatalf("create dir: %v", err)
				}
				if err := os.WriteFile(fullPath, []byte("#!/bin/bash"), 0755); err != nil {
					t.Fatalf("write file: %v", err)
				}
			}

			cfg := &RuntimeConfig{
				Runtime: RuntimeSpec{
					Type: "rust",
					Command: CommandSpec{
						Template: "{binary} {args}",
					},
				},
			}

			rt := NewGenericRuntime(skillPath, cfg)
			binary := rt.findBinary()

			wantPath := filepath.Join(skillPath, tt.wantBinary)
			if binary != wantPath {
				t.Errorf("findBinary() = %s, want %s", binary, wantPath)
			}
		})
	}
}

func TestDenoRuntime_Execute(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("deno"); err != nil {
		t.Skip("deno not installed")
	}

	skillPath := t.TempDir()
	scriptsDir := filepath.Join(skillPath, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatalf("create scripts dir: %v", err)
	}

	denoScript := filepath.Join(scriptsDir, "generate.ts")
	if err := os.WriteFile(denoScript, []byte(`#!/usr/bin/env deno
console.log(JSON.stringify({ status: "ok", deno: true }));
`), 0755); err != nil {
		t.Fatalf("write deno script: %v", err)
	}

	cfg := &RuntimeConfig{
		Runtime: RuntimeSpec{
			Type: "deno",
			Command: CommandSpec{
				Template: "deno run --allow-all {script} {args}",
			},
		},
	}

	rt := NewGenericRuntime(skillPath, cfg)
	result, err := rt.Execute(context.Background(), map[string]any{"test": true})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Success {
		t.Fatalf("Execute().Success = false, want true; error = %s", result.Error)
	}
}

func TestDenoRuntime_FindScript(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		files      []string
		wantScript string
	}{
		{
			name:       "prefers main.ts in scripts/",
			files:      []string{"scripts/main.ts"},
			wantScript: "scripts/main.ts",
		},
		{
			name:       "finds mod.ts",
			files:      []string{"scripts/mod.ts"},
			wantScript: "scripts/mod.ts",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			skillPath := t.TempDir()
			for _, f := range tt.files {
				fullPath := filepath.Join(skillPath, f)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatalf("create dir: %v", err)
				}
				if err := os.WriteFile(fullPath, []byte("// test"), 0644); err != nil {
					t.Fatalf("write file: %v", err)
				}
			}

			cfg := &RuntimeConfig{
				Runtime: RuntimeSpec{
					Type: "deno",
					Command: CommandSpec{
						Template: "deno run --allow-all {script} {args}",
					},
				},
			}

			rt := NewGenericRuntime(skillPath, cfg)
			script := rt.findScript()

			wantPath := filepath.Join(skillPath, tt.wantScript)
			if script != wantPath {
				t.Errorf("findScript() = %s, want %s", script, wantPath)
			}
		})
	}
}

func TestRuntimeConfig_Validate_SupportedTypes(t *testing.T) {
	t.Parallel()

	validTypes := []string{
		"python", "node", "go", "shell", "docker", "http", "command",
		"typescript", "tsx", "bun", "rust", "deno",
	}

	for _, runtimeType := range validTypes {
		runtimeType := runtimeType
		t.Run(runtimeType, func(t *testing.T) {
			t.Parallel()

			cfg := &RuntimeConfig{
				Runtime: RuntimeSpec{
					Type: runtimeType,
					Command: CommandSpec{
						Template: "echo test",
					},
				},
			}

			if err := cfg.Validate(); err != nil {
				t.Errorf("Validate() error = %v for type %s", err, runtimeType)
			}
		})
	}
}

func TestRuntimeConfig_Validate_UnsupportedTypes(t *testing.T) {
	t.Parallel()

	invalidTypes := []string{
		"ruby", "php", "perl", "lua", "kotlin",
	}

	for _, runtimeType := range invalidTypes {
		runtimeType := runtimeType
		t.Run(runtimeType, func(t *testing.T) {
			t.Parallel()

			cfg := &RuntimeConfig{
				Runtime: RuntimeSpec{
					Type: runtimeType,
					Command: CommandSpec{
						Template: "echo test",
					},
				},
			}

			if err := cfg.Validate(); err == nil {
				t.Errorf("Validate() expected error for type %s, got nil", runtimeType)
			}
		})
	}
}

func TestFindScript_AllSupportedTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		runtimeType string
		fallbacks   []string
	}{
		{runtimeType: "typescript", fallbacks: []string{"main.ts", "index.ts", "generate.ts"}},
		{runtimeType: "bun", fallbacks: []string{"main.ts", "index.ts", "run.ts"}},
		{runtimeType: "deno", fallbacks: []string{"main.ts", "mod.ts", "index.ts"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.runtimeType, func(t *testing.T) {
			t.Parallel()

			skillPath := t.TempDir()
			scriptsDir := filepath.Join(skillPath, "scripts")
			if err := os.MkdirAll(scriptsDir, 0755); err != nil {
				t.Fatalf("create scripts dir: %v", err)
			}

			if len(tt.fallbacks) > 0 {
				scriptPath := filepath.Join(scriptsDir, tt.fallbacks[0])
				if err := os.WriteFile(scriptPath, []byte("// test"), 0644); err != nil {
					t.Fatalf("write file: %v", err)
				}
			}

			cfg := &RuntimeConfig{
				Runtime: RuntimeSpec{
					Type: tt.runtimeType,
					Command: CommandSpec{
						Template: "echo {script}",
					},
				},
			}

			rt := NewGenericRuntime(skillPath, cfg)
			script := rt.findScript()

			if script == "" {
				t.Errorf("findScript() returned empty for type %s", tt.runtimeType)
			}

			if _, err := os.Stat(script); os.IsNotExist(err) {
				t.Errorf("findScript() returned non-existent path: %s", script)
			}
		})
	}
}
