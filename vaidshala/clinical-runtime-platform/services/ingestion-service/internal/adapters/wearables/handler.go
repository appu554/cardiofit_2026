package wearables

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// Handler dispatches inbound wearable payloads to the appropriate adapter
// based on the :provider path parameter.
type Handler struct {
	healthConnect *HealthConnectAdapter
	ultrahuman    *UltrahumanAdapter
	appleHealth   *AppleHealthAdapter
	logger        *zap.Logger
}

// NewHandler creates a wearable ingest handler with all three adapters.
func NewHandler(logger *zap.Logger) *Handler {
	return &Handler{
		healthConnect: &HealthConnectAdapter{},
		ultrahuman:    &UltrahumanAdapter{},
		appleHealth:   &AppleHealthAdapter{},
		logger:        logger,
	}
}

// HandleIngest is the Gin handler for POST /wearables/:provider.
// It reads the provider param and delegates to the matching adapter.
func (h *Handler) HandleIngest(c *gin.Context) {
	provider := c.Param("provider")

	var observations []canonical.CanonicalObservation
	var err error

	switch provider {
	case "health_connect":
		var payload HealthConnectPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		observations, err = h.healthConnect.Convert(payload)

	case "ultrahuman":
		var payload UltrahumanCGMPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		observations, err = h.ultrahuman.Convert(payload)

	case "apple_health":
		var payload AppleHealthPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		observations, err = h.appleHealth.Convert(payload)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported wearable provider: " + provider})
		return
	}

	if err != nil {
		h.logger.Error("wearable conversion failed",
			zap.String("provider", provider),
			zap.Error(err),
		)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("wearable data ingested",
		zap.String("provider", provider),
		zap.Int("observation_count", len(observations)),
	)

	c.JSON(http.StatusOK, gin.H{
		"status":            "accepted",
		"observation_count": len(observations),
		"observations":      observations,
	})
}
