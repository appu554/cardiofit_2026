package review

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestExtractReviewerID_Valid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expected := uuid.New()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("X-User-ID", expected.String())

	got := extractReviewerID(c)
	if got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestExtractReviewerID_Invalid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("X-User-ID", "not-a-uuid")

	got := extractReviewerID(c)
	if got != uuid.Nil {
		t.Errorf("expected uuid.Nil, got %s", got)
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestExtractReviewerID_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	// No X-User-ID header set.

	got := extractReviewerID(c)
	if got != uuid.Nil {
		t.Errorf("expected uuid.Nil, got %s", got)
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}
