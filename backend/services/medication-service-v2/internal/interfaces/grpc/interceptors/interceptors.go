package interceptors

import (
	"context"
	"fmt"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor provides request/response logging
type LoggingInterceptor struct {
	logger *zap.Logger
}

// MetricsInterceptor provides request metrics collection
type MetricsInterceptor struct {
	requestCount    map[string]int64
	requestDuration map[string]time.Duration
	errorCount      map[string]int64
	mu              sync.RWMutex
}

// RecoveryInterceptor provides panic recovery
type RecoveryInterceptor struct {
	logger *zap.Logger
}

// RateLimitInterceptor provides rate limiting
type RateLimitInterceptor struct {
	limiter   *rate.Limiter
	burst     int
	mu        sync.RWMutex
	clientMap map[string]*rate.Limiter
}

// NewLoggingInterceptor creates a new logging interceptor
func NewLoggingInterceptor(logger *zap.Logger) *LoggingInterceptor {
	return &LoggingInterceptor{
		logger: logger,
	}
}

// UnaryServerInterceptor returns logging interceptor for unary calls
func (i *LoggingInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		// Extract client information
		clientInfo := i.extractClientInfo(ctx)
		
		i.logger.Info("gRPC request started",
			zap.String("method", info.FullMethod),
			zap.String("client_ip", clientInfo.IP),
			zap.String("user_agent", clientInfo.UserAgent))

		// Call handler
		resp, err := handler(ctx, req)
		
		duration := time.Since(startTime)
		
		// Log response
		if err != nil {
			i.logger.Error("gRPC request failed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.Error(err),
				zap.String("client_ip", clientInfo.IP))
		} else {
			i.logger.Info("gRPC request completed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.String("client_ip", clientInfo.IP))
		}

		return resp, err
	}
}

// StreamServerInterceptor returns logging interceptor for stream calls
func (i *LoggingInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		startTime := time.Now()

		// Extract client information
		clientInfo := i.extractClientInfo(stream.Context())
		
		i.logger.Info("gRPC stream started",
			zap.String("method", info.FullMethod),
			zap.String("client_ip", clientInfo.IP),
			zap.String("user_agent", clientInfo.UserAgent))

		// Call handler
		err := handler(srv, stream)
		
		duration := time.Since(startTime)
		
		// Log completion
		if err != nil {
			i.logger.Error("gRPC stream failed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.Error(err),
				zap.String("client_ip", clientInfo.IP))
		} else {
			i.logger.Info("gRPC stream completed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.String("client_ip", clientInfo.IP))
		}

		return err
	}
}

// ClientInfo holds client connection information
type ClientInfo struct {
	IP        string
	UserAgent string
}

// extractClientInfo extracts client information from context
func (i *LoggingInterceptor) extractClientInfo(ctx context.Context) ClientInfo {
	clientInfo := ClientInfo{}

	// Extract peer information
	if peer, ok := peer.FromContext(ctx); ok {
		clientInfo.IP = peer.Addr.String()
	}

	// Extract metadata
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if userAgent := md.Get("user-agent"); len(userAgent) > 0 {
			clientInfo.UserAgent = userAgent[0]
		}
		// Check for forwarded IP
		if forwardedFor := md.Get("x-forwarded-for"); len(forwardedFor) > 0 {
			clientInfo.IP = forwardedFor[0]
		} else if realIP := md.Get("x-real-ip"); len(realIP) > 0 {
			clientInfo.IP = realIP[0]
		}
	}

	return clientInfo
}

// NewMetricsInterceptor creates a new metrics interceptor
func NewMetricsInterceptor() *MetricsInterceptor {
	return &MetricsInterceptor{
		requestCount:    make(map[string]int64),
		requestDuration: make(map[string]time.Duration),
		errorCount:      make(map[string]int64),
	}
}

// UnaryServerInterceptor returns metrics interceptor for unary calls
func (i *MetricsInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		// Call handler
		resp, err := handler(ctx, req)
		
		duration := time.Since(startTime)
		method := info.FullMethod

		// Update metrics
		i.mu.Lock()
		i.requestCount[method]++
		i.requestDuration[method] += duration
		if err != nil {
			i.errorCount[method]++
		}
		i.mu.Unlock()

		return resp, err
	}
}

