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
	if err := appendDataToDB(db); err != nil {
		log.Fatalf("appendDataToDB failed: %v", err)
	}

	setupAPI(db)
	if err = http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Couldn't setup the server", err)
	}
	log.Println("HTTP listening at :8080")
}

func setupAPI(db *mongo.Database) {
	http.HandleFunc("/api1", api1)
	http.HandleFunc("/api2", api2)
	//http.HandleFunc("/fill", appendDataToDB)
}

func api1(w http.ResponseWriter, r *http.Request) {
	fmt.Println("API 1 was requested")
}

func api2(w http.ResponseWriter, r *http.Request) {
	fmt.Println("API 2 was requested")
}

func appendDataToDB(db *mongo.Database) error {
	fmt.Println("Filling DB uwu")

	path, ok := os.LookupEnv("MOVIE_CSV_PATH")
	if !ok {
		log.Fatal("MOVIE_CSV_PATH not set")
	}
	movieData, err := parseCSV(path, decodeMovie)
	fmt.Println("Total Movie", len(movieData), err)

	path, ok = os.LookupEnv("LINKS_CSV_PATH")
	if !ok {
		log.Fatal("LINKS_CSV_PATH not set")
	}
	linksData, err := parseCSV(path, decodeLinks)
	fmt.Println("Total Links", len(linksData), err)

	path, ok = os.LookupEnv("RATING_CSV_PATH")
	if !ok {
		log.Fatal("RATING_CSV_PATH not set")
	}
	ratingsData, err := parseCSV(path, decodeRating)
	fmt.Println("Total Ratings", len(ratingsData), err)

	path, ok = os.LookupEnv("TAGS_CSV_PATH")
	if !ok {
		log.Fatal("TAGS_CSV_PATH not set")
	}
	tagsData, err := parseCSV(path, decodeTags)
	fmt.Println("Total Tags", len(tagsData), err)

	idxList := indexLinks(linksData)
	fillMovieWithLinks(movieData, idxList)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if err := createCollectionsIfExist(ctx, db); err != nil {
		log.Fatal(err)
	}

	if err := insertManyBatched(ctx, db.Collection("movies"), anySlice(movieData), 1000); err != nil {
		return fmt.Errorf("movies insert: %w", err)
	}

	if err := insertManyBatched(ctx, db.Collection("ratings"), anySlice(ratingsData), 1000); err != nil {
		return fmt.Errorf("ratings insert: %w", err)
	}

	if err := insertManyBatched(ctx, db.Collection("tag"), anySlice(tagsData), 1000); err != nil {
		return fmt.Errorf("tags insert: %w", err)
	}

	fmt.Println("DB is filled uwu")

	return nil
}
