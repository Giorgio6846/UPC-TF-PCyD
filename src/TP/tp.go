package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// We implement a simple user-based collaborative filtering recommender.
// Ratings are binary: `recommended` field from the dataset (true->1, false->0).

type datasetRow struct {
	app_id         int64
	author_steamid int64
	recommended    bool
}

func parseInt64(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseBool(s string) bool {
	s = strings.TrimSpace(s)
	if s == "1" {
		return true
	}
	if s == "0" {
		return false
	}
	return strings.EqualFold(s, "true")
}

// user->item->rating (0/1)
type userItemMatrix map[int64]map[int64]float64

// Read dataset and build a user-item map. We keep only app_id, author_steamid, recommended.
func readDataset(path string, sampleUsers, sampleItems int) (userItemMatrix, []int64, []int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// read header
	if _, err := reader.Read(); err != nil {
		return nil, nil, nil, err
	}

	m := make(userItemMatrix)
	userSet := make(map[int64]struct{})
	itemSet := make(map[int64]struct{})

	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, nil, err
		}
		if len(rec) < 17 { // dataset has many columns; author_steamid at 16 and app_id at 1
			continue
		}
		appID := parseInt64(rec[1])
		author := parseInt64(rec[16])
		recommended := parseBool(rec[8])

		if _, ok := m[author]; !ok {
			m[author] = make(map[int64]float64)
		}
		// rating: 1.0 for recommended, 0.0 otherwise
		var r float64
		if recommended {
			r = 1.0
		} else {
			r = 0.0
		}
		m[author][appID] = r
		userSet[author] = struct{}{}
		itemSet[appID] = struct{}{}
	}

	// convert sets to slices
	users := make([]int64, 0, len(userSet))
	for u := range userSet {
		users = append(users, u)
	}
	items := make([]int64, 0, len(itemSet))
	for it := range itemSet {
		items = append(items, it)
	}

	// Optionally sample users and items for faster experiments
	if sampleUsers > 0 && sampleUsers < len(users) {
		rand.Seed(42)
		perm := rand.Perm(len(users))[:sampleUsers]
		sampled := make(map[int64]struct{})
		for _, idx := range perm {
			sampled[users[idx]] = struct{}{}
		}
		// filter m
		newM := make(userItemMatrix)
		for u := range sampled {
			newM[u] = m[u]
		}
		m = newM
		users = users[:0]
		for u := range sampled {
			users = append(users, u)
		}
	}
	if sampleItems > 0 && sampleItems < len(items) {
		rand.Seed(43)
		perm := rand.Perm(len(items))[:sampleItems]
		sampled := make(map[int64]struct{})
		for _, idx := range perm {
			sampled[items[idx]] = struct{}{}
		}
		// filter item lists inside m
		for u := range m {
			for it := range m[u] {
				if _, ok := sampled[it]; !ok {
					delete(m[u], it)
				}
			}
		}
		items = items[:0]
		for it := range sampled {
			items = append(items, it)
		}
	}

	return m, users, items, nil
}

// Holdout: for each user with at least 2 items, remove one random item as test.
type testCase struct {
	user   int64
	item   int64
	actual float64
}

func buildHoldout(m userItemMatrix) (train userItemMatrix, tests []testCase) {
	rand.Seed(123)
	train = make(userItemMatrix)
	for u, its := range m {
		train[u] = make(map[int64]float64)
		for it, r := range its {
			train[u][it] = r
		}
	}

	for u, its := range m {
		if len(its) < 2 {
			continue
		}
		// pick one random item
		keys := make([]int64, 0, len(its))
		for it := range its {
			keys = append(keys, it)
		}
		it := keys[rand.Intn(len(keys))]
		actual := train[u][it]
		delete(train[u], it)
		tests = append(tests, testCase{user: u, item: it, actual: actual})
	}
	return train, tests
}

