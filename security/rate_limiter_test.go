package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterAllowsRequestsWithinLimit(t *testing.T) {
	rateLimiter := NewRateLimiter(
		2,
		time.Minute,
	)

	next := http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			w.WriteHeader(http.StatusOK)
		},
	)

	handler := rateLimiter.Middleware(next)

	for i := 0; i < 2; i++ {
		request := httptest.NewRequest(
			http.MethodGet,
			"/health",
			nil,
		)

		request.RemoteAddr = "127.0.0.1:12345"

		response := httptest.NewRecorder()

		handler.ServeHTTP(
			response,
			request,
		)

		if response.Code != http.StatusOK {
			t.Fatalf(
				"request %d status = %d, want %d",
				i+1,
				response.Code,
				http.StatusOK,
			)
		}

		if response.Header().Get(
			"X-RateLimit-Limit",
		) != "2" {
			t.Fatalf(
				"X-RateLimit-Limit = %q, want %q",
				response.Header().Get(
					"X-RateLimit-Limit",
				),
				"2",
			)
		}
	}
}

func TestRateLimiterRejectsRequestsOverLimit(t *testing.T) {
	rateLimiter := NewRateLimiter(
		2,
		time.Minute,
	)

	next := http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			w.WriteHeader(http.StatusOK)
		},
	)

	handler := rateLimiter.Middleware(next)

	for i := 0; i < 2; i++ {
		request := httptest.NewRequest(
			http.MethodGet,
			"/health",
			nil,
		)

		request.RemoteAddr = "127.0.0.1:12345"

		response := httptest.NewRecorder()

		handler.ServeHTTP(
			response,
			request,
		)

		if response.Code != http.StatusOK {
			t.Fatalf(
				"request %d status = %d, want %d",
				i+1,
				response.Code,
				http.StatusOK,
			)
		}
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	request.RemoteAddr = "127.0.0.1:12345"

	response := httptest.NewRecorder()

	handler.ServeHTTP(
		response,
		request,
	)

	if response.Code != http.StatusTooManyRequests {
		t.Fatalf(
			"status = %d, want %d",
			response.Code,
			http.StatusTooManyRequests,
		)
	}

	if response.Header().Get(
		"Retry-After",
	) == "" {
		t.Fatal(
			"expected Retry-After header",
		)
	}

	if response.Header().Get(
		"X-RateLimit-Limit",
	) != "2" {
		t.Fatalf(
			"X-RateLimit-Limit = %q, want %q",
			response.Header().Get(
				"X-RateLimit-Limit",
			),
			"2",
		)
	}
}

func TestRateLimiterSeparatesClientsByIP(t *testing.T) {
	rateLimiter := NewRateLimiter(
		1,
		time.Minute,
	)

	next := http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			w.WriteHeader(http.StatusOK)
		},
	)

	handler := rateLimiter.Middleware(next)

	firstRequest := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)
	firstRequest.RemoteAddr = "127.0.0.1:10001"

	firstResponse := httptest.NewRecorder()

	handler.ServeHTTP(
		firstResponse,
		firstRequest,
	)

	if firstResponse.Code != http.StatusOK {
		t.Fatalf(
			"first client status = %d, want %d",
			firstResponse.Code,
			http.StatusOK,
		)
	}

	secondRequest := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)
	secondRequest.RemoteAddr = "127.0.0.2:10002"

	secondResponse := httptest.NewRecorder()

	handler.ServeHTTP(
		secondResponse,
		secondRequest,
	)

	if secondResponse.Code != http.StatusOK {
		t.Fatalf(
			"second client status = %d, want %d",
			secondResponse.Code,
			http.StatusOK,
		)
	}
}
