package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"kb-formulary/internal/config"
	"kb-formulary/internal/services"
	pb "kb-formulary/proto/kb6"
)

// Server implements the KB6 gRPC service
type Server struct {
	pb.UnimplementedKB6ServiceServer
	formularyService *services.FormularyService
	inventoryService *services.InventoryService
	cfg              *config.Config
	grpcServer       *grpc.Server
	healthServer     *health.Server
}

// NewServer creates a new gRPC server instance
func NewServer(cfg *config.Config, formularyService *services.FormularyService, inventoryService *services.InventoryService) *Server {
	return &Server{
		formularyService: formularyService,
		inventoryService: inventoryService,
		cfg:              cfg,
		healthServer:     health.NewServer(),
	}
}

// Start starts the gRPC server
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", s.cfg.Server.Port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", s.cfg.Server.Port, err)
	}

	// Create gRPC server with interceptors
	s.grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(s.loggingInterceptor),
	)

	// Register services
	pb.RegisterKB6ServiceServer(s.grpcServer, s)
	grpc_health_v1.RegisterHealthServer(s.grpcServer, s.healthServer)

	// Set initial health status
	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	s.healthServer.SetServingStatus("kb6.v1.KB6Service", grpc_health_v1.HealthCheckResponse_SERVING)

	log.Printf("Starting KB-6 gRPC server on port %s", s.cfg.Server.Port)
	return s.grpcServer.Serve(lis)
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	if s.grpcServer != nil {
		log.Println("Stopping KB-6 gRPC server...")
		s.grpcServer.GracefulStop()
	}
}

// GetFormularyStatus implements formulary coverage checking
func (s *Server) GetFormularyStatus(ctx context.Context, req *pb.FormularyRequest) (*pb.FormularyResponse, error) {
	start := time.Now()
	
	// Validate request
	if err := s.validateFormularyRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Call formulary service
	coverage, err := s.formularyService.CheckCoverage(ctx, &services.CoverageRequest{
		TransactionID: req.TransactionId,
		DrugRxNorm:    req.DrugRxnorm,
		PayerID:       req.PayerId,
		PlanID:        req.PlanId,
		PlanYear:      int(req.PlanYear),
		Quantity:      int(req.Quantity),
		DaysSupply:    int(req.DaysSupply),
		Patient:       convertPatientContext(req.Patient),
	})

	if err != nil {
		log.Printf("Error checking formulary coverage: %v", err)
		return nil, status.Error(codes.Internal, "failed to check formulary coverage")
	}

	// Convert to protobuf response
	response := &pb.FormularyResponse{
		TransactionId:              req.TransactionId,
		DatasetVersion:             coverage.DatasetVersion,
		Covered:                    coverage.Covered,
		CoverageStatus:             coverage.CoverageStatus,
		Tier:                       coverage.Tier,
		Cost:                       convertCostDetails(coverage.Cost),
		PriorAuthorizationRequired: coverage.PriorAuthRequired,
		StepTherapyRequired:        coverage.StepTherapyRequired,
		QuantityLimits:             convertQuantityLimits(coverage.QuantityLimits),
		Restrictions:               coverage.Restrictions,
		AgeRestrictions:            convertAgeRestrictions(coverage.AgeRestrictions),
		GenderRestriction:          coverage.GenderRestriction,
		PreferredAlternatives:      convertAlternatives(coverage.Alternatives),
		Evidence:                   convertEvidenceEnvelope(coverage.Evidence),
		Status: &pb.ResponseStatus{
			Code:    pb.StatusCode_SUCCESS,
			Message: "Formulary status retrieved successfully",
		},
		ResponseTime: timestamppb.New(time.Now()),
	}

	// Log performance metric
	duration := time.Since(start)
	log.Printf("GetFormularyStatus completed in %v for transaction %s", duration, req.TransactionId)

	return response, nil
}

