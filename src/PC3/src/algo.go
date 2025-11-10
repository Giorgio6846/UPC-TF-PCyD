package main

import "math"

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

//First solution via user -> movie/rating
