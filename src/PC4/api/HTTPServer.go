package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	ds "pc4/DistributedSystem"
	recommendersystem "pc4/RecommenderSystem"
	"pc4/auth"
	"pc4/database"
	"pc4/tools"
	"strconv"
	"strings"
	"time"
)

type ctxKey string

const EmailIDKey ctxKey = "email"

func SetupAPI() {
	AP, ok := os.LookupEnv("API_PORT")
	if !ok {
		log.Fatal("API_PORT not set")
	}

	mux := http.NewServeMux()

	//Auth API
	mux.HandleFunc("/auth/register", registerUser)
	mux.HandleFunc("/auth/login", loginUser)

	//Recommender API
	mux.Handle("/api/similarMovies", RequireJWT(http.HandlerFunc(similarMoviesSearch)))

	log.Println("HTTP listening at :" + AP)

	if err := http.ListenAndServe(":"+AP, mux); err != nil {
		fmt.Println("Couldn't setup the server", err)
	}
}

func similarMoviesSearch(w http.ResponseWriter, r *http.Request) {
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

func loginUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req tools.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	user, err := database.FindUserByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "user wasn't found", http.StatusUnauthorized)
		return
	}

	if err := auth.CheckPassword(user.Password, req.Password); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.CreateJWT(user.Email)
	if err != nil {
		http.Error(w, "could not create token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})

}

func registerUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req tools.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	fmt.Printf("Registering userId=%d email=%s\n", req.UserID, req.Email)

	hashedPw, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "invalid password", http.StatusBadRequest)
		return
	}

	req.Password = hashedPw

	exists, err := database.UserExists(req.UserID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "user not created", http.StatusNotFound)
		return
	}

	if err = database.CreateUserWeb(r.Context(), req); err != nil {
		http.Error(w, "user couldn't be created", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "user %s registered", req.Email)
}

func RequireJWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		tokenString = strings.TrimSpace(tokenString)
		if tokenString == "" {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}

		claims, err := auth.ParseJWT(tokenString)
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), EmailIDKey, claims.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
