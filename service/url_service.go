package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"goshort/model"
	"goshort/repository"
)

const (
	base62Alphabet        = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	shortCodeLength       = 6
	maxGenerationAttempts = 10
	minAliasLength        = 3
	maxAliasLength        = 32
)

var customAliasPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// URLService contains URL shortening business logic.
type URLService struct {
	repository repository.URLRepository
}

// NewURLService creates a new URL service.
func NewURLService(
	repository repository.URLRepository,
) *URLService {
	return &URLService{
		repository: repository,
	}
}

// CreateShortURL validates the destination URL, generates a unique
// short code, and stores the shortened URL.
func (s *URLService) CreateShortURL(
	ctx context.Context,
	originalURL string,
	userID int64,
) (model.URL, error) {
	if err := validateURL(originalURL); err != nil {
		return model.URL{}, err
	}

	if userID <= 0 {
		return model.URL{}, fmt.Errorf("invalid user ID")
	}

	for attempt := 0; attempt < maxGenerationAttempts; attempt++ {
		shortCode, err := generateShortCode(shortCodeLength)
		if err != nil {
			return model.URL{}, fmt.Errorf(
				"generate short code: %w",
				err,
			)
		}

		createdURL := model.URL{
			ShortCode:   shortCode,
			OriginalURL: strings.TrimSpace(originalURL),
			UserID:      &userID,
			IsActive:    true,
		}

		createdURL, err = s.repository.Create(
			ctx,
			createdURL,
		)

		if err == nil {
			return createdURL, nil
		}

		// Retry only when the short_code UNIQUE constraint
		// reports a collision.
		if !isUniqueViolation(err) {
			return model.URL{}, fmt.Errorf(
				"create shortened URL: %w",
				err,
			)
		}
	}

	return model.URL{}, fmt.Errorf(
		"unable to generate unique short code",
	)
}

func (s *URLService) CreateShortURLWithAlias(
	ctx context.Context,
	originalURL string,
	alias string,
	userID int64,
) (model.URL, error) {
	if err := validateURL(originalURL); err != nil {
		return model.URL{}, err
	}

	if userID <= 0 {
		return model.URL{}, fmt.Errorf("invalid user ID")
	}

	alias = strings.TrimSpace(alias)
	if len(alias) < minAliasLength || len(alias) > maxAliasLength {
		return model.URL{}, fmt.Errorf("alias must be between 3 and 32 characters")
	}
	if !customAliasPattern.MatchString(alias) {
		return model.URL{}, fmt.Errorf("alias may contain only letters, numbers, hyphens, and underscores")
	}

	createdURL := model.URL{
		ShortCode:   alias,
		CustomAlias: &alias,
		OriginalURL: strings.TrimSpace(originalURL),
		UserID:      &userID,
		IsActive:    true,
	}

	createdURL, err := s.repository.Create(ctx, createdURL)
	if err != nil {
		if isUniqueViolation(err) {
			return model.URL{}, fmt.Errorf("alias is already in use")
		}

		return model.URL{}, fmt.Errorf("create shortened URL: %w", err)
	}

	return createdURL, nil
}

// GetURLByID returns a URL by its database ID.
func (s *URLService) GetURLByID(
	ctx context.Context,
	id int64,
) (model.URL, error) {
	return s.repository.FindByID(
		ctx,
		id,
	)
}

func (s *URLService) ListURLs(
	ctx context.Context,
	userID int64,
) ([]model.URL, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user ID")
	}

	return s.repository.FindByUserID(ctx, userID)
}

// GetOriginalURL returns the destination URL for a short code.
func (s *URLService) GetOriginalURL(
	ctx context.Context,
	shortCode string,
) (model.URL, error) {
	return s.repository.FindByShortCode(
		ctx,
		shortCode,
	)
}

// UpdateURL updates an existing shortened URL.
func (s *URLService) UpdateURL(
	ctx context.Context,
	url model.URL,
) (model.URL, error) {
	if err := validateURL(url.OriginalURL); err != nil {
		return model.URL{}, err
	}

	return s.repository.Update(
		ctx,
		url,
	)
}

// DeleteURL permanently deletes a shortened URL.
func (s *URLService) DeleteURL(
	ctx context.Context,
	id int64,
) error {
	return s.repository.Delete(
		ctx,
		id,
	)
}

// validateURL validates the syntax and structure of a destination URL.
//
// GoShort accepts HTTP and HTTPS destinations.
// It does not perform DNS resolution here because the shortener
// stores the destination and redirects the client; it does not
// server-side fetch the destination URL.
//
// IP addresses are checked when the host itself is an IP so that
// obvious localhost/private-network destinations are rejected.
func validateURL(value string) error {
	value = strings.TrimSpace(value)

	if value == "" {
		return fmt.Errorf("URL is required")
	}

	parsedURL, err := url.ParseRequestURI(value)
	if err != nil {
		return fmt.Errorf("invalid URL")
	}

	if parsedURL.Scheme != "http" &&
		parsedURL.Scheme != "https" {
		return fmt.Errorf(
			"URL must use http or https",
		)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf(
			"URL host is required",
		)
	}

	// Credentials such as user:password@host are not accepted.
	if parsedURL.User != nil {
		return fmt.Errorf(
			"URL credentials are not allowed",
		)
	}

	host := strings.TrimSpace(
		parsedURL.Hostname(),
	)
	host = strings.ToLower(host)

	if host == "" {
		return fmt.Errorf(
			"URL host is required",
		)
	}
	if host == "localhost" ||
		strings.HasSuffix(host, ".localhost") ||
		host == "localhost.localdomain" {
		return fmt.Errorf(
			"URL host must not be localhost",
		)
	}

	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return fmt.Errorf(
				"URL host must be a public IP address",
			)
		}
	}

	// Reject obvious internal IP destinations when the host
	// is written directly as an IP address.
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return fmt.Errorf(
				"URL host must be a public IP address",
			)
		}
	}

	return nil
}

// isPublicIP reports whether an IP address is publicly routable
// rather than a local, private, loopback, multicast, or
// unspecified address.
func isPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	if ip.IsLoopback() {
		return false
	}

	if ip.IsPrivate() {
		return false
	}

	if ip.IsLinkLocalUnicast() {
		return false
	}

	if ip.IsLinkLocalMulticast() {
		return false
	}

	if ip.IsMulticast() {
		return false
	}

	if ip.IsUnspecified() {
		return false
	}

	if ip4 := ip.To4(); ip4 != nil {
		// IPv4 limited broadcast address.
		if ip4[0] == 255 &&
			ip4[1] == 255 &&
			ip4[2] == 255 &&
			ip4[3] == 255 {
			return false
		}
	}

	return true
}

// generateShortCode generates a cryptographically random Base62
// short code.
func generateShortCode(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf(
			"short code length must be positive",
		)
	}

	code := make([]byte, length)

	for i := 0; i < length; {
		var randomByte [1]byte

		if _, err := rand.Read(randomByte[:]); err != nil {
			return "", err
		}

		// 248 is the largest multiple of 62 below 256.
		// Rejecting values >= 248 removes modulo bias.
		if randomByte[0] >= 248 {
			continue
		}

		code[i] = base62Alphabet[int(randomByte[0])%len(base62Alphabet)]

		i++
	}

	return string(code), nil
}

// isUniqueViolation reports whether PostgreSQL rejected an
// operation because of a UNIQUE constraint.
func isUniqueViolation(err error) bool {
	var pgError *pgconn.PgError

	if !errors.As(err, &pgError) {
		return false
	}

	// PostgreSQL SQLSTATE 23505 = unique_violation.
	return pgError.Code == "23505"
}
