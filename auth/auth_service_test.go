package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"goshort/model"
)

type mockUserRepository struct {
	user model.User
	err  error
}

func (m *mockUserRepository) Create(
	ctx context.Context,
	user model.User,
) (model.User, error) {
	return user, nil
}

func (m *mockUserRepository) FindByEmail(
	ctx context.Context,
	email string,
) (model.User, error) {
	return m.user, m.err
}

func (m *mockUserRepository) FindByID(
	ctx context.Context,
	id int64,
) (model.User, error) {
	if m.err != nil {
		return model.User{}, m.err
	}

	if m.user.ID != id {
		return model.User{}, errors.New("user not found")
	}

	return m.user, nil
}

type mockRefreshTokenRepository struct {
	activeToken      model.RefreshToken
	findErr          error
	rotateCalls      int
	lastRevokedID    int64
	lastRotatedToken model.RefreshToken
}

func (m *mockRefreshTokenRepository) Create(
	ctx context.Context,
	token model.RefreshToken,
) (model.RefreshToken, error) {
	return token, nil
}

func (m *mockRefreshTokenRepository) FindActiveByHash(
	ctx context.Context,
	tokenHash string,
) (model.RefreshToken, error) {
	if m.findErr != nil {
		return model.RefreshToken{}, m.findErr
	}

	if tokenHash != m.activeToken.TokenHash {
		return model.RefreshToken{}, errors.New("invalid token")
	}

	return m.activeToken, nil
}

func (m *mockRefreshTokenRepository) Revoke(
	ctx context.Context,
	id int64,
) error {
	m.lastRevokedID = id
	return nil
}

func (m *mockRefreshTokenRepository) Rotate(
	ctx context.Context,
	oldTokenID int64,
	newToken model.RefreshToken,
) (model.RefreshToken, error) {
	m.rotateCalls++
	m.lastRevokedID = oldTokenID
	m.lastRotatedToken = newToken

	return newToken, nil
}

func TestAuthServiceRefreshRotatesRefreshToken(t *testing.T) {
	userRepository := &mockUserRepository{
		user: model.User{
			ID:           1,
			Email:        "test@goshort.com",
			PasswordHash: "unused",
			Role:         "user",
		},
	}

	rawRefreshToken := "old-refresh-token"

	refreshRepository := &mockRefreshTokenRepository{
		activeToken: model.RefreshToken{
			ID:        10,
			UserID:    1,
			TokenHash: hashToken(rawRefreshToken),
			ExpiresAt: time.Now().Add(time.Hour),
		},
	}

	service, err := NewAuthService(
		userRepository,
		refreshRepository,
		"test-secret",
	)
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}

	accessToken, newRefreshToken, err := service.Refresh(
		context.Background(),
		rawRefreshToken,
	)
	if err != nil {
		t.Fatalf("refresh token: %v", err)
	}

	if accessToken == "" {
		t.Fatal("expected access token")
	}

	if newRefreshToken == "" {
		t.Fatal("expected new refresh token")
	}

	if newRefreshToken == rawRefreshToken {
		t.Fatal("refresh token was not rotated")
	}

	if refreshRepository.rotateCalls != 1 {
		t.Fatalf(
			"Rotate calls = %d, want 1",
			refreshRepository.rotateCalls,
		)
	}

	if refreshRepository.lastRevokedID != 10 {
		t.Fatalf(
			"revoked token ID = %d, want 10",
			refreshRepository.lastRevokedID,
		)
	}

	if refreshRepository.lastRotatedToken.UserID != 1 {
		t.Fatalf(
			"new token user ID = %d, want 1",
			refreshRepository.lastRotatedToken.UserID,
		)
	}

	if refreshRepository.lastRotatedToken.TokenHash ==
		hashToken(rawRefreshToken) {
		t.Fatal("new refresh token reused the old token hash")
	}
}

func TestAuthServiceRefreshRejectsInvalidRefreshToken(t *testing.T) {
	userRepository := &mockUserRepository{
		user: model.User{
			ID:    1,
			Email: "test@goshort.com",
			Role:  "user",
		},
	}

	refreshRepository := &mockRefreshTokenRepository{
		findErr: errors.New("token not found"),
	}

	service, err := NewAuthService(
		userRepository,
		refreshRepository,
		"test-secret",
	)
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}

	accessToken, newRefreshToken, err := service.Refresh(
		context.Background(),
		"invalid-refresh-token",
	)

	if err == nil {
		t.Fatal("expected refresh error")
	}

	if accessToken != "" {
		t.Fatalf(
			"access token = %q, want empty",
			accessToken,
		)
	}

	if newRefreshToken != "" {
		t.Fatalf(
			"refresh token = %q, want empty",
			newRefreshToken,
		)
	}

	if refreshRepository.rotateCalls != 0 {
		t.Fatalf(
			"Rotate calls = %d, want 0",
			refreshRepository.rotateCalls,
		)
	}
}

func TestAuthServiceRefreshRejectsReusedToken(t *testing.T) {
	userRepository := &mockUserRepository{
		user: model.User{
			ID:    1,
			Email: "test@goshort.com",
			Role:  "user",
		},
	}

	rawRefreshToken := "single-use-refresh-token"

	refreshRepository := &mockRefreshTokenRepository{
		activeToken: model.RefreshToken{
			ID:        20,
			UserID:    1,
			TokenHash: hashToken(rawRefreshToken),
			ExpiresAt: time.Now().Add(time.Hour),
		},
	}

	service, err := NewAuthService(
		userRepository,
		refreshRepository,
		"test-secret",
	)
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}

	_, newRefreshToken, err := service.Refresh(
		context.Background(),
		rawRefreshToken,
	)
	if err != nil {
		t.Fatalf("first refresh failed: %v", err)
	}

	if newRefreshToken == "" {
		t.Fatal("expected replacement refresh token")
	}

	// Simulate the database state after rotation:
	// the original refresh token is no longer active.
	refreshRepository.findErr = errors.New(
		"refresh token revoked",
	)

	_, _, err = service.Refresh(
		context.Background(),
		rawRefreshToken,
	)

	if err == nil {
		t.Fatal("expected reused refresh token to be rejected")
	}

	if refreshRepository.rotateCalls != 1 {
		t.Fatalf(
			"Rotate calls = %d, want 1",
			refreshRepository.rotateCalls,
		)
	}
}
