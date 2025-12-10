package main

import (
	"log"
	"net/http"
	"os"
)

// main provides the HTTP server entrypoint for Cloud Run.
func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", Packer)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("server starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
