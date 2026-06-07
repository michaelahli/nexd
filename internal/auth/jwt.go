package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid token")

// Claims contains NEXTD JWT claims.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

// TokenManager creates and validates JWT access tokens.
type TokenManager struct {
	secret     []byte
	expiration time.Duration
	now        func() time.Time
}

// NewTokenManager creates a JWT token manager.
func NewTokenManager(secret string, expiration time.Duration) *TokenManager {
	return &TokenManager{
		secret:     []byte(secret),
		expiration: expiration,
		now:        time.Now,
	}
}

// Generate creates a signed JWT for the user.
func (m *TokenManager) Generate(user User) (string, time.Time, error) {
	if m == nil || len(m.secret) == 0 {
		return "", time.Time{}, fmt.Errorf("jwt secret is required")
	}
	if m.expiration <= 0 {
		return "", time.Time{}, fmt.Errorf("jwt expiration must be positive")
	}

	now := m.now().UTC()
	expiresAt := now.Add(m.expiration)
	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}

// Validate parses and validates a signed JWT.
func (m *TokenManager) Validate(tokenString string) (*Claims, error) {
	if m == nil || len(m.secret) == 0 {
		return nil, fmt.Errorf("jwt secret is required")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
