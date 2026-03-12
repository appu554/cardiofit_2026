package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "kb-23-decision-cards",
	})
}

func (s *Server) handleReadiness(c *gin.Context) {
	// Check DB and Redis
	dbErr := s.db.Health()
	cacheErr := s.cache.Health()

	if dbErr != nil || cacheErr != nil {
		status := gin.H{"status": "not_ready"}
		if dbErr != nil {
			status["database"] = dbErr.Error()
		}
		if cacheErr != nil {
			status["redis"] = cacheErr.Error()
		}
		c.JSON(http.StatusServiceUnavailable, status)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           "ready",
		"service":          "kb-23-decision-cards",
		"templates_loaded": s.templateLoader.Count(),
	})
}

func (s *Server) handleMetrics(c *gin.Context) {
	promhttp.Handler().ServeHTTP(c.Writer, c.Request)
}

func (s *Server) handleTemplateReload(c *gin.Context) {
	if err := s.templateLoader.Reload(); err != nil {
		s.log.Error("template reload failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "reload_failed",
			"message": err.Error(),
		})
		return
	}

	// Reload fragments from new templates
	s.fragmentLoader.LoadFromTemplates(s.templateLoader.List())
	s.metrics.TemplateReloads.Inc()
	s.metrics.TemplatesLoaded.Set(float64(s.templateLoader.Count()))

	c.JSON(http.StatusOK, gin.H{
		"status":           "reloaded",
		"templates_loaded": s.templateLoader.Count(),
	})
}
