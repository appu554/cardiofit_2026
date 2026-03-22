package ehr

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler exposes HTTP endpoints for EHR data ingestion (FHIR passthrough,
// HL7v2, and SFTP-polled CSV).
type Handler struct {
	fhirAdapter *FHIRRestAdapter
	sftpAdapter *SFTPAdapter
	logger      *zap.Logger
}

// NewHandler creates an EHR handler wired to the given adapters.
func NewHandler(fhirAdapter *FHIRRestAdapter, sftpAdapter *SFTPAdapter, logger *zap.Logger) *Handler {
	return &Handler{
		fhirAdapter: fhirAdapter,
		sftpAdapter: sftpAdapter,
		logger:      logger,
	}
}

// HandleFHIRPassthrough accepts a FHIR R4 Bundle via POST, parses it into
// canonical observations, and returns a 202 Accepted with an OperationOutcome.
func (h *Handler) HandleFHIRPassthrough(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("failed to read request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{
				"severity":    "error",
				"code":        "invalid",
				"diagnostics": "failed to read request body: " + err.Error(),
			}},
		})
		return
	}

	_, observations, err := h.fhirAdapter.ParseBundle(c.Request.Context(), body)
	if err != nil {
		h.logger.Warn("FHIR bundle parse failed", zap.Error(err))
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{
				"severity":    "error",
				"code":        "processing",
				"diagnostics": "bundle parse failed: " + err.Error(),
			}},
		})
		return
	}

	h.logger.Info("FHIR passthrough accepted",
		zap.Int("observation_count", len(observations)),
	)

	c.JSON(http.StatusAccepted, gin.H{
		"resourceType": "OperationOutcome",
		"issue": []gin.H{{
			"severity":    "information",
			"code":        "informational",
			"diagnostics": "bundle accepted",
			"details": gin.H{
				"observation_count": len(observations),
			},
		}},
	})
}

// HandleHL7v2 is a placeholder for HL7 v2 MLLP message ingestion. The MLLP
// parser is not yet implemented.
func (h *Handler) HandleHL7v2(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"resourceType": "OperationOutcome",
		"issue": []gin.H{{
			"severity":    "error",
			"code":        "not-supported",
			"diagnostics": "HL7v2 MLLP parser not implemented",
		}},
	})
}
