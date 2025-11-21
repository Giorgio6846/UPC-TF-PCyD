package api

import (
	"log"
	"net"
	"os"
)

type JobWorker struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Data string `json:"data"`
}

func StartWorker() {
	addr, ok := os.LookupEnv("ORCHESTRATOR_IP")
	if !ok {
		log.Fatal("ORCHESTRATOR_IP not set")
	}

	log.Println("Orchestrator IP", addr)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalf("dial error %v", err)
	}
	defer conn.Close()
	log.Printf("connected to orchestrator at %s", addr)
}