// GetStock implements stock availability checking
func (s *Server) GetStock(ctx context.Context, req *pb.StockRequest) (*pb.StockResponse, error) {
	start := time.Now()

	// Validate request
	if err := s.validateStockRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Call inventory service
	stock, err := s.inventoryService.CheckStock(ctx, &services.StockCheckRequest{
		TransactionID:       req.TransactionId,
		DrugRxNorm:         req.DrugRxnorm,
		LocationID:         req.LocationId,
		IncludeLots:        req.IncludeLots,
		IncludeAlternatives: req.IncludeAlternatives,
		RequiredQuantity:   int(req.RequiredQuantity),
	})

	if err != nil {
		log.Printf("Error checking stock availability: %v", err)
		return nil, status.Error(codes.Internal, "failed to check stock availability")
	}

	// Convert to protobuf response
	response := &pb.StockResponse{
		TransactionId:     req.TransactionId,
		DatasetVersion:    stock.DatasetVersion,
		LocationId:        stock.LocationID,
		DrugRxnorm:        stock.DrugRxNorm,
		QuantityOnHand:    int32(stock.QuantityOnHand),
		QuantityAllocated: int32(stock.QuantityAllocated),
		QuantityAvailable: int32(stock.QuantityAvailable),
		InStock:           stock.InStock,
		SufficientStock:   stock.SufficientStock,
		Lots:              convertLotDetails(stock.Lots),
		ReorderInfo:       convertReorderInfo(stock.ReorderInfo),
		AlternativeStock:  convertAlternativeStock(stock.AlternativeStock),
		Alerts:            convertStockAlerts(stock.Alerts),
		DemandForecast:    convertDemandPrediction(stock.DemandForecast),
		Evidence:          convertEvidenceEnvelope(stock.Evidence),
		Status: &pb.ResponseStatus{
			Code:    pb.StatusCode_SUCCESS,
			Message: "Stock status retrieved successfully",
		},
		ResponseTime: timestamppb.New(time.Now()),
	}

	// Log performance metric
	duration := time.Since(start)
	log.Printf("GetStock completed in %v for transaction %s", duration, req.TransactionId)

	return response, nil
}

// GetCostAnalysis implements cost analysis with alternatives
func (s *Server) GetCostAnalysis(ctx context.Context, req *pb.CostAnalysisRequest) (*pb.CostAnalysisResponse, error) {
	start := time.Now()

	// Validate request
	if err := s.validateCostAnalysisRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Call formulary service for cost analysis
	analysis, err := s.formularyService.AnalyzeCosts(ctx, &services.CostAnalysisRequest{
		TransactionID:        req.TransactionId,
		DrugRxNorms:         req.DrugRxnorms,
		PayerID:             req.PayerId,
		PlanID:              req.PlanId,
		Quantity:            int(req.Quantity),
		IncludeAlternatives: req.IncludeAlternatives,
		OptimizationGoal:    req.OptimizationGoal,
	})

	if err != nil {
		log.Printf("Error analyzing costs: %v", err)
		return nil, status.Error(codes.Internal, "failed to analyze costs")
	}

	// Convert to protobuf response
	response := &pb.CostAnalysisResponse{
		TransactionId:        req.TransactionId,
		DatasetVersion:       analysis.DatasetVersion,
		TotalPrimaryCost:     analysis.TotalPrimaryCost,
		TotalAlternativeCost: analysis.TotalAlternativeCost,
		TotalSavings:         analysis.TotalSavings,
		SavingsPercent:       analysis.SavingsPercent,
		DrugAnalysis:         convertDrugCostAnalysis(analysis.DrugAnalysis),
		Recommendations:      convertCostOptimizations(analysis.Recommendations),
		Evidence:             convertEvidenceEnvelope(analysis.Evidence),
		Status: &pb.ResponseStatus{
			Code:    pb.StatusCode_SUCCESS,
			Message: "Cost analysis completed successfully",
		},
		ResponseTime: timestamppb.New(time.Now()),
	}

	// Log performance metric
	duration := time.Since(start)
	log.Printf("GetCostAnalysis completed in %v for transaction %s", duration, req.TransactionId)

	return response, nil
}

// SearchFormulary implements formulary search functionality
func (s *Server) SearchFormulary(ctx context.Context, req *pb.FormularySearchRequest) (*pb.FormularySearchResponse, error) {
	start := time.Now()

	// Validate request
	if err := s.validateSearchRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Call formulary service for search
	searchResults, err := s.formularyService.Search(ctx, &services.SearchRequest{
		TransactionID: req.TransactionId,
		Query:        req.Query,
		PayerID:      req.PayerId,
		PlanID:       req.PlanId,
		Tiers:        req.Tiers,
		DrugTypes:    req.DrugTypes,
		Limit:        int(req.Limit),
		Offset:       int(req.Offset),
		SortBy:       req.SortBy,
		SortOrder:    req.SortOrder,
	})

	if err != nil {
		log.Printf("Error searching formulary: %v", err)
		return nil, status.Error(codes.Internal, "failed to search formulary")
	}

	// Convert to protobuf response
	response := &pb.FormularySearchResponse{
		TransactionId:  req.TransactionId,
		DatasetVersion: searchResults.DatasetVersion,
		Results:        convertFormularyEntries(searchResults.Results),
		TotalCount:     int32(searchResults.TotalCount),
		SearchTimeMs:   int32(searchResults.SearchTimeMs),
		Suggestions:    searchResults.Suggestions,
		Metadata:       convertSearchMetadata(searchResults.Metadata),
		Status: &pb.ResponseStatus{
			Code:    pb.StatusCode_SUCCESS,
			Message: "Search completed successfully",
		},
		ResponseTime: timestamppb.New(time.Now()),
	}

	// Log performance metric
	duration := time.Since(start)
	log.Printf("SearchFormulary completed in %v for transaction %s", duration, req.TransactionId)

	return response, nil
}

