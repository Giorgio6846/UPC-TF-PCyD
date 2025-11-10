package main

import (
	"math"
	"sort"
)

type neighbor struct {
	UserID int
	Score  float64
}

func buildUserVectors(ratings []Rating) (map[int]map[int]float64, map[int]float64) {
	userVec := make(map[int]map[int]float64)

	for _, x := range ratings {
		movie, ok := userVec[int(x.UserID)]
		if !ok {
			movie = make(map[int]float64)
			userVec[int(x.UserID)] = movie
		}
		movie[int(x.MovieID)] = x.Rating
	}

	norms := make(map[int]float64, len(userVec))
	for uid, vec := range userVec {
		var sum float64
		for _, v := range vec {
			sum += v * v
		}
		norms[uid] = math.Sqrt(sum)
	}
	return userVec, norms
}

// First solution via user -> movie/rating
func cosineSim(a, b map[int]float64, na, nb float64) float64 {
	if na == 0 || nb == 0 {
		return 0
	}

	if len(a) > len(b) {
		a, b = b, a
		na, nb = nb, na
	}

	var dot float64
	for k, va := range a {
		if vb, ok := b[k]; ok {
			dot += va * vb
		}
	}
	return dot / (na * nb)
}

func topKUsers(user int, userVecs map[int]map[int]float64, norms map[int]float64, K int) []neighbor {
	targetVector := userVecs[user]
	targetNeighbor := norms[user]

	out := make([]neighbor, 0, len(userVecs)-1)
	for userID, vector := range userVecs {
		if userID == user {
			continue
		}
		score := cosineSim(targetVector, vector, targetNeighbor, norms[userID])
		if score > 0 {
			out = append(out, neighbor{UserID: userID, Score: score})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > K {
		out = out[:K]
	}
	return out
}
