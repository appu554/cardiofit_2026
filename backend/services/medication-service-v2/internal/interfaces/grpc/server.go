package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/interfaces/grpc/auth"
	"medication-service-v2/internal/interfaces/grpc/interceptors"
	pb "medication-service-v2/proto/medication/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"go.uber.org/zap"
)

// Server represents the gRPC server
type Server struct {
	pb.UnimplementedMedicationServiceServer
	logger   *zap.Logger
	services *services.Services
	grpcServer *grpc.Server
	config   ServerConfig
}

// ServerConfig holds gRPC server configuration
type ServerConfig struct {
	Port                 int           `mapstructure:"port"`
	MaxConnectionIdle    time.Duration `mapstructure:"max_connection_idle"`
	MaxConnectionAge     time.Duration `mapstructure:"max_connection_age"`
	MaxConnectionAgeGrace time.Duration `mapstructure:"max_connection_age_grace"`
	Time                 time.Duration `mapstructure:"time"`
	Timeout              time.Duration `mapstructure:"timeout"`
	EnableReflection     bool          `mapstructure:"enable_reflection"`
	EnableAuth           bool          `mapstructure:"enable_auth"`
	AuthSecret           string        `mapstructure:"auth_secret"`
}

// DefaultServerConfig returns default configuration
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Port:                 50051,
		MaxConnectionIdle:    15 * time.Minute,
		MaxConnectionAge:     30 * time.Minute,
		MaxConnectionAgeGrace: 5 * time.Minute,
		Time:                 10 * time.Second,
		Timeout:              15 * time.Second,
		EnableReflection:     true,
		EnableAuth:           true,
	}
}

// NewServer creates a new gRPC server instance
func NewServer(
	logger *zap.Logger,
	services *services.Services,
	config ServerConfig,
) *Server {
	return &Server{
		logger:   logger,
		services: services,
		config:   config,
	}
}

// Start starts the gRPC server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// Configure keep-alive settings
	kasp := keepalive.ServerParameters{
		MaxConnectionIdle:     s.config.MaxConnectionIdle,
		MaxConnectionAge:      s.config.MaxConnectionAge,
		MaxConnectionAgeGrace: s.config.MaxConnectionAgeGrace,
		Time:                  s.config.Time,
		Timeout:               s.config.Timeout,
	}

	kaep := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second,
		PermitWithoutStream: true,
	}

	// Create interceptors
	authInterceptor := auth.NewAuthInterceptor(s.config.AuthSecret, s.logger)
	loggingInterceptor := interceptors.NewLoggingInterceptor(s.logger)
	metricsInterceptor := interceptors.NewMetricsInterceptor()
	recoveryInterceptor := interceptors.NewRecoveryInterceptor(s.logger)
	rateLimitInterceptor := interceptors.NewRateLimitInterceptor(1000, 100) // 1000 req/sec, burst 100

	// Configure server options
	serverOptions := []grpc.ServerOption{
		grpc.KeepaliveParams(kasp),
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor.UnaryServerInterceptor(),
			loggingInterceptor.UnaryServerInterceptor(),
			metricsInterceptor.UnaryServerInterceptor(),
			rateLimitInterceptor.UnaryServerInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			recoveryInterceptor.StreamServerInterceptor(),
			loggingInterceptor.StreamServerInterceptor(),
			metricsInterceptor.StreamServerInterceptor(),
		),
	}

	// Add auth interceptor if enabled
	if s.config.EnableAuth {
		serverOptions = append(serverOptions,
			grpc.ChainUnaryInterceptor(authInterceptor.UnaryServerInterceptor()),
			grpc.ChainStreamInterceptor(authInterceptor.StreamServerInterceptor()),
		)
	}

	// Create gRPC server
	s.grpcServer = grpc.NewServer(serverOptions...)

	// Register service
	pb.RegisterMedicationServiceServer(s.grpcServer, s)

	// Enable reflection if configured
	if s.config.EnableReflection {
		reflection.Register(s.grpcServer)
	}

	s.logger.Info("Starting gRPC server", 
		zap.String("address", addr),
		zap.Bool("auth_enabled", s.config.EnableAuth),
		zap.Bool("reflection_enabled", s.config.EnableReflection))

	return s.grpcServer.Serve(lis)
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.logger.Info("Shutting down gRPC server")
		s.grpcServer.GracefulStop()
	}
}

// Medication Proposal Management Implementation

