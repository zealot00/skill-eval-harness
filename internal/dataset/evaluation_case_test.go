package dataset

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestEvaluationCaseTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		field   string
		jsonTag string
		yamlTag string
	}{
		{name: "case id", field: "CaseID", jsonTag: "case_id", yamlTag: "case_id"},
		{name: "skill", field: "Skill", jsonTag: "skill", yamlTag: "skill"},
		{name: "input", field: "Input", jsonTag: "input", yamlTag: "input"},
		{name: "expected", field: "Expected", jsonTag: "expected", yamlTag: "expected"},
		{name: "tags", field: "Tags", jsonTag: "tags", yamlTag: "tags"},
	}

	evaluationCaseType := reflect.TypeOf(EvaluationCase{})

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			field, ok := evaluationCaseType.FieldByName(tt.field)
			if !ok {
				t.Fatalf("field %q not found", tt.field)
			}

			if got := field.Tag.Get("json"); got != tt.jsonTag {
				t.Fatalf("json tag = %q, want %q", got, tt.jsonTag)
			}

			if got := field.Tag.Get("yaml"); got != tt.yamlTag {
				t.Fatalf("yaml tag = %q, want %q", got, tt.yamlTag)
			}
		})
	}
}

func TestLoadCases(t *testing.T) {
	t.Parallel()

	t.Run("loads yaml files recursively", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		writeTestFile(t, filepath.Join(dir, "manifest.yaml"), `
dataset_name: sample-dataset
version: v1
owner: qa
description: sample
`)
		writeTestFile(t, filepath.Join(dir, "one.yaml"), `
case_id: case-1
skill: skill-a
input:
  prompt: alpha
expected:
  result: ok
tags:
  - smoke
`)
		writeTestFile(t, filepath.Join(dir, "nested", "two.yaml"), `
case_id: case-2
skill: skill-b
`)
		writeTestFile(t, filepath.Join(dir, "nested", "deep", "three.yaml"), `
case_id: case-3
skill: skill-c
`)
		writeTestFile(t, filepath.Join(dir, "ignore.txt"), "not yaml")

		cases, err := LoadCases(dir)
		if err != nil {
			t.Fatalf("LoadCases() error = %v", err)
		}

		if got := len(cases); got != 3 {
			t.Fatalf("len(cases) = %d, want 3", got)
		}
	})

	t.Run("missing case id returns error", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		writeTestFile(t, filepath.Join(dir, "invalid.yaml"), `
skill: skill-a
input:
  prompt: alpha
`)

		_, err := LoadCases(dir)
		if err == nil {
			t.Fatal("LoadCases() error = nil, want error")
		}

		if !strings.Contains(err.Error(), "case_id is required") {
			t.Fatalf("error = %q, want missing case_id message", err)
		}
	})
}

func TestFilterByTag(t *testing.T) {
	t.Parallel()

	cases := []EvaluationCase{
		{CaseID: "case-1", Tags: []string{"regression", "smoke"}},
		{CaseID: "case-2", Tags: []string{"smoke"}},
		{CaseID: "case-3", Tags: []string{"regression"}},
	}

	got := FilterByTag(cases, "regression")
	if len(got) != 2 {
		t.Fatalf("len(FilterByTag()) = %d, want 2", len(got))
	}

	if got[0].CaseID != "case-1" || got[1].CaseID != "case-3" {
		t.Fatalf("filtered cases = %#v, want case-1 and case-3", got)
	}
}

func TestFilterBySample(t *testing.T) {
	t.Parallel()

	cases := make([]EvaluationCase, 0, 10)
	for i := 0; i < 10; i++ {
		cases = append(cases, EvaluationCase{CaseID: string(rune('a' + i))})
	}

	got := FilterBySample(cases, 0.2, 42)
	if len(got) != 2 {
		t.Fatalf("len(FilterBySample()) = %d, want 2", len(got))
	}

	gotRepeat := FilterBySample(cases, 0.2, 42)
	if !reflect.DeepEqual(got, gotRepeat) {
		t.Fatalf("FilterBySample() not deterministic with seed: %#v != %#v", got, gotRepeat)
	}
}

func TestLoadManifest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "manifest.yaml"), `
dataset_name: sample-dataset
version: v1.2.3
owner: qa-team
description: regression pack
`)

	got, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}

	if got.DatasetName != "sample-dataset" {
		t.Fatalf("DatasetName = %q, want %q", got.DatasetName, "sample-dataset")
	}

	if got.Version != "v1.2.3" {
		t.Fatalf("Version = %q, want %q", got.Version, "v1.2.3")
	}

	if got.Owner != "qa-team" {
		t.Fatalf("Owner = %q, want %q", got.Owner, "qa-team")
	}

	if got.Description != "regression pack" {
		t.Fatalf("Description = %q, want %q", got.Description, "regression pack")
	}
}

func TestComputeDatasetHash(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "manifest.yaml"), `
dataset_name: sample-dataset
version: v1.2.3
owner: qa-team
description: regression pack
`)
	writeTestFile(t, filepath.Join(dir, "one.yaml"), `
case_id: case-1
skill: skill-a
`)

	hashOne, err := ComputeDatasetHash(dir)
	if err != nil {
		t.Fatalf("ComputeDatasetHash() error = %v", err)
	}

	writeTestFile(t, filepath.Join(dir, "one.yaml"), `
case_id: case-1
skill: skill-b
`)

	hashTwo, err := ComputeDatasetHash(dir)
	if err != nil {
		t.Fatalf("ComputeDatasetHash() error = %v", err)
	}

	if hashOne == hashTwo {
		t.Fatalf("dataset hash did not change: %q", hashOne)
	}
}

func writeTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}

	if err := os.WriteFile(path, []byte(strings.TrimSpace(contents)), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
