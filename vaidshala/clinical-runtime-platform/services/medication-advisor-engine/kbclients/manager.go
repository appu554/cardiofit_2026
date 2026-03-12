package kbclients

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// KBManager aggregates all Knowledge Base clients and provides unified access.
//
// ARCHITECTURE (CTO/CMO Directive):
// The KBManager now supports two modes:
// 1. Direct mode: Uses local HTTP clients (legacy, for backward compatibility)
// 2. Bridge mode: Uses KnowledgeBridge with parent module's KnowledgeSnapshotBuilder (PREFERRED)
//
// For production, ALWAYS use Bridge mode via NewKBManagerWithBridge().
type KBManager struct {
	// Bridge mode (PREFERRED): Uses parent module's KnowledgeSnapshotBuilder
	bridge *KnowledgeBridge

	// Direct mode (legacy): Uses local HTTP clients
	kb1Dosing       KB1DosingClient
	kb2Interactions KB2InteractionsClient
	kb3Guidelines   KB3GuidelinesClient
	kb4Safety       KB4SafetyClient
	kb5Monitoring   KB5MonitoringClient
	kb6Efficacy     KB6EfficacyClient
	kb16LabSafety   KB16LabSafetyClient // KB-16: Lab-based safety checks

	// Version tracking for Evidence Envelope
	versions     map[string]KBVersionInfo
	versionMutex sync.RWMutex

	// Health status
	lastHealthCheck time.Time
	healthStatus    map[string]bool

	// Mode indicator
	useBridge bool
}

// KBManagerConfig holds configuration for all KB clients
type KBManagerConfig struct {
	KB1URL  string `json:"kb1_url"`  // Dosing service
	KB2URL  string `json:"kb2_url"`  // Interactions service
	KB3URL  string `json:"kb3_url"`  // Guidelines service
	KB4URL  string `json:"kb4_url"`  // Safety service
	KB5URL  string `json:"kb5_url"`  // Monitoring service
	KB6URL  string `json:"kb6_url"`  // Efficacy service
	KB16URL string `json:"kb16_url"` // Lab Safety service (lab-based medication contraindications)

	// Common settings
	Timeout       time.Duration `json:"timeout"`
	RetryAttempts int           `json:"retry_attempts"`
}

