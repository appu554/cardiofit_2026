package abdm

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ConsentCollector struct {
	abhaClient *ABHAClient
	logger     *zap.Logger
}

func NewConsentCollector(abhaClient *ABHAClient, logger *zap.Logger) *ConsentCollector {
	return &ConsentCollector{abhaClient: abhaClient, logger: logger}
}

type ConsentPurpose string

const (
	PurposeCareMgmt     ConsentPurpose = "CAREMGT"
	PurposeBreakGlass   ConsentPurpose = "BTG"
	PurposePublicHealth ConsentPurpose = "PUBHLTH"
	PurposeInsurance    ConsentPurpose = "HPAYMT"
	PurposeResearch     ConsentPurpose = "DSRCH"
)

type ConsentStatus string

const (
	ConsentRequested ConsentStatus = "REQUESTED"
	ConsentGranted   ConsentStatus = "GRANTED"
	ConsentDenied    ConsentStatus = "DENIED"
	ConsentRevoked   ConsentStatus = "REVOKED"
	ConsentExpired   ConsentStatus = "EXPIRED"
)

type ConsentRequest struct {
	ID            uuid.UUID      `json:"id"`
	PatientID     uuid.UUID      `json:"patient_id"`
	ABHANumber    string         `json:"abha_number"`
	Purpose       ConsentPurpose `json:"purpose"`
	HITypes       []string       `json:"hi_types"`
	DateRangeFrom time.Time      `json:"date_range_from"`
	DateRangeTo   time.Time      `json:"date_range_to"`
	ExpiryDate    time.Time      `json:"expiry_date"`
	DPDPAConsent  bool           `json:"dpdpa_consent"`
	DPDPAConsentAt *time.Time    `json:"dpdpa_consent_at,omitempty"`
	ABDMConsentID string         `json:"abdm_consent_id,omitempty"`
	Status        ConsentStatus  `json:"status"`
	CreatedAt     time.Time      `json:"created_at"`
}

type DPDPAConsentData struct {
	PatientID       uuid.UUID `json:"patient_id"`
	ConsentVersion  string    `json:"consent_version"`
	PurposeOfUse    string    `json:"purpose_of_use"`
	DataCategories  []string  `json:"data_categories"`
	RetentionPeriod string    `json:"retention_period"`
	GrantedAt       time.Time `json:"granted_at"`
	Channel         string    `json:"channel"`
	IPAddress       string    `json:"ip_address,omitempty"`
	UserAgent       string    `json:"user_agent,omitempty"`
}

func (cc *ConsentCollector) CollectDPDPAConsent(data DPDPAConsentData) error {
	if data.ConsentVersion == "" {
		return fmt.Errorf("DPDPA consent version is required")
	}
	if len(data.DataCategories) == 0 {
		return fmt.Errorf("at least one data category is required")
	}

	cc.logger.Info("DPDPA consent collected",
		zap.String("patient_id", data.PatientID.String()),
		zap.String("version", data.ConsentVersion),
		zap.String("channel", data.Channel),
		zap.Strings("categories", data.DataCategories),
	)

	return nil
}

func (cc *ConsentCollector) InitiateABDMConsent(req ConsentRequest) (*ConsentRequest, error) {
	if !req.DPDPAConsent {
		return nil, fmt.Errorf("DPDPA consent must be collected before ABDM consent")
	}

	if req.ABHANumber == "" {
		return nil, fmt.Errorf("ABHA number is required for ABDM consent")
	}

	artifact := map[string]interface{}{
		"purpose": map[string]string{
			"text": string(req.Purpose),
			"code": string(req.Purpose),
		},
		"patient": map[string]string{
			"id": req.ABHANumber,
		},
		"hiTypes": req.HITypes,
		"permission": map[string]interface{}{
			"dateRange": map[string]string{
				"from": req.DateRangeFrom.Format(time.RFC3339),
				"to":   req.DateRangeTo.Format(time.RFC3339),
			},
			"dataEraseAt": req.ExpiryDate.Format(time.RFC3339),
			"frequency": map[string]interface{}{
				"unit":  "HOUR",
				"value": 1,
			},
		},
		"hiu": map[string]string{
			"id": "cardiofit-hiu",
		},
	}

	body, _ := json.Marshal(artifact)
	resp, err := cc.abhaClient.post("/api/v3/consent/request/init", json.RawMessage(body))
	if err != nil {
		return nil, fmt.Errorf("ABDM consent init: %w", err)
	}

	var result struct {
		ConsentRequestID string `json:"consentRequestId"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse consent response: %w", err)
	}

	req.ABDMConsentID = result.ConsentRequestID
	req.Status = ConsentRequested

	cc.logger.Info("ABDM consent request initiated",
		zap.String("consent_id", result.ConsentRequestID),
		zap.String("patient_id", req.PatientID.String()),
	)

	return &req, nil
}

func (cc *ConsentCollector) CheckConsentStatus(consentRequestID string) (ConsentStatus, error) {
	resp, err := cc.abhaClient.get("/api/v3/consent/request/status/" + consentRequestID)
	if err != nil {
		return "", fmt.Errorf("check consent status: %w", err)
	}

	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("parse consent status: %w", err)
	}

	return ConsentStatus(result.Status), nil
}
