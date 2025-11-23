package main

import (
	"context"
	"log"
	"os"
	ds "pc4/DistributedSystem"
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
		if err := database.InitMongo(); err != nil {
			log.Fatal(err)
		}
		defer database.CloseMongo(context.Background())

		fillDBCond, ok := os.LookupEnv("FILL_DB")
		if !ok {
			log.Fatal("FILL_DB not set")
		}
		if fillDBCond == "True" {
			database.AppendDataToDB()
		}

		if err := database.InitRedis(); err != nil {
			log.Fatal(err)
		}

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			ds.ServerWorkers()
		}()

		go func() {
			defer wg.Done()
			api.SetupAPI()
		}()

		wg.Wait()

	case "worker":
		time.Sleep(time.Second * 5)
		ds.StartWorker()
	}
}
