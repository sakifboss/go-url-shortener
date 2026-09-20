package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"goshort/auth"
	"goshort/model"
	"goshort/repository"

	"github.com/golang-jwt/jwt/v5"
)

const handlerTestJWTSecret = "handler-test-secret"

type testURLService struct {
	calls atomic.Int32
}

func (s *testURLService) CreateShortURL(
	_ context.Context,
	_ string,
	userID int64,
) (model.URL, error) {
	s.calls.Add(1)
	return model.URL{
		ID:          1,
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		UserID:      &userID,
		IsActive:    true,
	}, nil
}

func (s *testURLService) GetURLByID(
	_ context.Context,
	_ int64,
) (model.URL, error) {
	return model.URL{}, nil
}

func (s *testURLService) GetOriginalURL(
	_ context.Context,
	_ string,
) (model.URL, error) {
	return model.URL{}, nil
}

func (s *testURLService) UpdateURL(
	_ context.Context,
	url model.URL,
) (model.URL, error) {
	return url, nil
}

func (s *testURLService) DeleteURL(
	_ context.Context,
	_ int64,
) error {
	return nil
}

type testIdempotencyRepository struct {
	mu      sync.Mutex
	records map[string]*repository.IdempotencyRecord
}

func (r *testIdempotencyRepository) Get(
	_ context.Context,
	userID int64,
	key string,
) (*repository.IdempotencyRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record := r.records[idempotencyRecordKey(userID, key)]
	if record == nil {
		return nil, nil
	}

	copyRecord := *record
	copyRecord.ResponseBody = append([]byte(nil), record.ResponseBody...)
	return &copyRecord, nil
}

func (r *testIdempotencyRepository) Create(
	_ context.Context,
	record repository.IdempotencyRecord,
) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := idempotencyRecordKey(record.UserID, record.IdempotencyKey)
	if _, exists := r.records[key]; exists {
		return false, nil
	}

	record.ResponseBody = append([]byte(nil), record.ResponseBody...)
	r.records[key] = &record
	return true, nil
}

func idempotencyRecordKey(userID int64, key string) string {
	return string(rune(userID)) + ":" + key
}

func TestCreateURL_IdempotencyConcurrent(t *testing.T) {
	service := &testURLService{}
	repo := &testIdempotencyRepository{
		records: make(map[string]*repository.IdempotencyRecord),
	}
	handler := NewURLHandler(service, nil, repo)

	middleware, err := auth.NewAuthMiddleware(handlerTestJWTSecret)
	if err != nil {
		t.Fatal(err)
	}

	token := handlerTestToken(t)
	const requestCount = 20

	var waitGroup sync.WaitGroup
	responses := make(chan string, requestCount)
	statuses := make(chan int, requestCount)

	for i := 0; i < requestCount; i++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()

			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/urls",
				io.NopCloser(strings.NewReader(`{"url":"https://example.com"}`)),
			)
			request.Header.Set("Authorization", "Bearer "+token)
			request.Header.Set("Idempotency-Key", "same-key")

			response := httptest.NewRecorder()
			middleware.RequireAuth(
				http.HandlerFunc(handler.CreateURL),
			).ServeHTTP(response, request)
			responses <- response.Body.String()
			statuses <- response.Code
		}()
	}

	waitGroup.Wait()
	close(responses)
	close(statuses)

	var firstResponse string
	for response := range responses {
		if firstResponse == "" {
			firstResponse = response
		}
		if response != firstResponse {
			t.Errorf("response body differs: got %q, want %q", response, firstResponse)
		}
	}

	for status := range statuses {
		if status != http.StatusCreated {
			t.Errorf("status = %d, want %d", status, http.StatusCreated)
		}
	}

	if calls := service.calls.Load(); calls != 1 {
		t.Fatalf("CreateShortURL calls = %d, want 1", calls)
	}
}

func handlerTestToken(t *testing.T) string {
	t.Helper()

	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   "1",
		"iss":   "goshort",
		"aud":   "goshort-api",
		"email": "test@example.com",
		"role":  "user",
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	raw, err := token.SignedString([]byte(handlerTestJWTSecret))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
