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

type jsonResult struct {
	Rank   int     `json:"rank"`
	UserID int     `json:"userId"`
	Score  float64 `json:"score"`
}

func movieSearcher(w http.ResponseWriter, r *http.Request, db *mongo.Database) {
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

	exists, err := userExists(ctx, db, userID)
	if !exists {
		http.Error(w, "user not found", http.StatusNotFound)
	}

	ratings, err := fetchRating(ctx, db)
	userVecs, norms := buildUserVectors(ratings)

	top10K := topKUsers(userID, userVecs, norms, 10)

	results := make([]jsonResult, len(top10K))
	for i, n := range top10K {
		results[i] = jsonResult{
			Rank:   i + 1,
			UserID: n.UserID,
			Score:  n.Score,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