func (s *Server) CreateMedicationProposal(ctx context.Context, req *pb.CreateMedicationProposalRequest) (*pb.CreateMedicationProposalResponse, error) {
	s.logger.Debug("Creating medication proposal", zap.String("patient_id", req.PatientId))

	// Convert protobuf to domain models
	clinicalContext, err := convertPBClinicalContext(req.ClinicalContext)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid clinical context: %v", err)
	}

	medicationDetails, err := convertPBMedicationDetails(req.MedicationDetails)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid medication details: %v", err)
	}

	// Create proposal using service
	proposal, err := s.services.MedicationService.CreateProposal(ctx, services.CreateProposalRequest{
		PatientID:         parseUUID(req.PatientId),
		ProtocolID:        req.ProtocolId,
		Indication:        req.Indication,
		ClinicalContext:   clinicalContext,
		MedicationDetails: medicationDetails,
		CreatedBy:         req.CreatedBy,
	})
	if err != nil {
		s.logger.Error("Failed to create medication proposal", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create proposal: %v", err)
	}

	// Convert to protobuf response
	pbProposal, err := convertToProposalPB(proposal)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert proposal: %v", err)
	}

	return &pb.CreateMedicationProposalResponse{
		Proposal: pbProposal,
		Status: &pb.OperationStatus{
			Code:      200,
			Message:   "Proposal created successfully",
			Success:   true,
			Timestamp: timestampNow(),
		},
	}, nil
}

func (s *Server) GetMedicationProposal(ctx context.Context, req *pb.GetMedicationProposalRequest) (*pb.GetMedicationProposalResponse, error) {
	s.logger.Debug("Getting medication proposal", zap.String("proposal_id", req.ProposalId))

	proposalID := parseUUID(req.ProposalId)
	if proposalID == uuid.Nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid proposal ID")
	}

	proposal, err := s.services.MedicationService.GetProposal(ctx, proposalID)
	if err != nil {
		if isNotFoundError(err) {
			return nil, status.Errorf(codes.NotFound, "proposal not found")
		}
		s.logger.Error("Failed to get medication proposal", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get proposal: %v", err)
	}

	pbProposal, err := convertToProposalPB(proposal)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert proposal: %v", err)
	}

	return &pb.GetMedicationProposalResponse{
		Proposal: pbProposal,
		Status: &pb.OperationStatus{
			Code:      200,
			Message:   "Proposal retrieved successfully",
			Success:   true,
			Timestamp: timestampNow(),
		},
	}, nil
}

func (s *Server) UpdateMedicationProposal(ctx context.Context, req *pb.UpdateMedicationProposalRequest) (*pb.UpdateMedicationProposalResponse, error) {
	s.logger.Debug("Updating medication proposal", zap.String("proposal_id", req.ProposalId))

	proposalID := parseUUID(req.ProposalId)
	if proposalID == uuid.Nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid proposal ID")
	}

	// Convert protobuf proposal to domain model
	proposal, err := convertFromProposalPB(req.Proposal)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid proposal data: %v", err)
	}

	updatedProposal, err := s.services.MedicationService.UpdateProposal(ctx, proposalID, services.UpdateProposalRequest{
		Proposal:  proposal,
		UpdatedBy: req.UpdatedBy,
	})
	if err != nil {
		if isNotFoundError(err) {
			return nil, status.Errorf(codes.NotFound, "proposal not found")
		}
		s.logger.Error("Failed to update medication proposal", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to update proposal: %v", err)
	}

	pbProposal, err := convertToProposalPB(updatedProposal)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert proposal: %v", err)
	}

	return &pb.UpdateMedicationProposalResponse{
		Proposal: pbProposal,
		Status: &pb.OperationStatus{
			Code:      200,
			Message:   "Proposal updated successfully",
			Success:   true,
			Timestamp: timestampNow(),
		},
	}, nil
}

func (s *Server) DeleteMedicationProposal(ctx context.Context, req *pb.DeleteMedicationProposalRequest) (*pb.DeleteMedicationProposalResponse, error) {
	s.logger.Debug("Deleting medication proposal", zap.String("proposal_id", req.ProposalId))

	proposalID := parseUUID(req.ProposalId)
	if proposalID == uuid.Nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid proposal ID")
	}

	err := s.services.MedicationService.DeleteProposal(ctx, proposalID)
	if err != nil {
		if isNotFoundError(err) {
			return nil, status.Errorf(codes.NotFound, "proposal not found")
		}
		s.logger.Error("Failed to delete medication proposal", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to delete proposal: %v", err)
	}

	return &pb.DeleteMedicationProposalResponse{
		Status: &pb.OperationStatus{
			Code:      200,
			Message:   "Proposal deleted successfully",
			Success:   true,
			Timestamp: timestampNow(),
		},
	}, nil
}

