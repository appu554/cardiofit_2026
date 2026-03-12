package server

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Prometheus metrics for HTTP requests
var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "notification_http_request_duration_seconds",
			Help:    "HTTP request latency",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	grpcRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_grpc_requests_total",
			Help: "Total number of gRPC requests",
		},
		[]string{"method", "status"},
	)

	grpcRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "notification_grpc_request_duration_seconds",
			Help:    "gRPC request latency",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	notificationDeliveriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_deliveries_total",
			Help: "Total number of notification deliveries",
		},
		[]string{"channel", "status"},
	)

	escalationEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_escalation_events_total",
			Help: "Total number of escalation events",
		},
		[]string{"level"},
	)

	alertFatigueSuppressions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_alert_fatigue_suppressions_total",
			Help: "Total number of alerts suppressed due to fatigue",
		},
		[]string{"reason"},
	)

	kafkaMessagesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_kafka_messages_processed_total",
			Help: "Total number of Kafka messages processed",
		},
		[]string{"topic", "status"},
	)
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	requestIDKey contextKey = "request-id"
	userIDKey    contextKey = "user-id"
)

// HTTP Middleware Functions

// LoggingMiddleware logs HTTP requests with structured logging
func LoggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Generate or extract request ID
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}

			// Add request ID to context
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			r = r.WithContext(ctx)

			// Add request ID to response header
			w.Header().Set("X-Request-ID", requestID)

			// Create response wrapper to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(rw, r)

			// Log request details
			duration := time.Since(start)
			logger.Info("HTTP request",
				zap.String("request_id", requestID),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
				zap.Int("status", rw.statusCode),
				zap.Duration("duration", duration),
				zap.String("user_agent", r.UserAgent()),
			)
		})
	}
}

// MetricsMiddleware collects HTTP request metrics
func MetricsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create response wrapper to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(rw, r)

			// Record metrics
			duration := time.Since(start).Seconds()
			statusCode := fmt.Sprintf("%d", rw.statusCode)

			httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, statusCode).Inc()
			httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
		})
	}
}

// CORSMiddleware adds CORS headers to HTTP responses
func CORSMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			w.Header().Set("Access-Control-Max-Age", "3600")

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// TimeoutMiddleware adds request timeout handling
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)

			done := make(chan bool)
			go func() {
				next.ServeHTTP(w, r)
				done <- true
			}()

			select {
			case <-done:
				return
			case <-ctx.Done():
				w.WriteHeader(http.StatusRequestTimeout)
				w.Write([]byte(`{"error": "Request timeout"}`))
				return
			}
		})
	}
}

// RecoveryMiddleware recovers from panics in HTTP handlers
func RecoveryMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Get request ID from context
					requestID := ""
					if id := r.Context().Value(requestIDKey); id != nil {
						requestID = id.(string)
					}

					// Log panic with stack trace
					logger.Error("Panic recovered in HTTP handler",
						zap.String("request_id", requestID),
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.Any("error", err),
						zap.String("stack", string(debug.Stack())),
					)

					// Return error response
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error": "Internal server error"}`))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// gRPC Interceptor Functions

// UnaryLoggingInterceptor logs gRPC requests
func UnaryLoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Generate or extract request ID
		requestID := generateRequestID()
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			if ids := md.Get("request-id"); len(ids) > 0 {
				requestID = ids[0]
			}
		}

		// Add request ID to context
		ctx = context.WithValue(ctx, requestIDKey, requestID)

		// Add request ID to outgoing metadata
		header := metadata.New(map[string]string{"request-id": requestID})
		grpc.SendHeader(ctx, header)

		// Process request
		resp, err := handler(ctx, req)

		// Log request details
		duration := time.Since(start)
		statusCode := codes.OK
		if err != nil {
			statusCode = status.Code(err)
		}

		logger.Info("gRPC request",
			zap.String("request_id", requestID),
			zap.String("method", info.FullMethod),
			zap.String("status", statusCode.String()),
			zap.Duration("duration", duration),
		)

		return resp, err
	}
}

// UnaryMetricsInterceptor collects gRPC request metrics
func UnaryMetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Process request
		resp, err := handler(ctx, req)

		// Record metrics
		duration := time.Since(start).Seconds()
		statusCode := codes.OK
		if err != nil {
			statusCode = status.Code(err)
		}

		grpcRequestsTotal.WithLabelValues(info.FullMethod, statusCode.String()).Inc()
		grpcRequestDuration.WithLabelValues(info.FullMethod).Observe(duration)

		return resp, err
	}
}

// UnaryAuthInterceptor validates JWT tokens for gRPC requests
func UnaryAuthInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip authentication for health check methods
		if info.FullMethod == "/grpc.health.v1.Health/Check" {
			return handler(ctx, req)
		}

		// Extract metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		// Extract authorization token
		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "missing authorization token")
		}

		// TODO: Implement JWT validation
		// For now, we'll extract a mock user ID
		// In production, this should validate the JWT and extract the user ID
		userID := "mock-user-id"

		// Add user ID to context
		ctx = context.WithValue(ctx, userIDKey, userID)

		return handler(ctx, req)
	}
}

// UnaryRecoveryInterceptor recovers from panics in gRPC handlers
func UnaryRecoveryInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				// Get request ID from context
				requestID := ""
				if id := ctx.Value(requestIDKey); id != nil {
					requestID = id.(string)
				}

				// Log panic with stack trace
				logger.Error("Panic recovered in gRPC handler",
					zap.String("request_id", requestID),
					zap.String("method", info.FullMethod),
					zap.Any("error", r),
					zap.String("stack", string(debug.Stack())),
				)

				// Return error response
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// Helper Functions

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return uuid.New().String()
}

// responseWriter is a wrapper around http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RecordNotificationDelivery records a notification delivery metric
func RecordNotificationDelivery(channel, status string) {
	notificationDeliveriesTotal.WithLabelValues(channel, status).Inc()
}

// RecordEscalationEvent records an escalation event metric
func RecordEscalationEvent(level string) {
	escalationEventsTotal.WithLabelValues(level).Inc()
}

// RecordAlertFatigueSuppression records an alert fatigue suppression metric
func RecordAlertFatigueSuppression(reason string) {
	alertFatigueSuppressions.WithLabelValues(reason).Inc()
}

// RecordKafkaMessageProcessed records a Kafka message processing metric
func RecordKafkaMessageProcessed(topic, status string) {
	kafkaMessagesProcessed.WithLabelValues(topic, status).Inc()
}
