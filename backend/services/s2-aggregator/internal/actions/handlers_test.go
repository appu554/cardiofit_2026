package actions

import (
	"context"
	"strings"
	"testing"

	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/google/uuid"
)

func newHandlerHarness(t *testing.T) (*Handler, *InMemoryActionStore, *InMemorySessionStore, *InMemoryOverrideForwarder, SessionContext) {
	t.Helper()
	store := NewInMemoryActionStore()
	sessions := NewInMemorySessionStore()
	fwd := NewInMemoryOverrideForwarder()
	vb := aggregation.NewDefaultViewBuilder()
	h := NewHandler(store, sessions, fwd, vb)
	s, err := StartSession(context.Background(), uuid.New(), sessions)
	if err != nil {
		t.Fatalf("StartSession err = %v", err)
	}
	return h, store, sessions, fwd, s
}

func baseReq(s SessionContext, a Action) ActionRequest {
	return ActionRequest{
		Action:       a,
		PharmacistID: s.PharmacistID,
		ResidentID:   uuid.New(),
		SessionID:    s.SessionID,
		SubjectID:    uuid.New(),
	}
}

func TestHandlerExecuteOpenRecordsAndCountsSession(t *testing.T) {
	h, store, sessions, _, s := newHandlerHarness(t)
	ack, err := h.Execute(context.Background(), baseReq(s, ActionOpen))
	if err != nil {
		t.Fatalf("Execute err = %v", err)
	}
	if ack.ActionID == uuid.Nil || ack.AuditTraceID == uuid.Nil {
		t.Errorf("Acknowledgment missing ids: %+v", ack)
	}
	if store.Count() != 1 {
		t.Errorf("store.Count = %d, want 1", store.Count())
	}
	got, _ := sessions.Get(context.Background(), s.SessionID)
	if got.ActionCount != 1 {
		t.Errorf("session ActionCount = %d, want 1", got.ActionCount)
	}
}

func TestHandlerExecuteOverrideForwardsToKb32(t *testing.T) {
	h, store, _, fwd, s := newHandlerHarness(t)
	req := baseReq(s, ActionOverride)
	req.Reasoning = "goals-of-care alignment per family meeting"
	req.OverrideReasonCodeShort = "GCA"
	ack, err := h.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute err = %v", err)
	}
	if ack.ActionID == uuid.Nil {
		t.Error("ack missing ActionID")
	}
	if store.Count() != 1 {
		t.Errorf("store.Count = %d, want 1", store.Count())
	}
	if fwd.Count() != 1 {
		t.Errorf("forwarder.Count = %d, want 1", fwd.Count())
	}
	// Verify the stored row was canonicalised to populate both vocabs.
	r, _, _ := store.Last()
	if r.OverrideReasonCode != "goals_of_care_aligned" || r.OverrideReasonCodeShort != "GCA" {
		t.Errorf("override codes not canonicalised on store: %+v", r)
	}
}

func TestHandlerExecuteOpenComplexWorkspaceEscalatesAndPropagatesSentinel(t *testing.T) {
	h, store, _, _, s := newHandlerHarness(t)
	req := baseReq(s, ActionOpenComplexWorkspace)
	ack, err := h.Execute(context.Background(), req)
	if err == nil {
		t.Fatal("Execute err = nil, want escalation sentinel")
	}
	if !strings.Contains(err.Error(), "escalation not implemented at Layer 1") {
		t.Errorf("err = %v, want Addendum Part 6 sentinel", err)
	}
	// Ack is still returned and the audit row is still recorded.
	if ack.ActionID == uuid.Nil {
		t.Error("ack missing ActionID after escalation sentinel")
	}
	if store.Count() != 1 {
		t.Errorf("store.Count = %d, want 1 (audit row recorded even when escalation deferred)", store.Count())
	}
}

func TestHandlerExecuteAddNoteRequiresBody(t *testing.T) {
	h, store, _, _, s := newHandlerHarness(t)
	req := baseReq(s, ActionAddNote)
	if _, err := h.Execute(context.Background(), req); err == nil {
		t.Fatal("Execute err = nil, want ErrEmptyNoteBody")
	}
	if store.Count() != 0 {
		t.Errorf("store.Count = %d after rejected request, want 0", store.Count())
	}
	req.NoteBody = "trial period — revisit at next visit"
	if _, err := h.Execute(context.Background(), req); err != nil {
		t.Fatalf("Execute with note body err = %v", err)
	}
}

