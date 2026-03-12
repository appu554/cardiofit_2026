package grpc

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"github.com/sirupsen/logrus"

	"global-outbox-service-go/internal/config"
	"global-outbox-service-go/internal/database"
	"global-outbox-service-go/internal/database/models"
	"global-outbox-service-go/internal/circuitbreaker"
	pb "global-outbox-service-go/pkg/proto"
)

// Server implements the gRPC OutboxService
type Server struct {
	pb.UnimplementedOutboxServiceServer
	repo           *database.Repository
	circuitBreaker *circuitbreaker.MedicalCircuitBreaker
	config         *config.Config
	logger         *logrus.Logger
	grpcServer     *grpc.Server
}

// NewServer creates a new gRPC server
func NewServer(
	repo *database.Repository,
	circuitBreaker *circuitbreaker.MedicalCircuitBreaker,
	config *config.Config,
	logger *logrus.Logger,
) *Server {
	return &Server{
		repo:           repo,
		circuitBreaker: circuitBreaker,
		config:         config,
		logger:         logger,
	}
}

// Start starts the gRPC server
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.config.GRPCPort, err)
	}

	// Create gRPC server with options
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(4 * 1024 * 1024), // 4MB
		grpc.MaxSendMsgSize(4 * 1024 * 1024), // 4MB
	}

	s.grpcServer = grpc.NewServer(opts...)
	pb.RegisterOutboxServiceServer(s.grpcServer, s)

	s.logger.Infof("Starting gRPC server on port %d", s.config.GRPCPort)
	
	// Start serving in a goroutine
	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			s.logger.Errorf("gRPC server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the gRPC server
func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.logger.Info("Stopping gRPC server...")
		s.grpcServer.GracefulStop()
	}
}

// PublishEvent publishes an event to the outbox
func (s *Server) PublishEvent(ctx context.Context, req *pb.PublishEventRequest) (*pb.PublishEventResponse, error) {
	s.logger.Debugf("Received PublishEvent request for service: %s, event_type: %s", 
		req.ServiceName, req.EventType)

	// Validate request
	if err := s.validatePublishEventRequest(req); err != nil {
		s.logger.Warnf("Invalid PublishEvent request: %v", err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	// Parse medical context
	medicalContext := models.MedicalContext(req.MedicalContext)
	if medicalContext == "" {
		medicalContext = models.MedicalContextRoutine
	}

	// Create outbox event
	event := models.NewOutboxEvent(
		req.ServiceName,
		req.EventType,
		req.EventData,
		req.Topic,
		req.Priority,
		medicalContext,
	)

	// Set optional fields
	if req.CorrelationId != "" {
		event.CorrelationID = &req.CorrelationId
	}

	if req.Metadata != nil {
		metadata := make(models.Metadata)
		for k, v := range req.Metadata {
			metadata[k] = v
		}
		event.Metadata = metadata
	}

	// Insert event into database
	if err := s.repo.InsertEvent(ctx, event); err != nil {
		s.logger.Errorf("Failed to insert event: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to store event: %v", err)
	}

	s.logger.Infof("Successfully stored event %s for service %s", event.ID, event.ServiceName)

	// Return success response
	return &pb.PublishEventResponse{
		Success:   true,
		Message:   "Event successfully queued for publishing",
		EventId:   event.ID.String(),
		CreatedAt: event.CreatedAt.Unix(),
	}, nil
}

// HealthCheck returns the health status of the service
func (s *Server) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	s.logger.Debug("Received HealthCheck request")

	// Check database health
	dbHealth := s.repo.HealthCheck(ctx)
	
	status := "SERVING"
	message := "Service is healthy"
	details := make(map[string]string)

	// Check database status
	if dbHealthStatus, ok := dbHealth["status"].(string); ok && dbHealthStatus != "healthy" {
		status = "NOT_SERVING"
		message = "Database is unhealthy"
		if dbError, exists := dbHealth["error"].(string); exists {
			details["database_error"] = dbError
		}
	} else {
		details["database"] = "healthy"
	}

	// Add circuit breaker status
	cbStatus := s.circuitBreaker.GetStatus()
	details["circuit_breaker_enabled"] = strconv.FormatBool(cbStatus.Enabled)
	details["circuit_breaker_state"] = string(cbStatus.State)

	return &pb.HealthCheckResponse{
		Status:  status,
		Message: message,
		Details: details,
	}, nil
}

// GetOutboxStats returns statistics about the outbox queues
func (s *Server) GetOutboxStats(ctx context.Context, req *pb.GetOutboxStatsRequest) (*pb.GetOutboxStatsResponse, error) {
	s.logger.Debug("Received GetOutboxStats request")

	// Get outbox statistics
	stats, err := s.repo.GetOutboxStats(ctx)
	if err != nil {
		s.logger.Errorf("Failed to get outbox stats: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get statistics: %v", err)
	}

	// Get circuit breaker status
	cbStatus := s.circuitBreaker.GetStatus()

	// Convert to protobuf response
	response := &pb.GetOutboxStatsResponse{
		QueueDepths:       stats.QueueDepths,
		TotalProcessed_24H: stats.TotalProcessed24h,
		DeadLetterCount:   stats.DeadLetterCount,
		SuccessRates:      stats.SuccessRates,
		CircuitBreaker: &pb.CircuitBreakerStatus{
			Enabled:                  cbStatus.Enabled,
			State:                    string(cbStatus.State),
			CurrentLoad:              cbStatus.CurrentLoad,
			TotalRequests:            cbStatus.TotalRequests,
			FailedRequests:           cbStatus.FailedRequests,
			CriticalEventsProcessed:  cbStatus.CriticalEventsProcessed,
			NonCriticalEventsDropped: cbStatus.NonCriticalEventsDropped,
		},
	}

	if cbStatus.NextRetryAt != nil {
		response.CircuitBreaker.NextRetryAt = cbStatus.NextRetryAt.Unix()
	}

	return response, nil
}

// validatePublishEventRequest validates the publish event request
func (s *Server) validatePublishEventRequest(req *pb.PublishEventRequest) error {
	if req.ServiceName == "" {
		return fmt.Errorf("service_name is required")
	}

	if req.EventType == "" {
		return fmt.Errorf("event_type is required")
	}

	if req.EventData == "" {
		return fmt.Errorf("event_data is required")
	}

	if req.Topic == "" {
		return fmt.Errorf("topic is required")
	}

	// Validate priority range
	if req.Priority < 1 || req.Priority > 10 {
		req.Priority = 5 // Default priority
	}

	// Validate medical context
	medicalContext := models.MedicalContext(req.MedicalContext)
	validContexts := []models.MedicalContext{
		models.MedicalContextCritical,
		models.MedicalContextUrgent,
		models.MedicalContextRoutine,
		models.MedicalContextBackground,
	}

	if req.MedicalContext != "" {
		valid := false
		for _, validContext := range validContexts {
			if medicalContext == validContext {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid medical_context: %s", req.MedicalContext)
		}
	}

	return nil
}