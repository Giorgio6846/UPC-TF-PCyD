package api

import (
	"log"
	"net"
	"os"
	"sync"
)

type Job struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Data string `json:"data"`
}

type WorkerRegistry struct {
	mu      sync.Mutex
	workers map[string]net.Conn
}

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

	registry := NewWorkerRegistry()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go handleWorker(conn, registry)
	}
}

func handleWorker(conn net.Conn, registry *WorkerRegistry) {
	defer conn.Close()

	remote := conn.RemoteAddr().String()
	log.Printf("new connection from %s", remote)

	registry.Add(remote, conn)
	defer registry.Remove(remote)

	//decoder := json.NewDecoder(bufio.NewReader(conn))
	//for {
	//	var msg map[string]any
	//	if err := decoder.Decode(&msg); err != nil {
	//		log.Printf("worker %s disconnected: %v", remote, err)
	//	}
	//}
}
