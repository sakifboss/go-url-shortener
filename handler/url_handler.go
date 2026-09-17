package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"goshort/service"
)

// URLHandler handles HTTP requests related to URL shortening.
type URLHandler struct {
	service *service.URLService
}

// CreateURLRequest represents the JSON body accepted by the create endpoint.
type CreateURLRequest struct {
	URL string `json:"url"`
}

// CreateURLResponse represents the public API response for a created URL.
type CreateURLResponse struct {
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"`
}

// NewURLHandler creates an HTTP handler using the provided URL service.
func NewURLHandler(service *service.URLService) *URLHandler {
	return &URLHandler{
		service: service,
	}
}

// CreateURL handles POST /api/v1/urls.
//
// The handler is responsible for HTTP concerns only: reading JSON,
// calling the service, and constructing the HTTP response.
func (h *URLHandler) CreateURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var request CreateURLRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	createdURL, err := h.service.CreateShortURL(request.URL)
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	response := CreateURLResponse{
		ShortCode: createdURL.ShortCode,
		ShortURL:  "http://localhost:8080/" + createdURL.ShortCode,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(response)
}

// Redirect handles GET /:shortCode.
//
// The handler extracts the short code, asks the service for the
// destination, and performs the HTTP redirect.
func (h *URLHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	shortCode := strings.TrimPrefix(r.URL.Path, "/")

	if shortCode == "" {
		http.NotFound(w, r)
		return
	}

	storedURL, exists := h.service.GetOriginalURL(shortCode)
	if !exists {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, storedURL.OriginalURL, http.StatusFound)
}
