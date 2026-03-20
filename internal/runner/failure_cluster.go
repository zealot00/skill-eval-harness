package runner

import "fmt"

const failureClusterThreshold = 0.8

func assignFailureClusters(results []CaseRunResult) []CaseRunResult {
	clusterTexts := make([]string, 0)
	clusterIDs := make([]string, 0)

	for index := range results {
		if results[index].Success {
			continue
		}

		text := normalizeFailureText(results[index])
		if text == "" {
			continue
		}

		bestCluster := -1
		bestSimilarity := 0.0
		for clusterIndex, clusterText := range clusterTexts {
			similarity := cosineSimilarity(text, clusterText)
			if similarity > bestSimilarity {
				bestSimilarity = similarity
				bestCluster = clusterIndex
			}
		}

		if bestCluster >= 0 && bestSimilarity >= failureClusterThreshold {
			results[index].FailureClusterID = clusterIDs[bestCluster]
			continue
		}

		clusterID := fmt.Sprintf("cluster-%d", len(clusterIDs)+1)
		clusterTexts = append(clusterTexts, text)
		clusterIDs = append(clusterIDs, clusterID)
		results[index].FailureClusterID = clusterID
	}

	return results
}

func normalizeFailureText(result CaseRunResult) string {
	text := normalizeOutputText(result.Output)
	if result.Error == "" {
		return text
	}
	if text == "" {
		return result.Error
	}
	return text + " " + result.Error
}
