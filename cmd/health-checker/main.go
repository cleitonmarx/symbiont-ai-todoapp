package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	resp, err := http.Get(os.Args[1])
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		os.Exit(1)
	}
	if resp.StatusCode == 200 {
		os.Exit(0)
	}
	os.Exit(1)
}
