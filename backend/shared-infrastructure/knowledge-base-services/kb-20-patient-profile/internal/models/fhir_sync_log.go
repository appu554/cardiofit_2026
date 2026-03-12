package models

import (
	"time"

	"github.com/google/uuid"
)

// FHIRSyncLog records each FHIR resource sync operation for auditability.
type FHIRSyncLog struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ResourceType string    `gorm:"size:50;not null" json:"resource_type"`
	FHIRID       string    `gorm:"size:200;not null;column:fhir_id" json:"fhir_id"`
	Action       string    `gorm:"size:20;not null;check:action IN ('CREATED','UPDATED','SKIPPED')" json:"action"`
	SyncedAt     time.Time `gorm:"not null;default:now()" json:"synced_at"`
	Error        string    `gorm:"type:text" json:"error,omitempty"`
}