// Similarity metrics for sparse user vectors (maps)
func cosineSim(a, b map[int64]float64) float64 {
	var dot, na, nb float64
	for k, va := range a {
		na += va * va
		if vb, ok := b[k]; ok {
			dot += va * vb
		}
	}
	for _, vb := range b {
		nb += vb * vb
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func pearsonSim(a, b map[int64]float64) float64 {
	// compute over common items only
	var n int
	var sumA, sumB, sumA2, sumB2, sumAB float64
	for k, va := range a {
		if vb, ok := b[k]; ok {
			n++
			sumA += va
			sumB += vb
			sumA2 += va * va
			sumB2 += vb * vb
			sumAB += va * vb
		}
	}
	if n == 0 {
		return 0
	}
	num := float64(n)*sumAB - sumA*sumB
	den := math.Sqrt((float64(n)*sumA2 - sumA*sumA) * (float64(n)*sumB2 - sumB*sumB))
	if den == 0 {
		return 0
	}
	return num / den
}

// Pick similarity function by name
func getSimFunc(name string) func(map[int64]float64, map[int64]float64) float64 {
	switch name {
	case "cosine":
		return cosineSim
	case "pearson":
		return pearsonSim
	default:
		return cosineSim
	}
}

// Predict rating for (user,item) using neighbors in train.
// neighbors: all other users in train.
func predictSequential(train userItemMatrix, user int64, item int64, sim func(map[int64]float64, map[int64]float64) float64) float64 {
	uvec, ok := train[user]
	if !ok {
		return 0.0
	}
	var num, den float64
	for v, vec := range train {
		if v == user {
			continue
		}
		rv, has := vec[item]
		if !has {
			continue
		}
		s := sim(uvec, vec)
		if s == 0 {
			continue
		}
		num += s * rv
		den += math.Abs(s)
	}
	if den == 0 { // fallback to user's mean
		var sum float64
		var cnt float64
		for _, r := range uvec {
			sum += r
			cnt++
		}
		if cnt == 0 {
			return 0
		}
		return sum / cnt
	}
	return num / den
}

// Concurrent prediction: compute similarities in parallel using worker pool
func predictConcurrent(train userItemMatrix, user int64, item int64, sim func(map[int64]float64, map[int64]float64) float64, workers int) float64 {
	uvec, ok := train[user]
	if !ok {
		return 0.0
	}
	type job struct {
		vid int64
		vec map[int64]float64
	}
	jobs := make(chan job)
	results := make(chan struct{ num, den float64 })

	var wg sync.WaitGroup
	// workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if rv, ok := j.vec[item]; ok {
					s := sim(uvec, j.vec)
					if s != 0 {
						results <- struct{ num, den float64 }{num: s * rv, den: math.Abs(s)}
					}
				}
			}
		}()
	}

	// collector
	var collectorWg sync.WaitGroup
	collectorWg.Add(1)
	var num, den float64
	go func() {
		defer collectorWg.Done()
		for r := range results {
			num += r.num
			den += r.den
		}
	}()

	// send jobs
	go func() {
		for v, vec := range train {
			if v == user {
				continue
			}
			jobs <- job{vid: v, vec: vec}
		}
		close(jobs)
	}()

	wg.Wait()
	close(results)
	collectorWg.Wait()

	if den == 0 {
		var sum float64
		var cnt float64
		for _, r := range uvec {
			sum += r
			cnt++
		}
		if cnt == 0 {
			return 0
		}
		return sum / cnt
	}
	return num / den
}

// Evaluate a set of test cases using given predictor function. Returns RMSE, MAE, accuracy (threshold 0.5)
func evaluate(tests []testCase, predictor func(int64, int64) float64) (rmse, mae, acc float64) {
	var sumSq, sumAbs float64
	var correct int
	for _, tc := range tests {
		pred := predictor(tc.user, tc.item)
		err := pred - tc.actual
		sumSq += err * err
		if err < 0 {
			sumAbs += -err
		} else {
			sumAbs += err
		}
		if (pred >= 0.5) == (tc.actual >= 0.5) {
			correct++
		}
	}
	n := float64(len(tests))
	if n == 0 {
		return 0, 0, 0
	}
	rmse = math.Sqrt(sumSq / n)
	mae = sumAbs / n
	acc = float64(correct) / n
	return
}

func keysFromMap(m map[int64]map[int64]float64) []int64 {
	ks := make([]int64, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i] < ks[j] })
	return ks
}

// Compute pairwise similarities sequentially. Returns map[u][v]=sim for u<v (symmetric stored both ways).
func computeSimilaritiesSeq(train userItemMatrix, sim func(map[int64]float64, map[int64]float64) float64) map[int64]map[int64]float64 {
	users := keysFromMap(train)
	sims := make(map[int64]map[int64]float64)
	for i := 0; i < len(users); i++ {
		ui := users[i]
		if _, ok := sims[ui]; !ok {
			sims[ui] = make(map[int64]float64)
		}
		for j := i + 1; j < len(users); j++ {
			uj := users[j]
			s := sim(train[ui], train[uj])
			if s != 0 {
				sims[ui][uj] = s
				if _, ok := sims[uj]; !ok {
					sims[uj] = make(map[int64]float64)
				}
				sims[uj][ui] = s
			}
		}
	}
	return sims
}

