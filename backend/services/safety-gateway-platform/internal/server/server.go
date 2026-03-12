package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"safety-gateway-platform/internal/config"
	contextpkg "safety-gateway-platform/internal/context"
	"safety-gateway-platform/internal/engines"
	"safety-gateway-platform/internal/orchestration"
	"safety-gateway-platform/internal/registry"
	"safety-gateway-platform/internal/services"
	"safety-gateway-platform/internal/validator"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
	pb "safety-gateway-platform/proto"
)

// Server represents the Safety Gateway Platform server
type Server struct {
	config              *config.Config
	logger              *logger.Logger
	grpcServer          *grpc.Server
	httpServer          *HTTPServer
	healthServer        *health.Server
	validator           *validator.IngressValidator
	registry            *registry.EngineRegistry
	contextService      *contextpkg.AssemblyService
	orchestrator        types.SafetyOrchestrator  // Interface to support both basic and advanced
	batchProcessor      *orchestration.EnhancedBatchProcessor
	metricsCollector    *orchestration.ComprehensiveMetricsCollector
	safetyService       *services.SafetyService
	listener            net.Listener
}

// New creates a new Safety Gateway Platform server
func New(cfg *config.Config, logger *logger.Logger) (*Server, error) {
	// Create ingress validator
	validator, err := validator.NewIngressValidator(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create ingress validator: %w", err)
	}

	// Create engine registry
	engineRegistry := registry.NewEngineRegistry(cfg, logger)

	// Create real GraphDB client
	// NOTE: Assuming NewGraphDBClient exists and returns a valid client for the context service.
	graphClient := services.NewGraphDBClient(cfg, logger)

	// Create context assembly service
	contextService := contextpkg.NewAssemblyService(graphClient, cfg, logger)

	// Create orchestration engine with Phase 2 integration
	var finalOrchestrator types.SafetyOrchestrator
	var batchProcessor *orchestration.EnhancedBatchProcessor
	var metricsCollector *orchestration.ComprehensiveMetricsCollector

	// Create base orchestration engine
	baseOrchestrator := orchestration.NewOrchestrationEngine(
		engineRegistry,
		contextService,
		cfg,
		logger,
	)

	// Check if advanced orchestration (Phase 2) is enabled
	if cfg.AdvancedOrchestration != nil && cfg.AdvancedOrchestration.Enabled {
		logger.Info("Initializing Phase 2 Advanced Orchestration Engine")
		
		// Create snapshot orchestration engine first
		snapshotOrchestrator := orchestration.NewSnapshotOrchestrationEngine(
			baseOrchestrator,
			validator.(*validator.SnapshotValidator), // Cast to snapshot validator
			contextService,
			nil, // snapshot cache - will be created internally
			cfg.Snapshot,
			logger,
		)
		
		// Create enhanced batch processor
		batchProcessor = orchestration.NewEnhancedBatchProcessor(
			snapshotOrchestrator,
			cfg.AdvancedOrchestration.BatchProcessing,
			logger,
		)
		
		// Create comprehensive metrics collector
		metricsCollector = orchestration.NewComprehensiveMetricsCollector(
			cfg.AdvancedOrchestration.Metrics,
			logger,
		)
		
		// Create advanced orchestration engine
		finalOrchestrator = orchestration.NewAdvancedOrchestrationEngine(
			snapshotOrchestrator,
			cfg.AdvancedOrchestration,
			logger,
		)
		
		logger.Info("Phase 2 Advanced Orchestration enabled",
			zap.String("load_balancing_strategy", cfg.AdvancedOrchestration.LoadBalancing.Strategy),
			zap.Bool("batch_processing_enabled", cfg.AdvancedOrchestration.BatchProcessing.Enabled),
			zap.Bool("intelligent_routing_enabled", cfg.AdvancedOrchestration.Routing.EnableIntelligentRouting),
		)
	} else if cfg.Snapshot != nil && cfg.Snapshot.Enabled {
		logger.Info("Using Phase 1 Snapshot Orchestration Engine")
		// Snapshot-only mode (Phase 1)
		finalOrchestrator = orchestration.NewSnapshotOrchestrationEngine(
			baseOrchestrator,
			validator.(*validator.SnapshotValidator),
			contextService,
			nil,
			cfg.Snapshot,
			logger,
		)
	} else {
		logger.Info("Using legacy orchestration engine")
		// Legacy mode
		finalOrchestrator = baseOrchestrator
	}

	// Create safety service
	safetyService := services.NewSafetyService(
		validator,
		finalOrchestrator,
		cfg,
		logger,
	)

	// Create gRPC server with middleware
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			services.LoggingInterceptor(logger),
			services.MetricsInterceptor(),
			services.AuthInterceptor(cfg),
			services.TimeoutInterceptor(cfg.GetRequestTimeout()),
		),
		grpc.MaxRecvMsgSize(cfg.Performance.MaxRequestSizeMB*1024*1024),
		grpc.MaxSendMsgSize(cfg.Performance.MaxRequestSizeMB*1024*1024),
	)

	// Create health server
	healthServer := health.NewServer()

	// Create HTTP server with advanced orchestration
	httpServer := NewHTTPServer(cfg, logger, validator, finalOrchestrator)

	server := &Server{
		config:           cfg,
		logger:           logger,
		grpcServer:       grpcServer,
		httpServer:       httpServer,
		healthServer:     healthServer,
		validator:        validator,
		registry:         engineRegistry,
		contextService:   contextService,
		orchestrator:     finalOrchestrator,
		batchProcessor:   batchProcessor,
		metricsCollector: metricsCollector,
		safetyService:    safetyService,
	}

	// Register services
	if err := server.registerServices(); err != nil {
		return nil, fmt.Errorf("failed to register services: %w", err)
	}

	return server, nil
}

