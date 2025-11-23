package recommendersystem

import (
	"math"
	"pc4/tools"
)

func CosineSimilarityAlgoCPU(a, b tools.UserVector) float64 {
	var dot, magA, magB float64

	for movie, rA := range a {
		if rB, ok := b[movie]; ok {
			dot += rA * rB
		}
		magA += rA * rA
	}
	for _, rB := range b {
		magB += rB * rB
	}

	if magA == 0 || magB == 0 {
		return 0
	}

	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}
