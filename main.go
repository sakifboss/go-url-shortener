package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
)

// URL represents a shortened URL stored by the application.
type URL struct {
	ShortCode   string `json:"short_code"`
	OriginalURL string `json:"original_url"`
}

// CreateURLRequest represents the JSON payload for creating a short URL.
type CreateURLRequest struct {
	URL string `json:"url"`
}

var (
	urlStore = make(map[string]URL)
	storeMu  sync.RWMutex
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Hello, GoShort!")
}

func createURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var request CreateURLRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	parsedURL, err := url.ParseRequestURI(request.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	storeMu.Lock()

	shortCode := fmt.Sprintf("%d", len(urlStore)+1)

	createdURL := URL{
		ShortCode:   shortCode,
		OriginalURL: request.URL,
	}

	urlStore[shortCode] = createdURL

	storeMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(createdURL); err != nil {
		fmt.Println("Response encoding error:", err)
	}
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/hello", helloHandler)
	mux.HandleFunc("/api/v1/urls", createURLHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	fmt.Println("GoShort server starting on http://localhost:8080")

	err := server.ListenAndServe()

	if err != nil && err != http.ErrServerClosed {
		fmt.Println("Server error:", err)
	}
}
