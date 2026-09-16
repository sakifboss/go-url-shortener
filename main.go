package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// URL defines the data we need to store for each shortened URL.
type URL struct {
	ShortCode   string `json:"short_code"`
	OriginalURL string `json:"original_url"`
}

// CreateURLRequest defines the JSON payload accepted by the create-URL API.
type CreateURLRequest struct {
	URL string `json:"url"`
}

// We use an in-memory map for the MVP so URL creation and lookup
// can be implemented without introducing a database at this stage.
var urlStore = make(map[string]URL)

// HTTP requests are handled concurrently, so the mutex protects
// the shared URL store from unsafe concurrent access.
var storeMu sync.RWMutex

// Health endpoint: added to provide a simple way to verify
// that the GoShort HTTP server is running correctly.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

// Hello endpoint: kept as a simple HTTP handler while learning
// request routing and method handling in Go's net/http package.
func helloHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Hello, GoShort!")
}

// Create-URL endpoint: accepts an original URL, validates it,
// generates a short code, stores the mapping, and returns the result.
func createURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var request CreateURLRequest

	// Decode the client's JSON body so the API can work with
	// the submitted original URL as a Go value.
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate the submitted URL before storing it so invalid
	// destination URLs are not accepted by the service.
	parsedURL, err := url.ParseRequestURI(request.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	// Protect the shared store while generating and saving
	// the short-code mapping because multiple requests may arrive at once.
	storeMu.Lock()

	shortCode := fmt.Sprintf("%d", len(urlStore)+1)

	createdURL := URL{
		ShortCode:   shortCode,
		OriginalURL: request.URL,
	}

	urlStore[shortCode] = createdURL

	storeMu.Unlock()

	// Return the newly created URL as JSON with HTTP 201,
	// indicating that a new resource was successfully created.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(createdURL); err != nil {
		fmt.Println("Response encoding error:", err)
	}
}

// Redirect endpoint: accepts a short code, looks up its original URL,
// and redirects the client to the stored destination.
func redirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the short code from the request path.
	shortCode := strings.TrimPrefix(r.URL.Path, "/")

	if shortCode == "" {
		http.NotFound(w, r)
		return
	}

	// Read the stored mapping safely because the URL store
	// is shared between concurrent HTTP requests.
	storeMu.RLock()
	storedURL, exists := urlStore[shortCode]
	storeMu.RUnlock()

	// Return 404 when the requested short code does not exist.
	if !exists {
		http.NotFound(w, r)
		return
	}

	// Redirect the client to the original destination URL.
	http.Redirect(w, r, storedURL.OriginalURL, http.StatusFound)
}

func main() {
	// Create a dedicated router so each URL path can be connected
	// to its corresponding handler.
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/hello", helloHandler)

	// Register the URL creation API required for the MVP.
	mux.HandleFunc("/api/v1/urls", createURLHandler)

	// Register "/" as the catch-all route so paths such as /1
	// can be interpreted as short-code redirect requests.
	mux.HandleFunc("/", redirectHandler)

	// Create the HTTP server and attach our custom router.
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	fmt.Println("GoShort server starting on http://localhost:8080")

	// Start listening for HTTP requests.
	err := server.ListenAndServe()

	if err != nil && err != http.ErrServerClosed {
		fmt.Println("Server error:", err)
	}
}
