package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCosineSimilarity(t *testing.T) {
	tests := map[string]struct {
		vectorA     []float64
		vectorB     []float64
		wantScore   float64
		wantSuccess bool
	}{
		"identical-vectors-return-1.0": {
			vectorA:     []float64{1.0, 2.0, 3.0},
			vectorB:     []float64{1.0, 2.0, 3.0},
			wantScore:   1.0,
			wantSuccess: true,
		},
		"opposite-vectors-return-negative-1.0": {
			vectorA:     []float64{1.0, 2.0, 3.0},
			vectorB:     []float64{-1.0, -2.0, -3.0},
			wantScore:   -1.0,
			wantSuccess: true,
		},
		"orthogonal-vectors-return-0.0": {
			vectorA:     []float64{1.0, 0.0},
			vectorB:     []float64{0.0, 1.0},
			wantScore:   0.0,
			wantSuccess: true,
		},
		"similar-vectors-return-high-score": {
			vectorA:     []float64{1.0, 2.0, 3.0},
			vectorB:     []float64{2.0, 4.0, 6.0},
			wantScore:   1.0,
			wantSuccess: true,
		},
		"partially-similar-vectors": {
			vectorA:     []float64{1.0, 1.0, 0.0},
			vectorB:     []float64{1.0, 0.0, 1.0},
			wantScore:   0.5,
			wantSuccess: true,
		},
		"empty-first-vector-returns-false": {
			vectorA:     []float64{},
			vectorB:     []float64{1.0, 2.0, 3.0},
			wantScore:   0,
			wantSuccess: false,
		},
		"empty-second-vector-returns-false": {
			vectorA:     []float64{1.0, 2.0, 3.0},
			vectorB:     []float64{},
			wantScore:   0,
			wantSuccess: false,
		},
		"both-vectors-empty-returns-false": {
			vectorA:     []float64{},
			vectorB:     []float64{},
			wantScore:   0,
			wantSuccess: false,
		},
		"different-length-vectors-returns-false": {
			vectorA:     []float64{1.0, 2.0},
			vectorB:     []float64{1.0, 2.0, 3.0},
			wantScore:   0,
			wantSuccess: false,
		},
		"zero-vector-first-returns-false": {
			vectorA:     []float64{0.0, 0.0, 0.0},
			vectorB:     []float64{1.0, 2.0, 3.0},
			wantScore:   0,
			wantSuccess: false,
		},
		"zero-vector-second-returns-false": {
			vectorA:     []float64{1.0, 2.0, 3.0},
			vectorB:     []float64{0.0, 0.0, 0.0},
			wantScore:   0,
			wantSuccess: false,
		},
		"both-zero-vectors-returns-false": {
			vectorA:     []float64{0.0, 0.0},
			vectorB:     []float64{0.0, 0.0},
			wantScore:   0,
			wantSuccess: false,
		},
		"single-element-vectors": {
			vectorA:     []float64{5.0},
			vectorB:     []float64{3.0},
			wantScore:   1.0,
			wantSuccess: true,
		},
		"negative-values": {
			vectorA:     []float64{-1.0, -2.0, -3.0},
			vectorB:     []float64{-2.0, -4.0, -6.0},
			wantScore:   1.0,
			wantSuccess: true,
		},
		"mixed-positive-and-negative": {
			vectorA:     []float64{1.0, -1.0, 2.0},
			vectorB:     []float64{-1.0, 1.0, -2.0},
			wantScore:   -1.0,
			wantSuccess: true,
		},
		"high-dimensional-vectors": {
			vectorA:     []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8},
			vectorB:     []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8},
			wantScore:   1.0,
			wantSuccess: true,
		},
		"very-small-values": {
			vectorA:     []float64{0.0001, 0.0002, 0.0003},
			vectorB:     []float64{0.0001, 0.0002, 0.0003},
			wantScore:   1.0,
			wantSuccess: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			score, success := CosineSimilarity(tt.vectorA, tt.vectorB)

			assert.Equal(t, tt.wantSuccess, success)
			if tt.wantSuccess {
				assert.InDelta(t, tt.wantScore, score, 0.0001,
					"Expected score %.4f but got %.4f for %s",
					tt.wantScore, score)
			} else {
				assert.Equal(t, tt.wantScore, score,
					"Failed case should return 0 as score")
			}
		})
	}
}
