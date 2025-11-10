package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Rating struct {
	UserID  int32   `bson:"userId"`
	MovieID int32   `bson:"movieId"`
	Rating  float64 `bson:"rating"`
}

func createCollectionsIfExist(ctx context.Context, db *mongo.Database) error {

	movieSchema := bson.M{
		"$jsonSchema": bson.M{
			"bsonType":             "object",
			"required":             bson.A{"_id", "title", "genres"},
			"additionalProperties": false,
			"properties": bson.M{
				"_id":   bson.M{"bsonType": "int"},
				"imdb":  bson.M{"bsonType": "int"},
				"tmdb":  bson.M{"bsonType": "int"},
				"title": bson.M{"bsonType": "string"},
				"genres": bson.M{"bsonType": "array",
					"items": bson.M{"bsonType": "string"},
				},
			},
		},
	}

	ratingSchema := bson.M{
		"$jsonSchema": bson.M{
			"bsonType":             "object",
			"required":             bson.A{"userId", "movieId", "rating"},
			"additionalProperties": false,
			"properties": bson.M{
				"_id":     bson.M{"bsonType": "objectId"},
				"userId":  bson.M{"bsonType": "int"},
				"movieId": bson.M{"bsonType": "int"},
				"rating":  bson.M{"bsonType": "double", "minimum": 0.5, "maximum": 5.0},
			},
		},
	}

	tagSchema := bson.M{
		"$jsonSchema": bson.M{
			"bsonType":             "object",
			"required":             bson.A{"userId", "movieId", "tag"},
			"additionalProperties": false,
			"properties": bson.M{
				"_id":     bson.M{"bsonType": "objectId"},
				"userId":  bson.M{"bsonType": "int"},
				"movieId": bson.M{"bsonType": "int"},
				"tag":     bson.M{"bsonType": "string"},
			},
		},
	}

	if err := ensureCollection(ctx, db, "movies", movieSchema); err != nil {
		return err
	}

	if err := ensureCollection(ctx, db, "ratings", ratingSchema); err != nil {
		return err
	}

	// Ensure a unique index on (userId, movieId) to prevent duplicate
	// rating documents for the same user/movie pair.
	{
		ratingsColl := db.Collection("ratings")
		idxModel := mongo.IndexModel{
			Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "movieId", Value: 1}},
			Options: options.Index().SetUnique(true),
		}
		if _, err := ratingsColl.Indexes().CreateOne(ctx, idxModel); err != nil {
			return err
		}
	}

	if err := ensureCollection(ctx, db, "tags", tagSchema); err != nil {
		return err
	}

	return nil
}

func ensureCollection(ctx context.Context, db *mongo.Database, name string, validator bson.M) error {
	createOpts := options.CreateCollection().
		SetValidator(validator)

	if err := db.CreateCollection(ctx, name, createOpts); err != nil {
		var cmdErr mongo.CommandError
		if errors.As(err, &cmdErr) && cmdErr.Code == 48 { // NamespaceExists
			cmd := bson.D{
				{Key: "collMod", Value: name},
				{Key: "validator", Value: validator},
				{Key: "validationLevel", Value: "strict"},
				{Key: "validationAction", Value: "error"},
			}
			return db.RunCommand(ctx, cmd).Err()
		}
		return err
	}
	return nil
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

	if err := insertManyBatched(ctx, db.Collection("tags"), anySlice(tagsData), 1000); err != nil {
		return fmt.Errorf("tags insert: %w", err)
	}

	fmt.Println("DB is filled uwu")

	return nil
}

func insertManyBatched(ctx context.Context, coll *mongo.Collection, docs []interface{}, batchSize int) error {
	if len(docs) == 0 {
		return nil
	}
	opts := options.InsertMany().SetOrdered(false) // continue on dup key
	total := 0
	for start := 0; start < len(docs); start += batchSize {
		end := start + batchSize
		if end > len(docs) {
			end = len(docs)
		}
		batch := docs[start:end]
		_, err := coll.InsertMany(ctx, batch, opts)
		if err != nil {
			// Allow duplicate key bulk errors (common on reruns)
			var bwe mongo.BulkWriteException
			if errors.As(err, &bwe) {
				// If *all* errors are duplicate key, ignore; otherwise return
				nonDup := false
				for _, we := range bwe.WriteErrors {
					if we.Code != 11000 { // duplicate key
						nonDup = true
						break
					}
				}
				if nonDup {
					return fmt.Errorf("bulk write error: %w", err)
				}
				// else: ignore dupes and continue
			} else {
				return err
			}
		}
		total += (end - start)
	}
	fmt.Printf("Inserted (attempted) %d into %s\n", total, coll.Name())
	return nil
}

func fetchRating(ctx context.Context, db *mongo.Database) ([]Rating, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "userId", Value: 1},
			{Key: "movieId", Value: 1},
			{Key: "rating", Value: 1},
		}}},
	}

	cur, err := db.Collection("ratings").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []Rating
	for cur.Next(ctx) {
		var r Rating
		if err := cur.Decode(&r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, cur.Err()
}

// fetchMoviesMap returns a map from movieID -> Movie for the provided IDs.
// Performs a single query using $in to fetch all matching movie documents.
func fetchMoviesMap(ctx context.Context, db *mongo.Database, ids []int) (map[int]Movie, error) {
	if len(ids) == 0 {
		return map[int]Movie{}, nil
	}

	// Build interface slice of ids for BSON $in
	var idInterfaces []interface{}
	for _, id := range ids {
		idInterfaces = append(idInterfaces, id)
	}

	filter := bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: idInterfaces}}}}
	proj := bson.D{
		{Key: "_id", Value: 1},
		{Key: "title", Value: 1},
		{Key: "genres", Value: 1},
		{Key: "imdb", Value: 1},
		{Key: "tmdb", Value: 1},
	}

	cur, err := db.Collection("movies").Find(ctx, filter, options.Find().SetProjection(proj))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	out := make(map[int]Movie)
	for cur.Next(ctx) {
		var m Movie
		if err := cur.Decode(&m); err != nil {
			return nil, err
		}
		out[int(m.ID)] = m
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func userExists(ctx context.Context, db *mongo.Database, userID int) (bool, error) {
	n, err := db.Collection("ratings").CountDocuments(ctx, bson.D{{Key: "userId", Value: userID}})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
