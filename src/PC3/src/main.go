package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	uri, ok := os.LookupEnv("MONGODB_URI")
	if !ok {
		log.Fatal("MONGODB_URI not set")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	db := client.Database("dataset")
	//if err := appendDataToDB(db); err != nil {
	//	log.Fatalf("appendDataToDB failed: %v", err)
	//}

	setupAPI(db)
	if err = http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Couldn't setup the server", err)
	}
	log.Println("HTTP listening at :8080")
}

func setupAPI(db *mongo.Database) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	ratings, err := fetchRating(ctx, db)
	if err != nil {
		return
	}

	http.HandleFunc("/similarMoviesSeq", func(w http.ResponseWriter, r *http.Request) {
		movieSearcherSeq(w, r, db, ratings)
	})
	http.HandleFunc("/similarMoviesCon", func(w http.ResponseWriter, r *http.Request) {
		movieSearcherCon(w, r, db, ratings)
	})
}
