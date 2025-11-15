package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ClusterMessage struct {
	Target    int          `json:"target"`
	Neighbors []Similarity `json:"neighbors"`
}

type Similarity struct {
	UserID     int     `json:"userId"`
	Similarity float64 `json:"similarity"`
}

func connectMongo() *mongo.Collection {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    uri := "mongodb://hello:world@localhost:27017"
    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
    if err != nil {
        log.Fatal("Error conectando mongo API:", err)
    }

    return client.Database("movielens").Collection("recommendations")
}

func startTCPServer() {
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
    collection := connectMongo()

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
