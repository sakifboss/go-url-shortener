package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"goshort/auth"
	"goshort/database"
	"goshort/handler"
	"goshort/repository"
	"goshort/security"
	"goshort/service"
)

func main() {
	// ------------------------------------------------------------
	// Database
	// ------------------------------------------------------------

	// Create the PostgreSQL connection pool.
	db, err := database.NewPostgres()
	if err != nil {
		fmt.Println("Database connection error:", err)
		return
	}
	defer db.Close()

	// ------------------------------------------------------------
	// Configuration
	// ------------------------------------------------------------

	// JWT secret is required to sign and validate access tokens.
	jwtSecret := strings.TrimSpace(os.Getenv("GOSHORT_JWT_SECRET"))

	if jwtSecret == "" {
		fmt.Println("Configuration error: GOSHORT_JWT_SECRET is not set")
		return
	}

	// ------------------------------------------------------------
	// Repository layer
	// ------------------------------------------------------------

	urlRepository := repository.NewPostgresURLRepository(db)
	userRepository := repository.NewPostgresUserRepository(db)
	refreshTokenRepository := repository.NewPostgresRefreshTokenRepository(db)

	// ------------------------------------------------------------
	// Service layer
	// ------------------------------------------------------------

	urlService := service.NewURLService(urlRepository)

	authService, err := auth.NewAuthService(
		userRepository,
		refreshTokenRepository,
		jwtSecret,
	)
	if err != nil {
		fmt.Println("Authentication service error:", err)
		return
	}

	// ------------------------------------------------------------
	// Handler layer
	// ------------------------------------------------------------

	urlHandler := handler.NewURLHandler(urlService)
	authHandler := handler.NewAuthHandler(authService)

	// ------------------------------------------------------------
	// Authentication middleware
	// ------------------------------------------------------------

	authMiddleware, err := auth.NewAuthMiddleware(jwtSecret)
	if err != nil {
		fmt.Println("Authentication middleware error:", err)
		return
	}

	// ------------------------------------------------------------
	// Rate limiters
	// ------------------------------------------------------------

	// Global application rate limiter.
	// Allows up to 60 requests per client IP per minute.
	rateLimiter := security.NewRateLimiter(
		60,
		time.Minute,
	)

	// Login-specific rate limiter.
	// Allows up to 10 login requests per client IP per minute.
	// This adds protection against brute-force login attempts.
	loginRateLimiter := security.NewRateLimiter(
		10,
		time.Minute,
	)

	// ------------------------------------------------------------
	// HTTP routes
	// ------------------------------------------------------------

	mux := http.NewServeMux()

	// ------------------------------------------------------------
	// Public health/development endpoints
	// ------------------------------------------------------------

	// GET /health
	mux.HandleFunc("/health", healthHandler)

	// GET /hello
	mux.HandleFunc("/hello", helloHandler)

	// ------------------------------------------------------------
	// Authentication endpoints
	// ------------------------------------------------------------

	// POST /api/v1/auth/register
	mux.HandleFunc(
		"/api/v1/auth/register",
		authHandler.Register,
	)

	// POST /api/v1/auth/login
	//
	// Login has its own stricter rate limiter.
	mux.Handle(
		"/api/v1/auth/login",
		loginRateLimiter.Middleware(
			http.HandlerFunc(authHandler.Login),
		),
	)

	// POST /api/v1/auth/refresh
	mux.HandleFunc(
		"/api/v1/auth/refresh",
		authHandler.Refresh,
	)

	// POST /api/v1/auth/logout
	mux.HandleFunc(
		"/api/v1/auth/logout",
		authHandler.Logout,
	)

	// ------------------------------------------------------------
	// Protected URL endpoints
	// ------------------------------------------------------------

	// POST /api/v1/urls
	//
	// Requires a valid JWT access token.
	mux.Handle(
		"/api/v1/urls",
		authMiddleware.RequireAuth(
			http.HandlerFunc(urlHandler.CreateURL),
		),
	)

	// GET    /api/v1/urls/:id
	// PATCH  /api/v1/urls/:id
	// DELETE /api/v1/urls/:id
	//
	// Requires authentication.
	// The handlers additionally verify URL ownership.
	mux.Handle(
		"/api/v1/urls/",
		authMiddleware.RequireAuth(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					urlHandler.GetURL(w, r)

				case http.MethodPatch:
					urlHandler.UpdateURL(w, r)

				case http.MethodDelete:
					urlHandler.DeleteURL(w, r)

				default:
					http.Error(
						w,
						"Method Not Allowed",
						http.StatusMethodNotAllowed,
					)
				}
			}),
		),
	)

	// ------------------------------------------------------------
	// Public redirect endpoint
	// ------------------------------------------------------------

	// GET /:shortCode
	//
	// Redirects do not require authentication.
	mux.HandleFunc("/", urlHandler.Redirect)

	// ------------------------------------------------------------
	// Middleware chain
	// ------------------------------------------------------------

	// Request flow:
	//
	// Client
	//   ↓
	// Security middleware
	//   ↓
	// Global rate limiter
	//   ↓
	// HTTP router
	//   ↓
	// Authentication middleware (protected routes only)
	//   ↓
	// Handler
	//   ↓
	// Service
	//   ↓
	// Repository
	//   ↓
	// PostgreSQL
	//
	// Security middleware provides request-size limits and
	// security-related HTTP response headers.
	//
	// Global rate limiter limits excessive requests from a
	// client IP.
	handlerChain := security.Middleware(
		rateLimiter.Middleware(mux),
	)

	// ------------------------------------------------------------
	// HTTP server
	// ------------------------------------------------------------

	server := &http.Server{
		Addr:    ":8080",
		Handler: handlerChain,
	}

	fmt.Println("GoShort server starting on http://localhost:8080")

	if err := server.ListenAndServe(); err != nil &&
		err != http.ErrServerClosed {
		fmt.Println("Server error:", err)
	}
}

// ------------------------------------------------------------
// Health check
// ------------------------------------------------------------

// healthHandler verifies that the HTTP server is running.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

// ------------------------------------------------------------
// Hello endpoint
// ------------------------------------------------------------

// helloHandler is a simple development endpoint.
func helloHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Hello, GoShort!")
}
