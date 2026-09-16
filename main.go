package main

import (
	"fmt"
	"net/http" // Provides HTTP server and request/response functionality.
)

// healthHandler handles requests sent to the /health endpoint.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OK")
}

// helloHandler handles requests sent to the /hello endpoint.
func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello, GoShort!")
}

func main() {
	// Create a dedicated router for the application.
	mux := http.NewServeMux()

	// Register application routes.
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/hello", helloHandler)

	// Create an HTTP server and attach the application router.
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	fmt.Println("GoShort server starting on http://localhost:8080")

	// Start the HTTP server and wait for incoming requests.
	err := server.ListenAndServe()

	// Handle unexpected server errors.
	if err != nil && err != http.ErrServerClosed {
		fmt.Println("Server error:", err)
	}
}
