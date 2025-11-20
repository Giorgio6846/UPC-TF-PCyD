package main

import (
	"fmt"
	"log"
	"os"
	"pc4/api"
)

func main() {
	workerType, ok := os.LookupEnv("WORKER_TYPE")
	if !ok {
		log.Fatal("WORKER_TYPE not set")
	}

	if workerType == "orchestrator" {
		go api.StartTCPServer()
		select {}
	} else if workerType == "worker" {
		test := 1 + 1
		fmt.Println(test)
	}

}