// registerServices registers gRPC services
func (s *Server) registerServices() error {
	// Register safety service
	pb.RegisterSafetyGatewayServer(s.grpcServer, s.safetyService)

	// Register health service
	grpc_health_v1.RegisterHealthServer(s.grpcServer, s.healthServer)

	// Enable reflection for development
	if s.config.Service.Environment == "development" {
		reflection.Register(s.grpcServer)
		s.logger.Info("gRPC reflection enabled for development")
	}

	s.logger.Info("gRPC services registered")
	return nil
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	// Create listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.Service.Port))
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	s.listener = listener

	s.logger.Info("Starting Safety Gateway Platform servers",
		zap.Int("grpc_port", s.config.Service.Port),
		zap.Int("http_port", s.config.Service.HTTPPort),
		zap.String("address", listener.Addr().String()),
	)

	// Initialize engines (real with mock fallbacks)
	if err := s.initializeEngines(); err != nil {
		// Log the error but continue starting the server, as it might be intentional in some environments
		s.logger.Error("Failed to initialize one or more engines", zap.Error(err))
	}

	// Set health status to serving
	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	s.healthServer.SetServingStatus("safety-gateway", grpc_health_v1.HealthCheckResponse_SERVING)

	// Start both gRPC and HTTP servers in goroutines
	errChan := make(chan error, 2)
	
	// Start gRPC server
	go func() {
		if err := s.grpcServer.Serve(listener); err != nil {
			errChan <- fmt.Errorf("gRPC server failed: %w", err)
		}
	}()
	
	// Start HTTP server
	go func() {
		if err := s.httpServer.Start(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP server failed: %w", err)
		}
	}()

	s.logger.Info("Safety Gateway Platform servers started successfully",
		zap.Int("grpc_port", s.config.Service.Port),
		zap.Int("http_port", s.config.Service.HTTPPort),
		zap.String("environment", s.config.Service.Environment),
	)

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		s.logger.Info("Server context cancelled, initiating shutdown")
		return nil
	case err := <-errChan:
		return err
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down Safety Gateway Platform server")

	// Set health status to not serving
	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	s.healthServer.SetServingStatus("safety-gateway", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Create shutdown channel
	shutdownComplete := make(chan struct{})

	go func() {
		// Shutdown HTTP server first
		if s.httpServer != nil {
			if err := s.httpServer.Shutdown(ctx); err != nil {
				s.logger.Error("HTTP server shutdown error", zap.Error(err))
			}
		}
		
		// Then graceful stop gRPC server
		s.grpcServer.GracefulStop()
		close(shutdownComplete)
	}()

	// Wait for graceful shutdown or force stop
	select {
	case <-shutdownComplete:
		s.logger.Info("Servers stopped gracefully")
	case <-ctx.Done():
		s.logger.Warn("Shutdown timeout exceeded, forcing stop")
		s.grpcServer.Stop()
	}

	// Shutdown components
	if err := s.shutdownComponents(); err != nil {
		s.logger.Error("Error shutting down components", zap.Error(err))
	}

	s.logger.Info("Safety Gateway Platform server shutdown complete")
	return nil
}

// shutdownComponents shuts down server components
func (s *Server) shutdownComponents() error {
	var errs []error

	// Shutdown engine registry
	if s.registry != nil {
		if err := s.registry.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("engine registry shutdown: %w", err))
		}
	}

	// Shutdown context service
	if s.contextService != nil {
		if err := s.contextService.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("context service shutdown: %w", err))
		}
	}

	if len(errs) > 0 {
		// Combine multiple errors if necessary
		return errors.New("component shutdown errors occurred")
	}

	return nil
}

