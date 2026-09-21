package main

import (
	"context"
	"errors"
	"fmt"
	"goshort/worker"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"goshort/auth"
	"goshort/cache"
	"goshort/database"
	"goshort/handler"
	"goshort/repository"
	"goshort/security"
	"goshort/service"
	"goshort/web"
	"io/fs"
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

	clickEventRepository := repository.NewPostgresClickEventRepository(db)
	clickWorker := worker.NewClickWorkerPool(
		clickEventRepository,
		4,
		1000,
		5,
		100*time.Millisecond,
	)

	clickWorker.Start()

	// ------------------------------------------------------------
	// Service layer
	// ------------------------------------------------------------

	redisCache, err := cache.NewRedisCache()
	if err != nil {
		fmt.Println("Redis configuration error:", err)
		return
	}
	defer redisCache.Close()

	if err := redisCache.Ping(context.Background()); err != nil {
		fmt.Println("Redis connection error:", err)
		return
	}

	urlService := service.NewCachedURLService(
		urlRepository,
		redisCache,
	)
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

	idempotencyRepository :=
		repository.NewPostgresIdempotencyRepository(db)

	urlHandler := handler.NewURLHandler(
		urlService,
		clickWorker,
		idempotencyRepository,
		clickEventRepository,
	)
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

	dashboardFS, err := fs.Sub(web.Dashboard, "dashboard")
	if err != nil {
		fmt.Println("Dashboard setup error:", err)
		return
	}
	mux.Handle(
		"/dashboard/",
		http.StripPrefix(
			"/dashboard/",
			http.FileServer(http.FS(dashboardFS)),
		),
	)

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
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					urlHandler.ListURLs(w, r)
					return
				}
				urlHandler.CreateURL(w, r)
			}),
		),
	)

	// GET /api/v1/urls/:id/analytics
	mux.Handle(
		"/api/v1/urls/",
		authMiddleware.RequireAuth(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/analytics") {
					urlHandler.GetAnalytics(w, r)
					return
				}

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
			}),
		),
	)

	// GET    /api/v1/urls/:id
	// PATCH  /api/v1/urls/:id
	// DELETE /api/v1/urls/:id
	//
	// Requires authentication.
	// The handlers additionally verify URL ownership.

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
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "9000"
	}

	addr := ":" + port

	server := &http.Server{
		Addr:    addr,
		Handler: handlerChain,
	}

	fmt.Printf("GoShort server starting on %s\n", addr)

	go func() {
		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {

			fmt.Println("Server error:", err)
		}
	}()
	stop := make(chan os.Signal, 1)

	signal.Notify(
		stop,
		os.Interrupt,
		syscall.SIGTERM,
	)

	<-stop

	fmt.Println("Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		fmt.Println("Server shutdown error:", err)
	}

	clickWorker.Shutdown()

	fmt.Println("Click worker pool stopped")

	if err := redisCache.Close(); err != nil {
		fmt.Println("Redis close error:", err)
	}

	if err := db.Close(); err != nil {
		fmt.Println("Database close error:", err)
	}

	fmt.Println("GoShort shutdown complete")
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
