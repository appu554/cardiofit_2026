package whatsapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type Sender struct {
	apiURL        string
	accessToken   string
	phoneNumberID string
	httpClient    *http.Client
	logger        *zap.Logger
}

func NewSender(phoneNumberID, accessToken string, logger *zap.Logger) *Sender {
	return &Sender{
		apiURL:        fmt.Sprintf("https://graph.facebook.com/v21.0/%s/messages", phoneNumberID),
		accessToken:   accessToken,
		phoneNumberID: phoneNumberID,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		logger:        logger,
	}
}

func (s *Sender) SendTemplate(to string, templateID TemplateID, lang string, params []string) error {
	components := make([]map[string]interface{}, 0)
	if len(params) > 0 {
		parameters := make([]map[string]interface{}, len(params))
		for i, p := range params {
			parameters[i] = map[string]interface{}{
				"type": "text",
				"text": p,
			}
		}
		components = append(components, map[string]interface{}{
			"type":       "body",
			"parameters": parameters,
		})
	}

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "template",
		"template": map[string]interface{}{
			"name": string(templateID),
			"language": map[string]string{
				"code": lang,
			},
			"components": components,
		},
	}

	return s.send(payload)
}

func (s *Sender) SendInteractiveButtons(to, bodyText string, buttons []InteractiveButton) error {
	if len(buttons) > 3 {
		return fmt.Errorf("WhatsApp allows max 3 buttons, got %d", len(buttons))
	}

	waButtons := make([]map[string]interface{}, len(buttons))
	for i, b := range buttons {
		waButtons[i] = map[string]interface{}{
			"type": "reply",
			"reply": map[string]string{
				"id":    b.ID,
				"title": b.Title,
			},
		}
	}

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "interactive",
		"interactive": map[string]interface{}{
			"type": "button",
			"body": map[string]string{
				"text": bodyText,
			},
			"action": map[string]interface{}{
				"buttons": waButtons,
			},
		},
	}

	return s.send(payload)
}

func (s *Sender) SendText(to, text string) error {
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "text",
		"text": map[string]string{
			"body": text,
		},
	}
	return s.send(payload)
}

func (s *Sender) send(payload map[string]interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal WhatsApp payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("WhatsApp API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		s.logger.Error("WhatsApp API error",
			zap.Int("status", resp.StatusCode),
			zap.String("response", string(respBody)),
		)
		return fmt.Errorf("WhatsApp API returned %d: %s", resp.StatusCode, string(respBody))
	}

	s.logger.Debug("WhatsApp message sent",
		zap.String("to", fmt.Sprintf("%v", payload["to"])),
		zap.String("type", fmt.Sprintf("%v", payload["type"])),
	)
	return nil
}
