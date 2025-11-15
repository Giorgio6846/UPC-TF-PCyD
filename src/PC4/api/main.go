package main

func main() {
	go startTCPServer()
	select {}
}
