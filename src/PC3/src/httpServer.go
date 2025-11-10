package main

import (
	"fmt"
	"net/http"
)

func api1(w http.ResponseWriter, r *http.Request) {
	fmt.Println("API 1 was requested")
}
