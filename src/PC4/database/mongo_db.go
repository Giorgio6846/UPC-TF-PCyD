package database

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CollectionType int

const (
	Ratings CollectionType = iota
	Movies
	Recommendations
)

var collectionName = map[CollectionType]string{
	Ratings:         "ratings",
	Movies:          "movies",
	Recommendations: "recommendations",
}

func (ct CollectionType) String() string {
	return collectionName[ct]
}

func ConnectMongo(ct CollectionType) *mongo.Collection {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uri, ok := os.LookupEnv("MONGODB_URI")
	if !ok {
		log.Fatal("MONGODB_URI not set")
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}

	return client.Database("movielens").Collection(ct.String())
}
