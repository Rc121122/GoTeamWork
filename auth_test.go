package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Ensures JWT issuance and authentication succeed and enforce user matching.
func TestJWTIssueAndAuthenticate(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	app := NewApp("client")

	user := app.CreateUser("Tester")
	token, err := app.issueToken(user.ID)
	if err != nil {
		t.Fatalf("issueToken error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	authUser, err := app.authenticateRequest(req)
	if err != nil {
		t.Fatalf("authenticateRequest error: %v", err)
	}
	if authUser.ID != user.ID {
		t.Fatalf("expected user %s, got %s", user.ID, authUser.ID)
	}

	if _, err := enforceUserMatch(user.ID, authUser); err != nil {
		t.Fatalf("enforceUserMatch rejected matching IDs: %v", err)
	}
	if _, err := enforceUserMatch("someone-else", authUser); err == nil {
		t.Fatalf("enforceUserMatch should fail for mismatched IDs")
	}
}
