package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"goshort/model"
	"goshort/repository"
)

const (
	accessTokenLifetime  = 15 * time.Minute
	refreshTokenLifetime = 7 * 24 * time.Hour
	refreshTokenBytes    = 32
)

// ErrEmailAlreadyExists is returned when a registration
// attempt uses an email that already exists.
var ErrEmailAlreadyExists = errors.New("email already registered")

// AuthService contains authentication and token logic.
type AuthService struct {
	users         repository.UserRepository
	refreshTokens repository.RefreshTokenRepository
	jwtSecret     []byte
}

// NewAuthService creates the authentication service.
func NewAuthService(
	users repository.UserRepository,
	refreshTokens repository.RefreshTokenRepository,
	jwtSecret string,
) (*AuthService, error) {
	if strings.TrimSpace(jwtSecret) == "" {
		return nil, errors.New("JWT secret is required")
	}

	return &AuthService{
		users:         users,
		refreshTokens: refreshTokens,
		jwtSecret:     []byte(jwtSecret),
	}, nil
}

// Register creates a new user with a bcrypt password hash.
func (s *AuthService) Register(
	ctx context.Context,
	email string,
	password string,
) (model.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	if email == "" {
		return model.User{}, errors.New("email is required")
	}

	if err := validatePassword(password); err != nil {
		return model.User{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return model.User{}, fmt.Errorf(
			"hash password: %w",
			err,
		)
	}

	user := model.User{
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         "user",
	}

	createdUser, err := s.users.Create(ctx, user)
	if err != nil {
		if repository.IsUniqueViolation(err) {
			return model.User{}, ErrEmailAlreadyExists
		}

		return model.User{}, fmt.Errorf(
			"create user: %w",
			err,
		)
	}

	return createdUser, nil
}

// Login verifies credentials and creates access + refresh tokens.
func (s *AuthService) Login(
	ctx context.Context,
	email string,
	password string,
) (string, string, model.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return "", "", model.User{}, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	); err != nil {
		return "", "", model.User{}, errors.New("invalid credentials")
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return "", "", model.User{}, fmt.Errorf(
			"generate access token: %w",
			err,
		)
	}

	refreshToken, err := s.createRefreshToken(
		ctx,
		user.ID,
	)
	if err != nil {
		return "", "", model.User{}, err
	}

	return accessToken, refreshToken, user, nil
}

// Refresh validates the supplied refresh token, revokes it,
// and atomically creates a replacement refresh token.
func (s *AuthService) Refresh(
	ctx context.Context,
	rawRefreshToken string,
) (string, string, error) {
	rawRefreshToken = strings.TrimSpace(rawRefreshToken)

	if rawRefreshToken == "" {
		return "", "", errors.New("invalid refresh token")
	}

	tokenHash := hashToken(rawRefreshToken)

	storedToken, err := s.refreshTokens.FindActiveByHash(
		ctx,
		tokenHash,
	)
	if err != nil {
		return "", "", errors.New("invalid refresh token")
	}

	user, err := s.users.FindByID(
		ctx,
		storedToken.UserID,
	)
	if err != nil {
		return "", "", errors.New("invalid refresh token")
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return "", "", fmt.Errorf(
			"generate access token: %w",
			err,
		)
	}

	newRawRefreshToken, newStoredToken, err := s.generateRefreshToken(
		user.ID,
	)
	if err != nil {
		return "", "", err
	}

	_, err = s.refreshTokens.Rotate(
		ctx,
		storedToken.ID,
		newStoredToken,
	)
	if err != nil {
		return "", "", errors.New("invalid refresh token")
	}

	return accessToken, newRawRefreshToken, nil
}

// Logout revokes the supplied refresh token.
func (s *AuthService) Logout(
	ctx context.Context,
	rawRefreshToken string,
) error {
	rawRefreshToken = strings.TrimSpace(rawRefreshToken)

	if rawRefreshToken == "" {
		return errors.New("invalid refresh token")
	}

	tokenHash := hashToken(rawRefreshToken)

	storedToken, err := s.refreshTokens.FindActiveByHash(
		ctx,
		tokenHash,
	)
	if err != nil {
		return errors.New("invalid refresh token")
	}

	return s.refreshTokens.Revoke(
		ctx,
		storedToken.ID,
	)
}

// generateAccessToken creates a short-lived signed JWT.
func (s *AuthService) generateAccessToken(
	user model.User,
) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"sub":   fmt.Sprintf("%d", user.ID),
		"iss":   "goshort",
		"aud":   "goshort-api",
		"email": user.Email,
		"role":  user.Role,
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
		"exp":   now.Add(accessTokenLifetime).Unix(),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	return token.SignedString(s.jwtSecret)
}

// createRefreshToken generates a cryptographically random
// opaque token and stores only its SHA-256 hash.
func (s *AuthService) createRefreshToken(
	ctx context.Context,
	userID int64,
) (string, error) {
	rawToken, storedToken, err := s.generateRefreshToken(userID)
	if err != nil {
		return "", err
	}

	if _, err := s.refreshTokens.Create(
		ctx,
		storedToken,
	); err != nil {
		return "", fmt.Errorf(
			"store refresh token: %w",
			err,
		)
	}

	return rawToken, nil
}

// generateRefreshToken creates a raw refresh token and
// its database representation without storing it.
func (s *AuthService) generateRefreshToken(
	userID int64,
) (string, model.RefreshToken, error) {
	randomBytes := make([]byte, refreshTokenBytes)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", model.RefreshToken{}, fmt.Errorf(
			"generate refresh token: %w",
			err,
		)
	}

	rawToken := hex.EncodeToString(randomBytes)

	storedToken := model.RefreshToken{
		UserID:    userID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: time.Now().Add(refreshTokenLifetime),
	}

	return rawToken, storedToken, nil
}

// hashToken creates a deterministic SHA-256 hash for
// database lookup.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))

	return hex.EncodeToString(hash[:])
}
