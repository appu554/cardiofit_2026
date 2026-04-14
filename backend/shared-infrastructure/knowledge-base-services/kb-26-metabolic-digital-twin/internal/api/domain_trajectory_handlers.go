package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-26-metabolic-digital-twin/internal/models"
)

// getDomainTrajectory computes and returns the decomposed MHRI trajectory for a patient.
// GET /api/v1/kb26/mri/:patientId/domain-trajectory
func (s *Server) getDomainTrajectory(c *gin.Context) {
	patientID, err := uuid.Parse(c.Param("patientId"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid patient ID", "INVALID_PATIENT_ID", nil)
		return
	}

	// Fetch recent MRI scores (up to 10 most recent).
	mriScores, err := s.mriScorer.GetHistory(patientID, 10)
	if err != nil {
		s.logger.Error("failed to fetch MRI history", zap.Error(err))
		sendError(c, http.StatusInternalServerError, "failed to fetch MRI history", "DB_ERROR", nil)
		return
	}

	if len(mriScores) < 2 {
		sendSuccess(c, gin.H{
			"status":      "INSUFFICIENT_DATA",
			"message":     "need at least 2 MRI scores for trajectory computation",
			"data_points": len(mriScores),
		}, map[string]interface{}{
			"patient_id": patientID.String(),
		})
		return
	}

	points := buildTrajectoryPoints(mriScores)

	// Compute decomposed trajectory.
	trajectory := s.trajectoryEngine.Compute(patientID.String(), points)

	// Persist snapshot to history table (idempotent per patient per day).
	if err := s.persistDomainTrajectorySnapshot(patientID, &trajectory); err != nil {
		// Persistence failure should not block the response — log and continue.
		s.logger.Warn("failed to persist domain trajectory snapshot",
			zap.String("patient_id", patientID.String()),
			zap.Error(err))
	}

	sendSuccess(c, trajectory, map[string]interface{}{
		"patient_id":  patientID.String(),
		"data_points": len(points),
	})
}

// persistDomainTrajectorySnapshot writes a single DomainTrajectoryHistory row.
// Uses FirstOrCreate with Where+Assign to keep one snapshot per patient per day.
func (s *Server) persistDomainTrajectorySnapshot(patientID uuid.UUID, traj *models.DecomposedTrajectory) error {
	if s.db == nil {
		return nil // no-op if DB not wired (e.g., in tests)
	}

	history := models.DomainTrajectoryHistory{
		ID:              uuid.New().String(),
		PatientID:       patientID,
		SnapshotDate:    time.Now().UTC().Truncate(24 * time.Hour),
		WindowDays:      traj.WindowDays,
		CompositeSlope:  traj.CompositeSlope,
		GlucoseSlope:    traj.DomainSlopes[models.DomainGlucose].SlopePerDay,
		CardioSlope:     traj.DomainSlopes[models.DomainCardio].SlopePerDay,
		BodyCompSlope:   traj.DomainSlopes[models.DomainBodyComp].SlopePerDay,
		BehavioralSlope: traj.DomainSlopes[models.DomainBehavioral].SlopePerDay,
		HasDiscordance:  traj.HasDiscordantTrend,
		CreatedAt:       time.Now(),
	}
	if traj.DominantDriver != nil {
		history.DominantDriver = string(*traj.DominantDriver)
	}

	// Upsert: one snapshot per patient per day.
	err := s.db.DB.
		Where("patient_id = ? AND snapshot_date = ?", history.PatientID, history.SnapshotDate).
		Assign(history).
		FirstOrCreate(&history).Error

	if s.trajectoryMetrics != nil {
		if err != nil {
			s.trajectoryMetrics.PersistTotal.WithLabelValues("fail").Inc()
		} else {
			s.trajectoryMetrics.PersistTotal.WithLabelValues("ok").Inc()
		}
	}

	return err
}

// computeAndPersistTrajectory fetches the patient's recent MRI history,
// computes the decomposed domain trajectory, and persists a snapshot.
// Phase 6 P6-3 — invoked from the MHRI compute path so that trajectory
// runs on every natural MRI recomputation, not just on explicit
// GET /domain-trajectory requests. Best-effort: all failures are logged
// and swallowed so they never block the caller (typically getMRI).
//
// The publish to Kafka happens inside TrajectoryEngine.Compute via the
// configured TrajectoryPublisher — production wires the real Kafka
// publisher; tests use NoopTrajectoryPublisher.
func (s *Server) computeAndPersistTrajectory(patientID uuid.UUID) {
	mriScores, err := s.mriScorer.GetHistory(patientID, 10)
	if err != nil {
		s.logger.Warn("trajectory: failed to fetch MRI history",
			zap.String("patient_id", patientID.String()),
			zap.Error(err))
		return
	}
	if len(mriScores) < 2 {
		// Insufficient data is not an error — the patient just doesn't
		// have enough history yet. The TrajectoryEngine.Compute path
		// also handles this case, but we early-return here to avoid the
		// unnecessary persistence call.
		return
	}

	points := buildTrajectoryPoints(mriScores)
	trajectory := s.trajectoryEngine.Compute(patientID.String(), points)

	if err := s.persistDomainTrajectorySnapshot(patientID, &trajectory); err != nil {
		s.logger.Warn("trajectory: failed to persist snapshot",
			zap.String("patient_id", patientID.String()),
			zap.Error(err))
	}
}

// buildTrajectoryPoints converts a slice of MRIScore (DESC order from
// GetHistory) into a chronological slice of DomainTrajectoryPoint suitable
// for TrajectoryEngine.Compute. Phase 6 P6-3 — extracted so that both the
// explicit GET handler and the implicit MHRI compute path produce identical
// trajectory inputs.
func buildTrajectoryPoints(mriScores []models.MRIScore) []models.DomainTrajectoryPoint {
	points := make([]models.DomainTrajectoryPoint, len(mriScores))
	for i, score := range mriScores {
		j := len(mriScores) - 1 - i
		points[j] = models.DomainTrajectoryPoint{
			Timestamp:       score.ComputedAt,
			CompositeScore:  score.Score,
			GlucoseScore:    score.GlucoseDomain,
			CardioScore:     score.CardioDomain,
			BodyCompScore:   score.BodyCompDomain,
			BehavioralScore: score.BehavioralDomain,
		}
	}
	return points
}
