package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
)

type ClusterMessage struct {
	Target    int          `json:"target"`
	Neighbors []Similarity `json:"neighbors"`
}

type Similarity struct {
	UserID     int     `json:"userId"`
	Similarity float64 `json:"similarity"`
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

	fmt.Println("Terminado")
}
