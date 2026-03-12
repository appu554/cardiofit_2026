package circuitbreaker

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"global-outbox-service-go/internal/config"
	"global-outbox-service-go/internal/database/models"
)

// MedicalCircuitBreakerState represents the circuit breaker state
type MedicalCircuitBreakerState string

const (
	StateClosed   MedicalCircuitBreakerState = "CLOSED"
	StateOpen     MedicalCircuitBreakerState = "OPEN"
	StateHalfOpen MedicalCircuitBreakerState = "HALF_OPEN"
)

// MedicalCircuitBreaker implements medical-aware circuit breaker logic
type MedicalCircuitBreaker struct {
	mu     sync.RWMutex
	config *config.Config
	logger *logrus.Logger

	// State
	state       MedicalCircuitBreakerState
	failures    int64
	requests    int64
	nextRetryAt time.Time

	// Medical context tracking
	criticalEventsProcessed  int64
	nonCriticalEventsDropped int64

	// Thresholds
	maxQueueDepth      int
	criticalThreshold  float64
	recoveryTimeout    time.Duration
	enabled            bool
}

// NewMedicalCircuitBreaker creates a new medical-aware circuit breaker
func NewMedicalCircuitBreaker(config *config.Config, logger *logrus.Logger) *MedicalCircuitBreaker {
	return &MedicalCircuitBreaker{
		config:            config,
		logger:            logger,
		state:             StateClosed,
		maxQueueDepth:     config.MedicalCircuitBreakerMaxQueueDepth,
		criticalThreshold: config.MedicalCircuitBreakerCriticalThreshold,
		recoveryTimeout:   time.Duration(config.MedicalCircuitBreakerRecoveryTimeout) * time.Second,
		enabled:           config.MedicalCircuitBreakerEnabled,
	}
}

// ShouldProcessEvent determines if an event should be processed based on medical context
func (mcb *MedicalCircuitBreaker) ShouldProcessEvent(event *models.OutboxEvent, currentQueueDepth int) bool {
	if !mcb.enabled {
		return true
	}

	mcb.mu.Lock()
	defer mcb.mu.Unlock()

	// Always process critical and urgent medical events
	if event.IsCritical() || event.IsUrgent() {
		mcb.criticalEventsProcessed++
		mcb.logger.Debugf("Processing critical/urgent event %s (medical_context: %s)", 
			event.ID, event.MedicalContext)
		return true
	}

	// Check circuit breaker state
	mcb.updateState(currentQueueDepth)

	switch mcb.state {
	case StateClosed:
		// Normal operation - process all events
		return true

	case StateOpen:
		// Circuit breaker is open - drop non-critical events
		mcb.nonCriticalEventsDropped++
		mcb.logger.Warnf("Dropping non-critical event %s due to circuit breaker OPEN state", event.ID)
		return false

	case StateHalfOpen:
		// Test mode - process some events to test recovery
		if mcb.requests%10 == 0 { // Process every 10th request
			mcb.logger.Infof("Processing test event %s in HALF_OPEN state", event.ID)
			return true
		}
		mcb.nonCriticalEventsDropped++
		return false

	default:
		return true
	}
}

// RecordSuccess records a successful event processing
func (mcb *MedicalCircuitBreaker) RecordSuccess() {
	if !mcb.enabled {
		return
	}

	mcb.mu.Lock()
	defer mcb.mu.Unlock()

	mcb.requests++

	// If we're in half-open state and succeeding, consider closing
	if mcb.state == StateHalfOpen {
		mcb.logger.Info("Circuit breaker transitioning from HALF_OPEN to CLOSED")
		mcb.state = StateClosed
		mcb.failures = 0
		mcb.nextRetryAt = time.Time{}
	}
}

// RecordFailure records a failed event processing
func (mcb *MedicalCircuitBreaker) RecordFailure() {
	if !mcb.enabled {
		return
	}

	mcb.mu.Lock()
	defer mcb.mu.Unlock()

	mcb.requests++
	mcb.failures++

	// Open the circuit if failure rate is too high
	if mcb.getFailureRate() > mcb.criticalThreshold {
		if mcb.state == StateClosed {
			mcb.logger.Warnf("Circuit breaker opening due to high failure rate: %.2f", mcb.getFailureRate())
			mcb.state = StateOpen
			mcb.nextRetryAt = time.Now().Add(mcb.recoveryTimeout)
		}
	}
}

