package abdm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/cardiofit/ingestion-service/internal/crypto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ConsentStore abstracts persistence of ABDM consent artifacts so the
// HIU handler can verify consent validity before processing data.
type ConsentStore interface {
	GetConsent(consentID string) (*crypto.ConsentArtifact, error)
}

// ABDMDataPushPayload is the top-level request body sent by an HIP (or
// the ABDM gateway) when pushing health data to our HIU callback endpoint.
type ABDMDataPushPayload struct {
	TransactionID string          `json:"transaction_id"`
	ConsentID     string          `json:"consent_id"`
	KeyMaterial   ABDMKeyMaterial `json:"key_material"`
	Entries       []ABDMDataEntry `json:"entries"`
}

// ABDMDataEntry is a single encrypted FHIR bundle within a data push.
type ABDMDataEntry struct {
	CareContextRef string `json:"care_context_ref"`
	Content        string `json:"content"` // base64-encoded encrypted bytes
}

// ABDMKeyMaterial carries the sender's ephemeral public key and nonce
// required for X25519 decryption of the data entries.
type ABDMKeyMaterial struct {
	SenderPublicKey string `json:"sender_public_key"` // base64-encoded [32]byte
	Nonce           string `json:"nonce"`              // base64-encoded [24]byte
}

// HIUHandler processes incoming ABDM health data pushes, decrypts them
// using X25519, and converts the FHIR content into canonical observations.
type HIUHandler struct {
	keyPair      *crypto.X25519KeyPair
	consentStore ConsentStore
	logger       *zap.Logger
}

// NewHIUHandler creates a handler wired with our X25519 key pair and
// consent store for incoming ABDM data push processing.
func NewHIUHandler(keyPair *crypto.X25519KeyPair, consentStore ConsentStore, logger *zap.Logger) *HIUHandler {
	return &HIUHandler{
		keyPair:      keyPair,
		consentStore: consentStore,
		logger:       logger,
	}
}

// HandleDataPush is the gin handler for POST /v1/abdm/data/push. It
// decrypts each entry, parses the FHIR content, and returns 202 Accepted.
func (h *HIUHandler) HandleDataPush(c *gin.Context) {
	var payload ABDMDataPushPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		h.logger.Warn("abdm data push: invalid JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	// Verify consent is valid.
	consent, err := h.consentStore.GetConsent(payload.ConsentID)
	if err != nil {
		h.logger.Error("abdm data push: consent lookup failed",
			zap.String("consent_id", payload.ConsentID),
			zap.Error(err),
		)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "consent verification failed"})
		return
	}
	if err := crypto.VerifyConsentArtifact(*consent); err != nil {
		h.logger.Warn("abdm data push: consent invalid",
			zap.String("consent_id", payload.ConsentID),
			zap.Error(err),
		)
		c.JSON(http.StatusForbidden, gin.H{"error": "consent not valid: " + err.Error()})
		return
	}

	// Decode sender public key (32 bytes).
	senderPubBytes, err := base64.StdEncoding.DecodeString(payload.KeyMaterial.SenderPublicKey)
	if err != nil || len(senderPubBytes) != 32 {
		h.logger.Warn("abdm data push: invalid sender public key",
			zap.String("transaction_id", payload.TransactionID),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sender public key"})
		return
	}
	var senderPub [32]byte
	copy(senderPub[:], senderPubBytes)

	// Decode nonce (24 bytes).
	nonceBytes, err := base64.StdEncoding.DecodeString(payload.KeyMaterial.Nonce)
	if err != nil || len(nonceBytes) != 24 {
		h.logger.Warn("abdm data push: invalid nonce",
			zap.String("transaction_id", payload.TransactionID),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid nonce"})
		return
	}
	var nonce [24]byte
	copy(nonce[:], nonceBytes)

	// Decrypt and parse each entry.
	var observations []canonical.CanonicalObservation
	for i, entry := range payload.Entries {
		encrypted, err := base64.StdEncoding.DecodeString(entry.Content)
		if err != nil {
			h.logger.Warn("abdm data push: base64 decode failed for entry",
				zap.Int("entry_index", i),
				zap.Error(err),
			)
			continue
		}

		plaintext, err := h.keyPair.Decrypt(encrypted, senderPub, nonce)
		if err != nil {
			h.logger.Error("abdm data push: decryption failed for entry",
				zap.Int("entry_index", i),
				zap.Error(err),
			)
			continue
		}

		obs, err := parseFHIRContent(c.Request.Context(), plaintext, consent)
		if err != nil {
			h.logger.Warn("abdm data push: FHIR parse failed for entry",
				zap.Int("entry_index", i),
				zap.Error(err),
			)
			continue
		}
		observations = append(observations, obs...)
	}

	h.logger.Info("abdm data push processed",
		zap.String("transaction_id", payload.TransactionID),
		zap.String("consent_id", payload.ConsentID),
		zap.Int("observation_count", len(observations)),
	)

	c.JSON(http.StatusAccepted, gin.H{
		"status":            "accepted",
		"transaction_id":    payload.TransactionID,
		"observation_count": len(observations),
	})
}

// parseFHIRContent parses a decrypted FHIR Bundle JSON and extracts
// canonical observations with ABDM source metadata.
func parseFHIRContent(_ context.Context, data []byte, consent *crypto.ConsentArtifact) ([]canonical.CanonicalObservation, error) {
	var bundle struct {
		ResourceType string `json:"resourceType"`
		Entry        []struct {
			Resource json.RawMessage `json:"resource"`
		} `json:"entry"`
	}
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, fmt.Errorf("abdm: failed to parse FHIR bundle: %w", err)
	}
	if bundle.ResourceType != "Bundle" {
		return nil, fmt.Errorf("abdm: expected Bundle, got %s", bundle.ResourceType)
	}

	var observations []canonical.CanonicalObservation
	for _, entry := range bundle.Entry {
		var resource struct {
			ResourceType string `json:"resourceType"`
		}
		if err := json.Unmarshal(entry.Resource, &resource); err != nil {
			continue
		}

		if resource.ResourceType != "Observation" {
			continue
		}

		obs := canonical.CanonicalObservation{
			ID:              uuid.New(),
			SourceType:      canonical.SourceABDM,
			ObservationType: canonical.ObsABDMRecords,
			QualityScore:    0.90,
			RawPayload:      entry.Resource,
			ABDMContext: &canonical.ABDMContext{
				ConsentID:    consent.ConsentID,
				HIURequestID: consent.HIURequestID,
			},
		}
		observations = append(observations, obs)
	}

	return observations, nil
}
