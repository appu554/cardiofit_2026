package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create Gin router
	r := gin.Default()

	// Basic health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "medication-service-v2",
			"version": "1.0.0",
		})
	})

	// Basic medication endpoints - placeholder
	r.GET("/api/v1/medications", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Medication service is running",
			"data":    []interface{}{},
		})
	})

	r.POST("/api/v1/medications/proposals", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Create medication proposal endpoint",
			"status":  "implemented",
		})
	})

	// Start server
	port := "8005" // Different port from other services
	logger.Info("Starting medication-service-v2 HTTP server", zap.String("port", port))

	if err := r.Run(":" + port); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}