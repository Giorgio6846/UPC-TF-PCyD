package main

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"time"
)

type Job struct {
	id int
	a  []float64
	b  []float64
}

type Result struct {
	id     int
	score  float64
	metric string
}

func throwPanic(lenVecA, lenVecB int) {
	if lenVecA != lenVecB {
		panic("Los vectores deben tener la misma cantidad de valores")
	}
}

func jaccardIndex(vecA, vecB []float64) float64 {
	throwPanic(len(vecA), len(vecB))
	var inter, union int
	for i := range vecA {
		if vecA[i] != 0 || vecB[i] != 0 {
			union++
		}
		if vecA[i] != 0 && vecB[i] != 0 {
			inter++
		}
	}
	if union != 0 {
		return float64(inter) / float64(union)
	}
	return 0
}

func pearsonCorrelation(vecA, vecB []float64) float64 {
	var meanA, meanB, upper, cumX, cumY float64

	throwPanic(len(vecA), len(vecB))

	for i := range vecA {
		meanA += vecA[i]
		meanB += vecB[i]
	}

	meanA /= float64(len(vecA))
	meanB /= float64(len(vecB))

	for i := range vecA {
		upper += (vecA[i] - meanA) * (vecB[i] - meanB)
		cumX += math.Pow(vecA[i]-meanA, 2)
		cumY += math.Pow(vecB[i]-meanB, 2)
	}

	if cumX == 0 || cumY == 0 {
		return 0
	}
	return upper / (math.Sqrt(cumX) * math.Sqrt(cumY))
}

func cosineSimilarity(vecA, vecB []float64) float64 {
	var dotProduct, magA, magB float64

	throwPanic(len(vecA), len(vecB))

	for i := range vecA {
		dotProduct += vecA[i] * vecB[i]
		magA += vecA[i] * vecA[i]
		magB += vecB[i] * vecB[i]
	}

	if magA != 0 && magB != 0 {
		return dotProduct / (math.Sqrt(magA) * math.Sqrt(magB))
	}
	return 0
}

func makeRandomVector(n int, max float64) []float64 {
	v := make([]float64, n)
	for i := range v {
		v[i] = rand.Float64() * max
	}
	return v
}

func createDatasetVector(n int, dim int, max float64) [][]float64 {
	df := make([][]float64, n)
	for i := 0; i < n; i++ {
		df[i] = makeRandomVector(dim, max)
	}
	return df
}

func worker(id int, tasks <-chan Job, results chan<- Result) {
	for job := range tasks {
		scoreC := cosineSimilarity(job.a, job.b)
		scoreP := pearsonCorrelation(job.a, job.b)
		scoreJ := jaccardIndex(job.a, job.b)
		results <- Result{id: job.id, score: scoreC, metric: "Cosine"}
		results <- Result{id: job.id, score: scoreP, metric: "Pearson"}
		results <- Result{id: job.id, score: scoreJ, metric: "Jaccard"}
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	numWorkers := runtime.NumCPU()

	dataset := createDatasetVector(100, 1000, 5.0)
	numTasks := len(dataset)

	//Vector a comparar
	vecA := makeRandomVector(1000, 5.0)

	startC := time.Now()

	tasks := make(chan Job, numWorkers)
	results := make(chan Result, numTasks*3)

	for i := 0; i < numWorkers; i++ {
		go worker(i, tasks, results)
	}

	for j := 0; j < numTasks; j++ {
		tasks <- Job{id: j, a: vecA, b: dataset[j]}
	}
	close(tasks)

	for k := 0; k < numTasks*3; k++ {
		result := <-results
		fmt.Printf("Result: job=%d, score=%f, metric=%s\n", result.id, result.score, result.metric)
	}

	elapsedC := time.Since(startC)
	fmt.Printf("Tiempo transcurrido (concurrente): %f\n", elapsedC.Seconds())

	start := time.Now()
	for j := 0; j < numTasks; j++ {
		_ = cosineSimilarity(vecA, dataset[j])
		_ = pearsonCorrelation(vecA, dataset[j])
		_ = jaccardIndex(vecA, dataset[j])
	}
	elapsed := time.Since(start)
	fmt.Printf("Tiempo transcurrido (secuencial): %f\n", elapsed.Seconds())

	fmt.Printf("Speedup: %f\n", elapsed.Seconds()/elapsedC.Seconds())
}