// StreamServerInterceptor returns metrics interceptor for stream calls
func (i *MetricsInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		startTime := time.Now()

		// Call handler
		err := handler(srv, stream)
		
		duration := time.Since(startTime)
		method := info.FullMethod

		// Update metrics
		i.mu.Lock()
		i.requestCount[method]++
		i.requestDuration[method] += duration
		if err != nil {
			i.errorCount[method]++
		}
		i.mu.Unlock()

		return err
	}
}

// GetMetrics returns current metrics snapshot
func (i *MetricsInterceptor) GetMetrics() map[string]interface{} {
	i.mu.RLock()
	defer i.mu.RUnlock()

	metrics := make(map[string]interface{})
	
	for method, count := range i.requestCount {
		methodMetrics := map[string]interface{}{
			"request_count":   count,
			"error_count":     i.errorCount[method],
			"total_duration":  i.requestDuration[method],
		}
		
		if count > 0 {
			methodMetrics["average_duration"] = i.requestDuration[method] / time.Duration(count)
			methodMetrics["error_rate"] = float64(i.errorCount[method]) / float64(count)
		}
		
		metrics[method] = methodMetrics
	}

	return metrics
}

// ResetMetrics clears all metrics
func (i *MetricsInterceptor) ResetMetrics() {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.requestCount = make(map[string]int64)
	i.requestDuration = make(map[string]time.Duration)
	i.errorCount = make(map[string]int64)
}

// NewRecoveryInterceptor creates a new recovery interceptor
func NewRecoveryInterceptor(logger *zap.Logger) *RecoveryInterceptor {
	return &RecoveryInterceptor{
		logger: logger,
	}
}

// UnaryServerInterceptor returns recovery interceptor for unary calls
func (i *RecoveryInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				
				i.logger.Error("Panic recovered in gRPC handler",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.String("stack", string(stack)))

				// Return internal server error
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns recovery interceptor for stream calls
func (i *RecoveryInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				
				i.logger.Error("Panic recovered in gRPC stream handler",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
					zap.String("stack", string(stack)))

				// Return internal server error
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		return handler(srv, stream)
	}
}

// NewRateLimitInterceptor creates a new rate limiting interceptor
func NewRateLimitInterceptor(requestsPerSecond float64, burst int) *RateLimitInterceptor {
	return &RateLimitInterceptor{
		limiter:   rate.NewLimiter(rate.Limit(requestsPerSecond), burst),
		burst:     burst,
		clientMap: make(map[string]*rate.Limiter),
	}
}

// UnaryServerInterceptor returns rate limiting interceptor for unary calls
func (i *RateLimitInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Get client-specific limiter
		clientLimiter := i.getClientLimiter(ctx)
		
		// Check rate limit
		if !clientLimiter.Allow() {
			return nil, status.Errorf(codes.ResourceExhausted, 
				"rate limit exceeded: too many requests")
		}

		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns rate limiting interceptor for stream calls
func (i *RateLimitInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Get client-specific limiter
		clientLimiter := i.getClientLimiter(stream.Context())
		
		// Check rate limit
		if !clientLimiter.Allow() {
			return status.Errorf(codes.ResourceExhausted, 
				"rate limit exceeded: too many requests")
		}

		return handler(srv, stream)
	}
}

// getClientLimiter returns a rate limiter for the client
func (i *RateLimitInterceptor) getClientLimiter(ctx context.Context) *rate.Limiter {
	clientIP := i.extractClientIP(ctx)
	
	i.mu.RLock()
	limiter, exists := i.clientMap[clientIP]
	i.mu.RUnlock()
	
	if !exists {
		// Create new limiter for this client
		limiter = rate.NewLimiter(rate.Limit(100), i.burst) // Per-client limit
		
		i.mu.Lock()
		i.clientMap[clientIP] = limiter
		i.mu.Unlock()
	}
	
	return limiter
}

// extractClientIP extracts client IP from context
func (i *RateLimitInterceptor) extractClientIP(ctx context.Context) string {
	// Try to get real client IP from metadata first
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if forwardedFor := md.Get("x-forwarded-for"); len(forwardedFor) > 0 {
			return forwardedFor[0]
		}
		if realIP := md.Get("x-real-ip"); len(realIP) > 0 {
			return realIP[0]
		}
	}
	
	// Fallback to peer address
	if peer, ok := peer.FromContext(ctx); ok {
		return peer.Addr.String()
	}
	
	return "unknown"
}

