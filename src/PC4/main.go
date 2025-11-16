package main

import (
	"pc4/api"
)

func main() {
	go api.StartTCPServer()
	select {}
}
