package whatsapp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestSendInteractiveButtons_MaxThree(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	s := NewSender("123456", "token", logger)

	err := s.SendInteractiveButtons("919876543210", "test", []InteractiveButton{
		{ID: "1", Title: "A"},
		{ID: "2", Title: "B"},
		{ID: "3", Title: "C"},
		{ID: "4", Title: "D"},
	})
	if err == nil {
		t.Error("expected error for >3 buttons")
	}
}

func TestSendText_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing auth header")
		}
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["to"] != "919876543210" {
			t.Errorf("expected to=919876543210, got %v", payload["to"])
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"messages":[{"id":"wamid.sent"}]}`))
	}))
	defer srv.Close()

	logger, _ := zap.NewDevelopment()
	s := &Sender{
		apiURL:      srv.URL,
		accessToken: "test-token",
		httpClient:  srv.Client(),
		logger:      logger,
	}

	if err := s.SendText("919876543210", "Hello"); err != nil {
		t.Fatalf("SendText failed: %v", err)
	}
}

func TestSendInteractiveButtons_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["type"] != "interactive" {
			t.Errorf("expected type=interactive, got %v", payload["type"])
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"messages":[{"id":"wamid.sent"}]}`))
	}))
	defer srv.Close()

	logger, _ := zap.NewDevelopment()
	s := &Sender{
		apiURL:      srv.URL,
		accessToken: "token",
		httpClient:  srv.Client(),
		logger:      logger,
	}

	buttons := StandardYesNo(LangHindi)
	if err := s.SendInteractiveButtons("919876543210", "क्या आपको मधुमेह है?", buttons); err != nil {
		t.Fatalf("SendInteractiveButtons failed: %v", err)
	}
}

func TestStandardYesNo_Hindi(t *testing.T) {
	btns := StandardYesNo(LangHindi)
	if len(btns) != 2 {
		t.Fatalf("expected 2 buttons, got %d", len(btns))
	}
	if btns[0].Title != "हाँ" {
		t.Errorf("expected हाँ, got %s", btns[0].Title)
	}
	if btns[1].Title != "नहीं" {
		t.Errorf("expected नहीं, got %s", btns[1].Title)
	}
}

func TestStandardYesNoUnsure_English(t *testing.T) {
	btns := StandardYesNoUnsure(LangEnglish)
	if len(btns) != 3 {
		t.Fatalf("expected 3 buttons, got %d", len(btns))
	}
	if btns[2].Title != "Not sure" {
		t.Errorf("expected 'Not sure', got %s", btns[2].Title)
	}
}
