package whatsapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type mockFlowDispatcher struct {
	dispatched []ParsedMessage
}

func (m *mockFlowDispatcher) DispatchWhatsAppMessage(phone string, msg ParsedMessage) error {
	m.dispatched = append(m.dispatched, msg)
	return nil
}

type mockDedup struct {
	seen map[string]bool
}

func (m *mockDedup) IsDuplicate(id string) (bool, error) { return m.seen[id], nil }
func (m *mockDedup) Record(id string) error              { m.seen[id] = true; return nil }

func testHandler() (*WebhookHandler, *mockFlowDispatcher) {
	logger, _ := zap.NewDevelopment()
	dispatch := &mockFlowDispatcher{}
	dedup := &mockDedup{seen: make(map[string]bool)}
	h := NewWebhookHandler("test-secret", "test-verify-token", dispatch, dedup, logger)
	return h, dispatch
}

func signBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestHandleVerification_Success(t *testing.T) {
	h, _ := testHandler()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET",
		"/webhook?hub.mode=subscribe&hub.verify_token=test-verify-token&hub.challenge=challenge123", nil)
	h.HandleVerification(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "challenge123" {
		t.Errorf("expected challenge123, got %s", w.Body.String())
	}
}

func TestHandleVerification_BadToken(t *testing.T) {
	h, _ := testHandler()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET",
		"/webhook?hub.mode=subscribe&hub.verify_token=wrong&hub.challenge=c", nil)
	h.HandleVerification(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestParseIncomingMessage_Text(t *testing.T) {
	msg := IncomingMessage{
		ID:   "wamid.123",
		From: "919876543210",
		Type: "text",
		Text: &struct {
			Body string `json:"body"`
		}{Body: "178"},
	}
	parsed := ParseIncomingMessage(msg)
	if parsed.Type != MsgTypeText {
		t.Errorf("expected TEXT, got %s", parsed.Type)
	}
	if parsed.Text != "178" {
		t.Errorf("expected '178', got '%s'", parsed.Text)
	}
}

func TestParseIncomingMessage_ButtonReply(t *testing.T) {
	msg := IncomingMessage{
		ID:   "wamid.456",
		From: "919876543210",
		Type: "interactive",
		Interactive: &struct {
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
		}{
			Type: "button_reply",
			ButtonReply: &struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			}{ID: "yes_diabetes", Title: "Yes"},
		},
	}
	parsed := ParseIncomingMessage(msg)
	if parsed.Type != MsgTypeButtonReply {
		t.Errorf("expected BUTTON_REPLY, got %s", parsed.Type)
	}
	if parsed.ButtonID != "yes_diabetes" {
		t.Errorf("expected button ID 'yes_diabetes', got '%s'", parsed.ButtonID)
	}
}

func TestHandleIncoming_WithSignature(t *testing.T) {
	h, dispatch := testHandler()
	gin.SetMode(gin.TestMode)

	payload := WebhookPayload{
		Object: "whatsapp_business_account",
		Entry: []Entry{{
			ID: "entry1",
			Changes: []Change{{
				Field: "messages",
				Value: ChangeValue{
					Contacts: []Contact{{WaID: "919876543210"}},
					Messages: []IncomingMessage{{
						ID:   "wamid.789",
						From: "919876543210",
						Type: "text",
						Text: &struct {
							Body string `json:"body"`
						}{Body: "hello"},
					}},
				},
			}},
		}},
	}
	body, _ := json.Marshal(payload)
	sig := signBody("test-secret", body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/webhook", strings.NewReader(string(body)))
	c.Request.Header.Set("X-Hub-Signature-256", sig)
	h.HandleIncoming(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if len(dispatch.dispatched) != 1 {
		t.Fatalf("expected 1 dispatched message, got %d", len(dispatch.dispatched))
	}
	if dispatch.dispatched[0].Text != "hello" {
		t.Errorf("expected 'hello', got '%s'", dispatch.dispatched[0].Text)
	}
}

func TestHandleIncoming_BadSignature(t *testing.T) {
	h, _ := testHandler()
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/webhook", strings.NewReader(`{}`))
	c.Request.Header.Set("X-Hub-Signature-256", "sha256=invalid")
	h.HandleIncoming(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleIncoming_Dedup(t *testing.T) {
	h, dispatch := testHandler()
	gin.SetMode(gin.TestMode)

	payload := WebhookPayload{
		Object: "whatsapp_business_account",
		Entry: []Entry{{
			Changes: []Change{{
				Field: "messages",
				Value: ChangeValue{
					Messages: []IncomingMessage{{
						ID:   "wamid.dup",
						From: "919876543210",
						Type: "text",
						Text: &struct {
							Body string `json:"body"`
						}{Body: "test"},
					}},
				},
			}},
		}},
	}
	body, _ := json.Marshal(payload)

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/webhook", strings.NewReader(string(body)))
		h.appSecret = ""
		h.HandleIncoming(c)
	}

	if len(dispatch.dispatched) != 1 {
		t.Errorf("expected 1 dispatch (dedup), got %d", len(dispatch.dispatched))
	}
}
