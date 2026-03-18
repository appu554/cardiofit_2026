package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// DB row types for query handlers
// ---------------------------------------------------------------------------

// signalRow maps to columns available from the clinical_signals table.
type signalRow struct {
	EventID         string    `gorm:"column:event_id" json:"event_id"`
	PatientID       string    `gorm:"column:patient_id" json:"patient_id"`
	NodeID          string    `gorm:"column:node_id" json:"node_id"`
	NodeVersion     string    `gorm:"column:node_version" json:"node_version"`
	SignalType      string    `gorm:"column:signal_type" json:"signal_type"`
	StratumLabel    string    `gorm:"column:stratum_label" json:"stratum_label"`
	DataSufficiency string    `gorm:"column:data_sufficiency" json:"data_sufficiency"`
	MCUGateSuggestion string  `gorm:"column:mcu_gate_suggestion" json:"mcu_gate_suggestion"`
	PublishedToKB23 bool      `gorm:"column:published_to_kb23" json:"published_to_kb23"`
	EvaluatedAt     time.Time `gorm:"column:evaluated_at" json:"evaluated_at"`
}

func (signalRow) TableName() string { return "clinical_signals" }

// signalLatestRow maps columns from clinical_signals_latest joined with clinical_signals.
type signalLatestRow struct {
	PatientID   string    `gorm:"column:patient_id" json:"patient_id"`
	NodeID      string    `gorm:"column:node_id" json:"node_id"`
	SignalID     string    `gorm:"column:signal_id" json:"signal_id"`
	EvaluatedAt time.Time `gorm:"column:evaluated_at" json:"evaluated_at"`
}

func (signalLatestRow) TableName() string { return "clinical_signals_latest" }

// ---------------------------------------------------------------------------
// handleGetPatientSignals — GET /signals/patients/:id/signals
// ---------------------------------------------------------------------------

func (g *SignalHandlerGroup) handleGetPatientSignals(c *gin.Context) {
	patientID := c.Param("id")
	if patientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing patient id"})
		return
	}

	db := g.db.DB
	var rows []signalLatestRow
	if err := db.WithContext(c.Request.Context()).
		Table("clinical_signals_latest").
		Where("patient_id = ?", patientID).
		Find(&rows).Error; err != nil {
		g.log.Error("handleGetPatientSignals: db query failed",
			zap.String("patient_id", patientID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query patient signals"})
		return
	}

	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// handleGetSignalHistory — GET /signals/patients/:id/signals/:nodeId
// ---------------------------------------------------------------------------

func (g *SignalHandlerGroup) handleGetSignalHistory(c *gin.Context) {
	patientID := c.Param("id")
	nodeID := c.Param("nodeId")
	if patientID == "" || nodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing patient id or node id"})
		return
	}

	db := g.db.DB
	var rows []signalRow
	if err := db.WithContext(c.Request.Context()).
		Table("clinical_signals").
		Where("patient_id = ? AND node_id = ?", patientID, nodeID).
		Order("evaluated_at DESC").
		Limit(50).
		Find(&rows).Error; err != nil {
		g.log.Error("handleGetSignalHistory: db query failed",
			zap.String("patient_id", patientID),
			zap.String("node_id", nodeID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query signal history"})
		return
	}

	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// handleGetDeteriorationSummary — GET /signals/patients/:id/deterioration-summary
// ---------------------------------------------------------------------------

func (g *SignalHandlerGroup) handleGetDeteriorationSummary(c *gin.Context) {
	patientID := c.Param("id")
	if patientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing patient id"})
		return
	}

	db := g.db.DB
	// Join clinical_signals_latest with clinical_signals to filter by signal_type.
	type deteriorationRow struct {
		PatientID   string    `gorm:"column:patient_id" json:"patient_id"`
		NodeID      string    `gorm:"column:node_id" json:"node_id"`
		SignalID    string    `gorm:"column:signal_id" json:"signal_id"`
		SignalType  string    `gorm:"column:signal_type" json:"signal_type"`
		EvaluatedAt time.Time `gorm:"column:evaluated_at" json:"evaluated_at"`
	}

	var rows []deteriorationRow
	if err := db.WithContext(c.Request.Context()).
		Raw(`SELECT csl.patient_id, csl.node_id, csl.signal_id, cs.signal_type, csl.evaluated_at
		     FROM clinical_signals_latest csl
		     JOIN clinical_signals cs ON cs.signal_id = csl.signal_id
		     WHERE csl.patient_id = ? AND cs.signal_type = ?`,
			patientID, "DETERIORATION_SIGNAL").
		Scan(&rows).Error; err != nil {
		g.log.Error("handleGetDeteriorationSummary: db query failed",
			zap.String("patient_id", patientID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query deterioration summary"})
		return
	}

	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// Node listing and retrieval handlers
// ---------------------------------------------------------------------------

// handleListMonitoringNodes returns all loaded PM monitoring node definitions.
func (g *SignalHandlerGroup) handleListMonitoringNodes(c *gin.Context) {
	nodes := g.monitoringLoader.All()
	// Convert map to slice for a stable JSON array response.
	list := make([]interface{}, 0, len(nodes))
	for _, n := range nodes {
		list = append(list, n)
	}
	c.JSON(http.StatusOK, list)
}

// handleGetMonitoringNode returns a single PM monitoring node by ID.
func (g *SignalHandlerGroup) handleGetMonitoringNode(c *gin.Context) {
	nodeID := c.Param("nodeId")
	node := g.monitoringLoader.Get(nodeID)
	if node == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "monitoring node not found", "node_id": nodeID})
		return
	}
	c.JSON(http.StatusOK, node)
}

// handleListDeteriorationNodes returns all loaded MD deterioration node definitions.
func (g *SignalHandlerGroup) handleListDeteriorationNodes(c *gin.Context) {
	nodes := g.deteriorationLoader.All()
	list := make([]interface{}, 0, len(nodes))
	for _, n := range nodes {
		list = append(list, n)
	}
	c.JSON(http.StatusOK, list)
}

// handleGetDeteriorationNode returns a single MD deterioration node by ID.
func (g *SignalHandlerGroup) handleGetDeteriorationNode(c *gin.Context) {
	nodeID := c.Param("nodeId")
	node := g.deteriorationLoader.Get(nodeID)
	if node == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "deterioration node not found", "node_id": nodeID})
		return
	}
	c.JSON(http.StatusOK, node)
}
