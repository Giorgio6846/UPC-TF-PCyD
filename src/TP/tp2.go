package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"math/rand"
	"os"
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

func readSelectedColumns(path string, cols []string, maxRows int, maxUniqueApps int) (map[string][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, _ := reader.Read()

	colIndex := make(map[string]int)
	for i, h := range header {
		for _, c := range cols {
			if h == c {
				colIndex[c] = i
			}
		}
	}

	data := make(map[string][]string)
	for _, c := range cols {
		data[c] = []string{}
	}

	uniqueApps := make(map[string]bool)
	rowCount := 0

	for {
		if rowCount >= maxRows || len(uniqueApps) >= maxUniqueApps {
			break
		}

		record, err := reader.Read()
		if err != nil {
			break
		}

		appID := record[colIndex["app_id"]]
		uniqueApps[appID] = true

		for _, c := range cols {
			data[c] = append(data[c], record[colIndex[c]])
		}

		rowCount++
	}

	fmt.Printf("Leídas %d filas y %d juegos distintos\n", rowCount, len(uniqueApps))
	return data, nil
}

func buildUserProfiles(data map[string][]string) map[string]map[string]float64 {
	profiles := make(map[string]map[string]float64)
	for i := range data["author.steamid"] {
		user := data["author.steamid"][i]
		app := data["app_id"][i]
		rec := data["recommended"][i] == "True"

		if _, ok := profiles[user]; !ok {
			profiles[user] = make(map[string]float64)
		}
		if rec {
			profiles[user][app] = 1.0
		} else {
			profiles[user][app] = 0.0
		}
	}
	return profiles
}

func buildVectors(userA, userB map[string]float64) ([]float64, []float64) {
	allApps := make(map[string]bool)
	for app := range userA {
		allApps[app] = true
	}
	for app := range userB {
		allApps[app] = true
	}

	vecA := []float64{}
	vecB := []float64{}

	for app := range allApps {
		vecA = append(vecA, userA[app])
		vecB = append(vecB, userB[app])
	}

	return vecA, vecB
}

func getRandomActiveUsers(profiles map[string]map[string]float64, minReviews int) (string, string) {
	userIDs := []string{}
	for id, games := range profiles {
		if len(games) >= minReviews {
			userIDs = append(userIDs, id)
		}
	}

	if len(userIDs) < 2 {
		panic("No hay suficientes usuarios con reviews válidos")
	}

	rand.Seed(time.Now().UnixNano())
	i := rand.Intn(len(userIDs))
	j := rand.Intn(len(userIDs))
	for j == i { // evitar que sea el mismo
		j = rand.Intn(len(userIDs))
	}

	return userIDs[i], userIDs[j]
}

func main() {
	cols := []string{"author.steamid", "app_id", "recommended"}
	data, _ := readSelectedColumns("data/steam_reviews.csv", cols, 21_000_000, 2000)
	fmt.Printf("Primeros valores: %+v\n", data["author.steamid"][:5])

	profiles := buildUserProfiles(data)

	userAID, userBID := getRandomActiveUsers(profiles, 10)
	userA := profiles[userAID]
	userB := profiles[userBID]

	vecAU, vecBU := buildVectors(userA, userB)

	fmt.Println("Usuario A:", userAID)
	fmt.Println("Usuario B:", userBID)
	fmt.Println("Longitud de vectores:", len(vecAU))

	//Versión Concurrente
	numWorkers := runtime.NumCPU()
	tasks := make(chan Job, numWorkers)
	results := make(chan Result, 3)

	startC := time.Now()

	for i := 0; i < numWorkers; i++ {
		go worker(i, tasks, results)
	}

	tasks <- Job{id: 1, a: vecAU, b: vecBU}
	close(tasks)

	// Recibimos los 3 resultados
	for k := 0; k < 3; k++ {
		result := <-results
		fmt.Printf("[Concurrente] %s: %.6f\n", result.metric, result.score)
	}

	elapsedC := time.Since(startC)
	fmt.Printf("⏱ Tiempo concurrente: %.6f segundos\n", elapsedC.Seconds())

	//Versión Secuencial
	start := time.Now()

	cosine := cosineSimilarity(vecAU, vecBU)
	pearson := pearsonCorrelation(vecAU, vecBU)
	jaccard := jaccardIndex(vecAU, vecBU)

	elapsed := time.Since(start)
	fmt.Printf("\n[Secuencial] Cosine: %.6f\n", cosine)
	fmt.Printf("[Secuencial] Pearson: %.6f\n", pearson)
	fmt.Printf("[Secuencial] Jaccard: %.6f\n", jaccard)
	fmt.Printf("⏱ Tiempo secuencial: %.6f segundos\n", elapsed.Seconds())

	speedup := elapsed.Seconds() / elapsedC.Seconds()
	fmt.Printf("Speedup: %.4fx\n", speedup)
}
