package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

type ctxKey string

const EmailIDKey ctxKey = "email"

func SetupAPI() {
	AP, ok := os.LookupEnv("API_PORT")
	if !ok {
		log.Fatal("API_PORT not set")
	}

	mux := http.NewServeMux()

	// Auth API
	mux.HandleFunc("/auth/register", RegisterUser)
	mux.HandleFunc("/auth/login", LoginUser)

	// Recommender API
	mux.Handle("/api/similarMovies", RequireJWT(http.HandlerFunc(SimilarMoviesSearch)))
	mux.Handle("/api/similarUsers", RequireJWT(http.HandlerFunc(SimilarUsersSearch)))

	// Info API
	mux.Handle("/me", RequireJWT(http.HandlerFunc(ReturnUserID)))
	mux.Handle("/me/movies", RequireJWT(http.HandlerFunc(ReturnUserMovies)))

	// Resources API
	mux.Handle("/resource", RequireJWT(http.HandlerFunc(ResourcesConsumption)))

	log.Println("HTTP listening at :" + AP)

	// envolvemos el mux con el middleware CORS
	handler := CorsMiddleware(mux)

	if err := http.ListenAndServe(":"+AP, handler); err != nil {
		fmt.Println("Couldn't setup the server", err)
	}
}
