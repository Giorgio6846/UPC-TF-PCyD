package main

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func createCollectionsIfExist(ctx context.Context, db *mongo.Database) error {

	movieSchema := bson.M{
		"$jsonSchema": bson.M{
			"bsonType":             "object",
			"required":             bson.A{"_id", "title", "genres"},
			"additionalProperties": false,
			"properties": bson.M{
				"_id":    bson.M{"bsonType": "int"},
				"imdbId": bson.M{"bsonType": "int"},
				"tmdbId": bson.M{"bsonType": "int"},
				"title":  bson.M{"bsonType": "string"},
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

	if err := ensureCollection(ctx, db, "tag", tagSchema); err != nil {
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
			// Update validator on existing collection
			cmd := bson.D{
				{Key: "collMod", Value: name},
				{Key: "validator", Value: validator},
				{Key: "validationLevel", Value: "strict"},
				{Key: "validationAction", Value: "error"},
			}
			return db.RunCommand(ctx, cmd).Err()
		}
		// Some other error occurred creating the collection
		return err
	}
	return nil
}
