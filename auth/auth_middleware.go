package auth

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userClaimsKey contextKey = "goshort_user_claims"

// UserClaims represents authenticated user information extracted
// from a validated access token.
type UserClaims struct {
	UserID int64
	Email  string
	Role   string
}

// AuthMiddleware validates JWT access tokens before allowing
// requests to reach protected handlers.
type AuthMiddleware struct {
	jwtSecret []byte
}

// NewAuthMiddleware creates JWT authentication middleware.
func NewAuthMiddleware(jwtSecret string) (*AuthMiddleware, error) {
	if strings.TrimSpace(jwtSecret) == "" {
		return nil, errors.New("JWT secret is required")
	}

	return &AuthMiddleware{
		jwtSecret: []byte(jwtSecret),
	}, nil
}

// RequireAuth rejects requests without a valid access token.
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))

		if header == "" {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		parts := strings.Fields(header)

		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
			return
		}

		claims, err := m.parseAccessToken(parts[1])
		if err != nil {
			http.Error(w, "Invalid access token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(
			r.Context(),
			userClaimsKey,
			claims,
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CurrentUser returns authenticated user claims from the request context.
func CurrentUser(ctx context.Context) (UserClaims, bool) {
	claims, ok := ctx.Value(userClaimsKey).(UserClaims)
	return claims, ok
}

// parseAccessToken validates the JWT signature, algorithm,
// expiration, issued-at time, issuer and audience.
func (m *AuthMiddleware) parseAccessToken(
	rawToken string,
) (UserClaims, error) {

	token, err := jwt.Parse(
		rawToken,
		func(token *jwt.Token) (interface{}, error) {
			// The application signs access tokens with HS256.
			// Reject every other signing method.
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}

			return m.jwtSecret, nil
		},
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithIssuer("goshort"),
		jwt.WithAudience("goshort-api"),
	)

	if err != nil {
		return UserClaims{}, err
	}

	if !token.Valid {
		return UserClaims{}, errors.New("invalid token")
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return UserClaims{}, errors.New("invalid claims")
	}

	// The JWT subject contains the user's database ID as a string.
	subject, err := mapClaims.GetSubject()
	if err != nil || strings.TrimSpace(subject) == "" {
		return UserClaims{}, errors.New("invalid subject")
	}

	userID, err := strconv.ParseInt(subject, 10, 64)
	if err != nil || userID <= 0 {
		return UserClaims{}, errors.New("invalid user ID")
	}

	email, ok := mapClaims["email"].(string)
	if !ok || strings.TrimSpace(email) == "" {
		return UserClaims{}, errors.New("invalid email")
	}

	role, ok := mapClaims["role"].(string)
	if !ok || strings.TrimSpace(role) == "" {
		return UserClaims{}, errors.New("invalid role")
	}

	// Explicitly verify the expiration timestamp.
	expiration, err := token.Claims.GetExpirationTime()
	if err != nil || expiration == nil {
		return UserClaims{}, errors.New("invalid expiration")
	}

	if expiration.Time.Before(time.Now()) {
		return UserClaims{}, errors.New("token expired")
	}

	return UserClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
	}, nil
}

// RequireRole restricts a route to a specific role.
func (m *AuthMiddleware) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := CurrentUser(r.Context())
			if !ok {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			if user.Role != role {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
