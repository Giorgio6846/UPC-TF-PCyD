package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

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
	if err := createCollectionsIfExist(context.Background(), db); err != nil {
		log.Fatal(err)
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
}

func api1(w http.ResponseWriter, r *http.Request) {
	fmt.Println("API 1 was requested")
}

func api2(w http.ResponseWriter, r *http.Request) {
	fmt.Println("API 2 was requested")
}
