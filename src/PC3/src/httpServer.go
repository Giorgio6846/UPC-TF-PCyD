package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type jsonMovieResult struct {
	Rank    int      `json:"rank"`
	MovieID int      `json:"movieId"`
	Title   string   `json:"title"`
	Genres  []string `json:"genres"`
	IMDB    int32    `json:"imdb,omitempty"`
	TMDB    int32    `json:"tmdb,omitempty"`
	Score   float64  `json:"score"`
}

type responseJSON struct {
	Results     []jsonMovieResult `json:"results"`
	TimeMongoMs int64             `json:"timeMongoMs"`
	TimeAlgoMs  int64             `json:"timeAlgoMs"`
}

func movieSearcherSeq(w http.ResponseWriter, r *http.Request, db *mongo.Database) {
	fmt.Println("/similarMovies was requested")

	userId := r.URL.Query().Get("userId")
	fmt.Println("userId:", userId)

	userID, err := strconv.Atoi(userId)
	if err != nil {
		http.Error(w, "invalid userID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var mongoDuration time.Duration
	var algoDuration time.Duration

	// check user existence (mongo)
	t0 := time.Now()
	exists, err := userExists(ctx, db, userID)
	mongoDuration += time.Since(t0)
	if !exists {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	// fetch ratings (mongo)
	t0 = time.Now()
	ratings, err := fetchRating(ctx, db)
	mongoDuration += time.Since(t0)
	if err != nil {
		http.Error(w, "failed to fetch ratings", http.StatusInternalServerError)
		return
	}

	// build user vectors (algo)
	t0 = time.Now()
	userVecs, norms := buildUserVectors(ratings)
	algoDuration += time.Since(t0)

	// find top neighbors (algo)
	t0 = time.Now()
	top10Neighbor := topKUsers(userID, userVecs, norms, 10)
	algoDuration += time.Since(t0)

	// generate recommendations (algo)
	t0 = time.Now()
	recommendedMovies := RecommendFromTopK(userID, top10Neighbor, userVecs, 20, 0.05, 2)
	algoDuration += time.Since(t0)

	// Batch fetch movie metadata for all recommended movie IDs (mongo)
	ids := make([]int, 0, len(recommendedMovies))
	for _, r := range recommendedMovies {
		ids = append(ids, r.MovieID)
	}

	t0 = time.Now()
	moviesMap, err := fetchMoviesMap(ctx, db, ids)
	mongoDuration += time.Since(t0)
	if err != nil {
		http.Error(w, "failed to fetch movie metadata", http.StatusInternalServerError)
		return
	}

	results := make([]jsonMovieResult, len(recommendedMovies))
	for i, n := range recommendedMovies {
		m, ok := moviesMap[n.MovieID]
		var title string
		var genres []string
		var imdb int32
		var tmdb int32
		if ok {
			title = m.Title
			genres = m.Genres
			imdb = m.IMDB
			tmdb = m.TMDB
		}
		results[i] = jsonMovieResult{
			Rank:    i + 1,
			MovieID: n.MovieID,
			Title:   title,
			Genres:  genres,
			IMDB:    imdb,
			TMDB:    tmdb,
			Score:   n.Score,
		}
	}

	resp := responseJSON{
		Results:     results,
		TimeMongoMs: mongoDuration.Milliseconds(),
		TimeAlgoMs:  algoDuration.Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Println("failed encoding response:", err)
	}
}
