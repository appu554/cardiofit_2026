package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// listBPReadings handles GET /api/v1/patient/:id/bp-readings?since=RFC3339
// Returns paired SBP+DBP readings for the patient since the given time.
// If `since` is omitted, defaults to the last 30 days.
func (s *Server) listBPReadings(c *gin.Context) {
	patientID := c.Param("id")

	sinceStr := c.Query("since")
	var since time.Time
	if sinceStr == "" {
		since = time.Now().AddDate(0, 0, -30)
	} else {
		parsed, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid since parameter (expected RFC3339)"})
			return
		}
		since = parsed
	}

	readings, err := s.bpReadingQuery.FetchSince(patientID, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    readings,
		"metadata": gin.H{
			"patient_id": patientID,
			"since":      since.Format(time.RFC3339),
			"count":      len(readings),
		},
	})
}
