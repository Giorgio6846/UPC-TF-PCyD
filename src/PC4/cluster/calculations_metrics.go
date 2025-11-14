package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Rating struct {
	UserID  int     `bson:"userId"`
	MovieID int     `bson:"movieId"`
	Rating  float64 `bson:"rating"`
}

type UserVector map[int]float64

type Similarity struct {
	UserID     int
	Similarity float64
}


func connectMongo() *mongo.Collection {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	uri := "mongodb://hello:world@localhost:27017"
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}

	return client.Database("movielens").Collection("ratings")
}


func loadUserVectors(collection *mongo.Collection) (map[int]UserVector, error) {
	ctx := context.Background()

	cur, err := collection.Find(ctx, map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	userVectors := make(map[int]UserVector)

	for cur.Next(ctx) {
		var r Rating
		if err := cur.Decode(&r); err != nil {
			return nil, err
		}

		if _, ok := userVectors[r.UserID]; !ok {
			userVectors[r.UserID] = make(UserVector)
		}

		userVectors[r.UserID][r.MovieID] = r.Rating
	}

	return userVectors, nil
}

func cosineSimilarity(a, b UserVector) float64 {
	var dot, magA, magB float64

	for movie, rA := range a {
		if rB, ok := b[movie]; ok {
			dot += rA * rB
		}
		magA += rA * rA
	}
	for _, rB := range b {
		magB += rB * rB
	}

	if magA == 0 || magB == 0 {
		return 0
	}

	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}

func worker(id int, jobs <-chan int, results chan<- Similarity, target UserVector, allUsers map[int]UserVector, wg *sync.WaitGroup) {
	defer wg.Done()

	for userID := range jobs {
		sim := cosineSimilarity(target, allUsers[userID])
		results <- Similarity{UserID: userID, Similarity: sim}
	}
	fmt.Printf("Worker %d done\n", id)
}


func computeSimilarUsers(targetID int, nWorkers int) {
	collection := connectMongo()

	allUsers, _ := loadUserVectors(collection)
	target := allUsers[targetID]

	jobs := make(chan int, nWorkers)
	results := make(chan Similarity, len(allUsers))

	var wg sync.WaitGroup

	for i := 0; i < nWorkers; i++ {
		wg.Add(1)
		go worker(i, jobs, results, target, allUsers, &wg)
	}

	for userID := range allUsers {
		if userID != targetID {
			jobs <- userID
		}
	}

	close(jobs)
	wg.Wait()
	close(results)

	var sims []Similarity
	for s := range results {
		sims = append(sims, s)
	}

	sort.Slice(sims, func(i, j int) bool {
		return sims[i].Similarity > sims[j].Similarity
	})

	top := sims[:20]

	fmt.Println("\n20 usuarios más similares:")
	for _, s := range top {
		fmt.Printf("User %d  → similarity %.4f\n", s.UserID, s.Similarity)
	}
}


func main() {
	targetUser := 1         
	nWorkers := 20        

	computeSimilarUsers(targetUser, nWorkers)
}
