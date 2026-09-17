package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/url"
	"strings"

	"goshort/model"
	"goshort/repository"
)

const (
	base62Alphabet        = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	shortCodeLength       = 6
	maxGenerationAttempts = 10
)

// URLService contains the application logic for URL operations.
// Validation and short-code generation remain outside the HTTP layer.
type URLService struct {
	repository repository.URLRepository
}

// NewURLService creates a URL service with the required repository.
func NewURLService(repository repository.URLRepository) *URLService {
	return &URLService{
		repository: repository,
	}
}

// CreateShortURL validates the original URL, generates a short code,
// stores it in PostgreSQL, and returns the created URL.
func (s *URLService) CreateShortURL(
	ctx context.Context,
	originalURL string,
) (model.URL, error) {
	if err := validateURL(originalURL); err != nil {
		return model.URL{}, err
	}

	for attempt := 0; attempt < maxGenerationAttempts; attempt++ {
		shortCode, err := generateShortCode(shortCodeLength)
		if err != nil {
			return model.URL{}, fmt.Errorf("generate short code: %w", err)
		}

		createdURL := model.URL{
			ShortCode:   shortCode,
			OriginalURL: originalURL,
			IsActive:    true,
		}

		createdURL, err = s.repository.Create(ctx, createdURL)
		if err == nil {
			return createdURL, nil
		}

		// PostgreSQL's UNIQUE constraint protects short_code.
		// A collision can therefore be retried safely.
	}

	return model.URL{}, fmt.Errorf("unable to generate unique short code")
}

// GetURLByID retrieves a URL by database ID.
func (s *URLService) GetURLByID(
	ctx context.Context,
	id int64,
) (model.URL, error) {
	return s.repository.FindByID(ctx, id)
}

// GetOriginalURL retrieves an active, non-expired URL by short code.
func (s *URLService) GetOriginalURL(
	ctx context.Context,
	shortCode string,
) (model.URL, error) {
	return s.repository.FindByShortCode(ctx, shortCode)
}

// UpdateURL updates the editable fields of a shortened URL.
func (s *URLService) UpdateURL(
	ctx context.Context,
	url model.URL,
) (model.URL, error) {
	if err := validateURL(url.OriginalURL); err != nil {
		return model.URL{}, err
	}

	return s.repository.Update(ctx, url)
}

// DeleteURL permanently removes a shortened URL.
func (s *URLService) DeleteURL(
	ctx context.Context,
	id int64,
) error {
	return s.repository.Delete(ctx, id)
}

// validateURL checks that the supplied value is an absolute HTTP(S) URL.
func validateURL(value string) error {
	value = strings.TrimSpace(value)

	parsedURL, err := url.ParseRequestURI(value)
	if err != nil {
		return fmt.Errorf("invalid URL")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use http or https")
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("URL host is required")
	}

	return nil
}

// generateShortCode creates a cryptographically random Base62 code.
// Rejection sampling avoids modulo bias when mapping random bytes
// into the 62-character alphabet.
func generateShortCode(length int) (string, error) {
	code := make([]byte, length)

	for i := 0; i < length; {
		var randomByte [1]byte

		if _, err := rand.Read(randomByte[:]); err != nil {
			return "", err
		}

		if randomByte[0] >= 248 {
			continue
		}

		code[i] = base62Alphabet[int(randomByte[0])%len(base62Alphabet)]
		i++
	}

	return string(code), nil
}
