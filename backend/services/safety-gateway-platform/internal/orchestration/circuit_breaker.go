package orchestration

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/pkg/logger"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitStateClosed CircuitState = iota
	CircuitStateOpen
	CircuitStateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern for engine protection
type CircuitBreaker struct {
	config   config.CircuitBreakerConfig
	logger   *logger.Logger
	circuits map[string]*Circuit
	mutex    sync.RWMutex
}

// Circuit represents a single circuit for an engine
type Circuit struct {
	engineID         string
	state            CircuitState
	failures         int
	lastFailTime     time.Time
	lastSuccessTime  time.Time
	halfOpenCalls    int
	mutex            sync.Mutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(cfg config.CircuitBreakerConfig, logger *logger.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		config:   cfg,
		logger:   logger,
		circuits: make(map[string]*Circuit),
	}
}

// Execute executes a function through the circuit breaker
func (cb *CircuitBreaker) Execute(engineID string, fn func() error) error {
	circuit := cb.getOrCreateCircuit(engineID)
	
	circuit.mutex.Lock()
	defer circuit.mutex.Unlock()

	// Check circuit state
	switch circuit.state {
	case CircuitStateOpen:
		if cb.shouldAttemptReset(circuit) {
			circuit.state = CircuitStateHalfOpen
			circuit.halfOpenCalls = 0
			cb.logger.LogCircuitBreakerEvent(engineID, "open", "half_open", circuit.failures)
		} else {
			return fmt.Errorf("circuit breaker is open for engine %s", engineID)
		}
	case CircuitStateHalfOpen:
		if circuit.halfOpenCalls >= cb.config.HalfOpenMaxCalls {
			return fmt.Errorf("circuit breaker half-open limit exceeded for engine %s", engineID)
		}
		circuit.halfOpenCalls++
	}

	// Execute function
	err := fn()

	// Handle result
	if err != nil {
		cb.onFailure(circuit)
		return err
	}

	cb.onSuccess(circuit)
	return nil
}

// getOrCreateCircuit gets or creates a circuit for an engine
func (cb *CircuitBreaker) getOrCreateCircuit(engineID string) *Circuit {
	cb.mutex.RLock()
	circuit, exists := cb.circuits[engineID]
	cb.mutex.RUnlock()

	if exists {
		return circuit
	}

	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// Double-check after acquiring write lock
	if circuit, exists := cb.circuits[engineID]; exists {
		return circuit
	}

	circuit = &Circuit{
		engineID:        engineID,
		state:           CircuitStateClosed,
		failures:        0,
		lastSuccessTime: time.Now(),
	}

	cb.circuits[engineID] = circuit
	return circuit
}

// shouldAttemptReset checks if the circuit should attempt to reset
func (cb *CircuitBreaker) shouldAttemptReset(circuit *Circuit) bool {
	return time.Since(circuit.lastFailTime) > time.Duration(cb.config.ResetTimeoutSeconds)*time.Second
}

// onFailure handles a failure
func (cb *CircuitBreaker) onFailure(circuit *Circuit) {
	circuit.failures++
	circuit.lastFailTime = time.Now()

	oldState := circuit.state

	switch circuit.state {
	case CircuitStateClosed:
		if circuit.failures >= cb.config.FailureThreshold {
			circuit.state = CircuitStateOpen
			cb.logger.LogCircuitBreakerEvent(circuit.engineID, "closed", "open", circuit.failures)
		}
	case CircuitStateHalfOpen:
		circuit.state = CircuitStateOpen
		cb.logger.LogCircuitBreakerEvent(circuit.engineID, "half_open", "open", circuit.failures)
	}

	if oldState != circuit.state {
		cb.logger.Warn("Circuit breaker state changed due to failure",
			zap.String("engine_id", circuit.engineID),
			zap.String("old_state", cb.stateToString(oldState)),
			zap.String("new_state", cb.stateToString(circuit.state)),
			zap.Int("failure_count", circuit.failures),
		)
	}
}

