package service

import (
	"crypto/rand"
	"fmt"
	"net/url"

	"goshort/model"
	"goshort/repository"
)

const (
	// base62Alphabet contains the characters used to build short codes.
	base62Alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	// shortCodeLength controls the size of generated short codes.
	shortCodeLength = 6

	// maxGenerationAttempts prevents an endless loop if collisions occur.
	maxGenerationAttempts = 10
)

// URLService contains the application logic for URL shortening.
// It keeps validation and short-code generation outside the HTTP layer.
type URLService struct {
	repository repository.URLRepository
}

// NewURLService creates a URL service with the required repository.
func NewURLService(repository repository.URLRepository) *URLService {
	return &URLService{
		repository: repository,
	}
}

// CreateShortURL validates the original URL, generates a unique short code,
// stores the mapping, and returns the created URL.
func (s *URLService) CreateShortURL(originalURL string) (model.URL, error) {
	// Validate the destination before storing it so invalid URLs
	// never enter the application storage.
	parsedURL, err := url.ParseRequestURI(originalURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return model.URL{}, fmt.Errorf("invalid URL")
	}

	// Generate a short code and retry when a collision occurs.
	for attempt := 0; attempt < maxGenerationAttempts; attempt++ {
		shortCode, err := generateShortCode(shortCodeLength)
		if err != nil {
			return model.URL{}, fmt.Errorf("generate short code: %w", err)
		}

		// A collision means the generated code already belongs to another URL.
		// Retry with a new random code instead of overwriting existing data.
		if s.repository.Exists(shortCode) {
			continue
		}

		createdURL := model.URL{
			ShortCode:   shortCode,
			OriginalURL: originalURL,
		}

		if err := s.repository.Save(createdURL); err != nil {
			return model.URL{}, err
		}

		return createdURL, nil
	}

	return model.URL{}, fmt.Errorf("unable to generate unique short code")
}

// GetOriginalURL retrieves the original destination for a short code.
func (s *URLService) GetOriginalURL(shortCode string) (model.URL, bool) {
	return s.repository.FindByShortCode(shortCode)
}

// generateShortCode creates a random Base62 short code.
func generateShortCode(length int) (string, error) {
	randomBytes := make([]byte, length)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	code := make([]byte, length)

	for i, randomByte := range randomBytes {
		code[i] = base62Alphabet[int(randomByte)%len(base62Alphabet)]
	}

	return string(code), nil
}
