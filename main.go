package main

import (
	"fmt"
	"net/http" // Provides HTTP server and request/response functionality.
)

// healthHandler handles requests sent to the /health endpoint.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

func main() {
	// Register the /health route and connect it to healthHandler.
	http.HandleFunc("/health", healthHandler)

	// Create an HTTP server that listens on port 3000.
	server := &http.Server{
		Addr: ":8080",
	}

	fmt.Println("GoShort server starting on http://localhost:8080")

	// Start the HTTP server and wait for incoming requests.
	err := server.ListenAndServe()

	// Handle the error returned when the server stops.
	if err != nil && err != http.ErrServerClosed {
		fmt.Println("Server error:", err)
	}
}