func TestHandlerExecuteSafetyCriticalBypassMandatoryReasoning(t *testing.T) {
	h, store, _, _, s := newHandlerHarness(t)
	req := baseReq(s, ActionInvokeSafetyCriticalBypass)
	if _, err := h.Execute(context.Background(), req); err == nil {
		t.Fatal("bypass without reasoning err = nil, want ErrReasoningRequired")
	}
	req.Reasoning = "imminent harm — clinical override per attending"
	if _, err := h.Execute(context.Background(), req); err != nil {
		t.Fatalf("bypass with reasoning err = %v", err)
	}
	if store.Count() != 1 {
		t.Errorf("store.Count = %d, want 1", store.Count())
	}
}

func TestHandlerExecuteModifyRequiresReasoning(t *testing.T) {
	h, _, _, _, s := newHandlerHarness(t)
	req := baseReq(s, ActionModify)
	if _, err := h.Execute(context.Background(), req); err == nil {
		t.Fatal("modify without reasoning err = nil, want ErrReasoningRequired")
	}
}

func TestHandlerExecuteOverrideForwardFailurePreservesAuditRow(t *testing.T) {
	h, store, _, fwd, s := newHandlerHarness(t)
	fwd.FailNext()
	req := baseReq(s, ActionOverride)
	req.Reasoning = "patient declined per shared decision making"
	req.OverrideReasonCodeShort = "PPF"
	ack, err := h.Execute(context.Background(), req)
	if err == nil {
		t.Fatal("Execute err = nil, want forward failure")
	}
	if ack.ActionID == uuid.Nil {
		t.Error("ack missing ActionID despite forward failure")
	}
	if store.Count() != 1 {
		t.Errorf("store.Count = %d, want 1 (audit row precedes forwarder)", store.Count())
	}
}

func TestHandlerExecuteAllElevenActionsAcceptedWhenWellFormed(t *testing.T) {
	// Smoke test that every one of the eleven actions can be executed
	// with a minimally well-formed request — guards against the
	// per-action switch silently dropping a case.
	cases := []struct {
		action  Action
		mutator func(*ActionRequest)
	}{
		{ActionOpen, nil},
		{ActionModify, func(r *ActionRequest) { r.Reasoning = "dose adjusted for eGFR trend" }},
		{ActionDefer, nil},
		{ActionOverride, func(r *ActionRequest) {
			r.Reasoning = "deprescribing already underway per geriatrics consult"
			r.OverrideReasonCodeShort = "DUW"
		}},
		{ActionMarkReviewed, nil},
		{ActionFlagForFollowUp, nil},
		{ActionAddNote, func(r *ActionRequest) { r.NoteBody = "revisit at family meeting on Friday" }},
		{ActionOpenComplexWorkspace, nil}, // expects escalation sentinel
		{ActionDrillIntoSubstrate, nil},
		{ActionAcknowledgeRestraintSignal, nil},
		{ActionInvokeSafetyCriticalBypass, func(r *ActionRequest) {
			r.Reasoning = "imminent QT prolongation risk — clinician escalation"
		}},
	}
	if len(cases) != 11 {
		t.Fatalf("test covers %d actions, not 11", len(cases))
	}
	h, store, _, _, s := newHandlerHarness(t)
	for _, c := range cases {
		req := baseReq(s, c.action)
		if c.mutator != nil {
			c.mutator(&req)
		}
		_, err := h.Execute(context.Background(), req)
		if c.action == ActionOpenComplexWorkspace {
			if err == nil {
				t.Errorf("action %q: err = nil, want escalation sentinel", c.action)
			}
			continue
		}
		if err != nil {
			t.Errorf("action %q: Execute err = %v, want nil", c.action, err)
		}
	}
	if store.Count() != 11 {
		t.Errorf("store.Count = %d, want 11 (all eleven actions recorded incl. complex-workspace audit row)", store.Count())
	}
}
