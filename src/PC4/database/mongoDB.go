package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"pc4/tools"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var mongoClient *mongo.Client
var mongoDB *mongo.Database

func InitMongo() error {
	uri := os.Getenv("MONGODB_URI")
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}

	mongoClient = client
	mongoDB = client.Database("dataset")
	return nil
}

func GetDB() *mongo.Database {
	return mongoDB
}

func CloseMongo(ctx context.Context) error {
	if mongoClient != nil {
		return mongoClient.Disconnect(ctx)
	}
	return nil
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

	userSchema := bson.M{
		"$jsonSchema": bson.M{
			"bsonType":             "object",
			"required":             bson.A{"_id"},
			"additionalProperties": true,
			"properties": bson.M{
				"_id": bson.M{"bsonType": "int"},

				"email":    bson.M{"bsonType": "string", "pattern": "^.+@.+\\..+$"},
				"password": bson.M{"bsonType": "string"},
				"name":     bson.M{"bsonType": "string"},
				"lastName": bson.M{"bsonType": "string"},

				"ratingIds": bson.M{"bsonType": "array", "items": bson.M{"bsonType": "objectId"}},
				"tagIds":    bson.M{"bsonType": "array", "items": bson.M{"bsonType": "objectId"}},
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
		ratingsCollection := db.Collection("ratings")
		idxModel := mongo.IndexModel{
			Keys:    bson.D{{Key: "userId", Value: 1}, {Key: "movieId", Value: 1}},
			Options: options.Index().SetUnique(true),
		}
		if _, err := ratingsCollection.Indexes().CreateOne(ctx, idxModel); err != nil {
			return err
		}
	}

	if err := ensureCollection(ctx, db, "tags", tagSchema); err != nil {
		return err
	}

	if err := ensureCollection(ctx, db, "users", userSchema); err != nil {
		return err
	}

	{
		userCollection := db.Collection("users")
		idxModel := mongo.IndexModel{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		}
		if _, err := userCollection.Indexes().CreateOne(ctx, idxModel); err != nil {
			return err
		}
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

func AppendDataToDB() error {
	db := GetDB()

	fmt.Println("Filling DB if not filled")

	moviePath, ok := os.LookupEnv("MOVIE_CSV_PATH")
	if !ok {
		return fmt.Errorf("MOVIE_CSV_PATH not set")
	}

	linksPath, ok := os.LookupEnv("LINKS_CSV_PATH")
	if !ok {
		return fmt.Errorf("LINKS_CSV_PATH not set")
	}

	ratingsPath, ok := os.LookupEnv("RATING_CSV_PATH")
	if !ok {
		return fmt.Errorf("RATING_CSV_PATH not set")
	}

	tagsPath, ok := os.LookupEnv("TAGS_CSV_PATH")
	if !ok {
		return fmt.Errorf("TAGS_CSV_PATH not set")
	}

	var (
		movieData   []tools.Movie
		linksData   []tools.LinksParsed
		ratingsData []tools.Ratings
		tagsData    []tools.Tags

		errMovies, errLinks, errRatings, errTags error
	)

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		movieData, errMovies = parseCSV(moviePath, decodeMovie, 4)
	}()

	go func() {
		defer wg.Done()
		linksData, errLinks = parseCSV(linksPath, decodeLinks, 4)
	}()

	go func() {
		defer wg.Done()
		ratingsData, errRatings = parseCSV(ratingsPath, decodeRating, 4)
	}()

	go func() {
		defer wg.Done()
		tagsData, errTags = parseCSV(tagsPath, decodeTags, 4)
	}()

	wg.Wait()

	if errMovies != nil {
		return fmt.Errorf("parse movies: %w", errMovies)
	}
	if errLinks != nil {
		return fmt.Errorf("parse links: %w", errLinks)
	}
	if errRatings != nil {
		return fmt.Errorf("parse ratings: %w", errRatings)
	}
	if errTags != nil {
		return fmt.Errorf("parse tags: %w", errTags)
	}

	fmt.Println("Total Movie", len(movieData))
	fmt.Println("Total Links", len(linksData))
	fmt.Println("Total Ratings", len(ratingsData))
	fmt.Println("Total Tags", len(tagsData))

	idxList := IndexLinks(linksData)
	fillMovieWithLinks(movieData, idxList)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	var (
		errMoviesInsert  error
		errRatingsInsert error
		errTagsInsert    error
	)

	if err := createCollectionsIfExist(ctx, db); err != nil {
		log.Fatal(err)
	}

	wg.Add(3)

	go func() {
		defer wg.Done()
		errMoviesInsert = insertManyBatched(ctx, db.Collection("movies"), AnySlice(movieData), 1000)
	}()
	go func() {
		defer wg.Done()
		errRatingsInsert = insertManyBatched(ctx, db.Collection("ratings"), AnySlice(ratingsData), 1000)
	}()
	go func() {
		defer wg.Done()
		errTagsInsert = insertManyBatched(ctx, db.Collection("tags"), AnySlice(tagsData), 1000)
	}()

	wg.Wait()

	if errMoviesInsert != nil {
		return fmt.Errorf("movies insert: %w", errMoviesInsert)
	}
	if errRatingsInsert != nil {
		return fmt.Errorf("ratings insert: %w", errRatingsInsert)
	}
	if errTagsInsert != nil {
		return fmt.Errorf("tags insert: %w", errTagsInsert)
	}

	if err := BuildUsersFromRatingsAndTags(ctx, db); err != nil {
		log.Fatal(err)
	}

	log.Printf("DB filled")

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

func fetchRating(ctx context.Context, db *mongo.Database) ([]tools.Rating, error) {
	coll := db.Collection("ratings")

	opts := options.Find().
		SetProjection(bson.D{
			{Key: "_id", Value: 0},
			{Key: "userId", Value: 1},
			{Key: "movieId", Value: 1},
			{Key: "rating", Value: 1},
		}).
		SetBatchSize(5000)

	cur, err := coll.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	out := make([]tools.Rating, 0, 1_000_000)

	for cur.Next(ctx) {
		var r tools.Rating
		if err := cur.Decode(&r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, cur.Err()
}

// fetchMoviesMap returns a map from movieID -> Movie for the provided IDs.
// Performs a single query using $in to fetch all matching movie documents.
func fetchMoviesMap(ctx context.Context, db *mongo.Database, ids []int) (map[int]tools.Movie, error) {
	if len(ids) == 0 {
		return map[int]tools.Movie{}, nil
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

	out := make(map[int]tools.Movie)
	for cur.Next(ctx) {
		var m tools.Movie
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

func UserExists(userID int) (bool, error) {
	db := GetDB()

	n, err := db.Collection("users").CountDocuments(context.Background(), bson.D{{Key: "userId", Value: userID}})
	if err != nil {
		return false, err
	}

	return n > 0, nil
}

func IndexLinks(ls []tools.LinksParsed) map[int]tools.LinksParsed {
	idx := make(map[int]tools.LinksParsed, len(ls))
	for _, l := range ls {
		idx[l.MovieId] = l
	}
	return idx
}

func AnySlice[T any](in []T) []interface{} {
	out := make([]interface{}, len(in))
	for i := range in {
		out[i] = in[i]
	}
	return out
}

func BuildUsersFromRatingsAndTags(ctx context.Context, db *mongo.Database) error {
	usersColl := db.Collection("users")

	// 1) group ratings by userId and collect their _ids
	ratingCur, err := db.Collection("ratings").Aggregate(ctx, mongo.Pipeline{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$userId"},
			{Key: "ratingIds", Value: bson.D{{Key: "$push", Value: "$_id"}}},
		}}},
	})
	if err != nil {
		return err
	}
	defer ratingCur.Close(ctx)

	ratingMap := make(map[int32][]bson.ObjectID)
	for ratingCur.Next(ctx) {
		var row struct {
			UserID    int32           `bson:"_id"`
			RatingIDs []bson.ObjectID `bson:"ratingIds"`
		}
		if err := ratingCur.Decode(&row); err != nil {
			return err
		}
		ratingMap[row.UserID] = row.RatingIDs
	}
	if err := ratingCur.Err(); err != nil {
		return err
	}

	// 2) group tags by userId and collect their _ids
	tagCur, err := db.Collection("tags").Aggregate(ctx, mongo.Pipeline{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$userId"},
			{Key: "tagIds", Value: bson.D{{Key: "$push", Value: "$_id"}}},
		}}},
	})
	if err != nil {
		return err
	}
	defer tagCur.Close(ctx)

	tagMap := make(map[int32][]bson.ObjectID)
	for tagCur.Next(ctx) {
		var row struct {
			UserID int32           `bson:"_id"`
			TagIDs []bson.ObjectID `bson:"tagIds"`
		}
		if err := tagCur.Decode(&row); err != nil {
			return err
		}
		tagMap[row.UserID] = row.TagIDs
	}
	if err := tagCur.Err(); err != nil {
		return err
	}

	// 3) union of user IDs from both maps (fixes "tags-only" users being skipped)
	userIDs := make(map[int32]struct{}, len(ratingMap)+len(tagMap))
	for id := range ratingMap {
		userIDs[id] = struct{}{}
	}
	for id := range tagMap {
		userIDs[id] = struct{}{}
	}

	// 4) bulk upsert users
	ops := make([]mongo.WriteModel, 0, len(userIDs))
	for userId := range userIDs {
		rids := ratingMap[userId]
		tids := tagMap[userId]

		// If you prefer empty arrays instead of null:
		if rids == nil {
			rids = []bson.ObjectID{}
		}
		if tids == nil {
			tids = []bson.ObjectID{}
		}

		update := bson.D{{
			Key: "$set",
			Value: bson.D{
				{Key: "ratingIds", Value: rids},
				{Key: "tagIds", Value: tids},
			},
		}}

		ops = append(ops, mongo.NewUpdateOneModel().
			SetFilter(bson.D{{Key: "_id", Value: userId}}). // assumes users._id is int32
			SetUpdate(update).
			SetUpsert(true),
		)
	}

	if len(ops) > 0 {
		_, err = usersColl.BulkWrite(ctx, ops, options.BulkWrite().SetOrdered(false))
		if err != nil {
			return err
		}
	}

	return nil
}

func fillMovieWithLinks(ms []tools.Movie, idx map[int]tools.LinksParsed) {
	for i := range ms {
		if l, ok := idx[int(ms[i].ID)]; ok {
			ms[i].IMDB = l.Imdb
			ms[i].TMDB = l.Tmdb
		}
	}
}

func CreateUserWeb(ctx context.Context, jsonAPI tools.RegisterRequest) error {
	db := GetDB()
	usersColl := db.Collection("users")

	update := bson.D{{
		Key: "$set",
		Value: bson.D{
			{Key: "email", Value: jsonAPI.Email},
			{Key: "password", Value: jsonAPI.Password},
			{Key: "name", Value: jsonAPI.Name},
			{Key: "lastName", Value: jsonAPI.LastName},
		},
	}}

	_, err := usersColl.UpdateOne(ctx, bson.D{{Key: "_id", Value: jsonAPI.UserID}}, update)
	if err != nil {
		return fmt.Errorf("update user with info: %w", err)
	}
	return err
}

func FindUserByEmail(ctx context.Context, email string) (*tools.UserInfo, error) {
	db := GetDB()
	usersColl := db.Collection("users")

	filter := bson.D{{Key: "email", Value: email}}
	opts := options.FindOne().SetProjection(bson.D{
		{Key: "_id", Value: 0},
		{Key: "email", Value: 1},
		{Key: "password", Value: 1},
		{Key: "name", Value: 0},
		{Key: "lastName", Value: 0},
		{Key: "ratingIds", Value: 0},
		{Key: "tagIds", Value: 0},
	})

	var u tools.UserInfo
	err := usersColl.FindOne(ctx, filter, opts).Decode(&u)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return &u, nil

}
