package semantic

import "math"

// CosineSimilarity calculates cosine similarity between two vectors.
func CosineSimilarity(a, b []float64) (float64, bool) {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0, false
	}

	var dotProduct float64
	var normA float64
	var normB float64

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0, false
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)), true
}
