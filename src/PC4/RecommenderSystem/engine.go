package recommendersystem

import (
	"math"
	"pc4/tools"
	"sort"
)

func BuildUserVectors(ratings []tools.Rating) map[int]tools.UserVector {
	userVec := make(map[int]tools.UserVector)

	for _, x := range ratings {
		userID := int(x.UserID)
		movieID := int(x.MovieID)

		if _, ok := userVec[userID]; !ok {
			userVec[userID] = make(tools.UserVector)
		}

		userVec[userID][movieID] = x.Rating
	}

	return userVec
}

func UserVectorSplitter(targetVector tools.UserVector, m map[int]tools.UserVector, chunks int) []tools.SimilarityVector {
	if chunks <= 0 {
		return nil
	}

	result := make([]tools.SimilarityVector, chunks)
	for i := 0; i < chunks; i++ {
		result[i].TargetVector = targetVector
		result[i].UsersVector = make(map[int]tools.UserVector)
	}

	i := 0
	for k, v := range m {
		idx := i % chunks
		result[idx].UsersVector[k] = v
		i++
	}

	return result
}

func RecommendFromTopK(
	target int,
	neighbors []tools.Similarity,
	userVec map[int]tools.UserVector,
	topN int,
	minAbsSim float64,
	minNeighborsPerItem int,
) []tools.Recommended {

	if _, ok := userVec[target]; !ok {
		return nil
	}

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
		if math.Abs(nb.Similarity) < minAbsSim {
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
			a.num += nb.Similarity * (r - mv)
			a.den += math.Abs(nb.Similarity)
			a.cnt++
		}
	}

	recs := make([]tools.Recommended, 0, len(score))
	mu := userMean[target]
	for mid, a := range score {
		if a.den == 0 || a.cnt < minNeighborsPerItem {
			continue
		}
		pred := mu + (a.num / a.den)
		recs = append(recs, tools.Recommended{MovieID: mid, Score: pred, Count: a.cnt})
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

func userMeans(userVec map[int]tools.UserVector) tools.UserVector {
	m := make(tools.UserVector, len(userVec))
	for uid, vec := range userVec {
		var s float64
		for _, r := range vec {
			s += r
		}
		m[uid] = s / float64(len(vec))
	}
	return m
}
