package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Register routes
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/shorten", shortenHandler)
	http.HandleFunc("/redirect/", redirectHandler)

	// Start server
	port := ":8080"
	fmt.Printf("Server starting on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `
	<!DOCTYPE html>
	<html>
	<head>
		<title>URL Shortener</title>
	</head>
	<body>
		<h1>URL Shortener</h1>
		<form action="/shorten" method="POST">
			<input type="text" name="url" placeholder="Enter URL" required>
			<button type="submit">Shorten</button>
		</form>
	</body>
	</html>
	`)
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	url := r.FormValue("url")
	if url == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Generate short code
	shortCode := generateShortCode()

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Short URL</title>
	</head>
	<body>
		<h1>Short URL Created</h1>
		<p>Original URL: %s</p>
		<p>Short URL: <a href="/redirect/%s">http://localhost:8080/redirect/%s</a></p>
		<a href="/">Create another</a>
	</body>
	</html>
	`, url, shortCode, shortCode)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	shortCode := r.URL.Path[len("/redirect/"):]

	// For now, just return a simple message
	fmt.Fprintf(w, "Redirect handler for code: %s", shortCode)
}

func generateShortCode() string {
	// Simple implementation - can be enhanced with database
	return fmt.Sprintf("%d", len("shortCode"))
}
