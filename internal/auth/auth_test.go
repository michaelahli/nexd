package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPasswordHashAndCheck(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	if !CheckPassword(hash, "correct horse battery staple") {
		t.Fatal("expected password to match hash")
	}
	if CheckPassword(hash, "wrong password") {
		t.Fatal("expected wrong password to fail")
	}
}

func TestTokenManagerGenerateAndValidate(t *testing.T) {
	manager := NewTokenManager("test-secret", time.Hour)
	userID := uuid.New()
	user := User{ID: userID, Email: "user@example.com"}

	token, expiresAt, err := manager.Generate(user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	if token == "" {
		t.Fatal("expected token")
	}
	if expiresAt.IsZero() {
		t.Fatal("expected expiration")
	}

	claims, err := manager.Validate(token)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("expected user id %s, got %s", userID, claims.UserID)
	}
	if claims.Email != user.Email {
		t.Fatalf("expected email %s, got %s", user.Email, claims.Email)
	}
}

func TestTokenManagerRejectsInvalidToken(t *testing.T) {
	manager := NewTokenManager("test-secret", time.Hour)

	if _, err := manager.Validate("not-a-token"); err == nil {
		t.Fatal("expected invalid token error")
	}
}
