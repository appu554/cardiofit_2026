package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kb-patient-profile/internal/models"
)

// activateProtocolRequest is the request body for the activate-protocol endpoint.
type activateProtocolRequest struct {
	ProtocolID    string             `json:"protocol_id" binding:"required"`
	NumericFields map[string]float64 `json:"numeric_fields,omitempty"`
	BoolFields    map[string]bool    `json:"bool_fields,omitempty"`
}

// activateProtocolResponse wraps the primary activation result and any
// protocols that were auto-activated as a side-effect (G-4: VFRP → PRP).
type activateProtocolResponse struct {
	Protocol      *models.ProtocolState  `json:"protocol"`
	AutoActivated []*models.ProtocolState `json:"auto_activated,omitempty"`
}

func (s *Server) activateProtocol(c *gin.Context) {
	patientID := c.Param("id")

	var req activateProtocolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	state, err := s.protocolService.ActivateProtocol(patientID, req.ProtocolID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	resp := activateProtocolResponse{Protocol: state}

	// G-4: When M3-VFRP is activated, check whether the patient meets M3-PRP
	// entry criteria and, if so, auto-activate M3-PRP as a concurrent protocol.
	// A failure in auto-activation is best-effort: we log and continue so the
	// primary VFRP activation is never rolled back due to PRP unavailability.
	if req.ProtocolID == "M3-VFRP" && s.protocolRegistry != nil {
		numericFields := req.NumericFields
		if numericFields == nil {
			numericFields = map[string]float64{}
		}
		boolFields := req.BoolFields
		if boolFields == nil {
			boolFields = map[string]bool{}
		}

		eligible, _ := s.protocolRegistry.CheckEntry("M3-PRP", numericFields, boolFields)
		if eligible {
			prpState, prpErr := s.protocolService.ActivateProtocol(patientID, "M3-PRP")
			if prpErr != nil {
				// Already active or transient error — do not fail the VFRP response.
				c.Header("X-PRP-Auto-Activation-Error", prpErr.Error())
			} else {
				resp.AutoActivated = append(resp.AutoActivated, prpState)
			}
		}
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": resp})
}

func (s *Server) getActiveProtocols(c *gin.Context) {
	patientID := c.Param("id")
	protocols, err := s.protocolService.GetActiveProtocols(patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": protocols})
}

func (s *Server) transitionProtocolPhase(c *gin.Context) {
	patientID := c.Param("id")
	protocolID := c.Param("protocol_id")
	var req struct {
		NextPhase string `json:"next_phase" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	state, err := s.protocolService.TransitionPhase(patientID, protocolID, req.NextPhase)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": state})
}

type engagementSeasonResponse struct {
	PatientID   string `json:"patient_id"`
	Season      string `json:"season"`
	Number      int    `json:"season_number"`
	Protocol    string `json:"protocol"`
	Phase       string `json:"phase"`
	DaysInPhase int    `json:"days_in_phase"`
}

type seasonInfo struct {
	Name   string
	Number int
}

func mapPhaseToSeason(phase string) seasonInfo {
	switch phase {
	case "CONSOLIDATION":
		return seasonInfo{"CONSOLIDATION", 2}
	case "INDEPENDENCE":
		return seasonInfo{"INDEPENDENCE", 3}
	case "STABILITY":
		return seasonInfo{"STABILITY", 4}
	case "PARTNERSHIP":
		return seasonInfo{"PARTNERSHIP", 5}
	default:
		return seasonInfo{"CORRECTION", 1}
	}
}

func (s *Server) getEngagementSeason(c *gin.Context) {
	patientID := c.Param("id")

	protocols, err := s.protocolService.GetActiveProtocols(patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := engagementSeasonResponse{
		PatientID: patientID,
		Season:    "CORRECTION",
		Number:    1,
	}

	for _, p := range protocols {
		if p.ProtocolID == "M3-MAINTAIN" {
			season := mapPhaseToSeason(p.CurrentPhase)
			resp.Season = season.Name
			resp.Number = season.Number
			resp.Protocol = p.ProtocolID
			resp.Phase = p.CurrentPhase
			resp.DaysInPhase = p.DaysInPhase()
			break
		}
	}

	c.JSON(http.StatusOK, resp)
}
