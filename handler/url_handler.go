package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"goshort/auth"
	"goshort/model"
	"goshort/repository"
	"goshort/service"
)

// URLHandler handles HTTP requests related to URL operations.
type URLHandler struct {
	service *service.URLService
}

// CreateURLRequest represents the JSON body accepted by the create endpoint.
type CreateURLRequest struct {
	URL string `json:"url"`
}

// CreateURLResponse represents the public API response for a created URL.
type CreateURLResponse struct {
	ID        int64  `json:"id"`
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"`
}

// UpdateURLRequest represents the fields that can be updated.
type UpdateURLRequest struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at,omitempty"`
	IsActive  *bool  `json:"is_active,omitempty"`
}

// NewURLHandler creates an HTTP handler using the provided URL service.
func NewURLHandler(service *service.URLService) *URLHandler {
	return &URLHandler{
		service: service,
	}
}

// CreateURL handles POST /api/v1/urls.
func (h *URLHandler) CreateURL(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		http.Error(
			w,
			"Authentication required",
			http.StatusUnauthorized,
		)
		return
	}

	var request CreateURLRequest

	if err := decodeJSONBody(w, r, &request); err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	createdURL, err := h.service.CreateShortURL(
		r.Context(),
		request.URL,
		user.UserID,
	)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	response := CreateURLResponse{
		ID:        createdURL.ID,
		ShortCode: createdURL.ShortCode,
		ShortURL:  "http://localhost:8080/" + createdURL.ShortCode,
	}

	writeJSON(
		w,
		http.StatusCreated,
		response,
	)
}

// GetURL handles GET /api/v1/urls/:id.
func (h *URLHandler) GetURL(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	id, err := parseURLID(r.URL.Path)
	if err != nil {
		http.Error(
			w,
			"Invalid URL ID",
			http.StatusBadRequest,
		)
		return
	}

	storedURL, err := h.service.GetURLByID(
		r.Context(),
		id,
	)
	if err != nil {
		if repository.IsNotFound(err) {
			http.NotFound(w, r)
			return
		}

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	if !requireURLOwnership(
		w,
		r,
		storedURL,
	) {
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		storedURL,
	)
}

// UpdateURL handles PATCH /api/v1/urls/:id.
func (h *URLHandler) UpdateURL(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPatch {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	id, err := parseURLID(r.URL.Path)
	if err != nil {
		http.Error(
			w,
			"Invalid URL ID",
			http.StatusBadRequest,
		)
		return
	}

	existingURL, err := h.service.GetURLByID(
		r.Context(),
		id,
	)
	if err != nil {
		if repository.IsNotFound(err) {
			http.NotFound(w, r)
			return
		}

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	if !requireURLOwnership(
		w,
		r,
		existingURL,
	) {
		return
	}

	var request UpdateURLRequest

	if err := decodeJSONBody(
		w,
		r,
		&request,
	); err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	if request.URL != "" {
		existingURL.OriginalURL = request.URL
	}

	if request.IsActive != nil {
		existingURL.IsActive = *request.IsActive
	}

	// Expiration parsing will be added when the API contract
	// explicitly defines the accepted timestamp format.
	if request.ExpiresAt != "" {
		http.Error(
			w,
			"expires_at format is not supported yet",
			http.StatusBadRequest,
		)
		return
	}

	updatedURL, err := h.service.UpdateURL(
		r.Context(),
		existingURL,
	)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		updatedURL,
	)
}

// DeleteURL handles DELETE /api/v1/urls/:id.
func (h *URLHandler) DeleteURL(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodDelete {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	id, err := parseURLID(r.URL.Path)
	if err != nil {
		http.Error(
			w,
			"Invalid URL ID",
			http.StatusBadRequest,
		)
		return
	}

	storedURL, err := h.service.GetURLByID(
		r.Context(),
		id,
	)
	if err != nil {
		if repository.IsNotFound(err) {
			http.NotFound(w, r)
			return
		}

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	if !requireURLOwnership(
		w,
		r,
		storedURL,
	) {
		return
	}

	if err := h.service.DeleteURL(
		r.Context(),
		id,
	); err != nil {
		if repository.IsNotFound(err) {
			http.NotFound(w, r)
			return
		}

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Redirect handles GET /:shortCode.
func (h *URLHandler) Redirect(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	shortCode := strings.TrimPrefix(
		r.URL.Path,
		"/",
	)

	if shortCode == "" {
		http.NotFound(w, r)
		return
	}

	storedURL, err := h.service.GetOriginalURL(
		r.Context(),
		shortCode,
	)
	if err != nil {
		if repository.IsNotFound(err) {
			http.NotFound(w, r)
			return
		}

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	http.Redirect(
		w,
		r,
		storedURL.OriginalURL,
		http.StatusFound,
	)
}

// requireURLOwnership verifies that the authenticated user owns the URL.
func requireURLOwnership(
	w http.ResponseWriter,
	r *http.Request,
	storedURL model.URL,
) bool {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		http.Error(
			w,
			"Authentication required",
			http.StatusUnauthorized,
		)
		return false
	}

	if storedURL.UserID == nil ||
		*storedURL.UserID != user.UserID {
		http.Error(
			w,
			"Forbidden",
			http.StatusForbidden,
		)
		return false
	}

	return true
}

// parseURLID extracts the numeric URL ID from /api/v1/urls/:id.
func parseURLID(path string) (int64, error) {
	idText := strings.TrimPrefix(
		path,
		"/api/v1/urls/",
	)

	id, err := strconv.ParseInt(
		idText,
		10,
		64,
	)
	if err != nil || id <= 0 {
		return 0, strconv.ErrSyntax
	}

	return id, nil
}

// writeJSON writes a JSON response with the requested HTTP status.
func writeJSON(
	w http.ResponseWriter,
	status int,
	data interface{},
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
