package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"kb-formulary/internal/config"
	"kb-formulary/internal/handlers"
	"kb-formulary/internal/middleware"
	"kb-formulary/internal/services"
)

// HTTPServer represents the HTTP server for REST API endpoints
type HTTPServer struct {
	server           *http.Server
	config           *config.Config
	formularyHandler *handlers.FormularyHandler
	inventoryHandler *handlers.InventoryHandler
	paHandler        *handlers.PAHandler
	stHandler        *handlers.STHandler
	qlHandler        *handlers.QLHandler
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(
	cfg *config.Config,
	formularyService *services.FormularyService,
	inventoryService *services.InventoryService,
	paService *services.PAService,
	stService *services.STService,
	qlService *services.QLService,
) *HTTPServer {
	formularyHandler := handlers.NewFormularyHandler(formularyService)
	inventoryHandler := handlers.NewInventoryHandler(inventoryService)
	paHandler := handlers.NewPAHandler(paService)
	stHandler := handlers.NewSTHandler(stService)
	qlHandler := handlers.NewQLHandler(qlService)

	httpServer := &HTTPServer{
		config:           cfg,
		formularyHandler: formularyHandler,
		inventoryHandler: inventoryHandler,
		paHandler:        paHandler,
		stHandler:        stHandler,
		qlHandler:        qlHandler,
	}

	// Create HTTP server with configured timeouts
	mux := httpServer.setupRoutes()
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", getHTTPPort(cfg.Server.Port)),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	httpServer.server = server
	return httpServer
}

// setupRoutes configures all HTTP routes and middleware
func (s *HTTPServer) setupRoutes() http.Handler {
	mux := http.NewServeMux()

	// Apply middleware stack
	handler := middleware.Chain(
		mux,
		middleware.RequestLogging(),
		middleware.CORS(),
		middleware.RateLimit(),
		middleware.RequestTimeout(30*time.Second),
		middleware.Recovery(),
	)

	// Health check endpoints
	mux.HandleFunc("/health", s.globalHealthCheck)
	mux.HandleFunc("/health/formulary", s.formularyHandler.HealthCheck)
	mux.HandleFunc("/health/inventory", s.inventoryHandler.HealthCheck)

	// Metrics endpoint (if enabled)
	if s.config.Metrics.Enabled {
		mux.HandleFunc(s.config.Metrics.Path, middleware.MetricsHandler())
	}

	// API version prefix
	apiPrefix := "/api/v1"

	// Formulary endpoints
	mux.HandleFunc(apiPrefix+"/formulary/coverage", s.formularyHandler.GetCoverage)
	mux.HandleFunc(apiPrefix+"/formulary/alternatives", s.formularyHandler.GetAlternatives)
	mux.HandleFunc(apiPrefix+"/formulary/search", s.formularyHandler.SearchDrugs)
	mux.Handle(apiPrefix+"/formulary/info/", http.StripPrefix(apiPrefix+"/formulary/info", 
		http.HandlerFunc(s.formularyHandler.GetFormularyInfo)))
	
	// Cost Analysis endpoints
	mux.HandleFunc(apiPrefix+"/cost/analyze", s.formularyHandler.AnalyzeCosts)
	mux.HandleFunc(apiPrefix+"/cost/optimize", s.formularyHandler.OptimizeCosts)
	mux.HandleFunc(apiPrefix+"/cost/portfolio", s.formularyHandler.AnalyzePortfolioCosts)

	// Inventory endpoints
	mux.HandleFunc(apiPrefix+"/inventory/stock", s.inventoryHandler.GetStock)
	mux.HandleFunc(apiPrefix+"/inventory/availability", s.inventoryHandler.GetAvailability)
	mux.HandleFunc(apiPrefix+"/inventory/pricing", s.inventoryHandler.GetPricing)
	mux.HandleFunc(apiPrefix+"/inventory/alerts", s.inventoryHandler.GetLowStockAlerts)
	
	// Inventory reservation endpoints (handle both GET and POST/DELETE)
	mux.HandleFunc(apiPrefix+"/inventory/reserve", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			s.inventoryHandler.ReserveStock(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Reservation management endpoints with path handling
	mux.Handle(apiPrefix+"/inventory/reserve/", http.StripPrefix(apiPrefix,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				s.inventoryHandler.GetReservationStatus(w, r)
			case http.MethodDelete:
				s.inventoryHandler.ReleaseReservation(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})))

	// Prior Authorization endpoints
	mux.HandleFunc(apiPrefix+"/pa/requirements", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			s.paHandler.GetRequirements(w, r)
		case http.MethodPost:
			s.paHandler.GetRequirements(w, r) // Also accept POST for complex queries
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc(apiPrefix+"/pa/check", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodPost:
			s.paHandler.CheckPA(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc(apiPrefix+"/pa/submit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			s.paHandler.SubmitPA(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc(apiPrefix+"/pa/status", s.paHandler.GetStatus)
	mux.HandleFunc(apiPrefix+"/pa/pending", s.paHandler.ListPending)

	// Step Therapy endpoints
	mux.HandleFunc(apiPrefix+"/steptherapy/requirements", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodPost:
			s.stHandler.GetRequirements(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc(apiPrefix+"/steptherapy/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			s.stHandler.CheckStepTherapy(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc(apiPrefix+"/steptherapy/override", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			s.stHandler.RequestOverride(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Quantity Limit endpoints
	mux.HandleFunc(apiPrefix+"/quantitylimit/check", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			s.qlHandler.CheckQuantityLimits(w, r)
		case http.MethodPost:
			s.qlHandler.CheckQuantityLimitsPost(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc(apiPrefix+"/quantitylimit/limits", s.qlHandler.GetLimits)
	mux.HandleFunc(apiPrefix+"/quantitylimit/override", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			s.qlHandler.RequestOverride(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// API documentation endpoint
	mux.HandleFunc(apiPrefix+"/docs", s.apiDocumentation)
	
	// Root endpoint with service info
	mux.HandleFunc("/", s.serviceInfo)

	return handler
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	log.Printf("Starting HTTP server on port %s", getHTTPPort(s.config.Server.Port))
	return s.server.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *HTTPServer) Stop() error {
	log.Println("Stopping HTTP server...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	return s.server.Shutdown(ctx)
}

// globalHealthCheck provides overall service health status
func (s *HTTPServer) globalHealthCheck(w http.ResponseWriter, r *http.Request) {

	health := map[string]interface{}{
		"service":   "KB-6 Formulary Management Service",
		"version":   "1.0.0",
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"checks":    make(map[string]interface{}),
	}

	// Add component health checks here if needed
	// This would integrate with the services for detailed health status

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := writeJSON(w, health); err != nil {
		log.Printf("Error writing health check response: %v", err)
	}
}

// apiDocumentation provides basic API documentation
func (s *HTTPServer) apiDocumentation(w http.ResponseWriter, r *http.Request) {
	docs := map[string]interface{}{
		"service":     "KB-6 Formulary Management Service",
		"version":     "1.0.0",
		"description": "FHIR-compliant formulary, PA, step therapy, and inventory management service",
		"endpoints": map[string]interface{}{
			"formulary": map[string]string{
				"GET /api/v1/formulary/coverage":     "Get drug coverage information",
				"GET /api/v1/formulary/alternatives": "Get alternative medications",
				"GET /api/v1/formulary/search":       "Search formulary drugs",
				"GET /api/v1/formulary/info/{id}":    "Get formulary information",
			},
			"prior_authorization": map[string]string{
				"GET /api/v1/pa/requirements":  "Get PA requirements for a drug",
				"GET|POST /api/v1/pa/check":    "Check if PA is required with clinical context",
				"POST /api/v1/pa/submit":       "Submit a PA request",
				"GET /api/v1/pa/status":        "Get PA submission status",
				"GET /api/v1/pa/pending":       "List pending PA submissions",
			},
			"step_therapy": map[string]string{
				"GET /api/v1/steptherapy/requirements": "Get step therapy rules for a drug",
				"POST /api/v1/steptherapy/check":       "Check step therapy compliance with drug history",
				"POST /api/v1/steptherapy/override":    "Request step therapy override",
			},
			"quantity_limits": map[string]string{
				"GET|POST /api/v1/quantitylimit/check": "Validate prescription against quantity limits",
				"GET /api/v1/quantitylimit/limits":     "Get quantity limits for a drug",
				"POST /api/v1/quantitylimit/override":  "Request quantity limit override",
			},
			"inventory": map[string]string{
				"GET /api/v1/inventory/stock":            "Get stock information",
				"GET /api/v1/inventory/availability":     "Get drug availability",
				"GET /api/v1/inventory/pricing":          "Get pricing information",
				"POST /api/v1/inventory/reserve":         "Reserve stock",
				"GET /api/v1/inventory/reserve/{id}":     "Get reservation status",
				"DELETE /api/v1/inventory/reserve/{id}":  "Release reservation",
				"GET /api/v1/inventory/alerts":           "Get low stock alerts",
			},
			"cost": map[string]string{
				"POST /api/v1/cost/analyze":   "Analyze drug costs",
				"POST /api/v1/cost/optimize":  "Optimize costs with alternatives",
				"POST /api/v1/cost/portfolio": "Analyze medication portfolio costs",
			},
			"health": map[string]string{
				"GET /health":           "Global health check",
				"GET /health/formulary": "Formulary service health",
				"GET /health/inventory": "Inventory service health",
			},
		},
		"authentication": "Bearer token required in Authorization header",
		"rate_limiting":  "100 requests per minute per client",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := writeJSON(w, docs); err != nil {
		log.Printf("Error writing API docs response: %v", err)
	}
}

// serviceInfo provides basic service information at root endpoint
func (s *HTTPServer) serviceInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"service":     "KB-6 Formulary Management Service",
		"version":     "1.0.0",
		"description": "FHIR-compliant formulary, PA, step therapy, and inventory management",
		"endpoints": map[string]string{
			"api_docs":            "/api/v1/docs",
			"health_check":        "/health",
			"metrics":             s.config.Metrics.Path,
			"formulary_coverage":  "/api/v1/formulary/coverage",
			"pa_requirements":     "/api/v1/pa/requirements",
			"pa_check":            "/api/v1/pa/check",
			"steptherapy_check":   "/api/v1/steptherapy/check",
			"quantitylimit_check": "/api/v1/quantitylimit/check",
			"cost_analysis":       "/api/v1/cost/analyze",
			"cost_optimization":   "/api/v1/cost/optimize",
			"portfolio_analysis":  "/api/v1/cost/portfolio",
		},
		"capabilities": []string{
			"Prior Authorization (PA) requirements and submission",
			"Step Therapy validation and override",
			"Quantity Limit enforcement and override",
			"Formulary coverage and alternatives",
			"Inventory management and reservations",
			"Cost analysis and optimization",
		},
		"protocols": []string{"HTTP/REST", "gRPC"},
		"status":    "operational",
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := writeJSON(w, info); err != nil {
		log.Printf("Error writing service info response: %v", err)
	}
}

// getHTTPPort calculates HTTP port from gRPC port (gRPC port + 1)
func getHTTPPort(grpcPort string) string {
	// For KB-6, if gRPC is on 8086, HTTP will be on 8087
	switch grpcPort {
	case "8086":
		return "8087"
	default:
		// Fallback logic - just add 1 to the last digit if possible
		if len(grpcPort) > 0 {
			lastChar := grpcPort[len(grpcPort)-1]
			if lastChar >= '0' && lastChar < '9' {
				return grpcPort[:len(grpcPort)-1] + string(lastChar+1)
			}
		}
		return "8087" // default fallback
	}
}

// writeJSON is a helper function to write JSON responses
func writeJSON(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}