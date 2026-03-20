package runner

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
)

var tokenPattern = regexp.MustCompile(`[a-z0-9_]+`)

// DriftReport captures output similarity between two runs.
type DriftReport struct {
	Threshold         float64           `json:"threshold"`
	AverageSimilarity float64           `json:"average_similarity"`
	DriftDetected     bool              `json:"drift_detected"`
	Cases             []CaseDriftResult `json:"cases"`
}

// CaseDriftResult captures similarity and drift status for a single case index.
type CaseDriftResult struct {
	CaseIndex    int     `json:"case_index"`
	Similarity   float64 `json:"similarity"`
	Drift        bool    `json:"drift"`
	CurrentText  string  `json:"current_text"`
	BaselineText string  `json:"baseline_text"`
}

// ComputeDriftReport computes embedding-style cosine similarity across run outputs.
func ComputeDriftReport(current RunResult, baseline RunResult, threshold float64) DriftReport {
	caseCount := min(len(current.Results), len(baseline.Results))
	report := DriftReport{
		Threshold: threshold,
		Cases:     make([]CaseDriftResult, 0, caseCount),
	}

	if caseCount == 0 {
		return report
	}

	var totalSimilarity float64
	for index := 0; index < caseCount; index++ {
		currentText := normalizeOutputText(current.Results[index].Output)
		baselineText := normalizeOutputText(baseline.Results[index].Output)
		similarity := cosineSimilarity(currentText, baselineText)
		drift := similarity < threshold

		report.Cases = append(report.Cases, CaseDriftResult{
			CaseIndex:    index,
			Similarity:   roundFloat(similarity),
			Drift:        drift,
			CurrentText:  currentText,
			BaselineText: baselineText,
		})
		totalSimilarity += similarity
		if drift {
			report.DriftDetected = true
		}
	}

	report.AverageSimilarity = roundFloat(totalSimilarity / float64(caseCount))
	return report
}

func normalizeOutputText(output map[string]any) string {
	if len(output) == 0 {
		return ""
	}

	data, err := json.Marshal(output)
	if err != nil {
		return ""
	}

	text := strings.ToLower(string(data))
	tokens := tokenPattern.FindAllString(text, -1)
	return strings.Join(tokens, " ")
}

func cosineSimilarity(left string, right string) float64 {
	leftVector := textVector(left)
	rightVector := textVector(right)
	if len(leftVector) == 0 && len(rightVector) == 0 {
		return 1
	}
	if len(leftVector) == 0 || len(rightVector) == 0 {
		return 0
	}

	var dotProduct float64
	var leftNorm float64
	var rightNorm float64

	for token, leftValue := range leftVector {
		leftNorm += leftValue * leftValue
		dotProduct += leftValue * rightVector[token]
	}
	for _, rightValue := range rightVector {
		rightNorm += rightValue * rightValue
	}

	if leftNorm == 0 || rightNorm == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(leftNorm) * math.Sqrt(rightNorm))
}

func textVector(text string) map[string]float64 {
	vector := map[string]float64{}
	for _, token := range strings.Fields(text) {
		vector[token]++
	}
	return vector
}

// FormatDriftSummary returns a plain-text drift summary for terminal output.
func FormatDriftSummary(report DriftReport) string {
	lines := []string{
		"Drift report",
		fmt.Sprintf("average_similarity: %.6f", report.AverageSimilarity),
		fmt.Sprintf("drift_detected: %t", report.DriftDetected),
	}

	drifted := make([]int, 0)
	for _, result := range report.Cases {
		if result.Drift {
			drifted = append(drifted, result.CaseIndex)
		}
	}
	slices.Sort(drifted)
	if len(drifted) == 0 {
		lines = append(lines, "drift_cases: none")
	} else {
		indexes := make([]string, 0, len(drifted))
		for _, index := range drifted {
			indexes = append(indexes, fmt.Sprintf("%d", index))
		}
		lines = append(lines, "drift_cases: "+strings.Join(indexes, ","))
	}

	return strings.Join(lines, "\n") + "\n"
}
