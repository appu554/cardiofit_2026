package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"flow2-go-engine/internal/orb"
)

type MedicationRequest struct {
	RequestID         string   `json:"request_id"`
	PatientID         string   `json:"patient_id"`
	MedicationCode    string   `json:"medication_code"`
	MedicationName    string   `json:"medication_name"`
	PatientConditions []string `json:"patient_conditions"`
	Timestamp         time.Time `json:"timestamp"`
}

type TestResponse struct {
	RequestID       string                 `json:"request_id"`
	Status          string                 `json:"status"`
	IntentManifest  *orb.IntentManifest   `json:"intent_manifest,omitempty"`
	Error           string                 `json:"error,omitempty"`
	ExecutionTimeMs int64                  `json:"execution_time_ms"`
}

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	logger.Info("Starting Flow 2 Test Server")

	// Initialize ORB
	orbEngine, err := orb.NewOrchestratorRuleBase("../knowledge")
	if err != nil {
		logger.Fatalf("Failed to initialize ORB: %v", err)
	}

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Health endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"service": "flow2-test-server",
			"timestamp": time.Now().UTC(),
		})
	})

	// Test ORB endpoint
	router.POST("/api/v1/test/orb", func(c *gin.Context) {
		startTime := time.Now()

		var request MedicationRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(400, TestResponse{
				Status:          "error",
				Error:           fmt.Sprintf("Invalid request: %v", err),
				ExecutionTimeMs: time.Since(startTime).Milliseconds(),
			})
			return
		}

		// Convert to ORB request
		orbRequest := &orb.MedicationRequest{
			RequestID:         request.RequestID,
			PatientID:         request.PatientID,
			MedicationCode:    request.MedicationCode,
			MedicationName:    request.MedicationName,
			PatientConditions: request.PatientConditions,
			Timestamp:         request.Timestamp,
		}

		// Execute ORB
		intentManifest, err := orbEngine.ExecuteLocal(c.Request.Context(), orbRequest)
		if err != nil {
			c.JSON(500, TestResponse{
				RequestID:       request.RequestID,
				Status:          "error",
				Error:           fmt.Sprintf("ORB execution failed: %v", err),
				ExecutionTimeMs: time.Since(startTime).Milliseconds(),
			})
			return
		}

		c.JSON(200, TestResponse{
			RequestID:       request.RequestID,
			Status:          "success",
			IntentManifest:  intentManifest,
			ExecutionTimeMs: time.Since(startTime).Milliseconds(),
		})
	})

	// Test multiple medications endpoint
	router.POST("/api/v1/test/orb/batch", func(c *gin.Context) {
		startTime := time.Now()

		var requests []MedicationRequest
		if err := c.ShouldBindJSON(&requests); err != nil {
			c.JSON(400, gin.H{
				"status": "error",
				"error":  fmt.Sprintf("Invalid request: %v", err),
			})
			return
		}

		results := make([]TestResponse, len(requests))
		for i, request := range requests {
			orbRequest := &orb.MedicationRequest{
				RequestID:         request.RequestID,
				PatientID:         request.PatientID,
				MedicationCode:    request.MedicationCode,
				MedicationName:    request.MedicationName,
				PatientConditions: request.PatientConditions,
				Timestamp:         request.Timestamp,
			}

			intentManifest, err := orbEngine.ExecuteLocal(c.Request.Context(), orbRequest)
			if err != nil {
				results[i] = TestResponse{
					RequestID: request.RequestID,
					Status:    "error",
					Error:     fmt.Sprintf("ORB execution failed: %v", err),
				}
			} else {
				results[i] = TestResponse{
					RequestID:      request.RequestID,
					Status:         "success",
					IntentManifest: intentManifest,
				}
			}
		}

		c.JSON(200, gin.H{
			"status":           "success",
			"results":          results,
			"execution_time_ms": time.Since(startTime).Milliseconds(),
		})
	})

	// Knowledge base info endpoint
	router.GET("/api/v1/test/knowledge", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "available",
			"message": "Knowledge base loaded successfully",
		})
	})

	// Start server
	port := 8080
	logger.WithField("port", port).Info("Starting Flow 2 Test Server")
	
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		logger.Fatalf("Server failed to start: %v", err)
	}
}
