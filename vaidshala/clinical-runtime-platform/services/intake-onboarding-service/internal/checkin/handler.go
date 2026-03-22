package checkin

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Handler exposes HTTP endpoints for check-in operations.
type Handler struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewHandler creates a check-in handler.
func NewHandler(db *pgxpool.Pool, logger *zap.Logger) *Handler {
	return &Handler{db: db, logger: logger}
}

// StartCheckinRequest is the JSON body for starting a check-in session.
type StartCheckinRequest struct {
	CycleNumber int `json:"cycle_number"`
}

// FillSlotRequest is the JSON body for filling a check-in slot.
type FillSlotRequest struct {
	SlotName       string  `json:"slot_name"`
	Value          float64 `json:"value"`
	ExtractionMode string  `json:"extraction_mode"`
}

// HandleStartCheckin creates a new check-in session for a patient.
// POST /fhir/Patient/:id/$checkin
func (h *Handler) HandleStartCheckin(c *gin.Context) {
	patientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient ID"})
		return
	}

	var req StartCheckinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	session := &CheckinSession{
		ID:          uuid.New(),
		PatientID:   patientID,
		CycleNumber: req.CycleNumber,
		State:       CS1_SCHEDULED,
		SlotsTotal:  len(CheckinSlots()),
		ScheduledAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Transition to CS2_REMINDED then CS3_COLLECTING (immediate start).
	if err := session.Transition(CS2_REMINDED); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err := session.Transition(CS3_COLLECTING); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// Look up the encounter_id from the enrollment.
	var encounterID uuid.UUID
	err = h.db.QueryRow(c.Request.Context(),
		`SELECT encounter_id FROM enrollments WHERE patient_id = $1 AND state = 'ENROLLED' LIMIT 1`,
		patientID,
	).Scan(&encounterID)
	if err != nil {
		h.logger.Error("patient not enrolled", zap.Error(err))
		c.JSON(http.StatusConflict, gin.H{"error": "patient must be ENROLLED to start check-in"})
		return
	}

	// Persist to database.
	_, err = h.db.Exec(c.Request.Context(), `
		INSERT INTO checkin_sessions (id, patient_id, encounter_id, cycle_number, state, slots_filled, slots_total, scheduled_at, started_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		session.ID, session.PatientID, encounterID, session.CycleNumber, string(session.State),
		session.SlotsFilled, session.SlotsTotal, session.ScheduledAt,
		session.StartedAt, session.CreatedAt, session.UpdatedAt,
	)
	if err != nil {
		h.logger.Error("failed to create checkin session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	h.logger.Info("checkin session started",
		zap.String("session_id", session.ID.String()),
		zap.String("patient_id", patientID.String()),
		zap.Int("cycle", req.CycleNumber),
	)

	c.JSON(http.StatusCreated, gin.H{
		"session_id":   session.ID,
		"state":        session.State,
		"slots_total":  session.SlotsTotal,
		"slots_filled": session.SlotsFilled,
		"slots":        CheckinSlots(),
	})
}

// HandleFillCheckinSlot records a slot value for an active check-in session.
// POST /fhir/Encounter/:id/$checkin-slot
func (h *Handler) HandleFillCheckinSlot(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	var req FillSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate slot name exists in the slot definitions and capture domain.
	var slotDef *CheckinSlotDef
	for _, s := range CheckinSlots() {
		if s.Name == req.SlotName {
			s := s // capture loop var
			slotDef = &s
			break
		}
	}
	if slotDef == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown slot: " + req.SlotName})
		return
	}

	// Default extraction mode.
	extractionMode := req.ExtractionMode
	if extractionMode == "" {
		extractionMode = "PATIENT_REPORTED"
	}

	// Load session state from DB.
	var state string
	var slotsFilled int
	var patientID uuid.UUID
	err = h.db.QueryRow(c.Request.Context(), `
		SELECT state, slots_filled, patient_id FROM checkin_sessions WHERE id = $1`, sessionID,
	).Scan(&state, &slotsFilled, &patientID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "checkin session not found"})
		return
	}

	if CheckinState(state) != CS3_COLLECTING {
		c.JSON(http.StatusConflict, gin.H{"error": "session not in COLLECTING state"})
		return
	}

	// Record the slot event — columns match migration 003_checkin.sql.
	now := time.Now()
	_, err = h.db.Exec(c.Request.Context(), `
		INSERT INTO checkin_slot_events (id, session_id, patient_id, slot_name, domain, value, extraction_mode, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		uuid.New(), sessionID, patientID, req.SlotName, slotDef.Domain,
		fmt.Sprintf(`{"value": %v}`, req.Value), extractionMode, now,
	)
	if err != nil {
		h.logger.Error("failed to record slot event", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record slot"})
		return
	}

	// Update filled count.
	slotsFilled++
	_, _ = h.db.Exec(c.Request.Context(), `
		UPDATE checkin_sessions SET slots_filled = $1, updated_at = $2 WHERE id = $3`,
		slotsFilled, now, sessionID,
	)

	h.logger.Info("checkin slot filled",
		zap.String("session_id", sessionID.String()),
		zap.String("slot", req.SlotName),
		zap.Float64("value", req.Value),
	)

	c.JSON(http.StatusOK, gin.H{
		"session_id":   sessionID,
		"slot_name":    req.SlotName,
		"slots_filled": slotsFilled,
	})
}
