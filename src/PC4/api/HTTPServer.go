package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"pc4/auth"
	"pc4/database"
	"pc4/tools"
	"strconv"
	"strings"
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
	mux.HandleFunc("/register", registerUser)
	mux.HandleFunc("/login", loginUser)

	//Recommender API
	mux.Handle("/similarMovies", RequireJWT(http.HandlerFunc(similarMoviesSearch)))

	log.Println("HTTP listening at :" + AP)

	if err := http.ListenAndServe(":"+AP, nil); err != nil {
		fmt.Println("Couldn't setup the server", err)
	}
}

func similarMoviesSearch(w http.ResponseWriter, r *http.Request) {
	userId := r.URL.Query().Get("userId")
	fmt.Println("userId:", userId)

	userID, err := strconv.Atoi(userId)
	if err != nil {
		http.Error(w, "invalid userID", http.StatusBadRequest)
		return
	}

	exists, _ := database.UserExists(userID)
	if !exists {
		http.Error(w, "user not found", http.StatusNotFound)
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
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := auth.CheckPassword(user.Pasword, req.Password); err != nil {
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

	w.WriteHeader(http.StatusCreated)
	fmt.Println(w, "user %s registered", req.Email)

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
		http.Error(w, "user not found", http.StatusNotFound)
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
