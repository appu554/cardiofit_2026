package api

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// correlationIDMiddleware adds a correlation ID to each request.
func correlationIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		corrID := c.GetHeader("X-Correlation-ID")
		if corrID == "" {
			corrID = uuid.New().String()
		}
		c.Set("correlation_id", corrID)
		c.Writer.Header().Set("X-Correlation-ID", corrID)
		c.Next()
	}
}