func (s *Server) ListMedicationProposals(ctx context.Context, req *pb.ListMedicationProposalsRequest) (*pb.ListMedicationProposalsResponse, error) {
	s.logger.Debug("Listing medication proposals", 
		zap.String("patient_id", req.PatientId),
		zap.Int32("page", req.Page),
		zap.Int32("page_size", req.PageSize))

	// Build list request
	listReq := services.ListProposalsRequest{
		PatientID: parseUUIDOptional(req.PatientId),
		Status:    convertProposalStatusFromPB(req.Status),
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	result, err := s.services.MedicationService.ListProposals(ctx, listReq)
	if err != nil {
		s.logger.Error("Failed to list medication proposals", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list proposals: %v", err)
	}

	// Convert proposals to protobuf
	pbProposals := make([]*pb.MedicationProposal, len(result.Proposals))
	for i, proposal := range result.Proposals {
		pbProposal, err := convertToProposalPB(proposal)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert proposal: %v", err)
		}
		pbProposals[i] = pbProposal
	}

	return &pb.ListMedicationProposalsResponse{
		Proposals:  pbProposals,
		TotalCount: int32(result.TotalCount),
		Page:       int32(result.Page),
		PageSize:   int32(result.PageSize),
		Status: &pb.OperationStatus{
			Code:      200,
			Message:   fmt.Sprintf("Found %d proposals", len(pbProposals)),
			Success:   true,
			Timestamp: timestampNow(),
		},
	}, nil
}

func (s *Server) ValidateMedicationProposal(ctx context.Context, req *pb.ValidateMedicationProposalRequest) (*pb.ValidateMedicationProposalResponse, error) {
	s.logger.Debug("Validating medication proposal", zap.String("proposal_id", req.ProposalId))

	proposalID := parseUUID(req.ProposalId)
	if proposalID == uuid.Nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid proposal ID")
	}

	validation, err := s.services.MedicationService.ValidateProposal(ctx, services.ValidateProposalRequest{
		ProposalID:  proposalID,
		ValidatedBy: req.ValidatedBy,
	})
	if err != nil {
		if isNotFoundError(err) {
			return nil, status.Errorf(codes.NotFound, "proposal not found")
		}
		s.logger.Error("Failed to validate medication proposal", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to validate proposal: %v", err)
	}

	pbProposal, err := convertToProposalPB(validation.Proposal)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert proposal: %v", err)
	}

	pbValidationResults := make([]*pb.ValidationResult, len(validation.ValidationResults))
	for i, result := range validation.ValidationResults {
		pbValidationResults[i] = &pb.ValidationResult{
			Field:    result.Field,
			Message:  result.Message,
			Severity: result.Severity,
			Code:     result.Code,
		}
	}

	return &pb.ValidateMedicationProposalResponse{
		Proposal:          pbProposal,
		ValidationResults: pbValidationResults,
		Status: &pb.OperationStatus{
			Code:      200,
			Message:   "Proposal validated successfully",
			Success:   true,
			Timestamp: timestampNow(),
		},
	}, nil
}

// Recipe Resolver Operations Implementation

func (s *Server) ResolveRecipe(ctx context.Context, req *pb.ResolveRecipeRequest) (*pb.ResolveRecipeResponse, error) {
	s.logger.Debug("Resolving recipe", zap.String("protocol_id", req.ProtocolId))

	clinicalContext, err := convertPBClinicalContext(req.ClinicalContext)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid clinical context: %v", err)
	}

	result, err := s.services.RecipeResolverIntegration.ResolveRecipe(ctx, services.ResolveRecipeRequest{
		ProtocolID:      req.ProtocolId,
		ClinicalContext: clinicalContext,
		Parameters:      req.Parameters,
	})
	if err != nil {
		s.logger.Error("Failed to resolve recipe", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to resolve recipe: %v", err)
	}

	// Convert to protobuf
	pbRecipe, err := convertToRecipePB(result.Recipe)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert recipe: %v", err)
	}

	pbDosageRecs := make([]*pb.DosageRecommendation, len(result.DosageRecommendations))
	for i, rec := range result.DosageRecommendations {
		pbRec, err := convertToDosageRecommendationPB(rec)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert dosage recommendation: %v", err)
		}
		pbDosageRecs[i] = pbRec
	}

	pbSafetyConstraints := make([]*pb.SafetyConstraint, len(result.SafetyConstraints))
	for i, constraint := range result.SafetyConstraints {
		pbConstraint, err := convertToSafetyConstraintPB(constraint)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert safety constraint: %v", err)
		}
		pbSafetyConstraints[i] = pbConstraint
	}

	return &pb.ResolveRecipeResponse{
		Recipe:                pbRecipe,
		DosageRecommendations: pbDosageRecs,
		SafetyConstraints:     pbSafetyConstraints,
		Status: &pb.OperationStatus{
			Code:      200,
			Message:   "Recipe resolved successfully",
			Success:   true,
			Timestamp: timestampNow(),
		},
	}, nil
}

// Health Check Implementation

func (s *Server) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	health := s.services.HealthService.CheckHealth(ctx, req.DeepCheck)
	
	pbServiceHealth := make(map[string]*pb.ServiceHealth)
	for name, serviceHealth := range health.ServiceHealth {
		pbServiceHealth[name] = &pb.ServiceHealth{
			Name:           serviceHealth.Name,
			Status:         serviceHealth.Status,
			Message:        serviceHealth.Message,
			LastCheck:      timestampFromTime(serviceHealth.LastCheck),
			ResponseTimeMs: serviceHealth.ResponseTimeMs,
		}
	}

	return &pb.HealthCheckResponse{
		Status:        health.Status,
		ServiceHealth: pbServiceHealth,
		Timestamp:     timestampNow(),
	}, nil
}

// Additional service method implementations would follow similar patterns...
// For brevity, showing representative implementations above.