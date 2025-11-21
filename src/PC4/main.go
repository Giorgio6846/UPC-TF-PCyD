package main

import (
	"log"
	"os"
	"pc4/api"
	"pc4/database"
	"sync"
	"time"
)

func main() {
	workerType, ok := os.LookupEnv("WORKER_TYPE")
	if !ok {
		log.Fatal("WORKER_TYPE not set")
	}

	switch workerType {
	case "orchestrator":
		database.AppendDataToDB()

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			api.ServerWorkers()
		}()

		go func() {
			defer wg.Done()
			api.SetupAPI()
		}()

		wg.Wait()

	case "worker":
		time.Sleep(time.Second * 5)
		api.StartWorker()
	}
}
