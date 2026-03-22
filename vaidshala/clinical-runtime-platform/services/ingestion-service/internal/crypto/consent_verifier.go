package crypto

import (
	"fmt"
	"time"
)

// ConsentArtifact represents an ABDM consent artifact that authorizes
// health information exchange between an HIU and HIP. The artifact is
// digitally signed by the ABDM consent manager.
type ConsentArtifact struct {
	ConsentID    string    `json:"consent_id"`
	PatientID    string    `json:"patient_id"`
	HIURequestID string    `json:"hiu_request_id"`
	Purpose      string    `json:"purpose"`
	HITypes      []string  `json:"hi_types"`
	DateFrom     time.Time `json:"date_from"`
	DateTo       time.Time `json:"date_to"`
	ExpiresAt    time.Time `json:"expires_at"`
	Signature    string    `json:"signature"`
	Status       string    `json:"status"`
}

// VerifyConsentArtifact validates that a consent artifact is currently
// active and structurally valid. It checks the grant status, expiry,
// date range, and presence of a digital signature.
func VerifyConsentArtifact(artifact ConsentArtifact) error {
	if artifact.Status != "GRANTED" {
		return fmt.Errorf("consent %s has status %q, expected GRANTED", artifact.ConsentID, artifact.Status)
	}

	now := time.Now()
	if now.After(artifact.ExpiresAt) {
		return fmt.Errorf("consent %s expired at %s", artifact.ConsentID, artifact.ExpiresAt.Format(time.RFC3339))
	}

	if artifact.DateFrom.After(artifact.DateTo) {
		return fmt.Errorf("consent %s has invalid date range: from %s is after to %s",
			artifact.ConsentID,
			artifact.DateFrom.Format(time.RFC3339),
			artifact.DateTo.Format(time.RFC3339),
		)
	}

	if artifact.Signature == "" {
		return fmt.Errorf("consent %s has empty signature", artifact.ConsentID)
	}

	return nil
}
