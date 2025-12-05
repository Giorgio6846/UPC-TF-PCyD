package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	ds "pc4/DistributedSystem"
	recommendersystem "pc4/RecommenderSystem"
	"pc4/database"
	"pc4/tools"
	"strconv"
	"time"
)

func SimilarMoviesSearch(w http.ResponseWriter, r *http.Request) {
	userId := r.URL.Query().Get("userId")

	userID, err := strconv.Atoi(userId)
	if err != nil {
		http.Error(w, "invalid userID", http.StatusBadRequest)
		return
	}

	fmt.Println("userId:", userID)

	exists, err := database.UserExists(userID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	t0 := time.Now()

	db := database.GetDB()
	rdb := database.GetRedisDB()

	var userVec map[int]tools.UserVector
	key := fmt.Sprintf("user:%d", userID)
	count, _ := rdb.Exists(context.Background(), key).Result()

	if count > 0 {
		fmt.Println("cargando vectores desde Redis...")
		userVec, err = database.LoadFromRedis(context.Background(), rdb)
		if err != nil {
			log.Fatal("error from redis", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		fmt.Println("cargando vectores desde Mongo...")

		ratings, err := database.FetchRating(context.Background(), db)
		if err != nil {
			log.Fatal("couldn't connect to mongo", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		userVec = recommendersystem.BuildUserVectors(ratings)

		fmt.Println("guardando vectores de usuarios en Redis")
		database.SaveToRedis(rdb, userVec)
	}

	durationDB := time.Since(t0)
	t0 = time.Now()

	neighbors, err := ds.ComputeSimilarUsers(userID, 30, userVec)
	if err != nil {
		log.Fatal("couldn't connect to mongo", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	durationAlgo := time.Since(t0)
	t0 = time.Now()

	recommendedMovies := recommendersystem.RecommendFromTopK(userID, neighbors, userVec, 20, 0.05, 2)
	ids := make([]int, 0, len(recommendedMovies))
	for _, r := range recommendedMovies {
		fmt.Println(r.MovieID)
		ids = append(ids, r.MovieID)
	}

	moviesMap, err := database.FetchMoviesMap(context.Background(), db, ids)
	if err != nil {
		log.Fatal("couldn't connect to mongo", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	durationFetch := time.Since(t0)

	results := make([]tools.JsonMovieResult, len(recommendedMovies))
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
		results[i] = tools.JsonMovieResult{
			Rank:    i + 1,
			MovieID: n.MovieID,
			Title:   title,
			Genres:  genres,
			IMDB:    imdb,
			TMDB:    tmdb,
			Score:   n.Score,
		}
	}

	resp := tools.ResponseMovieJSON{
		Results:            results,
		DurationDB:         float32(durationDB.Milliseconds()),
		DurationAlgo:       float32(durationAlgo.Milliseconds()),
		DurationMovieFetch: float32(durationFetch.Milliseconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Println("failed encoding response:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func SimilarUsersSearch(w http.ResponseWriter, r *http.Request) {
	userId := r.URL.Query().Get("userId")

	userID, err := strconv.Atoi(userId)
	if err != nil {
		http.Error(w, "invalid userID", http.StatusBadRequest)
		return
	}

	fmt.Println("userId:", userID)

	exists, err := database.UserExists(userID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	t0 := time.Now()

	db := database.GetDB()
	rdb := database.GetRedisDB()

	var userVec map[int]tools.UserVector
	key := fmt.Sprintf("user:%d", userID)
	count, _ := rdb.Exists(context.Background(), key).Result()

	if count > 0 {
		fmt.Println("cargando vectores desde Redis...")
		userVec, err = database.LoadFromRedis(context.Background(), rdb)
		if err != nil {
			log.Fatal("error from redis", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		fmt.Println("cargando vectores desde Mongo...")

		ratings, err := database.FetchRating(context.Background(), db)
		if err != nil {
			log.Fatal("couldn't connect to mongo", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		userVec = recommendersystem.BuildUserVectors(ratings)

		fmt.Println("guardando vectores de usuarios en Redis")
		database.SaveToRedis(rdb, userVec)
	}

	durationDB := time.Since(t0)
	t0 = time.Now()

	neighbors, err := ds.ComputeSimilarUsers(userID, 30, userVec)
	if err != nil {
		log.Fatal("couldn't connect to mongo", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if len(neighbors) > 20 {
		neighbors = neighbors[:20]
	}

	durationAlgo := time.Since(t0)

	resp := tools.ResponseSimlarities{
		SimilarUsers: neighbors,
		DurationDB:   float32(durationDB.Milliseconds()),
		DurationAlgo: float32(durationAlgo.Milliseconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Println("failed encoding response:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
