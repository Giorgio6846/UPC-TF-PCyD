package ds

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	recommendersystem "pc4/RecommenderSystem"
	"pc4/database"
	"pc4/tools"
	"sort"
	"sync"
)

type WorkerRegistry struct {
	mu      sync.Mutex
	workers map[string]net.Conn
}

var registry *WorkerRegistry
var responses chan tools.JobWorker

func NewWorkerRegistry() *WorkerRegistry {
	return &WorkerRegistry{
		workers: make(map[string]net.Conn),
	}
}

func (r *WorkerRegistry) Add(remote string, conn net.Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workers[remote] = conn
	log.Printf("registered %s from %s", remote, conn.RemoteAddr().String())
}

func (r *WorkerRegistry) Remove(remote string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.workers, remote)
	log.Printf("removed worker %s", remote)
}

func (r *WorkerRegistry) Get(remote string) (net.Conn, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	conn, ok := r.workers[remote]
	return conn, ok
}

func (r *WorkerRegistry) List() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.workers))
	for k := range r.workers {
		out = append(out, k)
	}
	return out
}

// GetWorkers returns the list of currently registered workers.
// It is exported for other packages (e.g., API) to query worker status.
func GetWorkers() []string {
	if registry == nil {
		return []string{}
	}
	return registry.List()
}

func ServerWorkers() {
	WP, ok := os.LookupEnv("WORKER_PORT")
	if !ok {
		log.Fatal("WORKER_PORT not set")
	}

	l, err := net.Listen("tcp", ":"+WP)
	if err != nil {
		log.Printf("listen error: %v", err)
	}

	defer l.Close()
	log.Printf("orchestrator listening on :%s", WP)

	registry = NewWorkerRegistry()
	responses = make(chan tools.JobWorker, 100)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go handleWorker(conn, registry, responses)
	}
}

func handleWorker(conn net.Conn, registry *WorkerRegistry, responses chan<- tools.JobWorker) {
	defer conn.Close()

	remote := conn.RemoteAddr().String()
	log.Printf("new connection from %s", remote)

	registry.Add(remote, conn)
	defer registry.Remove(remote)

	decoder := json.NewDecoder(bufio.NewReader(conn))
	for {
		var jobResp tools.JobWorker
		if err := decoder.Decode(&jobResp); err != nil {
			log.Printf("worker %s disconnected: %v", remote, err)
			return
		}
		responses <- jobResp
	}
}

func DispatchJobs(registry *WorkerRegistry, chunks []tools.SimilarityVector, responses <-chan tools.JobWorker) ([]tools.Similarity, error) {
	workers := registry.List()
	if len(workers) == 0 {
		return nil, fmt.Errorf("no workers registered")
	}

	expected := len(chunks)

	for i, chunk := range chunks {
		b, err := json.Marshal(chunk)
		if err != nil {
			return nil, err
		}

		job := tools.JobWorker{
			WorkerID: fmt.Sprintf("worker-%d", i),
			JobType:  "compute_similarity",
			Data:     string(b),
		}

		remote := workers[i%len(workers)]
		conn, ok := registry.Get(remote)
		if !ok {
			continue
		}

		encoder := json.NewEncoder(conn)
		if err := encoder.Encode(job); err != nil {
			log.Printf("failed to send job to %s: %v", remote, err)
			registry.Remove(remote)
			continue
		}

		log.Printf("dispatched chunk to %d to %s", i, remote)
	}

	merged := make([]tools.Similarity, 0)
	received := 0

	for received < expected {
		resp := <-responses

		if resp.JobType != "compute_similarity" {
			continue
		}

		var sims []tools.Similarity
		if err := json.Unmarshal([]byte(resp.Data), &sims); err != nil {
			log.Printf("bad response from %s: %v", resp.WorkerID, err)
			continue
		}

		merged = append(merged, sims...)
		received++
	}

	return merged, nil
}

func ComputeSimilarUsers(userID int, chunks int, userVec map[int]tools.UserVector) ([]tools.Similarity, error) {
	exists, err := database.UserExists(userID)
	if !exists {
		return nil, err
	}

	targetVec := userVec[userID]
	userVecSplit := recommendersystem.UserVectorSplitter(targetVec, userVec, chunks)

	mergedSimilarities, err := DispatchJobs(registry, userVecSplit, responses)
	if err != nil {
		log.Println("Error dispatching jobs:", err)
		return nil, err
	}

	sort.Slice(mergedSimilarities, func(i, j int) bool {
		return mergedSimilarities[i].Similarity > mergedSimilarities[j].Similarity
	})

	return mergedSimilarities, nil
}