// updateState updates the circuit breaker state based on current conditions
func (mcb *MedicalCircuitBreaker) updateState(currentQueueDepth int) {
	now := time.Now()

	switch mcb.state {
	case StateClosed:
		// Check if we should open due to queue depth or load
		currentLoad := float64(currentQueueDepth) / float64(mcb.maxQueueDepth)
		if currentLoad > mcb.criticalThreshold {
			mcb.logger.Warnf("Circuit breaker opening due to high queue depth: %d/%d (%.2f)", 
				currentQueueDepth, mcb.maxQueueDepth, currentLoad)
			mcb.state = StateOpen
			mcb.nextRetryAt = now.Add(mcb.recoveryTimeout)
		}

	case StateOpen:
		// Check if recovery timeout has passed
		if now.After(mcb.nextRetryAt) {
			mcb.logger.Info("Circuit breaker transitioning from OPEN to HALF_OPEN")
			mcb.state = StateHalfOpen
		}

	case StateHalfOpen:
		// Half-open state is managed by success/failure recording
		break
	}
}

// getFailureRate calculates the current failure rate
func (mcb *MedicalCircuitBreaker) getFailureRate() float64 {
	if mcb.requests == 0 {
		return 0.0
	}
	return float64(mcb.failures) / float64(mcb.requests)
}

// GetStatus returns the current circuit breaker status
func (mcb *MedicalCircuitBreaker) GetStatus() *models.CircuitBreakerStatus {
	mcb.mu.RLock()
	defer mcb.mu.RUnlock()

	status := &models.CircuitBreakerStatus{
		Enabled:                  mcb.enabled,
		State:                    models.CircuitBreakerState(mcb.state),
		CurrentLoad:              mcb.getCurrentLoad(),
		TotalRequests:            mcb.requests,
		FailedRequests:           mcb.failures,
		CriticalEventsProcessed:  mcb.criticalEventsProcessed,
		NonCriticalEventsDropped: mcb.nonCriticalEventsDropped,
	}

	if mcb.state == StateOpen && !mcb.nextRetryAt.IsZero() {
		status.NextRetryAt = &mcb.nextRetryAt
	}

	return status
}

// getCurrentLoad calculates current system load (simplified)
func (mcb *MedicalCircuitBreaker) getCurrentLoad() float64 {
	if mcb.requests == 0 {
		return 0.0
	}

	// Simple load calculation based on failure rate
	// In production, this could include more sophisticated metrics
	return mcb.getFailureRate()
}

// Reset resets the circuit breaker state (for testing/admin purposes)
func (mcb *MedicalCircuitBreaker) Reset() {
	if !mcb.enabled {
		return
	}

	mcb.mu.Lock()
	defer mcb.mu.Unlock()

	mcb.state = StateClosed
	mcb.failures = 0
	mcb.requests = 0
	mcb.nextRetryAt = time.Time{}
	mcb.criticalEventsProcessed = 0
	mcb.nonCriticalEventsDropped = 0

	mcb.logger.Info("Medical circuit breaker has been reset")
}

// IsEnabled returns whether the circuit breaker is enabled
func (mcb *MedicalCircuitBreaker) IsEnabled() bool {
	return mcb.enabled
}

// Enable enables the circuit breaker
func (mcb *MedicalCircuitBreaker) Enable() {
	mcb.mu.Lock()
	defer mcb.mu.Unlock()
	mcb.enabled = true
	mcb.logger.Info("Medical circuit breaker enabled")
}

// Disable disables the circuit breaker
func (mcb *MedicalCircuitBreaker) Disable() {
	mcb.mu.Lock()
	defer mcb.mu.Unlock()
	mcb.enabled = false
	mcb.logger.Info("Medical circuit breaker disabled")
}