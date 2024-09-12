package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	// Create a new mux router
	mux := http.NewServeMux()

	// Serve static files from the "static" directory
	mux.Handle("/", http.FileServer(http.Dir("./static")))

	// Define the port from the environment variable or default to 8080
	port := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	// Start the server
	log.Printf("Server started on :%s\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %s", err)
	}
}
