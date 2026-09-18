package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"goshort/auth"
)

type AuthHandler struct {
	authService *auth.AuthService
}

func NewAuthHandler(
	authService *auth.AuthService,
) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type authResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

func (h *AuthHandler) Register(
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

	var request registerRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(
			w,
			"Invalid JSON",
			http.StatusBadRequest,
		)
		return
	}

	user, err := h.authService.Register(
		r.Context(),
		request.Email,
		request.Password,
	)
	if err != nil {
		if errors.Is(err, auth.ErrEmailAlreadyExists) {
			http.Error(
				w,
				"Email already registered",
				http.StatusConflict,
			)
			return
		}

		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
		},
	)
}

func (h *AuthHandler) Login(
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

	var request loginRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(
			w,
			"Invalid JSON",
			http.StatusBadRequest,
		)
		return
	}

	accessToken, refreshToken, _, err := h.authService.Login(
		r.Context(),
		request.Email,
		request.Password,
	)
	if err != nil {
		http.Error(
			w,
			"Invalid credentials",
			http.StatusUnauthorized,
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		authResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
		},
	)
}

func (h *AuthHandler) Refresh(
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

	var request refreshRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(
			w,
			"Invalid JSON",
			http.StatusBadRequest,
		)
		return
	}

	request.RefreshToken = strings.TrimSpace(
		request.RefreshToken,
	)

	if request.RefreshToken == "" {
		http.Error(
			w,
			"refresh_token is required",
			http.StatusBadRequest,
		)
		return
	}

	accessToken, refreshToken, err := h.authService.Refresh(
		r.Context(),
		request.RefreshToken,
	)
	if err != nil {
		http.Error(
			w,
			"Invalid refresh token",
			http.StatusUnauthorized,
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		authResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
		},
	)
}

func (h *AuthHandler) Logout(
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

	var request logoutRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(
			w,
			"Invalid JSON",
			http.StatusBadRequest,
		)
		return
	}

	request.RefreshToken = strings.TrimSpace(
		request.RefreshToken,
	)

	if request.RefreshToken == "" {
		http.Error(
			w,
			"refresh_token is required",
			http.StatusBadRequest,
		)
		return
	}

	if err := h.authService.Logout(
		r.Context(),
		request.RefreshToken,
	); err != nil {
		http.Error(
			w,
			"Invalid refresh token",
			http.StatusUnauthorized,
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
