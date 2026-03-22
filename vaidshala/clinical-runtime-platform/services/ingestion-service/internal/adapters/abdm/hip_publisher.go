package abdm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cardiofit/ingestion-service/internal/crypto"
	"go.uber.org/zap"
)

// HIPDataPushRequest contains the data and routing information needed
// to push encrypted FHIR bundles to an HIU via the ABDM gateway.
type HIPDataPushRequest struct {
	TransactionID  string   `json:"transaction_id"`
	ConsentID      string   `json:"consent_id"`
	HIUCallbackURL string   `json:"hiu_callback_url"`
	PatientID      string   `json:"patient_id"`
	FHIRBundles    [][]byte `json:"fhir_bundles"`
}

// HIPPublisher handles outbound health data publishing when our system
// acts as a Health Information Provider (HIP) in the ABDM network.
type HIPPublisher struct {
	keyPair     *crypto.X25519KeyPair
	abdmBaseURL string
	accessToken string
	httpClient  *http.Client
	logger      *zap.Logger
}

// NewHIPPublisher creates a publisher configured with our HIP key pair
// and ABDM gateway credentials.
func NewHIPPublisher(keyPair *crypto.X25519KeyPair, abdmBaseURL, accessToken string, logger *zap.Logger) *HIPPublisher {
	return &HIPPublisher{
		keyPair:     keyPair,
		abdmBaseURL: abdmBaseURL,
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// PublishHealthData encrypts each FHIR bundle with the HIU's public key
// and POSTs the encrypted payload to the HIU's callback URL.
func (p *HIPPublisher) PublishHealthData(req HIPDataPushRequest, hiuPublicKey [32]byte) error {
	var entries []ABDMDataEntry
	var nonce [24]byte
	var keyMaterial ABDMKeyMaterial

	for i, bundle := range req.FHIRBundles {
		encrypted, n, err := p.keyPair.Encrypt(bundle, hiuPublicKey)
		if err != nil {
			return fmt.Errorf("hip: encrypt bundle %d failed: %w", i, err)
		}
		// Use the nonce from the first encryption for the key material.
		if i == 0 {
			nonce = n
			keyMaterial = ABDMKeyMaterial{
				SenderPublicKey: base64.StdEncoding.EncodeToString(p.keyPair.PublicKey[:]),
				Nonce:           base64.StdEncoding.EncodeToString(nonce[:]),
			}
		}

		entries = append(entries, ABDMDataEntry{
			CareContextRef: fmt.Sprintf("patient/%s/bundle/%d", req.PatientID, i),
			Content:        base64.StdEncoding.EncodeToString(encrypted),
		})
	}

	payload := ABDMDataPushPayload{
		TransactionID: req.TransactionID,
		ConsentID:     req.ConsentID,
		KeyMaterial:   keyMaterial,
		Entries:       entries,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("hip: marshal payload failed: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, req.HIUCallbackURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("hip: create request failed: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.accessToken)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("hip: POST to HIU callback failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hip: HIU callback returned %d: %s", resp.StatusCode, string(respBody))
	}

	p.logger.Info("hip: health data published",
		zap.String("transaction_id", req.TransactionID),
		zap.String("consent_id", req.ConsentID),
		zap.Int("bundle_count", len(req.FHIRBundles)),
	)

	return nil
}

// NotifyABDMDataAvailable posts a notification to the ABDM gateway
// indicating that health information is ready for collection.
func (p *HIPPublisher) NotifyABDMDataAvailable(transactionID, consentID string) error {
	notification := map[string]interface{}{
		"notification": map[string]interface{}{
			"transaction_id": transactionID,
			"consent_id":     consentID,
			"status":         "TRANSFERRED",
			"notified_at":    time.Now().UTC().Format(time.RFC3339),
		},
	}

	body, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("hip: marshal notification failed: %w", err)
	}

	url := fmt.Sprintf("%s/api/v3/health-information/notify", p.abdmBaseURL)
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("hip: create notification request failed: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.accessToken)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("hip: ABDM notification POST failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hip: ABDM notification returned %d: %s", resp.StatusCode, string(respBody))
	}

	p.logger.Info("hip: ABDM notified of data availability",
		zap.String("transaction_id", transactionID),
		zap.String("consent_id", consentID),
	)

	return nil
}
