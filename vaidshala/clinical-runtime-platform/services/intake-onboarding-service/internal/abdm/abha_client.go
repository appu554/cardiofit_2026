package abdm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type ABHAClient struct {
	baseURL      string
	clientID     string
	clientSecret string
	httpClient   *http.Client
	logger       *zap.Logger
	token        *accessToken
}

type accessToken struct {
	Token     string
	ExpiresAt time.Time
}

type ABHAConfig struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	IsSandbox    bool
}

func NewABHAClient(cfg ABHAConfig, logger *zap.Logger) *ABHAClient {
	return &ABHAClient{
		baseURL:      cfg.BaseURL,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		logger:       logger,
	}
}

func (c *ABHAClient) InitAadhaarOTP(aadhaarNumber string) (string, error) {
	if err := c.ensureToken(); err != nil {
		return "", err
	}

	payload := map[string]string{"aadhaar": aadhaarNumber}
	resp, err := c.post("/api/v3/enrollment/request/otp", payload)
	if err != nil {
		return "", fmt.Errorf("init aadhaar OTP: %w", err)
	}

	var result struct {
		TxnID string `json:"txnId"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("parse aadhaar OTP response: %w", err)
	}

	c.logger.Info("ABHA Aadhaar OTP initiated", zap.String("txn_id", result.TxnID))
	return result.TxnID, nil
}

func (c *ABHAClient) VerifyAadhaarOTP(txnID, otp string) (*ABHAAccount, error) {
	if err := c.ensureToken(); err != nil {
		return nil, err
	}

	payload := map[string]string{
		"txnId": txnID,
		"otp":   otp,
	}
	resp, err := c.post("/api/v3/enrollment/enrol/byAadhaar", payload)
	if err != nil {
		return nil, fmt.Errorf("verify aadhaar OTP: %w", err)
	}

	var account ABHAAccount
	if err := json.Unmarshal(resp, &account); err != nil {
		return nil, fmt.Errorf("parse ABHA account: %w", err)
	}

	c.logger.Info("ABHA account created",
		zap.String("abha_number", account.ABHANumber),
		zap.String("abha_address", account.ABHAAddress),
	)
	return &account, nil
}

func (c *ABHAClient) LinkABHA(abhaNumber string) (string, error) {
	if err := c.ensureToken(); err != nil {
		return "", err
	}

	payload := map[string]string{"abhaNumber": abhaNumber}
	resp, err := c.post("/api/v3/profile/link/init", payload)
	if err != nil {
		return "", fmt.Errorf("link ABHA init: %w", err)
	}

	var result struct {
		TxnID string `json:"txnId"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("parse link init response: %w", err)
	}

	return result.TxnID, nil
}

func (c *ABHAClient) ConfirmLinkABHA(txnID, otp string) (*ABHAProfile, error) {
	if err := c.ensureToken(); err != nil {
		return nil, err
	}

	payload := map[string]string{
		"txnId": txnID,
		"otp":   otp,
	}
	resp, err := c.post("/api/v3/profile/link/confirm", payload)
	if err != nil {
		return nil, fmt.Errorf("confirm ABHA link: %w", err)
	}

	var profile ABHAProfile
	if err := json.Unmarshal(resp, &profile); err != nil {
		return nil, fmt.Errorf("parse ABHA profile: %w", err)
	}

	c.logger.Info("ABHA linked successfully",
		zap.String("abha_number", profile.ABHANumber),
	)
	return &profile, nil
}

func (c *ABHAClient) FetchProfile(abhaNumber string) (*ABHAProfile, error) {
	if err := c.ensureToken(); err != nil {
		return nil, err
	}

	resp, err := c.get("/api/v3/profile/account/" + abhaNumber)
	if err != nil {
		return nil, fmt.Errorf("fetch ABHA profile: %w", err)
	}

	var profile ABHAProfile
	if err := json.Unmarshal(resp, &profile); err != nil {
		return nil, fmt.Errorf("parse ABHA profile: %w", err)
	}

	return &profile, nil
}

type ABHAAccount struct {
	ABHANumber  string `json:"ABHANumber"`
	ABHAAddress string `json:"preferredAbhaAddress"`
	Name        string `json:"name"`
	Gender      string `json:"gender"`
	DOB         string `json:"dayOfBirth"`
	MOB         string `json:"monthOfBirth"`
	YOB         string `json:"yearOfBirth"`
	Mobile      string `json:"mobile"`
	Token       string `json:"token"`
}

type ABHAProfile struct {
	ABHANumber  string `json:"healthIdNumber"`
	ABHAAddress string `json:"healthId"`
	Name        string `json:"name"`
	Gender      string `json:"gender"`
	DOB         string `json:"dateOfBirth"`
	Mobile      string `json:"mobile"`
	Address     string `json:"address"`
	State       string `json:"stateName"`
	District    string `json:"districtName"`
}

func (c *ABHAClient) ensureToken() error {
	if c.token != nil && time.Now().Before(c.token.ExpiresAt) {
		return nil
	}

	payload := map[string]string{
		"clientId":     c.clientID,
		"clientSecret": c.clientSecret,
		"grantType":    "client_credentials",
	}
	body, _ := json.Marshal(payload)

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/v3/token/generate",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("ABDM token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ABDM token failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		AccessToken string `json:"accessToken"`
		ExpiresIn   int    `json:"expiresIn"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("parse ABDM token: %w", err)
	}

	c.token = &accessToken{
		Token:     result.AccessToken,
		ExpiresAt: time.Now().Add(time.Duration(result.ExpiresIn) * time.Second),
	}
	c.logger.Debug("ABDM access token refreshed")
	return nil
}

func (c *ABHAClient) post(path string, payload interface{}) ([]byte, error) {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != nil {
		req.Header.Set("Authorization", "Bearer "+c.token.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ABDM API %s returned %d: %s", path, resp.StatusCode, string(data))
	}
	return data, nil
}

func (c *ABHAClient) get(path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.token != nil {
		req.Header.Set("Authorization", "Bearer "+c.token.Token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ABDM API %s returned %d: %s", path, resp.StatusCode, string(data))
	}
	return data, nil
}
