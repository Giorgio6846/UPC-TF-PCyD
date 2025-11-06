package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"time"
)

type ProgramInfo struct {
	amountGoroutines int
	universe         int
	seed             int64
}

type Pairs struct {
	user  []int64
	games [][]int64

	rows, columns int
}

type algorithmResult struct {
	algorithm string
	timeSeq   time.Duration
	answerSeq float64
	timeCon   time.Duration
	answerCon float64
}

func fillUser(cols, universe int, r *rand.Rand) []int64 {
	user := make([]int64, cols)

	for i := range user {
		user[i] = int64(r.Intn(universe))
	}

	return user
}

func fillGames(rows, cols, universe int, r *rand.Rand) [][]int64 {
	games := make([][]int64, rows)

	for i := 0; i < rows; i++ {
		row := make([]int64, cols)
		for j := range row {
			row[j] = int64(r.Intn(universe))
		}
		games[i] = row
	}

	return games
}

//func fillPairs(programInfo ProgramInfo, pairs *Pairs) {
//	var wg sync.WaitGroup
//
//	wg.Add(2)
//	go func() {
//		defer wg.Done()
//		r := rand.New(rand.NewSource(int64(programInfo.seed)))
//		pairs.user = fillTensorR1(pairs.columns, programInfo.universe, r)
//	}()
//	go func() {
//		defer wg.Done()
//		r := rand.New(rand.NewSource(int64(programInfo.seed) + 100))
//		pairs.games = fillTensorR2(pairs.rows, pairs.columns, programInfo.universe, r)
//	}()
//	wg.Wait()
//}

func sumArray(numArray []int64) float64 {
	var sum int64
	for _, num := range numArray {
		sum += num
	}
	return float64(sum)
}

func sumSquareArray(numArray []int64) float64 {
	var sum int64
	for _, num := range numArray {
		sum += num * num
	}
	return float64(sum)
}

func dotProduct(numArrayA, numArrayB []int64) float64 {
	dot := int64(0)

	for i := 0; i < len(numArrayA); i++ {
		dot += numArrayA[i] * numArrayB[i]
	}

	return float64(dot)
}

func cosine(user []int64, game []int64) float64 {
	dotProduct := dotProduct(user, game)
	nA := math.Sqrt(sumSquareArray(user))
	nB := math.Sqrt(sumSquareArray(game))

	return dotProduct / (nA * nB)
}

func pearson(user []int64, game []int64) float64 {
	dotProduct := dotProduct(user, game)
	sumA := sumArray(user)
	sumB := sumArray(game)
	sum2A := sumSquareArray(user)
	sum2B := sumSquareArray(game)

	lenVector := float64(len(user))

	return (lenVector*dotProduct - sumA*sumB) / math.Sqrt((lenVector*sum2A-sumA*sumA)*(lenVector*sum2B-sumB*sumB))
}

