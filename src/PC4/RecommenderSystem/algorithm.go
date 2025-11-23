package recommendersystem

import (
	"fmt"
	"pc4/tools"
	"sync"
)

func CosineSimilarity(target tools.UserVector, userVectors map[int]tools.UserVector, nWorkers int) []tools.Similarity {
	var wg sync.WaitGroup

	results := make(chan tools.Similarity, len(userVectors))
	jobs := make(chan int, nWorkers)

	for i := 0; i < nWorkers; i++ {
		wg.Add(1)
		go worker(i, jobs, results, target, userVectors, &wg)
	}

	for userID := range userVectors {
		jobs <- userID
	}
	close(jobs)
	wg.Wait()
	close(results)

	var similarities []tools.Similarity
	for s := range results {
		similarities = append(similarities, s)
	}

	return similarities
}

func worker(id int, jobs <-chan int, results chan<- tools.Similarity, target tools.UserVector, userVectors map[int]tools.UserVector, wg *sync.WaitGroup) {
	defer wg.Done()

	for userID := range jobs {
		sim := CosineSimilarityAlgoCPU(target, userVectors[userID]) //necesitas acceder al rating como tal
		results <- tools.Similarity{UserID: userID, Similarity: sim}
		fmt.Printf("Worker %d procesó user %d con similitud %.4f\n", id, userID, sim)
	}
}
