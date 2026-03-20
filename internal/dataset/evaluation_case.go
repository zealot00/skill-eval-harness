package dataset

import (
	"crypto/sha256"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// EvaluationCase defines a single skill evaluation input and expected output.
type EvaluationCase struct {
	CaseID   string         `json:"case_id" yaml:"case_id"`
	Skill    string         `json:"skill" yaml:"skill"`
	Input    map[string]any `json:"input" yaml:"input"`
	Expected map[string]any `json:"expected" yaml:"expected"`
	Tags     []string       `json:"tags" yaml:"tags"`
}

// Manifest describes dataset metadata for a case directory.
type Manifest struct {
	DatasetName string `json:"dataset_name" yaml:"dataset_name"`
	Version     string `json:"version" yaml:"version"`
	Owner       string `json:"owner" yaml:"owner"`
	Description string `json:"description" yaml:"description"`
}

// LoadCases recursively loads YAML evaluation cases from dir.
func LoadCases(dir string) ([]EvaluationCase, error) {
	var paths []string

	if err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %q: %w", path, err)
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".yaml" {
			return nil
		}

		if filepath.Base(path) == "manifest.yaml" {
			return nil
		}

		paths = append(paths, path)
		return nil
	}); err != nil {
		return nil, err
	}

	sort.Strings(paths)

	cases := make([]EvaluationCase, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", path, err)
		}

		var evaluationCase EvaluationCase
		if err := yaml.Unmarshal(data, &evaluationCase); err != nil {
			return nil, fmt.Errorf("unmarshal %q: %w", path, err)
		}

		if err := validateEvaluationCase(evaluationCase, path); err != nil {
			return nil, err
		}

		cases = append(cases, evaluationCase)
	}

	return cases, nil
}

// LoadManifest reads dataset metadata from manifest.yaml in dir.
func LoadManifest(dir string) (Manifest, error) {
	path := filepath.Join(dir, "manifest.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read %q: %w", path, err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("unmarshal %q: %w", path, err)
	}

	return manifest, nil
}

// ComputeDatasetHash returns a SHA256 over all YAML files in the dataset directory.
func ComputeDatasetHash(dir string) (string, error) {
	var paths []string

	if err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %q: %w", path, err)
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".yaml" {
			return nil
		}

		paths = append(paths, path)
		return nil
	}); err != nil {
		return "", err
	}

	sort.Strings(paths)

	hash := sha256.New()
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %q: %w", path, err)
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return "", fmt.Errorf("rel %q: %w", path, err)
		}

		if _, err := hash.Write([]byte(relPath)); err != nil {
			return "", fmt.Errorf("hash path %q: %w", relPath, err)
		}
		if _, err := hash.Write([]byte{0}); err != nil {
			return "", fmt.Errorf("hash separator: %w", err)
		}
		if _, err := hash.Write(data); err != nil {
			return "", fmt.Errorf("hash data %q: %w", path, err)
		}
		if _, err := hash.Write([]byte{0}); err != nil {
			return "", fmt.Errorf("hash separator: %w", err)
		}
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// FilterByTag returns only cases containing the provided tag.
func FilterByTag(cases []EvaluationCase, tag string) []EvaluationCase {
	if strings.TrimSpace(tag) == "" {
		return cases
	}

	filtered := make([]EvaluationCase, 0, len(cases))
	for _, evaluationCase := range cases {
		for _, caseTag := range evaluationCase.Tags {
			if caseTag == tag {
				filtered = append(filtered, evaluationCase)
				break
			}
		}
	}

	return filtered
}

// FilterBySample returns an approximate random sample of cases.
func FilterBySample(cases []EvaluationCase, sample float64, seed int64) []EvaluationCase {
	if sample <= 0 || len(cases) == 0 {
		return cases
	}

	if sample >= 1 {
		return cases
	}

	target := int(math.Round(float64(len(cases)) * sample))
	if target < 1 {
		target = 1
	}
	if target >= len(cases) {
		return cases
	}

	sourceSeed := seed
	if sourceSeed == 0 {
		sourceSeed = time.Now().UnixNano()
	}

	indexes := make([]int, len(cases))
	for index := range cases {
		indexes[index] = index
	}

	random := rand.New(rand.NewSource(sourceSeed))
	random.Shuffle(len(indexes), func(i, j int) {
		indexes[i], indexes[j] = indexes[j], indexes[i]
	})

	indexes = indexes[:target]
	sort.Ints(indexes)

	filtered := make([]EvaluationCase, 0, len(indexes))
	for _, index := range indexes {
		filtered = append(filtered, cases[index])
	}

	return filtered
}

func validateEvaluationCase(evaluationCase EvaluationCase, path string) error {
	if strings.TrimSpace(evaluationCase.CaseID) == "" {
		return fmt.Errorf("validate %q: case_id is required", path)
	}

	if strings.TrimSpace(evaluationCase.Skill) == "" {
		return fmt.Errorf("validate %q: skill is required", path)
	}

	return nil
}
