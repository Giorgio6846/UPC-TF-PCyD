package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Movie struct{
	MovieID int `bson:"movieId"`
	Title string `bson:"title"`
	Genres string `bson:"genres"`
}

type Rating struct {
	UserID  int     `bson:"userId"`
	MovieID int     `bson:"movieId"`
	Rating  float64 `bson:"rating"`
}

const (
	chunkSize  = 100000
	numWorkers = 50
)


func connectMongo(typeConnection int) *mongo.Collection {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uri := "mongodb://hello:world@localhost:27017"
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}

	if typeConnection == 1 {
		collection := client.Database("movielens").Collection("movies")
		fmt.Println("✅ Conectado a MongoDB y lista la colección 'movies'")
		return collection
	}
	
	collection := client.Database("movielens").Collection("ratings")
	fmt.Println("✅ Conectado a MongoDB y lista la colección 'ratings'")
	return collection
}

func workerRating(id int, jobs <-chan [][]string, wg *sync.WaitGroup, collection *mongo.Collection) {
	defer wg.Done()
	ctx := context.Background()

	for records := range jobs {
		var ratings []interface{} // lo unico que acepta mongo creo 
		for _, record := range records {
			userId, _ := strconv.Atoi(record[0])
			movieId, _ := strconv.Atoi(record[1])
			rating, _ := strconv.ParseFloat(record[2], 64)
			ratings = append(ratings, Rating{UserID: userId, MovieID: movieId, Rating: rating})
		}

		if len(ratings) > 0 {
			collection.InsertMany(ctx, ratings)
			fmt.Printf("Worker %d inserto %d registros en Ratings\n", id, len(ratings))
		}
	}
}

func workerMovie(id int, jobs <-chan [][]string, wg *sync.WaitGroup, collection *mongo.Collection) {
	defer wg.Done()
	ctx := context.Background()

	for records := range jobs {
		var movies []interface{}
		for _, record := range records {
			movieId, _ := strconv.Atoi(record[0])
			title := record[1]
			genres := record[2]
			movies = append(movies, Movie{MovieID: movieId, Title: title, Genres: genres})
		}

		if len(movies) > 0 {
			collection.InsertMany(ctx, movies)
			fmt.Printf("Worker %d inserto %d registros en Movies\n", id, len(movies))
		}
	}
}

func main() {
	collectionMovies := connectMongo(1)
	collectionRatings := connectMongo(2)
	
	fileRating, _ := os.Open("./data/ratings_clean.csv")
	fileMovie, _ := os.Open("./data/movies_clean.csv")

	defer fileMovie.Close()
	defer fileRating.Close()

	readerRating := csv.NewReader(bufio.NewReader(fileRating))
	readerMovie := csv.NewReader(bufio.NewReader(fileMovie))

	jobsRating := make(chan [][]string, numWorkers)
	jobsMovie := make(chan [][]string, numWorkers)

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go workerRating(i, jobsRating, &wg, collectionRatings)
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go workerMovie(i, jobsMovie, &wg, collectionMovies)
	}

	var batchMovie [][]string
	for {
		record, err := readerMovie.Read()
		if err != nil {
			break
		}
		batchMovie = append(batchMovie, record)
		if len(batchMovie) == chunkSize {
			jobsMovie <- batchMovie
			batchMovie = nil
		}
	}

	if len(batchMovie) > 0 {
		jobsMovie <- batchMovie
	}

	close(jobsMovie)


	var batchRating [][]string
	for {
		record, err := readerRating.Read()
		if err != nil {
			break
		}
		batchRating = append(batchRating, record)
		if len(batchRating) == chunkSize {
			jobsRating <- batchRating
			batchRating = nil
		}
	}

	if len(batchRating) > 0 {
		jobsRating	 <- batchRating
	}

	close(jobsRating)

	wg.Wait()
	fmt.Println("Procesamiento completado")
}