// NewKBManager creates a new KB Manager with all clients
func NewKBManager(config KBManagerConfig) (*KBManager, error) {
	manager := &KBManager{
		versions:     make(map[string]KBVersionInfo),
		healthStatus: make(map[string]bool),
	}

	var initErrors []error

	// Initialize KB-1 Dosing client
	if config.KB1URL != "" {
		client, err := NewKB1DosingClient(ClientConfig{
			BaseURL:       config.KB1URL,
			Timeout:       getTimeout(config.Timeout),
			RetryAttempts: getRetryAttempts(config.RetryAttempts),
			RetryDelay:    100 * time.Millisecond,
		})
		if err != nil {
			initErrors = append(initErrors, fmt.Errorf("KB-1 init failed: %w", err))
		} else {
			manager.kb1Dosing = client
			manager.versions["KB-1"] = KBVersionInfo{ServiceName: "KB-1 Dosing", Version: "1.0.0"}
		}
	}

	// Initialize KB-2 Interactions client
	if config.KB2URL != "" {
		client, err := NewKB2InteractionsClient(ClientConfig{
			BaseURL:       config.KB2URL,
			Timeout:       getTimeout(config.Timeout),
			RetryAttempts: getRetryAttempts(config.RetryAttempts),
			RetryDelay:    100 * time.Millisecond,
		})
		if err != nil {
			initErrors = append(initErrors, fmt.Errorf("KB-2 init failed: %w", err))
		} else {
			manager.kb2Interactions = client
			manager.versions["KB-2"] = KBVersionInfo{ServiceName: "KB-2 Interactions", Version: "1.0.0"}
		}
	}

	// Initialize KB-3 Guidelines client
	if config.KB3URL != "" {
		client, err := NewKB3GuidelinesClient(ClientConfig{
			BaseURL:       config.KB3URL,
			Timeout:       getTimeout(config.Timeout),
			RetryAttempts: getRetryAttempts(config.RetryAttempts),
			RetryDelay:    100 * time.Millisecond,
		})
		if err != nil {
			initErrors = append(initErrors, fmt.Errorf("KB-3 init failed: %w", err))
		} else {
			manager.kb3Guidelines = client
			manager.versions["KB-3"] = KBVersionInfo{ServiceName: "KB-3 Guidelines", Version: "1.0.0"}
		}
	}

	// Initialize KB-4 Safety client
	if config.KB4URL != "" {
		client, err := NewKB4SafetyClient(ClientConfig{
			BaseURL:       config.KB4URL,
			Timeout:       getTimeout(config.Timeout),
			RetryAttempts: getRetryAttempts(config.RetryAttempts),
			RetryDelay:    100 * time.Millisecond,
		})
		if err != nil {
			initErrors = append(initErrors, fmt.Errorf("KB-4 init failed: %w", err))
		} else {
			manager.kb4Safety = client
			manager.versions["KB-4"] = KBVersionInfo{ServiceName: "KB-4 Safety", Version: "1.0.0"}
		}
	}

	// Initialize KB-5 Monitoring client
	if config.KB5URL != "" {
		client, err := NewKB5MonitoringClient(ClientConfig{
			BaseURL:       config.KB5URL,
			Timeout:       getTimeout(config.Timeout),
			RetryAttempts: getRetryAttempts(config.RetryAttempts),
			RetryDelay:    100 * time.Millisecond,
		})
		if err != nil {
			initErrors = append(initErrors, fmt.Errorf("KB-5 init failed: %w", err))
		} else {
			manager.kb5Monitoring = client
			manager.versions["KB-5"] = KBVersionInfo{ServiceName: "KB-5 Monitoring", Version: "1.0.0"}
		}
	}

	// Initialize KB-6 Efficacy client
	if config.KB6URL != "" {
		client, err := NewKB6EfficacyClient(ClientConfig{
			BaseURL:       config.KB6URL,
			Timeout:       getTimeout(config.Timeout),
			RetryAttempts: getRetryAttempts(config.RetryAttempts),
			RetryDelay:    100 * time.Millisecond,
		})
		if err != nil {
			initErrors = append(initErrors, fmt.Errorf("KB-6 init failed: %w", err))
		} else {
			manager.kb6Efficacy = client
			manager.versions["KB-6"] = KBVersionInfo{ServiceName: "KB-6 Efficacy", Version: "1.0.0"}
		}
	}

	// Initialize KB-16 Lab Safety client
	if config.KB16URL != "" {
		client, err := NewKB16LabSafetyClient(ClientConfig{
			BaseURL:       config.KB16URL,
			Timeout:       getTimeout(config.Timeout),
			RetryAttempts: getRetryAttempts(config.RetryAttempts),
			RetryDelay:    100 * time.Millisecond,
		})
		if err != nil {
			initErrors = append(initErrors, fmt.Errorf("KB-16 init failed: %w", err))
		} else {
			manager.kb16LabSafety = client
			manager.versions["KB-16"] = KBVersionInfo{ServiceName: "KB-16 Lab Safety", Version: "1.0.0"}
		}
	}

	// Log initialization errors but don't fail - allow partial KB usage
	if len(initErrors) > 0 {
		for _, err := range initErrors {
			fmt.Printf("Warning: %v\n", err)
		}
	}

	return manager, nil
}

// Dosing returns the KB-1 Dosing client
func (m *KBManager) Dosing() KB1DosingClient {
	return m.kb1Dosing
}

// Interactions returns the KB-2 Interactions client
func (m *KBManager) Interactions() KB2InteractionsClient {
	return m.kb2Interactions
}

// Guidelines returns the KB-3 Guidelines client
func (m *KBManager) Guidelines() KB3GuidelinesClient {
	return m.kb3Guidelines
}

// Safety returns the KB-4 Safety client
func (m *KBManager) Safety() KB4SafetyClient {
	return m.kb4Safety
}

// Monitoring returns the KB-5 Monitoring client
func (m *KBManager) Monitoring() KB5MonitoringClient {
	return m.kb5Monitoring
}

// Efficacy returns the KB-6 Efficacy client
func (m *KBManager) Efficacy() KB6EfficacyClient {
	return m.kb6Efficacy
}

// LabSafety returns the KB-16 Lab Safety client
func (m *KBManager) LabSafety() KB16LabSafetyClient {
	return m.kb16LabSafety
}

// GetVersions returns current KB versions for Evidence Envelope
func (m *KBManager) GetVersions() map[string]string {
	m.versionMutex.RLock()
	defer m.versionMutex.RUnlock()

	versions := make(map[string]string)
	for k, v := range m.versions {
		versions[k] = v.Version
	}
	return versions
}