type pairJob struct{ i, j int }
type pairRes struct {
	ui, uj int64
	s      float64
}

// Compute pairwise similarities concurrently using worker pool.
func computeSimilaritiesCon(train userItemMatrix, sim func(map[int64]float64, map[int64]float64) float64, workers int) map[int64]map[int64]float64 {
	users := keysFromMap(train)
	jobs := make(chan pairJob, 1024)
	results := make(chan pairRes, 1024)

	// spawn workers
	var wg sync.WaitGroup
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				ui := users[job.i]
				uj := users[job.j]
				s := sim(train[ui], train[uj])
				if s != 0 {
					results <- pairRes{ui, uj, s}
				}
			}
		}()
	}

	// feed jobs
	go func() {
		for i := 0; i < len(users); i++ {
			for j := i + 1; j < len(users); j++ {
				jobs <- pairJob{i, j}
			}
		}
		close(jobs)
	}()

	// collector
	go func() {
		wg.Wait()
		close(results)
	}()

	sims := make(map[int64]map[int64]float64)
	for r := range results {
		if _, ok := sims[r.ui]; !ok {
			sims[r.ui] = make(map[int64]float64)
		}
		if _, ok := sims[r.uj]; !ok {
			sims[r.uj] = make(map[int64]float64)
		}
		sims[r.ui][r.uj] = r.s
		sims[r.uj][r.ui] = r.s
	}

	return sims
}

// Return top-k pairs (ui,uj,s) from sims map
func topKPairs(sims map[int64]map[int64]float64, k int) []struct {
	ui, uj int64
	s      float64
} {
	type p struct {
		ui, uj int64
		s      float64
	}
	list := make([]p, 0, 1024)
	seen := make(map[string]struct{})
	for u, m := range sims {
		for v, s := range m {
			// ensure pair uniqueness (u<v)
			if u >= v {
				continue
			}
			key := fmt.Sprintf("%d_%d", u, v)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			list = append(list, p{ui: u, uj: v, s: s})
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].s > list[j].s })
	if k > len(list) {
		k = len(list)
	}
	res := make([]struct {
		ui, uj int64
		s      float64
	}, k)
	for i := 0; i < k; i++ {
		res[i] = struct {
			ui, uj int64
			s      float64
		}{list[i].ui, list[i].uj, list[i].s}
	}
	return res
}

func countPairs(sims map[int64]map[int64]float64) int {
	cnt := 0
	for u, m := range sims {
		for v := range m {
			if u < v {
				cnt++
			}
		}
	}
	return cnt
}

// Parallel CSV reader: one goroutine reads records from csv.Reader and N workers parse
// the records and update the shared userItemMatrix. This amortizes parsing cost.
func readDatasetParallel(path string, sampleUsers, sampleItems, workers int) (userItemMatrix, []int64, []int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.ReuseRecord = true
	reader.FieldsPerRecord = -1

	// read header
	if _, err := reader.Read(); err != nil {
		return nil, nil, nil, err
	}

	//Canales
	// channels
	recCh := make(chan []string, 4096)

	//Zona critica :D
	// shared map and mutex
	m := make(userItemMatrix)
	var mu sync.Mutex

	// worker goroutines
	var wg sync.WaitGroup
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rec := range recCh {
				if len(rec) < 17 {
					continue
				}
				appID := parseInt64(rec[1])
				author := parseInt64(rec[16])
				recommended := parseBool(rec[8])

				var r float64
				if recommended {
					r = 1.0
				} else {
					r = 0.0
				}

				mu.Lock()
				if _, ok := m[author]; !ok {
					m[author] = make(map[int64]float64)
				}
				m[author][appID] = r
				mu.Unlock()
			}
		}()
	}

	// reader goroutine: copy records (because ReuseRecord=true) and send to recCh
	readErr := make(chan error, 1)
	go func() {
		defer close(recCh)
		for {
			rec, err := reader.Read()
			if err == io.EOF {
				readErr <- nil
				return
			}
			if err != nil {
				readErr <- err
				return
			}
			// copy record because reader may reuse underlying buffer
			recCopy := make([]string, len(rec))
			copy(recCopy, rec)
			recCh <- recCopy
		}
	}()

	// wait workers finish
	wg.Wait()
	if err := <-readErr; err != nil {
		return nil, nil, nil, err
	}

	// convert sets to slices
	users := make([]int64, 0, len(m))
	itemsSet := make(map[int64]struct{})
	for u, its := range m {
		users = append(users, u)
		for it := range its {
			itemsSet[it] = struct{}{}
		}
	}
	items := make([]int64, 0, len(itemsSet))
	for it := range itemsSet {
		items = append(items, it)
	}

	// Optional sampling (same semantics as readDataset)
	if sampleUsers > 0 && sampleUsers < len(users) {
		rand.Seed(42)
		perm := rand.Perm(len(users))[:sampleUsers]
		sampled := make(map[int64]struct{})
		for _, idx := range perm {
			sampled[users[idx]] = struct{}{}
		}
		newM := make(userItemMatrix)
		for u := range sampled {
			newM[u] = m[u]
		}
		m = newM
		users = users[:0]
		for u := range sampled {
			users = append(users, u)
		}
	}
	if sampleItems > 0 && sampleItems < len(items) {
		rand.Seed(43)
		perm := rand.Perm(len(items))[:sampleItems]
		sampled := make(map[int64]struct{})
		for _, idx := range perm {
			sampled[items[idx]] = struct{}{}
		}
		for u := range m {
			for it := range m[u] {
				if _, ok := sampled[it]; !ok {
					delete(m[u], it)
				}
			}
		}
		items = items[:0]
		for it := range sampled {
			items = append(items, it)
		}
	}

	return m, users, items, nil
}

