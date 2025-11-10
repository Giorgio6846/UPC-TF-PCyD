package main

import (
	"math"
	"sort"
)

type neighbor struct {
	UserID int
	Score  float64
}

type recommended struct {
	MovieID int
	Score   float64
	Count   int
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

func RecommendFromTopK(
	target int,
	neighbors []neighbor,
	userVec map[int]map[int]float64,
	topN int,
	minAbsSim float64,
	minNeighborsPerItem int,
) []recommended {
	userMean := userMeans(userVec)

	seen := make(map[int]bool, len(userVec[target]))
	for m := range userVec[target] {
		seen[m] = true
	}

	type agg struct {
		num, den float64
		cnt      int
	}
	score := make(map[int]*agg)
	for _, nb := range neighbors {
		if math.Abs(nb.Score) < minAbsSim {
			continue
		}
		rv := userVec[nb.UserID]
		if rv == nil {
			continue
		}
		mv := userMean[nb.UserID]
		for mid, r := range rv {
			if seen[mid] {
				continue
			}
			a := score[mid]
			if a == nil {
				a = &agg{}
				score[mid] = a
			}
			a.num += nb.Score * (r - mv)
			a.den += math.Abs(nb.Score)
			a.cnt++
		}
	}

	recs := make([]recommended, 0, len(score))
	mu := userMean[target]
	for mid, a := range score {
		if a.den == 0 || a.cnt < minNeighborsPerItem {
			continue
		}
		pred := mu + (a.num / a.den)
		recs = append(recs, recommended{MovieID: mid, Score: pred, Count: a.cnt})
	}

	sort.Slice(recs, func(i, j int) bool {
		if recs[i].Score == recs[j].Score {
			return recs[i].Count > recs[j].Count
		}
		return recs[i].Score > recs[j].Score
	})
	if topN > 0 && topN < len(recs) {
		return recs[:topN]
	}
	return recs
}

func userMeans(userVec map[int]map[int]float64) map[int]float64 {
	m := make(map[int]float64, len(userVec))
	for uid, vec := range userVec {
		var s float64
		for _, r := range vec {
			s += r
		}
		m[uid] = s / float64(len(vec))
	}
	return m
}