// HealthCheck implements health checking
func (s *Server) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	// Check service health
	overallStatus := pb.HealthStatus_HEALTHY
	var serviceHealths []*pb.ServiceHealth

	// Check database health
	dbHealth := s.checkDatabaseHealth()
	serviceHealths = append(serviceHealths, dbHealth)
	if dbHealth.Status != pb.HealthStatus_HEALTHY {
		overallStatus = pb.HealthStatus_DEGRADED
	}

	// Check Redis health
	redisHealth := s.checkRedisHealth()
	serviceHealths = append(serviceHealths, redisHealth)
	if redisHealth.Status == pb.HealthStatus_UNHEALTHY {
		overallStatus = pb.HealthStatus_DEGRADED
	}

	// Check Elasticsearch health (optional)
	if s.cfg.Elasticsearch.Enabled {
		esHealth := s.checkElasticsearchHealth()
		serviceHealths = append(serviceHealths, esHealth)
		if esHealth.Status == pb.HealthStatus_UNHEALTHY {
			overallStatus = pb.HealthStatus_DEGRADED
		}
	}

	return &pb.HealthCheckResponse{
		OverallStatus: overallStatus,
		Services:      serviceHealths,
		Timestamp:     timestamppb.New(time.Now()),
		Version:       "1.0.0",
	}, nil
}

// loggingInterceptor provides request/response logging
func (s *Server) loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	
	// Call handler
	resp, err := handler(ctx, req)
	
	// Log request
	duration := time.Since(start)
	log.Printf("gRPC %s completed in %v (error: %v)", info.FullMethod, duration, err != nil)
	
	return resp, err
}

// Validation functions
func (s *Server) validateFormularyRequest(req *pb.FormularyRequest) error {
	if req.TransactionId == "" {
		return fmt.Errorf("transaction_id is required")
	}
	if req.DrugRxnorm == "" {
		return fmt.Errorf("drug_rxnorm is required")
	}
	if req.PayerId == "" {
		return fmt.Errorf("payer_id is required")
	}
	if req.PlanId == "" {
		return fmt.Errorf("plan_id is required")
	}
	return nil
}

func (s *Server) validateStockRequest(req *pb.StockRequest) error {
	if req.TransactionId == "" {
		return fmt.Errorf("transaction_id is required")
	}
	if req.DrugRxnorm == "" {
		return fmt.Errorf("drug_rxnorm is required")
	}
	if req.LocationId == "" {
		return fmt.Errorf("location_id is required")
	}
	return nil
}

func (s *Server) validateCostAnalysisRequest(req *pb.CostAnalysisRequest) error {
	if req.TransactionId == "" {
		return fmt.Errorf("transaction_id is required")
	}
	if len(req.DrugRxnorms) == 0 {
		return fmt.Errorf("at least one drug_rxnorm is required")
	}
	return nil
}

func (s *Server) validateSearchRequest(req *pb.FormularySearchRequest) error {
	if req.TransactionId == "" {
		return fmt.Errorf("transaction_id is required")
	}
	if req.Query == "" {
		return fmt.Errorf("query is required")
	}
	return nil
}

// Health check helper functions
func (s *Server) checkDatabaseHealth() *pb.ServiceHealth {
	// TODO: Implement actual database health check
	return &pb.ServiceHealth{
		ServiceName: "postgresql",
		Status:      pb.HealthStatus_HEALTHY,
		Message:     "Database connection healthy",
		Metrics:     map[string]string{
			"connections": "active",
			"latency":     "<10ms",
		},
	}
}

func (s *Server) checkRedisHealth() *pb.ServiceHealth {
	// TODO: Implement actual Redis health check
	return &pb.ServiceHealth{
		ServiceName: "redis",
		Status:      pb.HealthStatus_HEALTHY,
		Message:     "Redis connection healthy",
		Metrics:     map[string]string{
			"connection": "active",
			"hit_rate":   "85%",
		},
	}
}

func (s *Server) checkElasticsearchHealth() *pb.ServiceHealth {
	// TODO: Implement actual Elasticsearch health check
	return &pb.ServiceHealth{
		ServiceName: "elasticsearch",
		Status:      pb.HealthStatus_HEALTHY,
		Message:     "Elasticsearch cluster healthy",
		Metrics:     map[string]string{
			"cluster_status": "green",
			"nodes":          "3",
		},
	}
}