func main() {
	// flags
	algorithm := flag.String("algorithm", "cosine", "Similarity algorithm: cosine|pearson")
	csvPath := flag.String("data", "data/steam_reviews.csv", "Path to CSV dataset (keep as provided)")
	sampleUsers := flag.Int("sample_users", 0, "Sample number of users (0 = all)")
	sampleItems := flag.Int("sample_items", 0, "Sample number of items (0 = all)")
	workers := flag.Int("num_cores", 10, "Number of cores")
	flag.Parse()

	fmt.Printf("Reading dataset %s (sample users=%d items=%d)\n", *csvPath, *sampleUsers, *sampleItems)
	matrix, _, items, err := readDatasetParallel(*csvPath, *sampleUsers, *sampleItems, *workers)
	if err != nil {
		fmt.Println("Error reading dataset:", err)
		return
	}
	fmt.Printf("Users=%d Items=%d\n", len(matrix), len(items))

	// build holdout
	train, tests := buildHoldout(matrix)
	fmt.Printf("Train users=%d  Test cases=%d\n", len(train), len(tests))

	simFunc := getSimFunc(*algorithm)

	// Instead of training/prediction, compute user-user similarities
	fmt.Println("Computing user-user similarities...")
	// limit number of users to compute pairwise similarities when dataset is large
	usersAll := keysFromMap(train)
	maxUsersForPairs := 1000
	var trainSub userItemMatrix
	if len(usersAll) > maxUsersForPairs {
		fmt.Printf("Too many users (%d), sampling first %d for pairwise similarities\n", len(usersAll), maxUsersForPairs)
		trainSub = make(userItemMatrix)
		for i := 0; i < maxUsersForPairs; i++ {
			u := usersAll[i]
			trainSub[u] = train[u]
		}
	} else {
		trainSub = train
	}

	// Sequential similarity
	t0 := time.Now()
	simsSeq := computeSimilaritiesSeq(trainSub, simFunc)
	durSeq := time.Since(t0)
	fmt.Printf("Sequential similarities computed in %v, pairs=%d\n", durSeq, countPairs(simsSeq))
	top := topKPairs(simsSeq, 10)
	fmt.Println("Top-10 similar user pairs (seq):")
	for _, p := range top {
		fmt.Printf("%d - %d : %.4f\n", p.ui, p.uj, p.s)
	}

	// Concurrent similarity
	// reuse workers variable from above
	t0 = time.Now()
	simsCon := computeSimilaritiesCon(trainSub, simFunc, *workers)
	durCon := time.Since(t0)
	fmt.Printf("Concurrent similarities computed in %v (workers=%d), pairs=%d\n", durCon, workers, countPairs(simsCon))
	top = topKPairs(simsCon, 10)
	fmt.Println("Top-10 similar user pairs (con):")
	for _, p := range top {
		fmt.Printf("%d - %d : %.4f\n", p.ui, p.uj, p.s)
	}

	fmt.Println("Done")
}
