package main

import (
	"math"
	"runtime"
	"sort"
	"sync"
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

func topKUsersConcurrent(user int, userVecs map[int]map[int]float64, norms map[int]float64, K int, workers int) []neighbor {
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}

	targetVector := userVecs[user]
	targetNorm := norms[user]

	type job struct {
		uid int
		vec map[int]float64
	}
	jobs := make(chan job)
	results := make(chan neighbor, 1024)

	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		for j := range jobs {
			score := cosineSim(targetVector, j.vec, targetNorm, norms[j.uid])
			if score > 0 {
				results <- neighbor{UserID: j.uid, Score: score}
			}
		}
	}

	// Start workers
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go worker()
	}

	// Feed jobs
	go func() {
		for uid, vec := range userVecs {
			if uid == user {
				continue
			}
			jobs <- job{uid: uid, vec: vec}
		}
		close(jobs)
	}()

	// Close results when workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect, sort, trim
	out := make([]neighbor, 0, len(userVecs)-1)
	for nb := range results {
		out = append(out, nb)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > K {
		out = out[:K]
	}
	return out
}

// --- Concurrent recommendation aggregation ---

func RecommendFromTopKConcurrent(
	target int,
	neighbors []neighbor,
	userVec map[int]map[int]float64,
	topN int,
	minAbsSim float64,
	minNeighborsPerItem int,
	workers int,
) []recommended {
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}

	userMean := userMeans(userVec)

	// Movies already rated by target
	seen := make(map[int]bool, len(userVec[target]))
	for m := range userVec[target] {
		seen[m] = true
	}

	type agg struct {
		num, den float64
		cnt      int
	}

	// Partition neighbors into roughly equal chunks
	chunks := splitNeighbors(neighbors, workers)

	type partial = map[int]agg // item -> partial agg

	partials := make(chan partial, workers)
	var wg sync.WaitGroup

	// Each worker builds a local map to avoid lock contention
	worker := func(chunk []neighbor) {
		defer wg.Done()
		loc := make(partial, 256)

		for _, nb := range chunk {
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
				a := loc[mid]
				a.num += nb.Score * (r - mv)
				a.den += math.Abs(nb.Score)
				a.cnt++
				loc[mid] = a
			}
		}
		partials <- loc
	}

	wg.Add(len(chunks))
	for _, ch := range chunks {
		go worker(ch)
	}

	go func() {
		wg.Wait()
		close(partials)
	}()

	// Merge partials serially (no locks needed here)
	global := make(map[int]agg, 4096)
	for p := range partials {
		for mid, a := range p {
			g := global[mid]
			g.num += a.num
			g.den += a.den
			g.cnt += a.cnt
			global[mid] = g
		}
	}

	// Build final list
	recs := make([]recommended, 0, len(global))
	mu := userMean[target]
	for mid, a := range global {
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

func splitNeighbors(ns []neighbor, parts int) [][]neighbor {
	if parts <= 1 || len(ns) == 0 {
		return [][]neighbor{ns}
	}
	if parts > len(ns) {
		parts = len(ns)
	}
	chunks := make([][]neighbor, parts)
	step := (len(ns) + parts - 1) / parts
	for i := 0; i < parts; i++ {
		start := i * step
		if start >= len(ns) {
			chunks[i] = nil
			continue
		}
		end := start + step
		if end > len(ns) {
			end = len(ns)
		}
		chunks[i] = ns[start:end]
	}
	return chunks
}
