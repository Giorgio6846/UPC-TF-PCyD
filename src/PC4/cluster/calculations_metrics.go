package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"pc4/database"
	"sort"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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

func sendToCoordinator(targetID int, top []Similarity) error {
	data := map[string]interface{}{
		"target":    targetID,
		"neighbors": top,
	}
	jsonBytes, _ := json.Marshal(data)
	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write(jsonBytes)
	if err != nil {
		return err
	}
	fmt.Println("Resultados enviados al coordinador")
	return nil
}

func loadUserVectors(collection *mongo.Collection) (map[int]UserVector, error) {
	ctx := context.Background()
	cur, err := collection.Find(ctx, bson.M{}) //aca devuelve todo los docs
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	userVectors := make(map[int]UserVector)

	for cur.Next(ctx) {
		var r Rating
		cur.Decode(&r)
		if _, ok := userVectors[r.UserID]; !ok {
			userVectors[r.UserID] = make(UserVector) //Se inicializa el uservector paRa luego poblarlo
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
		sim := cosineSimilarity(target, allUsers[userID]) //necesitas acceder al rating como tal
		results <- Similarity{UserID: userID, Similarity: sim}
		//fmt.Printf("Worker %d procesó user %d con similitud %.4f\n", id, userID, sim)
	}
}

func computeSimilarUsers(targetID int, nWorkers int) {
	rdb := connectRedis()
	collection := database.ConnectMongo(database.Ratings)
	var allUsers map[int]UserVector

	count, _ := rdb.Exists(ctx, "user:1").Result()

	if count > 0 {
		fmt.Println("cargando vectores desde Redis...")
		allUsers = loadFromRedis(rdb)
	} else {
		fmt.Println("cargando vectores desde Mongo...")
		allUsers, _ = loadUserVectors(collection)

		fmt.Println("guardando vectores de usuarios en Redis")
		saveToRedis(rdb, allUsers)
	}

	target, ok := allUsers[targetID]
	if !ok {
		log.Fatalf("El usuario %d no existe en la DB.", targetID)
	}

	calcStart := time.Now()

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

	calcElapsed := time.Since(calcStart)
	fmt.Printf("tiempo de solo calculos de similitud: %s\n\n", calcElapsed)

	var sims []Similarity
	for s := range results {
		sims = append(sims, s)
	}

	sort.Slice(sims, func(i, j int) bool {
		return sims[i].Similarity > sims[j].Similarity
	})

	top := sims[:20] //top 20

	sendToCoordinator(targetID, top)

	//for _, s := range top {
	//fmt.Printf("User %d  → similarity %.4f\n", s.UserID, s.Similarity)
	//}
}

func main() {

	startAll := time.Now()
	targetUser := 100
	nWorkers := 300

	computeSimilarUsers(targetUser, nWorkers)

	totalElapsed := time.Since(startAll)
	fmt.Printf("\tempo TOTAL de ejecucion: %s\n", totalElapsed)
}