// onSuccess handles a success
func (cb *CircuitBreaker) onSuccess(circuit *Circuit) {
	circuit.lastSuccessTime = time.Now()
	oldState := circuit.state

	switch circuit.state {
	case CircuitStateHalfOpen:
		circuit.state = CircuitStateClosed
		circuit.failures = 0
		cb.logger.LogCircuitBreakerEvent(circuit.engineID, "half_open", "closed", 0)
	case CircuitStateClosed:
		// Reset failure count on successful execution
		if circuit.failures > 0 {
			circuit.failures = 0
		}
	}

	if oldState != circuit.state {
		cb.logger.Info("Circuit breaker reset to closed state",
			zap.String("engine_id", circuit.engineID),
			zap.String("old_state", cb.stateToString(oldState)),
			zap.String("new_state", cb.stateToString(circuit.state)),
		)
	}
}

// GetCircuitState returns the current state of a circuit
func (cb *CircuitBreaker) GetCircuitState(engineID string) CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	if circuit, exists := cb.circuits[engineID]; exists {
		circuit.mutex.Lock()
		defer circuit.mutex.Unlock()
		return circuit.state
	}

	return CircuitStateClosed // Default state for non-existent circuits
}

// GetCircuitStats returns statistics for a circuit
func (cb *CircuitBreaker) GetCircuitStats(engineID string) map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	circuit, exists := cb.circuits[engineID]
	if !exists {
		return map[string]interface{}{
			"exists": false,
		}
	}

	circuit.mutex.Lock()
	defer circuit.mutex.Unlock()

	return map[string]interface{}{
		"exists":             true,
		"engine_id":          circuit.engineID,
		"state":              cb.stateToString(circuit.state),
		"failure_count":      circuit.failures,
		"last_fail_time":     circuit.lastFailTime,
		"last_success_time":  circuit.lastSuccessTime,
		"half_open_calls":    circuit.halfOpenCalls,
	}
}

// GetAllCircuitStats returns statistics for all circuits
func (cb *CircuitBreaker) GetAllCircuitStats() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	stats := make(map[string]interface{})
	
	for engineID := range cb.circuits {
		stats[engineID] = cb.GetCircuitStats(engineID)
	}

	stats["total_circuits"] = len(cb.circuits)
	stats["config"] = map[string]interface{}{
		"failure_threshold":      cb.config.FailureThreshold,
		"reset_timeout_seconds":  cb.config.ResetTimeoutSeconds,
		"half_open_max_calls":    cb.config.HalfOpenMaxCalls,
	}

	return stats
}

// ResetCircuit manually resets a circuit to closed state
func (cb *CircuitBreaker) ResetCircuit(engineID string) error {
	cb.mutex.RLock()
	circuit, exists := cb.circuits[engineID]
	cb.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("circuit for engine %s does not exist", engineID)
	}

	circuit.mutex.Lock()
	defer circuit.mutex.Unlock()

	oldState := circuit.state
	circuit.state = CircuitStateClosed
	circuit.failures = 0
	circuit.halfOpenCalls = 0

	cb.logger.Info("Circuit breaker manually reset",
		zap.String("engine_id", engineID),
		zap.String("old_state", cb.stateToString(oldState)),
		zap.String("new_state", "closed"),
	)

	cb.logger.LogCircuitBreakerEvent(engineID, cb.stateToString(oldState), "closed", 0)

	return nil
}

// IsCircuitOpen checks if a circuit is open
func (cb *CircuitBreaker) IsCircuitOpen(engineID string) bool {
	return cb.GetCircuitState(engineID) == CircuitStateOpen
}

// stateToString converts circuit state to string
func (cb *CircuitBreaker) stateToString(state CircuitState) string {
	switch state {
	case CircuitStateClosed:
		return "closed"
	case CircuitStateOpen:
		return "open"
	case CircuitStateHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

// Shutdown shuts down the circuit breaker
func (cb *CircuitBreaker) Shutdown() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.logger.Info("Circuit breaker shutting down", zap.Int("total_circuits", len(cb.circuits)))
	
	// Clear all circuits
	cb.circuits = make(map[string]*Circuit)
}
