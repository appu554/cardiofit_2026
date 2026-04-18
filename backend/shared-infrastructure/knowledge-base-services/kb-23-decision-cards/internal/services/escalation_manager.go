package services

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"kb-23-decision-cards/internal/models"
)

// EscalationManager orchestrates the Gap 15 escalation protocol:
// routing a decision card through tier classification, channel selection,
// notification dispatch, and lifecycle tracking.
type EscalationManager struct {
	router       *EscalationRouter
	tracker      *AcknowledgmentTracker
	channels     map[string]NotificationChannel // "push" -> PushChannelStub, etc.
	auditService *AuditService
	db           *gorm.DB // nil-safe for tests
	log          *zap.Logger

	// Tier timeout config (tier -> minutes). Zero means no timeout.
	tierTimeouts map[string]int

	// Gap 19: lifecycle tracker for T0→T4 detection lifecycle.
	lifecycleTracker *LifecycleTracker
}

// NewEscalationManager constructs the manager with all dependencies injected.
func NewEscalationManager(
	router *EscalationRouter,
	tracker *AcknowledgmentTracker,
	channels map[string]NotificationChannel,
	auditService *AuditService,
	db *gorm.DB,
	log *zap.Logger,
) *EscalationManager {
	if log == nil {
		log = zap.NewNop()
	}
	return &EscalationManager{
		router:       router,
		tracker:      tracker,
		channels:     channels,
		auditService: auditService,
		db:           db,
		log:          log,
		tierTimeouts: map[string]int{
			"SAFETY":    30,
			"IMMEDIATE": 240,
			"URGENT":    1440,
			"ROUTINE":   0,
		},
	}
}

// SetLifecycleTracker injects the lifecycle tracker for Gap 19 T0→T4 tracking.
func (m *EscalationManager) SetLifecycleTracker(t *LifecycleTracker) {
	m.lifecycleTracker = t
}

// HandleCardCreated is the primary entry point: given a newly generated
// decision card, it routes, selects channels, dispatches, persists, and
// returns the EscalationEvent (or nil if the card was suppressed / informational).
func (m *EscalationManager) HandleCardCreated(
	card *models.DecisionCard,
	paiTier string,
	paiScore float64,
) *models.EscalationEvent {
	// Step 1 — build router input.
	input := EscalationRouterInput{
		CardDifferentialID: card.PrimaryDifferentialID,
		MCUGate:            string(card.MCUGate),
		PAITier:            paiTier,
		PAIScore:           paiScore,
		PatientID:          card.PatientID.String(),
	}

	// Step 2 — route.
	result := m.router.RouteCard(input)
	if result == nil || result.Tier == models.TierInformational || result.Suppressed {
		m.log.Debug("escalation suppressed or informational",
			zap.String("patient_id", card.PatientID.String()),
			zap.String("differential", card.PrimaryDifferentialID),
			zap.Bool("suppressed", result != nil && result.Suppressed),
		)
		return nil
	}

	tier := string(result.Tier)
	now := time.Now()

	// Step 3 — build event.
	cardID := card.CardID
	event := &models.EscalationEvent{
		ID:                uuid.New(),
		PatientID:         card.PatientID.String(),
		CardID:            &cardID,
		TriggerType:       string(models.TriggerCardGenerated),
		EscalationTier:    tier,
		CurrentState:      string(models.StatePending),
		EscalationLevel:   1,
		PAIScoreAtTrigger: paiScore,
		PAITierAtTrigger:  paiTier,
		PrimaryReason:     truncate(card.ClinicianSummary, 200),
		SuggestedAction:   truncate(card.MCUGateRationale, 200),
	}

	// Set timeout based on tier.
	if mins, ok := m.tierTimeouts[tier]; ok && mins > 0 {
		timeout := now.Add(time.Duration(mins) * time.Minute)
		event.TimeoutAt = &timeout
	}

	// Gap 19 T0 — record detection into lifecycle tracker.
	var lifecycle *models.DetectionLifecycle
	if m.lifecycleTracker != nil {
		lifecycle = m.lifecycleTracker.RecordT0(
			card.PrimaryDifferentialID,
			string(card.MCUGate),
			card.PatientID.String(),
			string(result.Tier),
			"KB-23",
			&card.CardID,
			&event.ID,
		)
	}

	// Step 4 — select channels.
	sel := SelectChannels(result.Tier, nil, now)

	// Step 5 — dispatch notifications (unless suppressed by quiet hours).
	if !sel.Suppressed {
		notification := EscalationNotification{
			EscalationID:    event.ID.String(),
			PatientID:       event.PatientID,
			Tier:            tier,
			PrimaryReason:   event.PrimaryReason,
			SuggestedAction: event.SuggestedAction,
			CardID:          cardID.String(),
		}

		if sel.Simultaneous {
			// SAFETY tier: dispatch to all primary channels simultaneously.
			m.dispatchAll(event, sel.PrimaryChannels, notification)
		} else {
			// Other tiers: dispatch sequentially.
			m.dispatchSequential(event, sel.PrimaryChannels, notification)
		}
	}

	// Gap 19 T1 — record delivery into lifecycle tracker.
	if m.lifecycleTracker != nil && lifecycle != nil {
		m.lifecycleTracker.RecordT1(lifecycle, time.Now())
	}

	// Step 6 — persist if DB available.
	if m.db != nil {
		if err := m.db.Create(event).Error; err != nil {
			m.log.Error("failed to persist escalation event",
				zap.Error(err),
				zap.String("event_id", event.ID.String()),
			)
		}
	}

	// Step 7 — audit trail.
	if m.auditService != nil {
		_ = m.auditService.Append(
			event.PatientID,
			"ESCALATION_CREATED",
			"KB-23",
			event,
			now,
		)
	}

	return event
}

