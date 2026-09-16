package main

import (
	"fmt"
	"net/http" // Provides HTTP server and request/response functionality.
)

func main() {
	// Create an HTTP server that listens on port 3000.
	server := &http.Server{
		Addr: ":3000",
	}

	fmt.Println("GoShort server starting on http://localhost:3000")

	// Start the HTTP server and wait for incoming requests.
	err := server.ListenAndServe()

	// Handle the error returned when the server stops.
	if err != nil && err != http.ErrServerClosed {
		fmt.Println("Server error:", err)
	}
}
