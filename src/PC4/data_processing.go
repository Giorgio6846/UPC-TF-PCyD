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

func connectMongo() *mongo.Collection {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uri := "mongodb://hello:world@localhost:27017"
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}

	collection := client.Database("movielens").Collection("ratings")
	fmt.Println("✅ Conectado a MongoDB y lista la colección 'ratings'")
	return collection
}

type Rating struct {
	UserID  int     `bson:"userId"`
	MovieID int     `bson:"movieId"`
	Rating  float64 `bson:"rating"`
}

const (
	chunkSize  = 100000
	numWorkers = 10
)

func worker(id int, jobs <-chan [][]string, wg *sync.WaitGroup, collection *mongo.Collection) {
	defer wg.Done()
	ctx := context.Background()

	for records := range jobs {
		var ratings []interface{}
		for _, record := range records {
			userId, _ := strconv.Atoi(record[0])
			movieId, _ := strconv.Atoi(record[1])
			rating, _ := strconv.ParseFloat(record[2], 64)
			ratings = append(ratings, Rating{UserID: userId, MovieID: movieId, Rating: rating})
		}

		if len(ratings) > 0 {
			_, err := collection.InsertMany(ctx, ratings)
			if err != nil {
				log.Printf("Worker %d error al insertar: %v\n", id, err)
			} else {
				fmt.Printf("Worker %d inserto %d registros\n", id, len(ratings))
			}
		}
	}
}

// Falta hacer uan funcion para que lea los csv distintos como ratinfs y movies


func main() {
	collection := connectMongo()

	file, err := os.Open("./data/ratings_clean.csv")
	if err != nil {
		log.Fatal("Error abriendo CSV:", err)
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))

	jobs := make(chan [][]string, numWorkers)
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(i, jobs, &wg, collection)
	}

	var batch [][]string
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		batch = append(batch, record)
		if len(batch) == chunkSize {
			jobs <- batch
			batch = nil
		}
	}

	if len(batch) > 0 {
		jobs <- batch
	}

	close(jobs)
	wg.Wait()
	fmt.Println("Procesamiento completado")
}