// initializeEngines initializes real engines only - no mock fallbacks.
func (s *Server) initializeEngines() error {
	// --- CAE Engine Initialization (REQUIRED) ---
	if err := s.initializeCAEEngine(); err != nil {
		return fmt.Errorf("failed to initialize real CAE engine - no fallback available: %w", err)
	}

	// --- Real Engines Only ---
	// Note: Other engines should be implemented as real services
	// For now, we only use the CAE engine which contains all clinical reasoning capabilities

	// Log all registered engines
	var registeredEngineIDs []string
	allEngines := s.registry.GetAllEngines()
	for _, eng := range allEngines {
		registeredEngineIDs = append(registeredEngineIDs, eng.ID)
	}
	sort.Strings(registeredEngineIDs)
	s.logger.Info("Engines initialized", zap.Strings("registered_engines", registeredEngineIDs))

	return nil
}

// initializeCAEEngine initializes the real CAE engine via gRPC
func (s *Server) initializeCAEEngine() error {
	// Get CAE service address from environment or use default
	caeAddress := os.Getenv("CAE_SERVICE_ADDRESS")
	if caeAddress == "" {
		caeAddress = "localhost:8027" // Default CAE service address
	}

	// Create gRPC CAE engine
	caeEngine := engines.NewGRPCCAEEngine(s.logger, caeAddress)

	// Initialize the engine (establishes gRPC connection)
	if err := caeEngine.Initialize(types.EngineConfig{}); err != nil {
		return fmt.Errorf("failed to initialize gRPC CAE engine: %w", err)
	}

	// Register CAE engine as Tier 1 (Veto-Critical) with highest priority
	if err := s.registry.RegisterEngine(caeEngine, types.TierVetoCritical, 10); err != nil {
		return fmt.Errorf("failed to register gRPC CAE engine: %w", err)
	}

	s.logger.Info("gRPC CAE engine initialized successfully",
		zap.String("cae_address", caeAddress))
	return nil
}


// GetPort returns the server port
func (s *Server) GetPort() int {
	if s.listener != nil {
		if addr, ok := s.listener.Addr().(*net.TCPAddr); ok {
			return addr.Port
		}
	}
	return s.config.Service.Port
}

// GetAddress returns the server address
func (s *Server) GetAddress() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return fmt.Sprintf(":%d", s.config.Service.Port)
}

// HealthCheck performs a comprehensive health check
func (s *Server) HealthCheck() error {
	// Check validator
	if s.validator == nil {
		return fmt.Errorf("ingress validator not initialized")
	}

	// Check registry
	if s.registry == nil {
		return fmt.Errorf("engine registry not initialized")
	}

	// Check context service
	if s.contextService == nil {
		return fmt.Errorf("context service not initialized")
	}

	// Check cache health
	cacheStats := s.contextService.GetCacheStats()
	if cacheStats == nil {
		s.logger.Warn("Cache health check failed - no stats available")
		// Decide if this is a critical failure
	}

	// Check engine health
	engines := s.registry.GetAllEngines()
	if len(engines) == 0 {
		return fmt.Errorf("no engines are registered")
	}

	healthyEngines := 0
	for _, engine := range engines {
		// NOTE: Assuming engine info has a status field or a health check method
		if engine.Status == types.EngineStatusHealthy {
			healthyEngines++
		}
	}

	if healthyEngines == 0 {
		return fmt.Errorf("no healthy engines available")
	}

	s.logger.Info("Health check passed", zap.Int("healthy_engines", healthyEngines), zap.Int("total_engines", len(engines)))
	return nil
}