package whatsapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// WebhookHandler processes incoming WhatsApp Business API webhooks.
type WebhookHandler struct {
	appSecret    string
	verifyToken  string
	flowDispatch FlowDispatcher
	dedup        MessageDeduplicator
	logger       *zap.Logger
}

// FlowDispatcher routes parsed WhatsApp messages to the intake flow engine.
type FlowDispatcher interface {
	DispatchWhatsAppMessage(patientPhone string, msg ParsedMessage) error
}

// MessageDeduplicator checks and records message IDs to prevent double-processing.
type MessageDeduplicator interface {
	IsDuplicate(messageID string) (bool, error)
	Record(messageID string) error
}

// NewWebhookHandler creates a WhatsApp webhook handler.
func NewWebhookHandler(
	appSecret, verifyToken string,
	flowDispatch FlowDispatcher,
	dedup MessageDeduplicator,
	logger *zap.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		appSecret:    appSecret,
		verifyToken:  verifyToken,
		flowDispatch: flowDispatch,
		dedup:        dedup,
		logger:       logger,
	}
}

// HandleVerification handles the GET webhook verification challenge from WhatsApp.
func (h *WebhookHandler) HandleVerification(c *gin.Context) {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode == "subscribe" && token == h.verifyToken {
		h.logger.Info("WhatsApp webhook verified")
		c.String(http.StatusOK, challenge)
		return
	}
	h.logger.Warn("WhatsApp webhook verification failed",
		zap.String("mode", mode),
		zap.String("token_match", fmt.Sprintf("%v", token == h.verifyToken)),
	)
	c.JSON(http.StatusForbidden, gin.H{"error": "verification failed"})
}

// HandleIncoming processes incoming WhatsApp messages (POST webhook).
func (h *WebhookHandler) HandleIncoming(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("failed to read webhook body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	signature := c.GetHeader("X-Hub-Signature-256")
	if !h.verifySignature(body, signature) {
		h.logger.Warn("WhatsApp webhook signature mismatch")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error("failed to parse webhook payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}
			for _, msg := range change.Value.Messages {
				if err := h.processMessage(msg, change.Value.Contacts); err != nil {
					h.logger.Error("failed to process WhatsApp message",
						zap.String("message_id", msg.ID),
						zap.Error(err),
					)
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "received"})
}

func (h *WebhookHandler) processMessage(msg IncomingMessage, contacts []Contact) error {
	isDup, err := h.dedup.IsDuplicate(msg.ID)
	if err != nil {
		h.logger.Warn("dedup check failed, processing anyway", zap.Error(err))
	}
	if isDup {
		h.logger.Debug("duplicate WhatsApp message, skipping", zap.String("id", msg.ID))
		return nil
	}

	parsed := ParseIncomingMessage(msg)

	phone := msg.From
	if phone == "" && len(contacts) > 0 {
		phone = contacts[0].WaID
	}

	if err := h.flowDispatch.DispatchWhatsAppMessage(phone, parsed); err != nil {
		return fmt.Errorf("dispatch message from %s: %w", phone, err)
	}

	_ = h.dedup.Record(msg.ID)
	return nil
}

func (h *WebhookHandler) verifySignature(body []byte, signature string) bool {
	if h.appSecret == "" {
		return true
	}
	if signature == "" {
		return false
	}

	parts := strings.SplitN(signature, "=", 2)
	if len(parts) != 2 || parts[0] != "sha256" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.appSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(parts[1]))
}

// --- WhatsApp Business API webhook payload types ---

type WebhookPayload struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

type Entry struct {
	ID      string   `json:"id"`
	Changes []Change `json:"changes"`
}

type Change struct {
	Field string      `json:"field"`
	Value ChangeValue `json:"value"`
}

type ChangeValue struct {
	MessagingProduct string            `json:"messaging_product"`
	Metadata         Metadata          `json:"metadata"`
	Contacts         []Contact         `json:"contacts"`
	Messages         []IncomingMessage `json:"messages"`
	Statuses         []StatusUpdate    `json:"statuses,omitempty"`
}

type Metadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type Contact struct {
	Profile struct {
		Name string `json:"name"`
	} `json:"profile"`
	WaID string `json:"wa_id"`
}

type IncomingMessage struct {
	ID        string `json:"id"`
	From      string `json:"from"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Text      *struct {
		Body string `json:"body"`
	} `json:"text,omitempty"`
	Interactive *struct {
		Type        string `json:"type"`
		ButtonReply *struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"button_reply,omitempty"`
		ListReply *struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"list_reply,omitempty"`
	} `json:"interactive,omitempty"`
	Location *struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Name      string  `json:"name,omitempty"`
		Address   string  `json:"address,omitempty"`
	} `json:"location,omitempty"`
}

type StatusUpdate struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type MessageType string

const (
	MsgTypeText        MessageType = "TEXT"
	MsgTypeButtonReply MessageType = "BUTTON_REPLY"
	MsgTypeListReply   MessageType = "LIST_REPLY"
	MsgTypeLocation    MessageType = "LOCATION"
	MsgTypeUnsupported MessageType = "UNSUPPORTED"
)

type ParsedMessage struct {
	ID        string      `json:"id"`
	From      string      `json:"from"`
	Timestamp string      `json:"timestamp"`
	Type      MessageType `json:"type"`
	Text      string      `json:"text,omitempty"`
	ButtonID  string      `json:"button_id,omitempty"`
	ListID    string      `json:"list_id,omitempty"`
	Latitude  float64     `json:"latitude,omitempty"`
	Longitude float64     `json:"longitude,omitempty"`
}

func ParseIncomingMessage(msg IncomingMessage) ParsedMessage {
	parsed := ParsedMessage{
		ID:        msg.ID,
		From:      msg.From,
		Timestamp: msg.Timestamp,
	}

	switch msg.Type {
	case "text":
		parsed.Type = MsgTypeText
		if msg.Text != nil {
			parsed.Text = msg.Text.Body
		}
	case "interactive":
		if msg.Interactive != nil {
			switch msg.Interactive.Type {
			case "button_reply":
				parsed.Type = MsgTypeButtonReply
				if msg.Interactive.ButtonReply != nil {
					parsed.ButtonID = msg.Interactive.ButtonReply.ID
					parsed.Text = msg.Interactive.ButtonReply.Title
				}
			case "list_reply":
				parsed.Type = MsgTypeListReply
				if msg.Interactive.ListReply != nil {
					parsed.ListID = msg.Interactive.ListReply.ID
					parsed.Text = msg.Interactive.ListReply.Title
				}
			default:
				parsed.Type = MsgTypeUnsupported
			}
		}
	case "location":
		parsed.Type = MsgTypeLocation
		if msg.Location != nil {
			parsed.Latitude = msg.Location.Latitude
			parsed.Longitude = msg.Location.Longitude
		}
	default:
		parsed.Type = MsgTypeUnsupported
	}

	return parsed
}
