package ds

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	recommendersystem "pc4/RecommenderSystem"
	"pc4/tools"
	"strings"
	"time"
)

func StartWorker() {
	addr, ok := os.LookupEnv("ORCHESTRATOR_IP")
	if !ok {
		log.Fatal("ORCHESTRATOR_IP not set")
	}

	for {
		log.Println("Orchestrator IP", addr)

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			log.Fatalf("dial error %v (retrying...)", err)
			time.Sleep(2 * time.Second)
			continue
		}

		log.Printf("connected to orchestrator at %s", addr)
		if err := waitWork(conn); err != nil {
			log.Printf("connection ended %v", err)
		}

		_ = conn.Close()
		log.Println("reconnecting")
		time.Sleep(1 * time.Second)
	}

}

func waitWork(conn net.Conn) error {
	reader := bufio.NewReader(conn)
	decode := json.NewDecoder(reader)

	for {
		var job tools.JobWorker
		if err := decode.Decode(&job); err != nil {
			if err == io.EOF {
				return err
			}
			log.Printf("decode error: %v", err)
			continue
		}
		//log.Printf("received job: %+v", job)
		go doWork(conn, job)
	}
}

func doWork(conn net.Conn, job tools.JobWorker) {
	switch job.JobType {
	case "compute_similarity":
		similarityResponse := cosineSimilarityWorker(job.Data)
		sr, err := json.Marshal(similarityResponse)
		if err != nil {
			log.Println("failed to encode:", err)
			return
		}

		jobResponse := tools.JobWorker{
			WorkerID: job.WorkerID,
			JobType:  job.JobType,
			Data:     string(sr),
		}

		encoder := json.NewEncoder(conn)
		if err := encoder.Encode(jobResponse); err != nil {
			log.Println("failed to send result:", err)
		}

	default:
		log.Fatal("Type not implemented")
	}

}

func cosineSimilarityWorker(jobData string) []tools.Similarity {
	var chunk tools.SimilarityVector

	reader := strings.NewReader(jobData)
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&chunk); err != nil {
		log.Println("failed to decode chunk:", err)
	}

	similarity := recommendersystem.CosineSimilarity(chunk.TargetVector, chunk.UsersVector, 2)
	return similarity
}
