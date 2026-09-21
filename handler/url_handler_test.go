package handler

import (
	"context"
	"encoding/json"
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

func (s *testURLService) CreateShortURLWithAlias(
	ctx context.Context,
	originalURL string,
	alias string,
	userID int64,
) (model.URL, error) {
	return s.CreateShortURL(ctx, originalURL, userID)
}

func (s *testURLService) GetURLByID(
	_ context.Context,
	_ int64,
) (model.URL, error) {
	userID := int64(1)
	return model.URL{ID: 1, UserID: &userID}, nil
}

func (s *testURLService) ListURLs(
	_ context.Context,
	_ int64,
) ([]model.URL, error) {
	return []model.URL{}, nil
}

type testAnalyticsRepository struct{}

func (testAnalyticsRepository) GetAnalytics(
	_ context.Context,
	_ int64,
) (model.ClickAnalytics, error) {
	return model.ClickAnalytics{
		TotalClicks: 3,
		ByDevice: []model.AnalyticsCount{
			{Name: "mobile", Count: 2},
			{Name: "desktop", Count: 1},
		},
		ByReferrer: []model.AnalyticsCount{
			{Name: "direct", Count: 3},
		},
		DailyClicks: []model.DailyClickCount{
			{Date: "2026-09-21", Count: 3},
		},
	}, nil
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

func (r *testIdempotencyRepository) Reserve(
	_ context.Context,
	userID int64,
	key string,
) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	recordKey := idempotencyRecordKey(userID, key)
	if _, exists := r.records[recordKey]; exists {
		return false, nil
	}

	r.records[recordKey] = &repository.IdempotencyRecord{
		UserID:         userID,
		IdempotencyKey: key,
		ResponseBody:   []byte(`{}`),
		StatusCode:     0,
		State:          "pending",
	}
	return true, nil
}

func (r *testIdempotencyRepository) Complete(
	_ context.Context,
	record repository.IdempotencyRecord,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stored := r.records[idempotencyRecordKey(record.UserID, record.IdempotencyKey)]
	if stored == nil {
		return context.Canceled
	}

	*stored = record
	stored.ResponseBody = append([]byte(nil), record.ResponseBody...)
	return nil
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

func TestGetAnalyticsRequiresOwnershipAndReturnsAggregates(t *testing.T) {
	service := &testURLService{}
	handler := NewURLHandler(
		service,
		nil,
		nil,
		testAnalyticsRepository{},
	)

	middleware, err := auth.NewAuthMiddleware(handlerTestJWTSecret)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/urls/1/analytics",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+handlerTestToken(t))
	response := httptest.NewRecorder()

	middleware.RequireAuth(
		http.HandlerFunc(handler.GetAnalytics),
	).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var analytics model.ClickAnalytics
	if err := json.Unmarshal(response.Body.Bytes(), &analytics); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if analytics.TotalClicks != 3 {
		t.Fatalf("total clicks = %d, want 3", analytics.TotalClicks)
	}
}

func TestListURLsRequiresAuthentication(t *testing.T) {
	service := &testURLService{}
	handler := NewURLHandler(service, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/urls", nil)
	response := httptest.NewRecorder()

	handler.ListURLs(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
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
