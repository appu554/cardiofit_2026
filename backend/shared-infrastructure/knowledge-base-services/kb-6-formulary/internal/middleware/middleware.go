package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"sync"
	"time"
)

// Middleware represents a middleware function
type Middleware func(http.Handler) http.Handler

// Chain applies multiple middleware functions in sequence
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// RequestLogging logs HTTP requests and responses
func RequestLogging() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// Wrap the response writer to capture status
			wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
			
			next.ServeHTTP(wrapped, r)
			
			duration := time.Since(start)
			log.Printf("HTTP %s %s - %d - %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
		})
	}
}

// CORS adds Cross-Origin Resource Sharing headers
func CORS() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Max-Age", "86400")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit implements basic rate limiting per IP
func RateLimit() Middleware {
	type client struct {
		requests int
		lastSeen time.Time
	}
	
	clients := make(map[string]*client)
	mu := sync.Mutex{}
	
	// Clean up old entries every minute
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			mu.Lock()
			now := time.Now()
			for ip, c := range clients {
				if now.Sub(c.lastSeen) > 1*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)
			
			mu.Lock()
			c, exists := clients[ip]
			if !exists {
				c = &client{requests: 0, lastSeen: time.Now()}
				clients[ip] = c
			}
			
			// Reset counter if more than a minute has passed
			if time.Since(c.lastSeen) > 1*time.Minute {
				c.requests = 0
			}
			
			c.requests++
			c.lastSeen = time.Now()
			
			// Rate limit: 100 requests per minute
			if c.requests > 100 {
				mu.Unlock()
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			mu.Unlock()
			
			next.ServeHTTP(w, r)
		})
	}
}

// RequestTimeout adds a timeout to requests
func RequestTimeout(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// Recovery recovers from panics and returns a 500 error
func Recovery() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("Panic recovered: %v\n%s", err, debug.Stack())
					
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					
					errorResponse := map[string]interface{}{
						"error":     "Internal server error",
						"timestamp": time.Now().UTC(),
						"path":      r.URL.Path,
					}
					
					json.NewEncoder(w).Encode(errorResponse)
				}
			}()
			
			next.ServeHTTP(w, r)
		})
	}
}

// MetricsHandler provides a basic metrics endpoint
func MetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics := map[string]interface{}{
			"service":        "kb-6-formulary",
			"version":        "1.0.0",
			"uptime_seconds": time.Since(startTime).Seconds(),
			"timestamp":      time.Now().UTC(),
			"go_version":     "go1.21+",
			"status":         "healthy",
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
	}
}

// Authentication middleware for Bearer token validation
func Authentication() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for health checks and public endpoints
			if r.URL.Path == "/health" || r.URL.Path == "/metrics" || r.URL.Path == "/" {
				next.ServeHTTP(w, r)
				return
			}
			
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}
			
			// Basic Bearer token validation (in production, integrate with auth service)
			if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}
			
			token := authHeader[7:]
			if !isValidToken(token) {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// getClientIP extracts client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// isValidToken validates Bearer tokens (placeholder implementation)
func isValidToken(token string) bool {
	// In production, this would validate against your auth service
	// For now, accept any non-empty token longer than 10 characters
	return len(token) > 10
}

// startTime tracks when the service started for uptime metrics
var startTime = time.Now()

// RequestID middleware adds a unique request ID to each request
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := generateRequestID()
			w.Header().Set("X-Request-ID", requestID)
			
			// Add to context for use in handlers
			ctx := context.WithValue(r.Context(), "request_id", requestID)
			r = r.WithContext(ctx)
			
			next.ServeHTTP(w, r)
		})
	}
}

// generateRequestID creates a simple request ID
func generateRequestID() string {
	return fmt.Sprintf("kb6-%d", time.Now().UnixNano())
}

// Security headers middleware
func SecurityHeaders() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			
			next.ServeHTTP(w, r)
		})
	}
}