func jaccard(user []int64, game []int64) float64 {
	setA := make(map[int64]struct{})
	setB := make(map[int64]struct{})

	for _, num := range user {
		setA[num] = struct{}{}
	}

	for _, num := range game {
		setB[num] = struct{}{}
	}

	intersection := 0
	for x := range setA {
		if _, found := setB[x]; found {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection

	return float64(intersection) / float64(union)
}

func cosineSeq(pairs *Pairs) []result {
	results := make([]result, pairs.rows)
	for i := 0; i < pairs.rows; i++ {
		game := pairs.games[i]
		results = append(results, result{index: i, value: cosine(pairs.user, game)})
	}
	return results
}

func pearsonSeq(pairs *Pairs) []result {
	results := make([]result, pairs.rows)
	for i := 0; i < pairs.rows; i++ {
		game := pairs.games[i]
		results = append(results, result{index: i, value: pearson(pairs.user, game)})
	}
	return results
}

func jaccardSeq(pairs *Pairs) []result {
	results := make([]result, pairs.rows)
	for i := 0; i < pairs.rows; i++ {
		game := pairs.games[i]
		results = append(results, result{index: i, value: jaccard(pairs.user, game)})
	}
	return results
}

func cosineCon(pairs *Pairs) []result  { return computeCon(pairs, cosine) }
func pearsonCon(pairs *Pairs) []result { return computeCon(pairs, pearson) }
func jaccardCon(pairs *Pairs) []result { return computeCon(pairs, jaccard) }

type job struct {
	index int
	game  []int64
}

type result struct {
	index int
	value float64
}

func gen(ctx context.Context, p *Pairs) <-chan job {
	ch := make(chan job)
	go func() {
		defer close(ch)
		for i := 0; i < p.rows; i++ {
			select {
			case <-ctx.Done():
				return
			case ch <- job{index: i, game: p.games[i]}:
			}
		}
	}()
	return ch
}

func workerMetric(
	ctx context.Context,
	user []int64,
	in <-chan job,
	out chan<- result,
	metric func([]int64, []int64) float64,
) {
	for j := range in {
		r := result{index: j.index, value: metric(user, j.game)}
		select {
		case <-ctx.Done():
			return
		case out <- r:
		}
	}
}

func computeCon(pairs *Pairs, metric func([]int64, []int64) float64) []result {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := gen(ctx, pairs)
	workers := runtime.NumCPU()

	out := make(chan result)
	go func() {
		defer close(out)
		done := make(chan struct{}, workers)
		for i := 0; i < workers; i++ {
			go func() {
				workerMetric(ctx, pairs.user, jobs, out, metric)
				done <- struct{}{}
			}()
		}
		for i := 0; i < workers; i++ {
			<-done
		}
	}()

	res := make([]result, pairs.rows)
	for r := range out {
		res[r.index] = r
	}
	return res
}

func printTime(result algorithmResult) {
	fmt.Printf("%s Seq %f \n", result.algorithm, result.answerSeq)
	fmt.Printf("Time duration %v \n", result.timeSeq)
	fmt.Printf("%s Con %f \n", result.algorithm, result.answerCon)
	fmt.Printf("Time duration %v \n \n", result.timeCon)
}

func mean(data []result) float64 {
	var sum float64
	for i := 0; i < len(data); i++ {
		sum += data[i].value
	}

	return sum / float64(len(data))
}

func main() {
	seed := flag.Int64("seed", 42, "Set RNG seed")
	//goroutines := flag.Int("goroutines", 100, "Amount of goroutines")
	rowsDim := flag.Int("row_dim", 1_000, "Row shape, x")
	colsDim := flag.Int("col_dim", 1_000, "Col shape, x")
	algorithm := flag.String("algorithm", "all", "Select Algorithm: cosine | pearson | jaccard | all")

	flag.Parse()

	pairs := &Pairs{
		rows:    *rowsDim,
		columns: *colsDim,
		games:   fillGames(*rowsDim, *colsDim, int(math.Max(float64(*rowsDim), float64(*colsDim))), rand.New(rand.NewSource(*seed))),
		user:    fillUser(*colsDim, int(math.Max(float64(*rowsDim), float64(*colsDim))), rand.New(rand.NewSource(*seed))),
	}

	//programInfo := ProgramInfo{amountGoroutines: *goroutines, seed: int64(*seed), universe: int(math.Max(float64(*rowsDim), float64(*colsDim)))}

	var algoRes algorithmResult
	switch *algorithm {
	case "cosine":
		algoRes.algorithm = *algorithm

		t0 := time.Now()
		algoRes.answerSeq = mean(cosineSeq(pairs))
		algoRes.timeSeq = time.Since(t0)

		t0 = time.Now()
		algoRes.answerSeq = mean(cosineCon(pairs))
		algoRes.timeCon = time.Since(t0)

		printTime(algoRes)
	case "pearson":
		algoRes.algorithm = *algorithm

		t0 := time.Now()
		algoRes.answerSeq = mean(pearsonSeq(pairs))
		algoRes.timeSeq = time.Since(t0)

		t0 = time.Now()
		algoRes.answerCon = mean(pearsonCon(pairs))
		algoRes.timeCon = time.Since(t0)

		printTime(algoRes)
	case "jaccard":
		algoRes.algorithm = *algorithm

		t0 := time.Now()
		algoRes.answerSeq = mean(jaccardSeq(pairs))
		algoRes.timeSeq = time.Since(t0)

		t0 = time.Now()
		algoRes.answerCon = mean(jaccardCon(pairs))
		algoRes.timeCon = time.Since(t0)

		printTime(algoRes)
	case "all":
		algoRes.algorithm = "cosine"

		t0 := time.Now()
		algoRes.answerSeq = mean(cosineSeq(pairs))
		algoRes.timeSeq = time.Since(t0)

		t0 = time.Now()
		algoRes.answerCon = mean(cosineCon(pairs))
		algoRes.timeCon = time.Since(t0)

		printTime(algoRes)

		algoRes.algorithm = "pearson"

		t0 = time.Now()
		algoRes.answerSeq = mean(pearsonSeq(pairs))
		algoRes.timeSeq = time.Since(t0)

		t0 = time.Now()
		algoRes.answerCon = mean(pearsonCon(pairs))
		algoRes.timeCon = time.Since(t0)

		printTime(algoRes)

		algoRes.algorithm = "jaccard"

		t0 = time.Now()
		algoRes.answerSeq = mean(jaccardSeq(pairs))
		algoRes.timeSeq = time.Since(t0)

		t0 = time.Now()
		algoRes.answerCon = mean(jaccardCon(pairs))
		algoRes.timeCon = time.Since(t0)

		printTime(algoRes)

	default:
		print("Tipo de algoritmo incorrecto")
	}
}
