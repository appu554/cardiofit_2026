package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/permissions"
)

// ---------------------------------------------------------------------------
// Plan tests (verbatim from Task 2 spec)
// ---------------------------------------------------------------------------

func TestJWTMiddleware_ExtractsViewerRole(t *testing.T) {
	viewerID := uuid.New()
	secret := "test-secret"
	token := signTestToken(t, secret, viewerID.String())

	var seen uuid.UUID
	handler := JWTMiddleware(secret)(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			seen, _ = permissions.ViewerRoleFrom(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if seen != viewerID {
		t.Errorf("viewer = %v, want %v", seen, viewerID)
	}
}

func TestJWTMiddleware_RejectsMissingHeader(t *testing.T) {
	handler := JWTMiddleware("secret")(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestJWTMiddleware_RejectsBadSubject(t *testing.T) {
	secret := "test-secret"
	token := signTestToken(t, secret, "not-a-uuid")
	handler := JWTMiddleware(secret)(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Augmentation tests
// ---------------------------------------------------------------------------

// TestJWTMiddleware_RejectsAlgNone confirms the classic alg:none JWT
// vulnerability (CVE pattern) is blocked. The strict HS256-only key function
// rejects any token whose signing method != HS256, which covers alg=none.
func TestJWTMiddleware_RejectsAlgNone(t *testing.T) {
	// Craft a raw alg=none token manually. jwt/v5 refuses to sign with none,
	// so we construct the three-part JWT string directly.
	viewerID := uuid.New()

	headerJSON, _ := json.Marshal(map[string]string{"alg": "none", "typ": "JWT"})
	payloadJSON, _ := json.Marshal(map[string]string{"sub": viewerID.String()})

	enc := base64.RawURLEncoding
	algNoneToken := strings.Join([]string{
		enc.EncodeToString(headerJSON),
		enc.EncodeToString(payloadJSON),
		"", // empty signature — the "none" algorithm
	}, ".")

	handler := JWTMiddleware("some-secret")(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+algNoneToken)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("alg=none: status = %d, want 401", w.Code)
	}
}

// TestJWTMiddleware_RejectsExpiredToken confirms jwt/v5 automatic exp enforcement.
// An expired token (exp in the past) must yield 401 regardless of signature validity.
func TestJWTMiddleware_RejectsExpiredToken(t *testing.T) {
	secret := "test-secret"
	past := time.Now().Add(-1 * time.Hour).Unix()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": uuid.New().String(),
		"exp": past,
	})
	tokenStr, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	handler := JWTMiddleware(secret)(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expired token: status = %d, want 401", w.Code)
	}
}

// TestJWTMiddleware_PanicsOnEmptySecret confirms the operator misconfiguration
// guard: JWTMiddleware panics at construction time if the secret is empty.
// This ensures server startup fails loudly rather than silently accepting
// any token (which is what an empty HMAC key would do).
func TestJWTMiddleware_PanicsOnEmptySecret(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty secret, got none")
		}
	}()
	JWTMiddleware("") // must panic before returning
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

// signTestToken signs a JWT with HS256 using secret and sub claim.
// No exp is set — the token is valid indefinitely (acceptable in test context).
//
// Note: tokens belong in Authorization headers only; URL query parameters
// are intentionally unsupported (see JWTMiddleware godoc).
func signTestToken(t *testing.T, secret, sub string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": sub})
	s, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}