// dispatchAll sends to all channels (used for SAFETY tier simultaneous dispatch).
func (m *EscalationManager) dispatchAll(
	event *models.EscalationEvent,
	channelNames []string,
	notification EscalationNotification,
) {
	for _, name := range channelNames {
		ch, ok := m.channels[name]
		if !ok {
			m.log.Warn("channel not registered", zap.String("channel", name))
			continue
		}
		result, err := ch.Send(notification)
		if err != nil {
			m.log.Error("channel send failed",
				zap.String("channel", name),
				zap.Error(err),
			)
			continue
		}
		if result.Status == "SENT" {
			m.tracker.RecordDelivery(event, name, result.MessageID)
		}
	}
}

// dispatchSequential sends to channels one at a time (used for non-SAFETY tiers).
func (m *EscalationManager) dispatchSequential(
	event *models.EscalationEvent,
	channelNames []string,
	notification EscalationNotification,
) {
	for _, name := range channelNames {
		ch, ok := m.channels[name]
		if !ok {
			m.log.Warn("channel not registered", zap.String("channel", name))
			continue
		}
		result, err := ch.Send(notification)
		if err != nil {
			m.log.Error("channel send failed",
				zap.String("channel", name),
				zap.Error(err),
			)
			continue
		}
		if result.Status == "SENT" {
			m.tracker.RecordDelivery(event, name, result.MessageID)
			break // sequential: stop after first successful delivery
		}
	}
}

// StartTimeoutChecker launches a background goroutine that periodically
// checks for timed-out escalation events and either escalates or expires them.
func (m *EscalationManager) StartTimeoutChecker(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.checkTimeouts()
			}
		}
	}()
}

func (m *EscalationManager) checkTimeouts() {
	if m.db == nil {
		return
	}
	now := time.Now()
	var pending []models.EscalationEvent
	m.db.Where("current_state IN (?, ?) AND timeout_at IS NOT NULL AND timeout_at < ?",
		models.StatePending, models.StateDelivered, now).Find(&pending)

	for i := range pending {
		result := m.tracker.CheckTimeout(&pending[i], now)
		if result.ShouldExpire {
			pending[i].CurrentState = string(models.StateExpired)
			m.db.Save(&pending[i])
			m.log.Info("escalation expired",
				zap.String("event_id", pending[i].ID.String()),
				zap.String("tier", pending[i].EscalationTier),
			)
		} else if result.ShouldEscalate {
			pending[i].CurrentState = string(models.StateEscalated)
			pending[i].EscalatedAt = &now
			m.db.Save(&pending[i])
			m.log.Info("escalation level increased",
				zap.String("event_id", pending[i].ID.String()),
				zap.Int("new_level", result.NextLevel),
			)
			// TODO: create level-2 event and re-dispatch (Sprint 2)
		}
	}
}

// HandlePAIUpdate processes a PAI score change for a patient, potentially
// resolving pending escalations if the patient's condition has improved.
func (m *EscalationManager) HandlePAIUpdate(patientID, newPAITier string, newPAIScore float64) {
	if m.db == nil {
		return
	}
	var active []models.EscalationEvent
	m.db.Where("patient_id = ? AND current_state IN (?, ?)",
		patientID, models.StatePending, models.StateDelivered).Find(&active)

	for i := range active {
		if m.tracker.HandlePAIImprovement(&active[i], newPAITier) {
			m.db.Save(&active[i])
			m.log.Info("escalation resolved by PAI improvement",
				zap.String("event_id", active[i].ID.String()),
				zap.String("new_pai_tier", newPAITier),
			)
			if m.auditService != nil {
				_ = m.auditService.Append(
					patientID,
					"ESCALATION_RESOLVED_PAI",
					"KB-23",
					active[i],
					time.Now(),
				)
			}
		}
	}
}

// truncate returns the first n characters of s, or s if shorter.
func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}
