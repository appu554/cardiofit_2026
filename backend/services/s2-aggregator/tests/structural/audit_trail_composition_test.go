// audit_trail_composition_test.go — v1.0 Part 17 Category 6
// (audit trail). Composition tests across the eleven pharmacist actions,
// visibility-class enforcement at the per-row + aggregate level, the
// escalation-event log-only invariant (runtime mirror of the AST-parse
// structural test), and drill-through audit emission.
package structural

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/actions"
	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/audit"
	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// elevenActions is the canonical ordered set per v1.0 Part 12.1.
var elevenActions = []actions.Action{
	actions.ActionOpen,
	actions.ActionModify,
	actions.ActionDefer,
	actions.ActionOverride,
	actions.ActionMarkReviewed,
	actions.ActionFlagForFollowUp,
	actions.ActionAddNote,
	actions.ActionOpenComplexWorkspace,
	actions.ActionDrillIntoSubstrate,
	actions.ActionAcknowledgeRestraintSignal,
	actions.ActionInvokeSafetyCriticalBypass,
}

// TestAudit_AllElevenActions_LoggedCorrectly — exercise each of the 11
// actions; assert an AuditEvent is emitted with EventType
// EventPharmacistAction.
func TestAudit_AllElevenActions_LoggedCorrectly(t *testing.T) {
	store := actions.NewInMemoryActionStore()
	sessions := actions.NewInMemorySessionStore()
	fwd := actions.NewInMemoryOverrideForwarder()
	vb := aggregation.NewDefaultViewBuilder()
	emitter := audit.NewMemoryEmitter()
	h := actions.NewHandler(store, sessions, fwd, vb).WithAuditEmitter(emitter)

	pid := uuid.New()
	sess, err := actions.StartSession(context.Background(), pid, sessions)
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}

	for _, a := range elevenActions {
		req := actions.ActionRequest{
			Action:       a,
			PharmacistID: pid,
			ResidentID:   uuid.New(),
			SessionID:    sess.SessionID,
			SubjectID:    uuid.New(),
		}
		// Populate per-action mandatory fields.
		switch a {
		case actions.ActionModify, actions.ActionInvokeSafetyCriticalBypass:
			req.Reasoning = "clinical-judgment-based decision per family discussion last week"
		case actions.ActionOverride:
			req.Reasoning = "goals-of-care alignment per documented family meeting"
			req.OverrideReasonCodeShort = "GCA"
		case actions.ActionDefer, actions.ActionAcknowledgeRestraintSignal:
			// optional reasoning — leave empty
		case actions.ActionAddNote:
			req.NoteBody = "monitoring already arranged via pathology"
		}

		_, execErr := h.Execute(context.Background(), req)
		// OpenComplexWorkspace + Override propagate sentinel/forwarder
		// results that are non-nil but the audit row + emitter
		// emission run BEFORE those side effects per Handler.Execute.
		if execErr != nil &&
			a != actions.ActionOpenComplexWorkspace {
			// Override forward succeeds (forwarder in-memory accepts);
			// only OpenComplexWorkspace returns sentinel ("not yet
			// implemented Layer 3").
			t.Errorf("Execute(%s) unexpected err: %v", a, execErr)
		}
	}

	// We expect one EventPharmacistAction per action; OpenComplexWorkspace
	// returned a sentinel but the audit row was emitted BEFORE the side
	// effect per handler contract.
	pharmacistEvents := emitter.EventsOfType(audit.EventPharmacistAction)
	if len(pharmacistEvents) != len(elevenActions) {
		t.Errorf("expected %d EventPharmacistAction rows; got %d", len(elevenActions), len(pharmacistEvents))
	}
	// Severity 1 (primary algorithmic / cognitive event) per Task 6 contract.
	for _, e := range pharmacistEvents {
		if e.Severity != 1 {
			t.Errorf("action audit row severity = %d; want 1", e.Severity)
		}
		if e.PharmacistID != pid {
			t.Errorf("PharmacistID mismatch on audit row")
		}
	}
}

