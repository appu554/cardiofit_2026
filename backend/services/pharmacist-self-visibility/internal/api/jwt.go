package api

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/permissions"
)

// JWTMiddleware returns an HTTP middleware that validates a Bearer JWT in the
// Authorization header and places the viewer-role UUID (from the `sub` claim)
// into the request context via permissions.WithViewerRole.
//
// Algorithm: strictly HS256. Tokens signed with any other algorithm — including
// the "alg:none" bypass (CVE pattern) — are rejected with 401.
//
// Expiry: jwt/v5 enforces the `exp` claim automatically. Expired tokens yield 401.
//
// Token location: Authorization header only (Bearer scheme). Query-parameter
// delivery is intentionally unsupported — tokens must not appear in URLs because
// they leak into access logs, browser history, and Referer headers.
//
// Startup guard: panics if secret is empty. This surfaces operator
// misconfiguration at server-start time rather than silently accepting all tokens.
func JWTMiddleware(secret string) func(http.Handler) http.Handler {
	if secret == "" {
		panic("JWTMiddleware: secret must not be empty — set JWT_SECRET in the environment")
	}
	secretBytes := []byte(secret)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract Bearer token from Authorization header.
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				WriteError(w, http.StatusUnauthorized, "missing_bearer", "Authorization Bearer token required")
				return
			}
			tokenStr := strings.TrimPrefix(auth, "Bearer ")

			// Parse and validate the token. The key function enforces HS256 strictly;
			// returning an error for any other method prevents algorithm-confusion attacks.
			tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
				if t.Method != jwt.SigningMethodHS256 {
					return nil, jwt.ErrTokenSignatureInvalid
				}
				return secretBytes, nil
			})
			if err != nil || !tok.Valid {
				WriteError(w, http.StatusUnauthorized, "invalid_token", "JWT verification failed")
				return
			}

			// Extract claims and validate the sub claim as a UUID.
			claims, ok := tok.Claims.(jwt.MapClaims)
			if !ok {
				WriteError(w, http.StatusUnauthorized, "invalid_claims", "claims malformed")
				return
			}
			sub, _ := claims["sub"].(string)
			viewerID, err := uuid.Parse(sub)
			if err != nil {
				WriteError(w, http.StatusUnauthorized, "invalid_subject", "sub must be a UUID")
				return
			}

			// Stuff viewer role into context for downstream permission checks.
			ctx := permissions.WithViewerRole(r.Context(), viewerID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
