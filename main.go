package main

import (
	"fmt"
	"net/http"

	"goshort/database"
	"goshort/handler"
	"goshort/repository"
	"goshort/service"
)

func main() {
	// Create the PostgreSQL connection pool.
	db, err := database.NewPostgres()
	if err != nil {
		fmt.Println("Database connection error:", err)
		return
	}
	defer db.Close()

	// Build the application layers:
	// PostgreSQL -> Repository -> Service -> HTTP Handler
	urlRepository := repository.NewPostgresURLRepository(db)
	urlService := service.NewURLService(urlRepository)
	urlHandler := handler.NewURLHandler(urlService)

	// Register HTTP routes.
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/hello", helloHandler)

	// URL creation endpoint.
	mux.HandleFunc("/api/v1/urls", urlHandler.CreateURL)

	// URL CRUD endpoints using /api/v1/urls/:id.
	mux.HandleFunc("/api/v1/urls/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			urlHandler.GetURL(w, r)
		case http.MethodPatch:
			urlHandler.UpdateURL(w, r)
		case http.MethodDelete:
			urlHandler.DeleteURL(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	// Redirect endpoint: GET /:shortCode
	mux.HandleFunc("/", urlHandler.Redirect)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	fmt.Println("GoShort server starting on http://localhost:8080")

	if err := server.ListenAndServe(); err != nil &&
		err != http.ErrServerClosed {
		fmt.Println("Server error:", err)
	}
}

// healthHandler verifies that the HTTP server is running.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

// helloHandler is a simple development endpoint.
func helloHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Hello, GoShort!")
}
