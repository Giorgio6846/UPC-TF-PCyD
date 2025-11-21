package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func SetupAPI() {
	AP, ok := os.LookupEnv("API_PORT")
	if !ok {
		log.Fatal("API_PORT not set")
	}

	if err := http.ListenAndServe(":"+AP, nil); err != nil {
		fmt.Println("Couldn't setup the server", err)
	}
	log.Println("HTTP listening at :" + AP)

	http.HandleFunc("/similarMovies", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("TEST")
	})
}