// Cleanup removes old client limiters to prevent memory leaks
func (i *RateLimitInterceptor) Cleanup() {
	i.mu.Lock()
	defer i.mu.Unlock()
	
	// In production, you might want to track last access time
	// and remove limiters that haven't been used for a while
	// For now, we'll keep it simple
	if len(i.clientMap) > 10000 { // Arbitrary limit
		// Clear half of the map
		newMap := make(map[string]*rate.Limiter)
		count := 0
		for k, v := range i.clientMap {
			if count < 5000 {
				newMap[k] = v
				count++
			} else {
				break
			}
		}
		i.clientMap = newMap
	}
}

// ValidationInterceptor provides request validation
type ValidationInterceptor struct {
	logger *zap.Logger
}

// NewValidationInterceptor creates a new validation interceptor
func NewValidationInterceptor(logger *zap.Logger) *ValidationInterceptor {
	return &ValidationInterceptor{
		logger: logger,
	}
}

// UnaryServerInterceptor returns validation interceptor for unary calls
func (i *ValidationInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Validate request based on method
		if err := i.validateRequest(req, info.FullMethod); err != nil {
			i.logger.Warn("Request validation failed",
				zap.String("method", info.FullMethod),
				zap.Error(err))
			return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
		}

		return handler(ctx, req)
	}
}

// validateRequest performs basic request validation
func (i *ValidationInterceptor) validateRequest(req interface{}, method string) error {
	// Add validation logic based on the request type and method
	// This is a placeholder implementation
	
	// Example: Check for nil requests
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	// Add more specific validation logic here
	// You might want to use a validation library like go-playground/validator
	
	return nil
}

// TraceInterceptor provides distributed tracing support
type TraceInterceptor struct {
	logger *zap.Logger
}

// NewTraceInterceptor creates a new trace interceptor
func NewTraceInterceptor(logger *zap.Logger) *TraceInterceptor {
	return &TraceInterceptor{
		logger: logger,
	}
}

// UnaryServerInterceptor returns tracing interceptor for unary calls
func (i *TraceInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract trace information from metadata
		traceID := i.extractTraceID(ctx)
		
		// Add trace ID to logger context
		logger := i.logger.With(zap.String("trace_id", traceID))
		
		logger.Debug("Processing gRPC request",
			zap.String("method", info.FullMethod))

		return handler(ctx, req)
	}
}

// extractTraceID extracts trace ID from metadata
func (i *TraceInterceptor) extractTraceID(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if traceID := md.Get("x-trace-id"); len(traceID) > 0 {
			return traceID[0]
		}
		if traceID := md.Get("traceid"); len(traceID) > 0 {
			return traceID[0]
		}
	}
	
	// Generate new trace ID if not provided
	return generateTraceID()
}

// generateTraceID generates a new trace ID
func generateTraceID() string {
	// Simple trace ID generation (in production, use proper trace ID format)
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// HIPAAInterceptor provides HIPAA compliance logging and auditing
type HIPAAInterceptor struct {
	logger *zap.Logger
}

// NewHIPAAInterceptor creates a new HIPAA compliance interceptor
func NewHIPAAInterceptor(logger *zap.Logger) *HIPAAInterceptor {
	return &HIPAAInterceptor{
		logger: logger,
	}
}

// UnaryServerInterceptor returns HIPAA interceptor for unary calls
func (i *HIPAAInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Log HIPAA audit event
		i.logHIPAAEvent(ctx, "grpc_request", info.FullMethod, map[string]interface{}{
			"request_type": "unary",
		})

		return handler(ctx, req)
	}
}

// logHIPAAEvent logs HIPAA compliance audit events
func (i *HIPAAInterceptor) logHIPAAEvent(ctx context.Context, eventType, method string, metadata map[string]interface{}) {
	auditData := map[string]interface{}{
		"event_type":   eventType,
		"method":       method,
		"timestamp":    time.Now().UTC(),
		"service":      "medication-service-v2",
		"component":    "grpc-server",
	}

	// Add custom metadata
	for k, v := range metadata {
		auditData[k] = v
	}

	// Extract client information
	if peer, ok := peer.FromContext(ctx); ok {
		auditData["client_address"] = peer.Addr.String()
	}

	// Extract user information from auth context if available
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if userID := md.Get("x-user-id"); len(userID) > 0 {
			auditData["user_id"] = userID[0]
		}
	}

	i.logger.Info("HIPAA Audit Event",
		zap.String("audit_type", "access"),
		zap.Any("audit_data", auditData))
}