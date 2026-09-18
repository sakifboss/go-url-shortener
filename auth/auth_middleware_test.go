package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testJWTSecret = "test-secret-for-goshort-auth-tests"

func TestAuthMiddlewareRequireAuth(t *testing.T) {
	middleware, err := NewAuthMiddleware(testJWTSecret)
	if err != nil {
		t.Fatalf("create middleware: %v", err)
	}

	tests := []struct {
		name           string
		token          string
		expectedStatus int
	}{
		{
			name:           "missing authorization header",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "malformed authorization header",
			token:          "Basic abc123",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid token",
			token:          "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "valid token",
			token:          createTestAccessToken(t, testJWTSecret),
			expectedStatus: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			next := http.HandlerFunc(func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				user, ok := CurrentUser(r.Context())

				if !ok {
					t.Error("expected authenticated user")
					return
				}

				if user.UserID != 1 {
					t.Errorf(
						"user ID = %d, want 1",
						user.UserID,
					)
				}

				w.WriteHeader(http.StatusOK)
			})

			handler := middleware.RequireAuth(next)

			request := httptest.NewRequest(
				http.MethodGet,
				"/protected",
				nil,
			)

			if test.token != "" {
				request.Header.Set(
					"Authorization",
					test.token,
				)
			}

			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.expectedStatus {
				t.Errorf(
					"status = %d, want %d",
					response.Code,
					test.expectedStatus,
				)
			}
		})
	}
}

func TestAuthMiddlewareRejectsWrongIssuer(t *testing.T) {
	middleware, err := NewAuthMiddleware(testJWTSecret)
	if err != nil {
		t.Fatalf("create middleware: %v", err)
	}

	now := time.Now()

	claims := jwt.MapClaims{
		"sub":   "1",
		"iss":   "wrong-issuer",
		"aud":   "goshort-api",
		"email": "test@goshort.com",
		"role":  "user",
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
		"exp":   now.Add(15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	rawToken, err := token.SignedString(
		[]byte(testJWTSecret),
	)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	handler := middleware.RequireAuth(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer "+rawToken,
	)

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Errorf(
			"status = %d, want %d",
			response.Code,
			http.StatusUnauthorized,
		)
	}
}

func TestAuthMiddlewareRejectsExpiredToken(t *testing.T) {
	middleware, err := NewAuthMiddleware(testJWTSecret)
	if err != nil {
		t.Fatalf("create middleware: %v", err)
	}

	now := time.Now()

	claims := jwt.MapClaims{
		"sub":   "1",
		"iss":   "goshort",
		"aud":   "goshort-api",
		"email": "test@goshort.com",
		"role":  "user",
		"iat":   now.Add(-30 * time.Minute).Unix(),
		"nbf":   now.Add(-30 * time.Minute).Unix(),
		"exp":   now.Add(-15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	rawToken, err := token.SignedString(
		[]byte(testJWTSecret),
	)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	handler := middleware.RequireAuth(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)

	request.Header.Set(
		"Authorization",
		"Bearer "+rawToken,
	)

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Errorf(
			"status = %d, want %d",
			response.Code,
			http.StatusUnauthorized,
		)
	}
}

func createTestAccessToken(
	t *testing.T,
	secret string,
) string {
	t.Helper()

	now := time.Now()

	claims := jwt.MapClaims{
		"sub":   "1",
		"iss":   "goshort",
		"aud":   "goshort-api",
		"email": "test@goshort.com",
		"role":  "user",
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
		"exp":   now.Add(15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	rawToken, err := token.SignedString(
		[]byte(secret),
	)
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}

	return "Bearer " + rawToken
}
