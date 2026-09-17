package service

import (
	"fmt"
	"net/url"
	"sync"

	"goshort/model"
	"goshort/repository"
)

// URLService contains the application logic for URL shortening.
// It keeps validation and short-code generation out of the HTTP layer.
type URLService struct {
	repository repository.URLRepository

	mu       sync.Mutex
	nextCode int
}

// NewURLService creates a URL service with the required repository.
func NewURLService(repository repository.URLRepository) *URLService {
	return &URLService{
		repository: repository,
		nextCode:   1,
	}
}

// CreateShortURL validates the original URL, generates a short code,
// stores the mapping, and returns the created URL.
func (s *URLService) CreateShortURL(originalURL string) (model.URL, error) {
	// Validate the destination before storing it so invalid URLs
	// never enter the application storage.
	parsedURL, err := url.ParseRequestURI(originalURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return model.URL{}, fmt.Errorf("invalid URL")
	}

	// Protect short-code generation because multiple requests
	// can execute the service concurrently.
	s.mu.Lock()
	shortCode := fmt.Sprintf("%d", s.nextCode)
	s.nextCode++
	s.mu.Unlock()

	createdURL := model.URL{
		ShortCode:   shortCode,
		OriginalURL: originalURL,
	}

	// Persist the URL through the repository instead of accessing
	// storage directly from the service.
	if err := s.repository.Save(createdURL); err != nil {
		return model.URL{}, err
	}

	return createdURL, nil
}

// GetOriginalURL retrieves the original destination for a short code.
func (s *URLService) GetOriginalURL(shortCode string) (model.URL, bool) {
	return s.repository.FindByShortCode(shortCode)
}
