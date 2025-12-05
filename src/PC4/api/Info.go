package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"pc4/auth"
	"pc4/database"
	"pc4/tools"
	"strings"
)

func ReturnUserID(w http.ResponseWriter, r *http.Request) {
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

	user, err := database.FindUserIDByEmail(r.Context(), claims.Email)
	if err != nil {
		http.Error(w, "user wasn't found", http.StatusUnauthorized)
		return
	}

	resp := tools.ResponseUserID{
		UserID:   int(user.UserID),
		Email:    user.Email,
		Name:     user.Name,
		LastName: user.LastName,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Println("failed encoding response:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func ReturnUserMovies(w http.ResponseWriter, r *http.Request) {
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

	user, err := database.FindUserIDByEmail(r.Context(), claims.Email)
	if err != nil {
		http.Error(w, "user wasn't found", http.StatusUnauthorized)
		return
	}

	movies, err := database.FindMoviesWithRatingsByUserID(r.Context(), user.UserID)
	if err != nil {
		http.Error(w, "user wasn't found", http.StatusUnauthorized)
		return
	}

	resp := tools.ResponseUserMovies{
		UserID:       user.UserID,
		MovieRatings: movies,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Println("failed encoding response:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
