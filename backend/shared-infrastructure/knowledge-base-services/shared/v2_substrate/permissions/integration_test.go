// Package permissions — integration test for Task 9 (Phase 1a).
//
// TestBottomUpAdoptionMotion_EndToEnd exercises the full substrate chain:
//
//  1. Pharmacist self-view via PharmacistView succeeds without any consent or
//     middleware records (POA/PDP self path).
//  2. Employer read rejected when both stores are empty (no ViewPermission).
//  3. Employer read rejected when ViewPermission exists but DataAggregationConsent
//     is absent (two-store check).
//  4. Employer read allowed when both ViewPermission and DataAggregationConsent
//     are present (happy path through Middleware.Wrap).
//  5. Pharmacist files a Contestation; the record is visible to both parties
//     (ListByPharmacist and ListByKPIType both return the same record).
package permissions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cardiofit/shared/v2_substrate/contestation"
	"github.com/cardiofit/shared/v2_substrate/permissions/views"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Fake PharmacistSource (satisfies views.PharmacistSource)
// ---------------------------------------------------------------------------

type fakePharmacistSource struct{}

func (f fakePharmacistSource) RIRTrajectoryFor(_ context.Context, pharmacistID uuid.UUID, _ int) (views.RIRTrajectory, error) {
	return views.RIRTrajectory{
		AuthorID: pharmacistID,
		Points: []views.TrajectoryPoint{
			{PeriodStart: "2026-03-01", RIR: 0.72},
			{PeriodStart: "2026-04-01", RIR: 0.81},
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Helper: build middleware with empty in-memory stores
// ---------------------------------------------------------------------------

func setupMiddleware(t *testing.T) (*Middleware, *InMemoryStore, *InMemoryDataConsentStore) {
	t.Helper()
	pstore := &InMemoryStore{}
	cstore := &InMemoryDataConsentStore{}
	audit := &NoopAuditEmitter{}
	mw := NewMiddleware(pstore, cstore, audit)
	return mw, pstore, cstore
}

// ---------------------------------------------------------------------------
// Integration test
// ---------------------------------------------------------------------------

func TestBottomUpAdoptionMotion_EndToEnd(t *testing.T) {
	now := time.Now().UTC()
	pharm := uuid.New()
	employer := uuid.New()
	ctx := context.Background()

	// -----------------------------------------------------------------------
	// Subtest 1: Pharmacist self-view succeeds without consent
	// -----------------------------------------------------------------------
	t.Run("pharmacist self-view succeeds without consent", func(t *testing.T) {
		src := fakePharmacistSource{}
		view := views.NewPharmacistView(src)

		traj, err := view.OwnRIRTrajectory(ctx, pharm, 28)
		if err != nil {
			t.Fatalf("expected success; got error: %v", err)
		}
		if traj.AuthorID != pharm {
			t.Errorf("expected trajectory AuthorID %v; got %v", pharm, traj.AuthorID)
		}
		if len(traj.Points) == 0 {
			t.Error("expected non-empty trajectory points")
		}
	})

	// -----------------------------------------------------------------------
	// Subtest 2: Employer denied without consent (empty stores)
	// -----------------------------------------------------------------------
	t.Run("employer denied with empty stores", func(t *testing.T) {
		mw, _, _ := setupMiddleware(t)

		called := false
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		handler := mw.Wrap("rir_class_specific", PFA, inner)

		req := httptest.NewRequest("GET", "/foo?subject_id="+pharm.String(), nil)
		req = req.WithContext(WithViewerRole(req.Context(), employer))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("empty stores: expected 403; got %d", w.Code)
		}
		if called {
			t.Error("inner handler must not be called when stores are empty")
		}
	})

	// -----------------------------------------------------------------------
	// Subtest 3: Employer denied with ViewPermission but missing consent
	// -----------------------------------------------------------------------
	t.Run("employer denied with permission but missing consent", func(t *testing.T) {
		mw, pstore, _ := setupMiddleware(t)

		gate := &AggregationGate{
			MinObservations:  10,
			TimeWindow:       90 * 24 * time.Hour,
			DelayWindow:      7 * 24 * time.Hour,
			ContractualBasis: "employer_contract_v1",
			ExplicitNotice:   true,
		}

		perm := ViewPermission{
			ID:           uuid.New(),
			SubjectID:    pharm,
			ViewerRoleID: employer,
			Scope: Scope{
				ViewType:      ViewTypeEmployer,
				Class:         PFA,
				ResourceTypes: []string{"rir_class_specific"},
				Gate:          gate,
			},
			GrantedAt:   now.Add(-48 * time.Hour),
			GrantedByID: pharm,
		}
		if _, err := pstore.Create(ctx, perm); err != nil {
			t.Fatalf("Create ViewPermission: %v", err)
		}

		// DataConsentStore is still empty — two-store check must fail.
		called := false
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		handler := mw.Wrap("rir_class_specific", PFA, inner)

		req := httptest.NewRequest("GET", "/foo?subject_id="+pharm.String(), nil)
		req = req.WithContext(WithViewerRole(req.Context(), employer))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("perm only, no consent: expected 403; got %d", w.Code)
		}
		if called {
			t.Error("inner handler must not be called when consent is absent")
		}
	})

	// -----------------------------------------------------------------------
	// Subtest 4: Employer allowed when both stores grant access
	// -----------------------------------------------------------------------
	t.Run("employer allowed when both stores grant access", func(t *testing.T) {
		mw, pstore, cstore := setupMiddleware(t)

		gate := &AggregationGate{
			MinObservations:  10,
			TimeWindow:       90 * 24 * time.Hour,
			DelayWindow:      7 * 24 * time.Hour,
			ContractualBasis: "employer_contract_v1",
			ExplicitNotice:   true,
		}

		perm := ViewPermission{
			ID:           uuid.New(),
			SubjectID:    pharm,
			ViewerRoleID: employer,
			Scope: Scope{
				ViewType:      ViewTypeEmployer,
				Class:         PFA,
				ResourceTypes: []string{"rir_class_specific"},
				Gate:          gate,
			},
			GrantedAt:   now.Add(-48 * time.Hour),
			GrantedByID: pharm,
		}
		if _, err := pstore.Create(ctx, perm); err != nil {
			t.Fatalf("Create ViewPermission: %v", err)
		}

		// DataAggregationConsent: granted 30 days ago, expires in 335 days.
		consent := DataAggregationConsent{
			ID:                uuid.New(),
			PharmacistID:      pharm,
			DataElement:       "rir_class_specific",
			AggregationTarget: "", // any target
			Purpose:           PurposeWorkforcePlanning,
			GrantedAt:         now.Add(-30 * 24 * time.Hour),
			ExpiresAt:         now.Add(335 * 24 * time.Hour),
		}
		if _, err := cstore.CreateConsent(ctx, consent); err != nil {
			t.Fatalf("CreateConsent: %v", err)
		}

		called := false
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		handler := mw.Wrap("rir_class_specific", PFA, inner)

		req := httptest.NewRequest("GET", "/foo?subject_id="+pharm.String(), nil)
		req = req.WithContext(WithViewerRole(req.Context(), employer))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("perm + consent: expected 200; got %d", w.Code)
		}
		if !called {
			t.Error("inner handler should have been called when both stores grant access")
		}
	})

	// -----------------------------------------------------------------------
	// Subtest 5: Pharmacist files contestation; record visible to both parties
	// -----------------------------------------------------------------------
	t.Run("contestation visible to both pharmacist and employer", func(t *testing.T) {
		cstore := &contestation.InMemoryStore{}

		c := contestation.Contestation{
			ID:                 uuid.New(),
			PharmacistID:       pharm,
			EmployerID:         employer,
			KPIType:            "rir_28d",
			KPISnapshot:        map[string]any{"rir": 0.42, "n": 35},
			PharmacistArgument: "The 28-day window included a facility-wide disruption period; the KPI does not reflect typical practice.",
			Status:             contestation.StatusOpen,
			FiledAt:            now,
		}
		if _, err := cstore.Create(ctx, c); err != nil {
			t.Fatalf("Create contestation: %v", err)
		}

		// Pharmacist view: ListByPharmacist.
		byPharmacist, err := cstore.ListByPharmacist(ctx, pharm)
		if err != nil {
			t.Fatalf("ListByPharmacist: %v", err)
		}
		if len(byPharmacist) != 1 {
			t.Fatalf("expected 1 contestation for pharmacist; got %d", len(byPharmacist))
		}

		// Employer view: ListByKPIType (employer filters by KPI type for open contestations).
		byKPI, err := cstore.ListByKPIType(ctx, "rir_28d", contestation.StatusOpen)
		if err != nil {
			t.Fatalf("ListByKPIType: %v", err)
		}
		if len(byKPI) != 1 {
			t.Fatalf("expected 1 contestation by KPI type; got %d", len(byKPI))
		}

		// Both queries must return the same record.
		if byPharmacist[0].ID != byKPI[0].ID {
			t.Errorf("pharmacist-view ID %v != employer-view ID %v — should be the same record",
				byPharmacist[0].ID, byKPI[0].ID)
		}
		if byPharmacist[0].ID != c.ID {
			t.Errorf("returned ID %v does not match filed contestation ID %v",
				byPharmacist[0].ID, c.ID)
		}
		if byPharmacist[0].Status != contestation.StatusOpen {
			t.Errorf("expected status %q; got %q", contestation.StatusOpen, byPharmacist[0].Status)
		}

		// Verify snapshot round-trips.
		snap := byPharmacist[0].KPISnapshot
		if snap["rir"] != 0.42 {
			t.Errorf("expected KPISnapshot[rir]=0.42; got %v", snap["rir"])
		}
	})
}
