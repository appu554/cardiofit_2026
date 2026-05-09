package api

import "net/http"

// JWTMiddleware returns an HTTP middleware that validates a Bearer JWT and
// places the viewer-role claim into the request context.
//
// STUB (Phase 1b Task 1): This implementation is a passthrough that does NOT
// validate the token. Task 2 will replace this with real JWT verification using
// permissions.WithViewerRole from the shared substrate.
func JWTMiddleware(secret string) func(http.Handler) http.Handler {
	_ = secret // consumed in Task 2
	return func(next http.Handler) http.Handler {
		return next
	}
}
