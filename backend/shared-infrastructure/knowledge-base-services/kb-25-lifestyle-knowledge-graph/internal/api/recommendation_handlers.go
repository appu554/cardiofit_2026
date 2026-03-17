package api

import (
	"kb-25-lifestyle-knowledge-graph/internal/services"

	"github.com/gin-gonic/gin"
)

func (s *Server) recommendLifestyle(c *gin.Context) {
	sendSuccess(c, gin.H{"status": "recommendation_engine_pending"}, nil)
}

func (s *Server) searchFood(c *gin.Context) {
	name := c.DefaultQuery("name", "")
	region := c.DefaultQuery("region", "")
	dietType := c.DefaultQuery("diet_type", "")
	limit := c.DefaultQuery("limit", "20")

	sendSuccess(c, gin.H{
		"query":   gin.H{"name": name, "region": region, "diet_type": dietType, "limit": limit},
		"results": []interface{}{},
	}, nil)
}

func (s *Server) getDietQuality(c *gin.Context) {
	patientID := c.Param("patientId")
	sendSuccess(c, gin.H{"patient_id": patientID, "diet_quality_score": 0, "status": "pending_data"}, nil)
}

func (s *Server) getExerciseRx(c *gin.Context) {
	patientID := c.Param("patientId")
	rx := services.GenerateExerciseRx(patientID)
	sendSuccess(c, rx, nil)
}