// TestAudit_VisibilityClassEnforcement_PharmacistA_CannotReadPharmacistB —
// cross-pharmacist PDP read attempt → ErrCrossPharmacistRead.
func TestAudit_VisibilityClassEnforcement_PharmacistA_CannotReadPharmacistB(t *testing.T) {
	a := uuid.New()
	b := uuid.New()
	// Pharmacist a attempting to read pharmacist b's row.
	if err := audit.EnforcePDPRead(a, b); !errors.Is(err, audit.ErrCrossPharmacistRead) {
		t.Errorf("expected ErrCrossPharmacistRead for cross-pharmacist PDP read; got %v", err)
	}
	// Pharmacist a reading own row succeeds.
	if err := audit.EnforcePDPRead(a, a); err != nil {
		t.Errorf("self-read should succeed; got %v", err)
	}
	// uuid.Nil on either side is rejected (prevents accidental
	// nil-attributed reads).
	if err := audit.EnforcePDPRead(uuid.Nil, a); !errors.Is(err, audit.ErrCrossPharmacistRead) {
		t.Error("nil requester should be rejected")
	}
	if err := audit.EnforcePDPRead(a, uuid.Nil); !errors.Is(err, audit.ErrCrossPharmacistRead) {
		t.Error("nil owner should be rejected")
	}
}

// TestAudit_AggregateRead_NonClinicalInformaticsRole_Denied — non-CI/
// non-ESC roles attempting aggregate → ErrSurveillanceAttempt.
func TestAudit_AggregateRead_NonClinicalInformaticsRole_Denied(t *testing.T) {
	cases := []string{"", "manager", "employer", "supervisor", "facility_admin"}
	for _, role := range cases {
		if err := audit.EnforcePDPAggregateRead(role); !errors.Is(err, audit.ErrSurveillanceAttempt) {
			t.Errorf("role=%q: expected ErrSurveillanceAttempt; got %v", role, err)
		}
	}
	// Authorised roles pass.
	for _, role := range []string{audit.RoleClinicalInformatics, audit.RoleEthicsSteeringCommittee} {
		if err := audit.EnforcePDPAggregateRead(role); err != nil {
			t.Errorf("role=%q should be authorised for aggregate read; got %v", role, err)
		}
	}
}

// TestAudit_EscalationEvent_LogOnly_NoReadPath — runtime mirror of the
// AST-parse structural assertion from
// no_surveillance_reader_test.go: assert that escalation events DO
// emit (so the write path works) and that the audit MemoryEmitter
// retains them but the audit package exposes NO function for filtering
// them by pharmacist.
//
// We assert the write-path works via the public Capture API and rely
// on the AST-parse test (TestNoSurveillanceReader_FunctionNames) for
// the no-read-path proof. This test catches the case where the write
// path silently drops escalation events.
func TestAudit_EscalationEvent_LogOnly_NoReadPath(t *testing.T) {
	mem := audit.NewMemoryEmitter()
	em := audit.NewEscalationEventEmitter(mem)
	ev := aggregation.EscalationEvent{
		PharmacistID: uuid.New(),
		ResidentID:   uuid.New(),
		SessionID:    uuid.New(),
		FromLayer:    1,
		ToLayer:      3,
		TriggeredBy:  aggregation.TriggerPharmacistInitiated,
		Timestamp:    time.Now().UTC(),
	}
	if err := em.Capture(context.Background(), ev); err != nil {
		t.Fatalf("Capture: %v", err)
	}
	// The MemoryEmitter (test fake) does retain — but the AUDIT
	// PACKAGE's public surface area does NOT expose a per-pharmacist
	// reader. Verified structurally by the AST-parse test.
	got := mem.EventsOfType(audit.EventCognitiveEscalation)
	if len(got) != 1 {
		t.Errorf("expected 1 cognitive_escalation row; got %d", len(got))
	}
	if got[0].Severity != 1 {
		t.Errorf("cognitive_escalation severity should be 1; got %d", got[0].Severity)
	}
	// Subject must be the canonical tag for log-search ergonomics.
	if got[0].Subject != "cognitive_escalation" {
		t.Errorf("cognitive_escalation Subject = %q; want %q", got[0].Subject, "cognitive_escalation")
	}
}

