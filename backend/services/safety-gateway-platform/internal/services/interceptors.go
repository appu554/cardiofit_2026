package services

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/pkg/logger"
)

// LoggingInterceptor provides request/response logging
func LoggingInterceptor(logger *logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		// Extract metadata
		md, _ := metadata.FromIncomingContext(ctx)
		requestID := getMetadataValue(md, "request-id")
		clientID := getMetadataValue(md, "client-id")

		requestLogger := logger
		if requestID != "" {
			requestLogger = logger.WithRequestID(requestID)
		}

		requestLogger.Info("gRPC request started",
			zap.String("method", info.FullMethod),
			zap.String("client_id", clientID),
		)

		// Call handler
		resp, err := handler(ctx, req)

		// Log completion
		duration := time.Since(startTime)
		if err != nil {
			requestLogger.Error("gRPC request failed",
				zap.String("method", info.FullMethod),
				zap.Int64("duration_ms", duration.Milliseconds()),
				zap.String("error", err.Error()),
			)
		} else {
			requestLogger.Info("gRPC request completed",
				zap.String("method", info.FullMethod),
				zap.Int64("duration_ms", duration.Milliseconds()),
			)
		}

		return resp, err
	}
}

// MetricsInterceptor provides request metrics collection
func MetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		// Call handler
		resp, err := handler(ctx, req)

		// Record metrics (this would integrate with Prometheus in production)
		duration := time.Since(startTime)
		
		// Extract status code
		statusCode := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				statusCode = st.Code()
			}
		}

		// Log metrics (in production, this would be sent to Prometheus)
		// For now, we'll just log the metrics
		// metricsLogger.Info("grpc_request_metrics",
		//     "method", info.FullMethod,
		//     "status_code", statusCode.String(),
		//     "duration_seconds", duration.Seconds(),
		// )

		_ = statusCode // Suppress unused variable warning
		_ = duration   // Suppress unused variable warning

		return resp, err
	}
}

// AuthInterceptor provides authentication and authorization
func AuthInterceptor(cfg *config.Config) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip auth for health checks
		if info.FullMethod == "/grpc.health.v1.Health/Check" ||
			info.FullMethod == "/safety_gateway.SafetyGateway/GetHealth" {
			return handler(ctx, req)
		}

		// Extract metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		// Check for authorization header
		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "missing authorization header")
		}

		// In production, this would validate the token
		// For now, we'll just check that it's not empty
		authToken := authHeaders[0]
		if authToken == "" {
			return nil, status.Errorf(codes.Unauthenticated, "empty authorization token")
		}

		// Add user information to context (mock implementation)
		userCtx := context.WithValue(ctx, "user_id", "mock_user")
		userCtx = context.WithValue(userCtx, "user_role", "clinician")

		return handler(userCtx, req)
	}
}

// TimeoutInterceptor enforces request timeouts
func TimeoutInterceptor(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Create context with timeout
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// Channel to receive handler result
		type result struct {
			resp interface{}
			err  error
		}
		resultChan := make(chan result, 1)

		// Run handler in goroutine
		go func() {
			resp, err := handler(timeoutCtx, req)
			resultChan <- result{resp: resp, err: err}
		}()

		// Wait for result or timeout
		select {
		case res := <-resultChan:
			return res.resp, res.err
		case <-timeoutCtx.Done():
			return nil, status.Errorf(codes.DeadlineExceeded, "request timeout exceeded")
		}
	}
}

// RateLimitInterceptor provides rate limiting (basic implementation)
func RateLimitInterceptor() grpc.UnaryServerInterceptor {
	// This would integrate with a proper rate limiter in production
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract client ID from metadata
		md, _ := metadata.FromIncomingContext(ctx)
		clientID := getMetadataValue(md, "client-id")

		// In production, check rate limit for client
		_ = clientID // Suppress unused variable warning

		// For now, just pass through
		return handler(ctx, req)
	}
}

// RecoveryInterceptor provides panic recovery
func RecoveryInterceptor(logger *logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("gRPC handler panic recovered",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// getMetadataValue safely extracts a value from gRPC metadata
func getMetadataValue(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}