// HealthCheck performs health check on all KB services
func (m *KBManager) HealthCheck(ctx context.Context) map[string]bool {
	status := make(map[string]bool)

	// Check each KB service
	if m.kb1Dosing != nil {
		status["KB-1"] = m.kb1Dosing.HealthCheck(ctx) == nil
	}
	if m.kb2Interactions != nil {
		status["KB-2"] = m.kb2Interactions.HealthCheck(ctx) == nil
	}
	if m.kb3Guidelines != nil {
		status["KB-3"] = m.kb3Guidelines.HealthCheck(ctx) == nil
	}
	if m.kb4Safety != nil {
		status["KB-4"] = m.kb4Safety.HealthCheck(ctx) == nil
	}
	if m.kb5Monitoring != nil {
		status["KB-5"] = m.kb5Monitoring.HealthCheck(ctx) == nil
	}
	if m.kb6Efficacy != nil {
		status["KB-6"] = m.kb6Efficacy.HealthCheck(ctx) == nil
	}
	if m.kb16LabSafety != nil {
		status["KB-16"] = m.kb16LabSafety.HealthCheck(ctx) == nil
	}

	m.healthStatus = status
	m.lastHealthCheck = time.Now()

	return status
}

// IsAvailable checks if a specific KB service is available
func (m *KBManager) IsAvailable(kbName string) bool {
	switch kbName {
	case "KB-1":
		return m.kb1Dosing != nil
	case "KB-2":
		return m.kb2Interactions != nil
	case "KB-3":
		return m.kb3Guidelines != nil
	case "KB-4":
		return m.kb4Safety != nil
	case "KB-5":
		return m.kb5Monitoring != nil
	case "KB-6":
		return m.kb6Efficacy != nil
	case "KB-16":
		return m.kb16LabSafety != nil
	}
	return false
}

// Close closes all KB client connections
func (m *KBManager) Close() error {
	var errors []error

	if m.kb1Dosing != nil {
		if err := m.kb1Dosing.Close(); err != nil {
			errors = append(errors, err)
		}
	}
	if m.kb2Interactions != nil {
		if err := m.kb2Interactions.Close(); err != nil {
			errors = append(errors, err)
		}
	}
	if m.kb3Guidelines != nil {
		if err := m.kb3Guidelines.Close(); err != nil {
			errors = append(errors, err)
		}
	}
	if m.kb4Safety != nil {
		if err := m.kb4Safety.Close(); err != nil {
			errors = append(errors, err)
		}
	}
	if m.kb5Monitoring != nil {
		if err := m.kb5Monitoring.Close(); err != nil {
			errors = append(errors, err)
		}
	}
	if m.kb6Efficacy != nil {
		if err := m.kb6Efficacy.Close(); err != nil {
			errors = append(errors, err)
		}
	}
	if m.kb16LabSafety != nil {
		if err := m.kb16LabSafety.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing KB clients: %v", errors)
	}
	return nil
}

// ============================================================================
// Bridge Mode (PREFERRED for production)
// ============================================================================

// NewKBManagerWithBridge creates a KBManager using the parent module's
// KnowledgeSnapshotBuilder pattern. This is the PREFERRED mode for production.
//
// In this mode, KB answers are pre-computed at snapshot build time and
// engines read from the snapshot - NO KB calls at execution time.
func NewKBManagerWithBridge(config KnowledgeBridgeConfig) (*KBManager, error) {
	bridge, err := NewKnowledgeBridge(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create knowledge bridge: %w", err)
	}

	return &KBManager{
		bridge:       bridge,
		useBridge:    true,
		versions:     make(map[string]KBVersionInfo),
		healthStatus: make(map[string]bool),
	}, nil
}

// NewDefaultKBManagerWithBridge creates a KBManager with default bridge configuration.
func NewDefaultKBManagerWithBridge() (*KBManager, error) {
	return NewKBManagerWithBridge(DefaultKnowledgeBridgeConfig())
}

// Bridge returns the KnowledgeBridge for accessing pre-computed KB answers.
// Returns nil if the manager was created in direct mode.
func (m *KBManager) Bridge() *KnowledgeBridge {
	return m.bridge
}

// UsesBridge returns true if the manager is in bridge mode.
func (m *KBManager) UsesBridge() bool {
	return m.useBridge
}

// ============================================================================
// Helper functions
// ============================================================================

func getTimeout(t time.Duration) time.Duration {
	if t == 0 {
		return 30 * time.Second
	}
	return t
}

func getRetryAttempts(n int) int {
	if n == 0 {
		return 3
	}
	return n
}
