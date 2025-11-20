package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"pc4/database"
	"time"
)

type ClusterMessage struct {
	Target    int          `json:"target"`
	Neighbors []Similarity `json:"neighbors"`
}

type Similarity struct {
	UserID     int     `json:"userId"`
	Similarity float64 `json:"similarity"`
}

func StartTCPServer() {
	ln, err := net.Listen("tcp", ":9000")
	if err != nil {
		panic(err)
	}

	fmt.Println("TCP 9000")

	for {
		conn, _ := ln.Accept()

		go handleConnection(conn)
	}
}

func saveRecommendationToMongo(target int, neighbors []Similarity) error {
	collection := database.ConnectMongo(database.Recommendations)

	doc := map[string]interface{}{
		"target":    target,
		"neighbors": neighbors,
		"createdAt": time.Now(),
	}

	_, err := collection.InsertOne(context.Background(), doc)
	return err
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	data, _ := io.ReadAll(conn)

	fmt.Println("datos recibidos")
	fmt.Println(string(data))

	var msg ClusterMessage
	json.Unmarshal(data, &msg)

	fmt.Printf("Usuario objetivo: %d\n", msg.Target)
	fmt.Println("vecinos más similares:")

	for _, n := range msg.Neighbors {
		fmt.Printf(" → User %d (sim=%.4f)\n", n.UserID, n.Similarity)
	}

	if err := saveRecommendationToMongo(msg.Target, msg.Neighbors); err != nil {
		fmt.Println("❌ Error guardando en Mongo:", err)
		return
	}

	fmt.Println("Terminado")
}
