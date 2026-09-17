package main

import (
	"fmt"
	"net/http"

	"goshort/handler"
	"goshort/repository"
	"goshort/service"
)

func main() {
	// Build the application from the storage layer upward so
	// each layer receives only the dependency it actually needs.
	urlRepository := repository.NewMemoryURLRepository()
	urlService := service.NewURLService(urlRepository)
	urlHandler := handler.NewURLHandler(urlService)

	// Keep routing in the HTTP entry point while the actual
	// application logic remains inside the handler and service layers.
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/hello", helloHandler)
	mux.HandleFunc("/api/v1/urls", urlHandler.CreateURL)
	mux.HandleFunc("/", urlHandler.Redirect)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	fmt.Println("GoShort server starting on http://localhost:8080")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Println("Server error:", err)
	}
}

// healthHandler provides a lightweight endpoint for checking
// whether the HTTP server is running.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

// helloHandler remains as a simple endpoint for basic HTTP testing.
func helloHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Hello, GoShort!")
}
