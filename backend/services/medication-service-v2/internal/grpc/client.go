package grpc

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "medication-service-v2/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Flow2Clients manages connections to both Flow2 engines
type Flow2Clients struct {
	GoEngineClient   pb.Flow2EngineClient
	RustEngineClient pb.Flow2EngineClient
	goConn          *grpc.ClientConn
	rustConn        *grpc.ClientConn
}

// NewFlow2Clients creates connections to both Flow2 engines
func NewFlow2Clients() (*Flow2Clients, error) {
	// Connect to Go engine (port 8080)
	goConn, err := grpc.Dial("localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Go engine: %v", err)
	}

	// Connect to Rust engine (port 8091 for gRPC)
	rustConn, err := grpc.Dial("localhost:8091", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		goConn.Close()
		return nil, fmt.Errorf("failed to connect to Rust engine: %v", err)
	}

	return &Flow2Clients{
		GoEngineClient:   pb.NewFlow2EngineClient(goConn),
		RustEngineClient: pb.NewFlow2EngineClient(rustConn),
		goConn:          goConn,
		rustConn:        rustConn,
	}, nil
}

// Close closes all gRPC connections
func (c *Flow2Clients) Close() {
	if c.goConn != nil {
		c.goConn.Close()
	}
	if c.rustConn != nil {
		c.rustConn.Close()
	}
}

// ExecuteRecipeWithGoEngine executes a recipe using the Go engine
func (c *Flow2Clients) ExecuteRecipeWithGoEngine(ctx context.Context, req *pb.RecipeExecutionRequest) (*pb.RecipeExecutionResponse, error) {
	log.Printf("🔄 Executing recipe %s with Go engine for patient %s", req.RecipeId, req.PatientId)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := c.GoEngineClient.ExecuteRecipe(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Go engine execution failed: %v", err)
	}

	log.Printf("✅ Go engine completed recipe %s in %dms", req.RecipeId, response.ExecutionTimeMs)
	return response, nil
}

// ExecuteRecipeWithRustEngine executes a recipe using the Rust engine
func (c *Flow2Clients) ExecuteRecipeWithRustEngine(ctx context.Context, req *pb.RecipeExecutionRequest) (*pb.RecipeExecutionResponse, error) {
	log.Printf("🦀 Executing recipe %s with Rust engine for patient %s", req.RecipeId, req.PatientId)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := c.RustEngineClient.ExecuteRecipe(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Rust engine execution failed: %v", err)
	}

	log.Printf("✅ Rust engine completed recipe %s in %dms", req.RecipeId, response.ExecutionTimeMs)
	return response, nil
}

// OptimizeDoseWithGoEngine optimizes dose using the Go engine
func (c *Flow2Clients) OptimizeDoseWithGoEngine(ctx context.Context, req *pb.DoseOptimizationRequest) (*pb.DoseOptimizationResponse, error) {
	log.Printf("🔄 Optimizing dose with Go engine for medication %s", req.MedicationCode)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := c.GoEngineClient.OptimizeDose(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Go engine dose optimization failed: %v", err)
	}

	if response.Recommendation != nil {
		log.Printf("✅ Go engine optimized dose: %.2f %s", response.Recommendation.DoseValue, response.Recommendation.DoseUnit)
	}
	return response, nil
}

// OptimizeDoseWithRustEngine optimizes dose using the Rust engine
func (c *Flow2Clients) OptimizeDoseWithRustEngine(ctx context.Context, req *pb.DoseOptimizationRequest) (*pb.DoseOptimizationResponse, error) {
	log.Printf("🦀 Optimizing dose with Rust engine for medication %s", req.MedicationCode)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := c.RustEngineClient.OptimizeDose(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Rust engine dose optimization failed: %v", err)
	}

	if response.Recommendation != nil {
		log.Printf("✅ Rust engine optimized dose: %.2f %s", response.Recommendation.DoseValue, response.Recommendation.DoseUnit)
	}
	return response, nil
}

// HealthCheckBothEngines checks health of both engines
func (c *Flow2Clients) HealthCheckBothEngines(ctx context.Context) (bool, bool, error) {
	healthReq := &pb.HealthCheckRequest{}

	// Check Go engine
	goHealthy := false
	if _, err := c.GoEngineClient.HealthCheck(ctx, healthReq); err == nil {
		goHealthy = true
		log.Printf("✅ Go engine is healthy")
	} else {
		log.Printf("❌ Go engine health check failed: %v", err)
	}

	// Check Rust engine
	rustHealthy := false
	if _, err := c.RustEngineClient.HealthCheck(ctx, healthReq); err == nil {
		rustHealthy = true
		log.Printf("✅ Rust engine is healthy")
	} else {
		log.Printf("❌ Rust engine health check failed: %v", err)
	}

	return goHealthy, rustHealthy, nil
}