// TestAudit_DrillThroughEvents_Logged — drill-through events fan out
// via the audit emitter. We assert by emitting a synthetic
// EventDrillThrough row directly (the drill-through audit wrapper
// emits this via the same Emitter interface).
func TestAudit_DrillThroughEvents_Logged(t *testing.T) {
	mem := audit.NewMemoryEmitter()
	evt := audit.AuditEvent{
		TraceID:      uuid.New(),
		EventType:    audit.EventDrillThrough,
		Severity:     3,
		PharmacistID: uuid.New(),
		ResidentID:   uuid.New(),
		SessionID:    uuid.New(),
		Subject:      "substrate_observation",
		Payload:      map[string]any{"observation_id": uuid.New().String()},
		OccurredAt:   time.Now(),
	}
	if err := mem.Emit(context.Background(), evt); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	got := mem.EventsOfType(audit.EventDrillThrough)
	if len(got) != 1 {
		t.Errorf("expected 1 drill_through row; got %d", len(got))
	}
}

// TestAudit_ViewRenderEmitter_RoundTrip — the aggregation-side
// ViewRenderEmitter adapter fans EmitViewRender into an AuditEvent
// with EventType=EventViewRender.
func TestAudit_ViewRenderEmitter_RoundTrip(t *testing.T) {
	mem := audit.NewMemoryEmitter()
	adapter := audit.NewViewRenderAdapter(mem)
	pid := uuid.New()
	rid := uuid.New()
	sid := uuid.New()
	asOf := time.Now()
	req := aggregation.WorkspaceRequest{
		ResidentID: rid, PharmacistID: pid, SessionID: sid,
		AsOf: asOf, EntryPath: aggregation.EntryPathWorklist,
	}
	if err := adapter.EmitViewRender(context.Background(), req, 1); err != nil {
		t.Fatalf("EmitViewRender: %v", err)
	}
	got := mem.EventsOfType(audit.EventViewRender)
	if len(got) != 1 {
		t.Fatalf("expected 1 view_render row; got %d", len(got))
	}
	if got[0].PharmacistID != pid || got[0].ResidentID != rid || got[0].SessionID != sid {
		t.Errorf("view_render audit row identity drift: %+v", got[0])
	}
	if layer, ok := got[0].Payload["layer"].(int); !ok || layer != 1 {
		t.Errorf("view_render payload layer mismatch; got %+v", got[0].Payload)
	}
	if ep, ok := got[0].Payload["entry_path"].(string); !ok || ep != string(aggregation.EntryPathWorklist) {
		t.Errorf("view_render payload entry_path mismatch; got %+v", got[0].Payload)
	}
}

// TestAudit_EthicsLogFanout — emitting through the EthicsLogEmitter
// puts a serialized AuditEvent into the EthicsLog with the AuditEvent's
// TraceID as DecisionID.
func TestAudit_EthicsLogFanout(t *testing.T) {
	logger := audit.NewMemoryLogger()
	em := audit.NewEthicsLogEmitter(logger)
	trace := uuid.New()
	evt := audit.AuditEvent{
		TraceID:    trace,
		EventType:  audit.EventPharmacistAction,
		Severity:   1,
		Subject:    "override",
		Payload:    map[string]any{"action": "override"},
		OccurredAt: time.Now(),
	}
	if err := em.Emit(context.Background(), evt); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	entries := logger.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 ethics_log entry; got %d", len(entries))
	}
	if entries[0].DecisionID != trace {
		t.Errorf("DecisionID drift: got %s want %s", entries[0].DecisionID, trace)
	}
	if entries[0].EntryType != substrate_types.EthicsEntryTypeDecision {
		t.Errorf("EntryType: got %s", entries[0].EntryType)
	}
	if !strings.Contains(entries[0].Description, `"EventType":"pharmacist_action"`) {
		t.Errorf("Description should carry serialized event_type; got %q", entries[0].Description)
	}
}
