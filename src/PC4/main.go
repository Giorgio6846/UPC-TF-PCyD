package main

import (
	"log"
	"os"
	"pc4/orchestrator"
	"pc4/worker"
	"time"
)

func main() {
	workerType, ok := os.LookupEnv("WORKER_TYPE")
	if !ok {
		log.Fatal("WORKER_TYPE not set")
	}

	switch workerType {
	case "orchestrator":
		orchestrator.StartOrchestrator()
	case "worker":
		time.Sleep(time.Second * 5)
		worker.StartWorker()
	}
}
