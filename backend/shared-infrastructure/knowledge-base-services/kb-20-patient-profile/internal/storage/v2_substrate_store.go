// Package storage provides kb-20-patient-profile's persistence layer,
// including the v2 substrate canonical-row implementations.
//
// V2SubstrateStore is implemented with raw database/sql + lib/pq rather
// than the kb-20 GORM stack, because v2 reads happen through the
// residents_v2 SQL view (rather than a GORM-mapped table) and the
// upsert into the legacy patient_profiles table needs control over
// per-column COALESCE/default semantics that GORM does not express
// cleanly. Other kb-20 storage packages continue to use GORM.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/cardiofit/shared/v2_substrate/delta"
	"github.com/cardiofit/shared/v2_substrate/evidence_trace"
	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
)

// V2SubstrateStore implements ResidentStore + PersonStore + RoleStore for
// kb-20-patient-profile. Reads go through the residents_v2 compatibility
// view; writes touch the underlying patient_profiles + persons + roles
// tables created by migration 008_part1.
type V2SubstrateStore struct {
	db               *sql.DB
	baselineProvider delta.BaselineProvider // injected via SetBaselineProvider; nil → all writes get Delta.Flag=no_baseline
	baselineStore    *BaselineStore         // optional: when non-nil, UpsertObservation recomputes baseline_state inside the same tx
	baselineLookback int                    // 0 → fall back to delta.DefaultBaselineLookbackDays
}

// NewV2SubstrateStore opens a Postgres connection at dsn and returns a
// ready-to-use store. The caller owns the connection lifecycle and should
// call Close() when done.
func NewV2SubstrateStore(dsn string) (*V2SubstrateStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &V2SubstrateStore{db: db}, nil
}

// NewV2SubstrateStoreWithDB constructs a store using an externally
// managed *sql.DB (useful for sharing a pool with other components).
func NewV2SubstrateStoreWithDB(db *sql.DB) *V2SubstrateStore {
	return &V2SubstrateStore{db: db}
}

// Close releases the underlying database connection.
func (s *V2SubstrateStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// rowScanner abstracts *sql.Row and *sql.Rows so a single scan helper can
// service both single-row Get* methods and per-row List* loops.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

// ============================================================================
// Resident
// ============================================================================

// residentColumns is the canonical column list selected by GetResident and
// the List* methods that materialise full Resident structs.
const residentColumns = `id, ihi, given_name, family_name, dob, sex, indigenous_status,
       facility_id, admission_date, care_intensity, sdms, status,
       created_at, updated_at`

// scanResident reads one row's columns (in residentColumns order) into a
// fully-populated Resident.
func scanResident(sc rowScanner) (models.Resident, error) {
	var (
		r           models.Resident
		ihi         sql.NullString
		givenName   sql.NullString
		familyName  sql.NullString
		dob         sql.NullTime
		sex         sql.NullString
		indigStatus sql.NullString
		facilityID  uuid.NullUUID
		admDate     sql.NullTime
		careInt     sql.NullString
		sdms        pq.StringArray
		status      sql.NullString
	)

	if err := sc.Scan(
		&r.ID, &ihi, &givenName, &familyName, &dob, &sex,
		&indigStatus, &facilityID, &admDate, &careInt,
		&sdms, &status, &r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		return models.Resident{}, err
	}

	if ihi.Valid {
		r.IHI = ihi.String
	}
	if givenName.Valid {
		r.GivenName = givenName.String
	}
	if familyName.Valid {
		r.FamilyName = familyName.String
	}
	if dob.Valid {
		r.DOB = dob.Time
	}
	if sex.Valid {
		r.Sex = sexFromLegacy(sex.String)
	}
	if indigStatus.Valid {
		r.IndigenousStatus = indigStatus.String
	}
	if facilityID.Valid {
		r.FacilityID = facilityID.UUID
	}
	if admDate.Valid {
		t := admDate.Time
		r.AdmissionDate = &t
	}
	if careInt.Valid {
		r.CareIntensity = careInt.String
	}
	if status.Valid {
		r.Status = status.String
	}
	if len(sdms) > 0 {
		ids := make([]uuid.UUID, 0, len(sdms))
		for _, s := range sdms {
			if u, err := uuid.Parse(s); err == nil {
				ids = append(ids, u)
			}
		}
		r.SDMs = ids
	}

	return r, nil
}

// GetResident reads a single Resident through the residents_v2 view.
func (s *V2SubstrateStore) GetResident(ctx context.Context, id uuid.UUID) (*models.Resident, error) {
	q := `SELECT ` + residentColumns + ` FROM residents_v2 WHERE id = $1`

	r, err := scanResident(s.db.QueryRowContext(ctx, q, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get resident %s: %w", id, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get resident %s: %w", id, err)
	}
	return &r, nil
}

// UpsertResident writes a Resident into the underlying patient_profiles
// row. It fills both the v2 substrate columns (given_name, family_name,
// dob, lifecycle_status, ihi, care_intensity, sdms, facility_id,
// indigenous_status, admission_date) and the legacy NOT NULL columns
// (patient_id, age, sex, dm_type) with sensible defaults derived from
// the Resident payload so existing constraints continue to hold.
//
// Sex round-trip is lossy: Resident uses FHIR AdministrativeGender codes
// (male|female|other|unknown), patient_profiles.sex is constrained to
// (M|F|OTHER). The mapper preserves semantics across the boundary.
func (s *V2SubstrateStore) UpsertResident(ctx context.Context, r models.Resident) (*models.Resident, error) {
	const q = `
		INSERT INTO patient_profiles
			(id, patient_id, age, sex, dm_type, ihi, given_name, family_name, dob,
			 indigenous_status, facility_id, admission_date, care_intensity, sdms,
			 lifecycle_status, active, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, COALESCE($5,'NONE'), $6, $7, $8, $9,
			 $10, $11, $12, $13, $14,
			 $15, $16, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			ihi               = EXCLUDED.ihi,
			given_name        = EXCLUDED.given_name,
			family_name       = EXCLUDED.family_name,
			dob               = EXCLUDED.dob,
			sex               = EXCLUDED.sex,
			age               = EXCLUDED.age,
			indigenous_status = EXCLUDED.indigenous_status,
			facility_id       = EXCLUDED.facility_id,
			admission_date    = EXCLUDED.admission_date,
			care_intensity    = EXCLUDED.care_intensity,
			sdms              = EXCLUDED.sdms,
			lifecycle_status  = EXCLUDED.lifecycle_status,
			active            = EXCLUDED.active,
			updated_at        = NOW()
	`

	var sdmsArg interface{}
	if len(r.SDMs) > 0 {
		ids := make([]string, len(r.SDMs))
		for i, u := range r.SDMs {
			ids[i] = u.String()
		}
		sdmsArg = pq.Array(ids)
	}

	patientID := r.ID.String() // legacy NOT NULL key — use UUID string when no ABHA
	age := computeAge(r.DOB)
	legacySex := sexToLegacy(r.Sex)
	active := r.Status == "" || r.Status == models.ResidentStatusActive

	if _, err := s.db.ExecContext(ctx, q,
		r.ID,                           // $1
		patientID,                      // $2
		age,                            // $3
		legacySex,                      // $4
		// $5 dm_type: legacy nullable column with no v2 Resident analog;
		// we pass nil and rely on COALESCE($5,'NONE') above to satisfy the
		// CHECK constraint without inventing a fake disease classification.
		nil,
		nilIfEmpty(r.IHI),              // $6
		nilIfEmpty(r.GivenName),        // $7
		nilIfEmpty(r.FamilyName),       // $8
		r.DOB,                          // $9
		nilIfEmpty(r.IndigenousStatus), // $10
		r.FacilityID,                   // $11
		r.AdmissionDate,                // $12
		nilIfEmpty(r.CareIntensity),    // $13
		sdmsArg,                        // $14
		nilIfEmpty(r.Status),           // $15
		active,                         // $16
	); err != nil {
		return nil, fmt.Errorf("upsert resident: %w", err)
	}

	return s.GetResident(ctx, r.ID)
}

// ListResidentsByFacility returns residents at the given facility, paged.
//
// One round-trip: SELECTs the full row set in a single query and scans
// each row in-process via scanResident. (No N+1 GetResident loop.)
func (s *V2SubstrateStore) ListResidentsByFacility(ctx context.Context, facilityID uuid.UUID, limit, offset int) ([]models.Resident, error) {
	q := `SELECT ` + residentColumns + `
		  FROM residents_v2
		 WHERE facility_id = $1
		 ORDER BY family_name NULLS LAST, given_name NULLS LAST
		 LIMIT $2 OFFSET $3`

	rows, err := s.db.QueryContext(ctx, q, facilityID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var residents []models.Resident
	for rows.Next() {
		r, err := scanResident(rows)
		if err != nil {
			return nil, err
		}
		residents = append(residents, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return residents, nil
}

// ============================================================================
// Person
// ============================================================================

// GetPerson reads a Person row by primary key.
func (s *V2SubstrateStore) GetPerson(ctx context.Context, id uuid.UUID) (*models.Person, error) {
	const q = `
		SELECT id, given_name, family_name, hpii, ahpra_registration, contact_details
		  FROM persons
		 WHERE id = $1`

	var (
		p       models.Person
		hpii    sql.NullString
		ahpra   sql.NullString
		contact []byte
	)
	if err := s.db.QueryRowContext(ctx, q, id).Scan(
		&p.ID, &p.GivenName, &p.FamilyName, &hpii, &ahpra, &contact,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get person %s: %w", id, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get person %s: %w", id, err)
	}
	if hpii.Valid {
		p.HPII = hpii.String
	}
	if ahpra.Valid {
		p.AHPRARegistration = ahpra.String
	}
	if len(contact) > 0 {
		p.ContactDetails = json.RawMessage(contact)
	}
	return &p, nil
}

// UpsertPerson writes a Person row.
func (s *V2SubstrateStore) UpsertPerson(ctx context.Context, p models.Person) (*models.Person, error) {
	const q = `
		INSERT INTO persons (id, given_name, family_name, hpii, ahpra_registration, contact_details, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			given_name         = EXCLUDED.given_name,
			family_name        = EXCLUDED.family_name,
			hpii               = EXCLUDED.hpii,
			ahpra_registration = EXCLUDED.ahpra_registration,
			contact_details    = EXCLUDED.contact_details,
			updated_at         = NOW()
	`

	var contactArg interface{}
	if len(p.ContactDetails) > 0 {
		contactArg = []byte(p.ContactDetails)
	}

	if _, err := s.db.ExecContext(ctx, q,
		p.ID, p.GivenName, p.FamilyName,
		nilIfEmpty(p.HPII), nilIfEmpty(p.AHPRARegistration), contactArg,
	); err != nil {
		return nil, fmt.Errorf("upsert person: %w", err)
	}
	return s.GetPerson(ctx, p.ID)
}

// GetPersonByHPII looks up a Person by HPII.
func (s *V2SubstrateStore) GetPersonByHPII(ctx context.Context, hpii string) (*models.Person, error) {
	const q = `SELECT id FROM persons WHERE hpii = $1`
	var id uuid.UUID
	if err := s.db.QueryRowContext(ctx, q, hpii).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get person by hpii %s: %w", hpii, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get person by hpii %s: %w", hpii, err)
	}
	return s.GetPerson(ctx, id)
}

// ============================================================================
// Role
// ============================================================================

// roleColumns is the canonical column list selected by GetRole and the
// List* methods that materialise full Role structs.
const roleColumns = `id, person_id, kind, qualifications, facility_id, valid_from, valid_to, evidence_url`

// scanRole reads one row's columns (in roleColumns order) into a fully-
// populated Role.
func scanRole(sc rowScanner) (models.Role, error) {
	var (
		r           models.Role
		quals       []byte
		facID       uuid.NullUUID
		validTo     sql.NullTime
		evidenceURL sql.NullString
	)
	if err := sc.Scan(
		&r.ID, &r.PersonID, &r.Kind, &quals, &facID, &r.ValidFrom, &validTo, &evidenceURL,
	); err != nil {
		return models.Role{}, err
	}
	if len(quals) > 0 {
		r.Qualifications = json.RawMessage(quals)
	}
	if facID.Valid {
		f := facID.UUID
		r.FacilityID = &f
	}
	if validTo.Valid {
		t := validTo.Time
		r.ValidTo = &t
	}
	if evidenceURL.Valid {
		r.EvidenceURL = evidenceURL.String
	}
	return r, nil
}

// GetRole reads a Role row.
func (s *V2SubstrateStore) GetRole(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	q := `SELECT ` + roleColumns + ` FROM roles WHERE id = $1`

	r, err := scanRole(s.db.QueryRowContext(ctx, q, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get role %s: %w", id, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get role %s: %w", id, err)
	}
	return &r, nil
}

// UpsertRole writes a Role row.
func (s *V2SubstrateStore) UpsertRole(ctx context.Context, r models.Role) (*models.Role, error) {
	const q = `
		INSERT INTO roles (id, person_id, kind, qualifications, facility_id, valid_from, valid_to, evidence_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			kind           = EXCLUDED.kind,
			qualifications = EXCLUDED.qualifications,
			facility_id    = EXCLUDED.facility_id,
			valid_from     = EXCLUDED.valid_from,
			valid_to       = EXCLUDED.valid_to,
			evidence_url   = EXCLUDED.evidence_url,
			updated_at     = NOW()
	`

	var qualsArg interface{}
	if len(r.Qualifications) > 0 {
		qualsArg = []byte(r.Qualifications)
	}
	var facIDArg interface{}
	if r.FacilityID != nil {
		facIDArg = *r.FacilityID
	}
	validFrom := r.ValidFrom
	if validFrom.IsZero() {
		validFrom = time.Now().UTC()
	}

	if _, err := s.db.ExecContext(ctx, q,
		r.ID, r.PersonID, r.Kind, qualsArg, facIDArg, validFrom, r.ValidTo, nilIfEmpty(r.EvidenceURL),
	); err != nil {
		return nil, fmt.Errorf("upsert role: %w", err)
	}
	return s.GetRole(ctx, r.ID)
}

// ListRolesByPerson returns all roles for a Person, newest validity first.
//
// One round-trip: full rows in a single SELECT, scanned in-process.
func (s *V2SubstrateStore) ListRolesByPerson(ctx context.Context, personID uuid.UUID) ([]models.Role, error) {
	q := `SELECT ` + roleColumns + ` FROM roles WHERE person_id = $1 ORDER BY valid_from DESC`
	rows, err := s.db.QueryContext(ctx, q, personID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []models.Role
	for rows.Next() {
		r, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return roles, nil
}

// ListActiveRolesByPersonAndFacility returns currently-active roles for a
// Person scoped to a facility (or portable / facility-agnostic roles).
//
// One round-trip: full rows in a single SELECT, scanned in-process.
func (s *V2SubstrateStore) ListActiveRolesByPersonAndFacility(ctx context.Context, personID, facilityID uuid.UUID) ([]models.Role, error) {
	q := `SELECT ` + roleColumns + ` FROM roles
		 WHERE person_id = $1
		   AND (facility_id IS NULL OR facility_id = $2)
		   AND valid_from <= NOW()
		   AND (valid_to IS NULL OR valid_to >= NOW())
		 ORDER BY valid_from DESC`

	rows, err := s.db.QueryContext(ctx, q, personID, facilityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []models.Role
	for rows.Next() {
		r, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return roles, nil
}

// ============================================================================
// Helpers
// ============================================================================

// nilIfEmpty returns nil for empty strings so that DB nullable columns receive
// SQL NULL rather than ''.
func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// computeAge derives age in years from DOB. Returns 0 for the zero time.
func computeAge(dob time.Time) int {
	if dob.IsZero() {
		return 0
	}
	now := time.Now().UTC()
	years := now.Year() - dob.Year()
	// adjust if birthday hasn't occurred yet this year
	if now.YearDay() < dob.YearDay() {
		years--
	}
	if years < 0 {
		return 0
	}
	return years
}

// sexToLegacy maps the FHIR AdministrativeGender code stored in
// models.Resident.Sex onto the legacy patient_profiles.sex CHECK
// constraint domain {M, F, OTHER}.
func sexToLegacy(s string) string {
	switch s {
	case "male", "M":
		return "M"
	case "female", "F":
		return "F"
	default:
		return "OTHER"
	}
}

// sexFromLegacy is the inverse of sexToLegacy. Note that the round-trip is
// lossy for "unknown" — it surfaces as "other" when read back.
func sexFromLegacy(s string) string {
	switch s {
	case "M":
		return "male"
	case "F":
		return "female"
	default:
		return "other"
	}
}

// ============================================================================
// MedicineUse
// ============================================================================

// medicineUseColumns is the canonical column list selected by GetMedicineUse
// and ListMedicineUsesByResident. Mirrors the projection of medicine_uses_v2.
const medicineUseColumns = `id, resident_id, amt_code, display_name, intent, target, stop_criteria,
       dose, route, frequency, prescriber_id, started_at, ended_at, status,
       created_at, updated_at`

// scanMedicineUse reads one row's columns (in medicineUseColumns order) into
// a fully-populated MedicineUse. Handles nullable amt_code/dose/route/
// frequency, nullable ended_at, optional prescriber_id, and JSONB
// intent/target/stop_criteria payloads.
func scanMedicineUse(sc rowScanner) (models.MedicineUse, error) {
	var (
		m            models.MedicineUse
		amtCode      sql.NullString
		dose         sql.NullString
		route        sql.NullString
		frequency    sql.NullString
		endedAt      sql.NullTime
		prescriberID uuid.NullUUID
		intentBytes  []byte
		targetBytes  []byte
		stopBytes    []byte
	)
	if err := sc.Scan(
		&m.ID, &m.ResidentID, &amtCode, &m.DisplayName,
		&intentBytes, &targetBytes, &stopBytes,
		&dose, &route, &frequency,
		&prescriberID, &m.StartedAt, &endedAt, &m.Status,
		&m.CreatedAt, &m.UpdatedAt,
	); err != nil {
		return models.MedicineUse{}, err
	}
	if amtCode.Valid {
		m.AMTCode = amtCode.String
	}
	if dose.Valid {
		m.Dose = dose.String
	}
	if route.Valid {
		m.Route = route.String
	}
	if frequency.Valid {
		m.Frequency = frequency.String
	}
	if endedAt.Valid {
		t := endedAt.Time
		m.EndedAt = &t
	}
	if prescriberID.Valid {
		p := prescriberID.UUID
		m.PrescriberID = &p
	}
	if len(intentBytes) > 0 {
		if err := json.Unmarshal(intentBytes, &m.Intent); err != nil {
			return models.MedicineUse{}, fmt.Errorf("unmarshal intent: %w", err)
		}
	}
	if len(targetBytes) > 0 {
		if err := json.Unmarshal(targetBytes, &m.Target); err != nil {
			return models.MedicineUse{}, fmt.Errorf("unmarshal target: %w", err)
		}
	}
	if len(stopBytes) > 0 {
		if err := json.Unmarshal(stopBytes, &m.StopCriteria); err != nil {
			return models.MedicineUse{}, fmt.Errorf("unmarshal stop_criteria: %w", err)
		}
	}
	return m, nil
}

// GetMedicineUse reads a single MedicineUse through the medicine_uses_v2 view.
func (s *V2SubstrateStore) GetMedicineUse(ctx context.Context, id uuid.UUID) (*models.MedicineUse, error) {
	q := `SELECT ` + medicineUseColumns + ` FROM medicine_uses_v2 WHERE id = $1`

	m, err := scanMedicineUse(s.db.QueryRowContext(ctx, q, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get medicine_use %s: %w", id, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get medicine_use %s: %w", id, err)
	}
	return &m, nil
}

// UpsertMedicineUse writes a MedicineUse into the underlying medication_states
// row. It populates BOTH v2 columns (resident_id, amt_code, display_name,
// intent JSONB, target JSONB, stop_criteria JSONB, prescriber_id,
// lifecycle_status) and the legacy NOT NULL columns (patient_id, drug_name,
// drug_class, is_active) so the constraints from migration 001 continue to
// hold for v2 writers.
//
// drug_class is set to a sentinel "UNKNOWN" because v2 MedicineUse does not
// carry class-level information; legacy class lookups against v2-written rows
// will surface as "UNKNOWN" by design.
func (s *V2SubstrateStore) UpsertMedicineUse(ctx context.Context, m models.MedicineUse) (*models.MedicineUse, error) {
	const q = `
		INSERT INTO medication_states
			(id, patient_id, drug_name, drug_class, route, frequency,
			 is_active, start_date, end_date,
			 amt_code, display_name, intent, target, stop_criteria,
			 prescriber_id, resident_id, lifecycle_status,
			 created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6,
			 $7, $8, $9,
			 $10, $11, $12, $13, $14,
			 $15, $16, $17,
			 NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			drug_name        = EXCLUDED.drug_name,
			drug_class       = EXCLUDED.drug_class,
			route            = EXCLUDED.route,
			frequency        = EXCLUDED.frequency,
			is_active        = EXCLUDED.is_active,
			start_date       = EXCLUDED.start_date,
			end_date         = EXCLUDED.end_date,
			amt_code         = EXCLUDED.amt_code,
			display_name     = EXCLUDED.display_name,
			intent           = EXCLUDED.intent,
			target           = EXCLUDED.target,
			stop_criteria    = EXCLUDED.stop_criteria,
			prescriber_id    = EXCLUDED.prescriber_id,
			resident_id      = EXCLUDED.resident_id,
			lifecycle_status = EXCLUDED.lifecycle_status,
			updated_at       = NOW()
	`

	intentJSON, err := json.Marshal(m.Intent)
	if err != nil {
		return nil, fmt.Errorf("marshal intent: %w", err)
	}
	targetJSON, err := json.Marshal(m.Target)
	if err != nil {
		return nil, fmt.Errorf("marshal target: %w", err)
	}
	stopJSON, err := json.Marshal(m.StopCriteria)
	if err != nil {
		return nil, fmt.Errorf("marshal stop_criteria: %w", err)
	}

	// Legacy patient_id is VARCHAR(100); v2 writers always have a resident_id,
	// so we surface its UUID string in the legacy column to keep the NOT NULL
	// constraint satisfied without minting a parallel identity.
	patientID := m.ResidentID.String()
	// Legacy drug_name NOT NULL — fall back to v2 display_name.
	drugName := m.DisplayName
	// Legacy drug_class NOT NULL — v2 carries no class info; sentinel.
	const drugClass = "UNKNOWN"
	isActive := m.Status == models.MedicineUseStatusActive

	var prescriberArg interface{}
	if m.PrescriberID != nil {
		prescriberArg = *m.PrescriberID
	}

	if _, err := s.db.ExecContext(ctx, q,
		m.ID,                         // $1
		patientID,                    // $2
		drugName,                     // $3
		drugClass,                    // $4
		nilIfEmpty(m.Route),          // $5
		nilIfEmpty(m.Frequency),      // $6
		isActive,                     // $7
		m.StartedAt,                  // $8
		m.EndedAt,                    // $9
		nilIfEmpty(m.AMTCode),        // $10
		nilIfEmpty(m.DisplayName),    // $11
		intentJSON,                   // $12
		targetJSON,                   // $13
		stopJSON,                     // $14
		prescriberArg,                // $15
		m.ResidentID,                 // $16
		nilIfEmpty(m.Status),         // $17
	); err != nil {
		return nil, fmt.Errorf("upsert medicine_use: %w", err)
	}

	return s.GetMedicineUse(ctx, m.ID)
}

// ListMedicineUsesByResident returns medicine uses for a Resident, paged.
//
// One round-trip: SELECTs the full row set in a single query and scans each
// row in-process via scanMedicineUse. (No N+1 GetMedicineUse loop.)
func (s *V2SubstrateStore) ListMedicineUsesByResident(ctx context.Context, residentID uuid.UUID, limit, offset int) ([]models.MedicineUse, error) {
	q := `SELECT ` + medicineUseColumns + `
		  FROM medicine_uses_v2
		 WHERE resident_id = $1
		 ORDER BY started_at DESC
		 LIMIT $2 OFFSET $3`

	rows, err := s.db.QueryContext(ctx, q, residentID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var uses []models.MedicineUse
	for rows.Next() {
		m, err := scanMedicineUse(rows)
		if err != nil {
			return nil, err
		}
		uses = append(uses, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return uses, nil
}

// ============================================================================
// Observation
// ============================================================================

// SetBaselineProvider injects the delta.BaselineProvider used by
// UpsertObservation. Must be called before UpsertObservation; if unset,
// UpsertObservation falls back to Delta.Flag=no_baseline for every write.
func (s *V2SubstrateStore) SetBaselineProvider(bp delta.BaselineProvider) {
	s.baselineProvider = bp
}

// SetBaselineStore wires the persistent BaselineStore into the observation
// write path. When set, UpsertObservation runs the observation INSERT and
// the baseline_state recompute inside a single Postgres transaction so the
// running baseline is always consistent with persisted observations.
//
// Leave unset (nil) to preserve the legacy behaviour of writing the
// observation row only — the running baseline is then expected to come
// from a different source (e.g. the in-memory provider used by unit tests).
func (s *V2SubstrateStore) SetBaselineStore(bs *BaselineStore) {
	s.baselineStore = bs
}

// SetBaselineLookbackDays overrides the default 14-day lookback window used
// when recomputing baselines on observation insert. Pass 0 to restore the
// default.
func (s *V2SubstrateStore) SetBaselineLookbackDays(days int) {
	s.baselineLookback = days
}

// observationColumns matches the projection of observations_v2 (which UNIONs
// observations + lab_entries with kind='lab').
const observationColumns = `id, resident_id, loinc_code, snomed_code, kind,
       value, value_text, unit, observed_at, source_id, delta, created_at`

// scanObservation reads one row's columns (in observationColumns order) into
// a fully-populated Observation. Handles nullable LOINC/SNOMED, pointer-nullable
// Value, optional ValueText/Unit, optional SourceID, and JSONB Delta payload.
func scanObservation(sc rowScanner) (models.Observation, error) {
	var (
		o          models.Observation
		loinc      sql.NullString
		snomed     sql.NullString
		value      sql.NullFloat64
		valueText  sql.NullString
		unit       sql.NullString
		sourceID   uuid.NullUUID
		deltaBytes []byte
	)
	if err := sc.Scan(
		&o.ID, &o.ResidentID, &loinc, &snomed, &o.Kind,
		&value, &valueText, &unit, &o.ObservedAt,
		&sourceID, &deltaBytes, &o.CreatedAt,
	); err != nil {
		return models.Observation{}, err
	}
	if loinc.Valid {
		o.LOINCCode = loinc.String
	}
	if snomed.Valid {
		o.SNOMEDCode = snomed.String
	}
	if value.Valid {
		v := value.Float64
		o.Value = &v
	}
	if valueText.Valid {
		o.ValueText = valueText.String
	}
	if unit.Valid {
		o.Unit = unit.String
	}
	if sourceID.Valid {
		sid := sourceID.UUID
		o.SourceID = &sid
	}
	if len(deltaBytes) > 0 {
		var d models.Delta
		if err := json.Unmarshal(deltaBytes, &d); err != nil {
			return models.Observation{}, fmt.Errorf("unmarshal delta: %w", err)
		}
		o.Delta = &d
	}
	return o, nil
}

// GetObservation reads a single Observation through the observations_v2 view.
func (s *V2SubstrateStore) GetObservation(ctx context.Context, id uuid.UUID) (*models.Observation, error) {
	q := `SELECT ` + observationColumns + ` FROM observations_v2 WHERE id = $1`
	o, err := scanObservation(s.db.QueryRowContext(ctx, q, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get observation %s: %w", id, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get observation %s: %w", id, err)
	}
	return &o, nil
}

// vitalTypeKey resolves an Observation to the BaselineProvider vital-type key.
// Priority: LOINC code, then SNOMED code, then a fallback derived from Kind.
// kb-26's AcuteRepository keys on LOINC for vitals/labs and on a model-internal
// string for weight/mobility — this resolver mirrors that precedence.
func vitalTypeKey(o models.Observation) string {
	if o.LOINCCode != "" {
		return o.LOINCCode
	}
	if o.SNOMEDCode != "" {
		return o.SNOMEDCode
	}
	return o.Kind
}

// UpsertObservation writes an Observation row, computing Delta first via the
// injected delta.BaselineProvider. If the provider is unset OR returns
// delta.ErrNoBaseline OR returns any other error, the resulting Delta has
// DirectionalFlag = no_baseline and the row still persists (writes are NOT
// blocked by baseline unavailability).
//
// Writes go to the greenfield observations table. Reads come back through the
// observations_v2 view (which UNIONs lab_entries) — so v2 writers see their
// own writes via GetObservation immediately.
func (s *V2SubstrateStore) UpsertObservation(ctx context.Context, o models.Observation) (*models.Observation, error) {
	// Resolve baseline (best-effort; failures degrade to no_baseline).
	var baseline *delta.Baseline
	if s.baselineProvider != nil && o.Value != nil && o.Kind != models.ObservationKindBehavioural {
		bl, err := s.baselineProvider.FetchBaseline(ctx, o.ResidentID, vitalTypeKey(o))
		if err == nil {
			baseline = bl
		}
		// err (incl. ErrNoBaseline) → baseline stays nil → ComputeDelta yields no_baseline
	}
	d := delta.ComputeDelta(o, baseline)
	o.Delta = &d

	deltaJSON, err := json.Marshal(o.Delta)
	if err != nil {
		return nil, fmt.Errorf("marshal delta: %w", err)
	}

	const q = `
		INSERT INTO observations
			(id, resident_id, loinc_code, snomed_code, kind,
			 value, value_text, unit, observed_at, source_id, delta, created_at)
		VALUES
			($1, $2, $3, $4, $5,
			 $6, $7, $8, $9, $10, $11, NOW())
		ON CONFLICT (id) DO UPDATE SET
			resident_id = EXCLUDED.resident_id,
			loinc_code  = EXCLUDED.loinc_code,
			snomed_code = EXCLUDED.snomed_code,
			kind        = EXCLUDED.kind,
			value       = EXCLUDED.value,
			value_text  = EXCLUDED.value_text,
			unit        = EXCLUDED.unit,
			observed_at = EXCLUDED.observed_at,
			source_id   = EXCLUDED.source_id,
			delta       = EXCLUDED.delta
	`

	var valueArg interface{}
	if o.Value != nil {
		valueArg = *o.Value
	}
	var sourceArg interface{}
	if o.SourceID != nil {
		sourceArg = *o.SourceID
	}

	// Wrap observation INSERT + baseline recompute in a single transaction
	// when a BaselineStore is wired. This guarantees the running baseline
	// row in baseline_state is always consistent with the persisted
	// observation set: either both succeed and commit, or both roll back.
	// Without this, a partial state (observation written but baseline
	// stale, or vice versa) would be a correctness bug per the plan.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx for upsert observation: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, q,
		o.ID,                    // $1
		o.ResidentID,            // $2
		nilIfEmpty(o.LOINCCode), // $3
		nilIfEmpty(o.SNOMEDCode), // $4
		o.Kind,                  // $5
		valueArg,                // $6
		nilIfEmpty(o.ValueText), // $7
		nilIfEmpty(o.Unit),      // $8
		o.ObservedAt,            // $9
		sourceArg,               // $10
		deltaJSON,               // $11
	); err != nil {
		return nil, fmt.Errorf("upsert observation: %w", err)
	}

	// Baseline recompute side-effect. Skipped for non-numeric observations
	// (no Value, behavioural kind) — they don't contribute to a baseline,
	// and the recompute SQL would be a wasted round-trip. Also skipped
	// when no BaselineStore is wired (legacy/in-memory test wiring).
	if s.baselineStore != nil && o.Value != nil && o.Kind != models.ObservationKindBehavioural {
		lookback := s.baselineLookback
		if lookback <= 0 {
			lookback = delta.DefaultBaselineLookbackDays
		}
		if _, err := s.baselineStore.RecomputeAndUpsertTx(ctx, tx, o.ResidentID, vitalTypeKey(o), lookback); err != nil {
			// delta.ErrNoBaseline is NOT an error here — it just means
			// the running window doesn't have enough samples yet. The
			// baseline_state row was still upserted with confidence=
			// insufficient_data; commit the tx so that accounting row
			// is preserved.
			if !errors.Is(err, delta.ErrNoBaseline) {
				return nil, fmt.Errorf("recompute baseline: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit observation tx: %w", err)
	}
	committed = true

	return s.GetObservation(ctx, o.ID)
}

// ListObservationsByResident returns observations for a resident, paged.
// One round-trip; no N+1.
func (s *V2SubstrateStore) ListObservationsByResident(ctx context.Context, residentID uuid.UUID, limit, offset int) ([]models.Observation, error) {
	q := `SELECT ` + observationColumns + `
          FROM observations_v2
         WHERE resident_id = $1
         ORDER BY observed_at DESC
         LIMIT $2 OFFSET $3`
	rows, err := s.db.QueryContext(ctx, q, residentID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Observation
	for rows.Next() {
		o, err := scanObservation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ListObservationsByResidentAndKind filters ListObservationsByResident on kind.
func (s *V2SubstrateStore) ListObservationsByResidentAndKind(ctx context.Context, residentID uuid.UUID, kind string, limit, offset int) ([]models.Observation, error) {
	q := `SELECT ` + observationColumns + `
          FROM observations_v2
         WHERE resident_id = $1 AND kind = $2
         ORDER BY observed_at DESC
         LIMIT $3 OFFSET $4`
	rows, err := s.db.QueryContext(ctx, q, residentID, kind, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Observation
	for rows.Next() {
		o, err := scanObservation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ============================================================================
// Event
// ============================================================================

// eventColumns is the canonical column list selected by GetEvent and the
// List* methods that materialise full Event structs.
const eventColumns = `id, event_type, occurred_at, occurred_at_facility,
       resident_id, reported_by_ref, witnessed_by_refs, severity,
       description_structured, description_free_text,
       related_observations, related_medication_uses,
       triggered_state_changes, reportable_under,
       created_at, updated_at`

// scanEvent reads one row's columns (in eventColumns order) into a fully-
// populated Event. Handles nullable occurred_at_facility/severity/
// description_free_text/description_structured, UUID[] arrays for witness
// + related-entity refs, and JSONB triggered_state_changes.
func scanEvent(sc rowScanner) (models.Event, error) {
	var (
		e             models.Event
		facID         uuid.NullUUID
		severity      sql.NullString
		freeText      sql.NullString
		descStruct    []byte
		witnesses     pq.StringArray
		relObs        pq.StringArray
		relMed        pq.StringArray
		tscBytes      []byte
		reportable    pq.StringArray
	)
	if err := sc.Scan(
		&e.ID, &e.EventType, &e.OccurredAt, &facID,
		&e.ResidentID, &e.ReportedByRef, &witnesses, &severity,
		&descStruct, &freeText,
		&relObs, &relMed,
		&tscBytes, &reportable,
		&e.CreatedAt, &e.UpdatedAt,
	); err != nil {
		return models.Event{}, err
	}
	if facID.Valid {
		f := facID.UUID
		e.OccurredAtFacility = &f
	}
	if severity.Valid {
		e.Severity = severity.String
	}
	if freeText.Valid {
		e.DescriptionFreeText = freeText.String
	}
	if len(descStruct) > 0 {
		e.DescriptionStructured = json.RawMessage(descStruct)
	}
	if len(witnesses) > 0 {
		e.WitnessedByRefs = parseStringUUIDs(witnesses)
	}
	if len(relObs) > 0 {
		e.RelatedObservations = parseStringUUIDs(relObs)
	}
	if len(relMed) > 0 {
		e.RelatedMedicationUses = parseStringUUIDs(relMed)
	}
	if len(tscBytes) > 0 {
		var tscs []models.TriggeredStateChange
		if err := json.Unmarshal(tscBytes, &tscs); err != nil {
			return models.Event{}, fmt.Errorf("unmarshal triggered_state_changes: %w", err)
		}
		if len(tscs) > 0 {
			e.TriggeredStateChanges = tscs
		}
	}
	if len(reportable) > 0 {
		e.ReportableUnder = []string(reportable)
	}
	return e, nil
}

// parseStringUUIDs converts a pq.StringArray of UUID-formatted strings into
// a []uuid.UUID, dropping malformed entries (Postgres-side data integrity
// already enforces UUID typing on the columns we use this with).
func parseStringUUIDs(in pq.StringArray) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(in))
	for _, s := range in {
		if u, err := uuid.Parse(s); err == nil {
			out = append(out, u)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// uuidsToStrings converts []uuid.UUID to []string for pq.Array binding.
func uuidsToStrings(in []uuid.UUID) []string {
	out := make([]string, len(in))
	for i, u := range in {
		out[i] = u.String()
	}
	return out
}

// GetEvent reads a single Event by primary key.
func (s *V2SubstrateStore) GetEvent(ctx context.Context, id uuid.UUID) (*models.Event, error) {
	q := `SELECT ` + eventColumns + ` FROM events WHERE id = $1`
	e, err := scanEvent(s.db.QueryRowContext(ctx, q, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get event %s: %w", id, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get event %s: %w", id, err)
	}
	return &e, nil
}

// UpsertEvent inserts (or updates by id) one Event row. Marshals the JSONB
// columns and binds UUID[] arrays via pq.Array.
func (s *V2SubstrateStore) UpsertEvent(ctx context.Context, e models.Event) (*models.Event, error) {
	const q = `
		INSERT INTO events
			(id, event_type, occurred_at, occurred_at_facility,
			 resident_id, reported_by_ref, witnessed_by_refs, severity,
			 description_structured, description_free_text,
			 related_observations, related_medication_uses,
			 triggered_state_changes, reportable_under,
			 created_at, updated_at)
		VALUES
			($1, $2, $3, $4,
			 $5, $6, $7, $8,
			 $9, $10,
			 $11, $12,
			 $13, $14,
			 NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			event_type              = EXCLUDED.event_type,
			occurred_at             = EXCLUDED.occurred_at,
			occurred_at_facility    = EXCLUDED.occurred_at_facility,
			resident_id             = EXCLUDED.resident_id,
			reported_by_ref         = EXCLUDED.reported_by_ref,
			witnessed_by_refs       = EXCLUDED.witnessed_by_refs,
			severity                = EXCLUDED.severity,
			description_structured  = EXCLUDED.description_structured,
			description_free_text   = EXCLUDED.description_free_text,
			related_observations    = EXCLUDED.related_observations,
			related_medication_uses = EXCLUDED.related_medication_uses,
			triggered_state_changes = EXCLUDED.triggered_state_changes,
			reportable_under        = EXCLUDED.reportable_under,
			updated_at              = NOW()
	`

	var facArg interface{}
	if e.OccurredAtFacility != nil {
		facArg = *e.OccurredAtFacility
	}

	var descStructArg interface{}
	if len(e.DescriptionStructured) > 0 {
		descStructArg = []byte(e.DescriptionStructured)
	}

	tscJSON, err := json.Marshal(e.TriggeredStateChanges)
	if err != nil {
		return nil, fmt.Errorf("marshal triggered_state_changes: %w", err)
	}
	if len(e.TriggeredStateChanges) == 0 {
		// Persist '[]' (not 'null') to match the column DEFAULT and avoid
		// nullable-vs-empty drift on read.
		tscJSON = []byte("[]")
	}

	if _, err := s.db.ExecContext(ctx, q,
		e.ID,                                      // $1
		e.EventType,                               // $2
		e.OccurredAt,                              // $3
		facArg,                                    // $4
		e.ResidentID,                              // $5
		e.ReportedByRef,                           // $6
		pq.Array(uuidsToStrings(e.WitnessedByRefs)), // $7
		nilIfEmpty(e.Severity),                    // $8
		descStructArg,                             // $9
		nilIfEmpty(e.DescriptionFreeText),         // $10
		pq.Array(uuidsToStrings(e.RelatedObservations)),   // $11
		pq.Array(uuidsToStrings(e.RelatedMedicationUses)), // $12
		tscJSON,                                   // $13
		pq.Array(e.ReportableUnder),               // $14
	); err != nil {
		return nil, fmt.Errorf("upsert event: %w", err)
	}
	return s.GetEvent(ctx, e.ID)
}

// ListEventsByResident returns events for a resident, newest first.
func (s *V2SubstrateStore) ListEventsByResident(ctx context.Context, residentID uuid.UUID, limit, offset int) ([]models.Event, error) {
	q := `SELECT ` + eventColumns + `
		  FROM events
		 WHERE resident_id = $1
		 ORDER BY occurred_at DESC
		 LIMIT $2 OFFSET $3`
	rows, err := s.db.QueryContext(ctx, q, residentID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Event
	for rows.Next() {
		ev, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

// ListEventsByType returns events of a given event_type within an optional
// [from, to) date range. A zero `from` or `to` is treated as no bound.
func (s *V2SubstrateStore) ListEventsByType(ctx context.Context, eventType string, from, to time.Time, limit, offset int) ([]models.Event, error) {
	// Build the WHERE incrementally so that zero bounds drop cleanly.
	where := "event_type = $1"
	args := []interface{}{eventType}
	idx := 2
	if !from.IsZero() {
		where += fmt.Sprintf(" AND occurred_at >= $%d", idx)
		args = append(args, from)
		idx++
	}
	if !to.IsZero() {
		where += fmt.Sprintf(" AND occurred_at < $%d", idx)
		args = append(args, to)
		idx++
	}
	q := `SELECT ` + eventColumns + `
		  FROM events
		 WHERE ` + where + `
		 ORDER BY occurred_at DESC
		 LIMIT $` + fmt.Sprintf("%d", idx) + ` OFFSET $` + fmt.Sprintf("%d", idx+1)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Event
	for rows.Next() {
		ev, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

// ============================================================================
// EvidenceTrace
// ============================================================================

// evidenceTraceNodeColumns is the canonical column list for SELECT/scan.
const evidenceTraceNodeColumns = `id, state_machine, state_change_type,
       recorded_at, occurred_at,
       actor_role_ref, actor_person_ref, authority_basis_ref,
       inputs, reasoning_summary, outputs,
       resident_ref, created_at`

// scanEvidenceTraceNode reads one row's columns (in evidenceTraceNodeColumns
// order) into a fully-populated EvidenceTraceNode.
func scanEvidenceTraceNode(sc rowScanner) (models.EvidenceTraceNode, error) {
	var (
		n          models.EvidenceTraceNode
		roleRef    uuid.NullUUID
		personRef  uuid.NullUUID
		authRef    uuid.NullUUID
		residentR  uuid.NullUUID
		inputs     []byte
		reasoning  []byte
		outputs    []byte
	)
	if err := sc.Scan(
		&n.ID, &n.StateMachine, &n.StateChangeType,
		&n.RecordedAt, &n.OccurredAt,
		&roleRef, &personRef, &authRef,
		&inputs, &reasoning, &outputs,
		&residentR, &n.CreatedAt,
	); err != nil {
		return models.EvidenceTraceNode{}, err
	}
	if roleRef.Valid {
		u := roleRef.UUID
		n.Actor.RoleRef = &u
	}
	if personRef.Valid {
		u := personRef.UUID
		n.Actor.PersonRef = &u
	}
	if authRef.Valid {
		u := authRef.UUID
		n.Actor.AuthorityBasisRef = &u
	}
	if residentR.Valid {
		u := residentR.UUID
		n.ResidentRef = &u
	}
	if len(inputs) > 0 {
		var ins []models.TraceInput
		if err := json.Unmarshal(inputs, &ins); err != nil {
			return models.EvidenceTraceNode{}, fmt.Errorf("unmarshal inputs: %w", err)
		}
		if len(ins) > 0 {
			n.Inputs = ins
		}
	}
	if len(outputs) > 0 {
		var outs []models.TraceOutput
		if err := json.Unmarshal(outputs, &outs); err != nil {
			return models.EvidenceTraceNode{}, fmt.Errorf("unmarshal outputs: %w", err)
		}
		if len(outs) > 0 {
			n.Outputs = outs
		}
	}
	if len(reasoning) > 0 && string(reasoning) != "null" {
		var rs models.ReasoningSummary
		if err := json.Unmarshal(reasoning, &rs); err != nil {
			return models.EvidenceTraceNode{}, fmt.Errorf("unmarshal reasoning_summary: %w", err)
		}
		n.ReasoningSummary = &rs
	}
	return n, nil
}

// GetEvidenceTraceNode reads a single EvidenceTraceNode by primary key.
func (s *V2SubstrateStore) GetEvidenceTraceNode(ctx context.Context, id uuid.UUID) (*models.EvidenceTraceNode, error) {
	q := `SELECT ` + evidenceTraceNodeColumns + ` FROM evidence_trace_nodes WHERE id = $1`
	n, err := scanEvidenceTraceNode(s.db.QueryRowContext(ctx, q, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get evidence_trace_node %s: %w", id, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get evidence_trace_node %s: %w", id, err)
	}
	return &n, nil
}

// UpsertEvidenceTraceNode inserts (or updates by id) one EvidenceTraceNode.
// Marshals JSONB columns and binds nullable UUID columns via interface{}.
func (s *V2SubstrateStore) UpsertEvidenceTraceNode(ctx context.Context, n models.EvidenceTraceNode) (*models.EvidenceTraceNode, error) {
	const q = `
		INSERT INTO evidence_trace_nodes
			(id, state_machine, state_change_type,
			 recorded_at, occurred_at,
			 actor_role_ref, actor_person_ref, authority_basis_ref,
			 inputs, reasoning_summary, outputs,
			 resident_ref, created_at)
		VALUES
			($1, $2, $3,
			 $4, $5,
			 $6, $7, $8,
			 $9, $10, $11,
			 $12, NOW())
		ON CONFLICT (id) DO UPDATE SET
			state_machine       = EXCLUDED.state_machine,
			state_change_type   = EXCLUDED.state_change_type,
			recorded_at         = EXCLUDED.recorded_at,
			occurred_at         = EXCLUDED.occurred_at,
			actor_role_ref      = EXCLUDED.actor_role_ref,
			actor_person_ref    = EXCLUDED.actor_person_ref,
			authority_basis_ref = EXCLUDED.authority_basis_ref,
			inputs              = EXCLUDED.inputs,
			reasoning_summary   = EXCLUDED.reasoning_summary,
			outputs             = EXCLUDED.outputs,
			resident_ref        = EXCLUDED.resident_ref
	`

	inputsJSON, err := json.Marshal(n.Inputs)
	if err != nil {
		return nil, fmt.Errorf("marshal inputs: %w", err)
	}
	if len(n.Inputs) == 0 {
		inputsJSON = []byte("[]")
	}
	outputsJSON, err := json.Marshal(n.Outputs)
	if err != nil {
		return nil, fmt.Errorf("marshal outputs: %w", err)
	}
	if len(n.Outputs) == 0 {
		outputsJSON = []byte("[]")
	}
	var reasoningArg interface{}
	if n.ReasoningSummary != nil {
		b, err := json.Marshal(n.ReasoningSummary)
		if err != nil {
			return nil, fmt.Errorf("marshal reasoning_summary: %w", err)
		}
		reasoningArg = b
	}

	var roleArg, personArg, authArg, residentArg interface{}
	if n.Actor.RoleRef != nil {
		roleArg = *n.Actor.RoleRef
	}
	if n.Actor.PersonRef != nil {
		personArg = *n.Actor.PersonRef
	}
	if n.Actor.AuthorityBasisRef != nil {
		authArg = *n.Actor.AuthorityBasisRef
	}
	if n.ResidentRef != nil {
		residentArg = *n.ResidentRef
	}

	if _, err := s.db.ExecContext(ctx, q,
		n.ID,                // $1
		n.StateMachine,      // $2
		n.StateChangeType,   // $3
		n.RecordedAt,        // $4
		n.OccurredAt,        // $5
		roleArg,             // $6
		personArg,           // $7
		authArg,             // $8
		inputsJSON,          // $9
		reasoningArg,        // $10
		outputsJSON,         // $11
		residentArg,         // $12
	); err != nil {
		return nil, fmt.Errorf("upsert evidence_trace_node: %w", err)
	}
	return s.GetEvidenceTraceNode(ctx, n.ID)
}

// InsertEvidenceTraceEdge inserts an edge (idempotent on PK collision).
// The (from_node, to_node, edge_kind) primary key in the schema means a
// repeat insert is a no-op rather than an error.
func (s *V2SubstrateStore) InsertEvidenceTraceEdge(ctx context.Context, e evidence_trace.Edge) error {
	const q = `
		INSERT INTO evidence_trace_edges (from_node, to_node, edge_kind, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (from_node, to_node, edge_kind) DO NOTHING`
	if _, err := s.db.ExecContext(ctx, q, e.From, e.To, string(e.Kind)); err != nil {
		return fmt.Errorf("insert evidence_trace_edge: %w", err)
	}
	return nil
}

// TraceForward returns the distinct EvidenceTrace nodes reachable from
// startNode by following outgoing edges, capped at maxDepth hops.
//
// Implementation: a recursive CTE over evidence_trace_edges. UNION (not
// UNION ALL) deduplicates as it expands, which handles cycles in the
// graph automatically; the depth cap is a defence-in-depth limit even so.
func (s *V2SubstrateStore) TraceForward(ctx context.Context, startNode uuid.UUID, maxDepth int) ([]models.EvidenceTraceNode, error) {
	if maxDepth <= 0 {
		return nil, fmt.Errorf("trace_forward: maxDepth must be > 0")
	}
	const q = `
		WITH RECURSIVE downstream AS (
			SELECT to_node, 1 AS depth
			  FROM evidence_trace_edges
			 WHERE from_node = $1
			UNION
			SELECT e.to_node, d.depth + 1
			  FROM evidence_trace_edges e
			  JOIN downstream d ON e.from_node = d.to_node
			 WHERE d.depth < $2
		)
		SELECT ` + evidenceTraceNodeColumns + `
		  FROM evidence_trace_nodes
		 WHERE id IN (SELECT to_node FROM downstream)`
	rows, err := s.db.QueryContext(ctx, q, startNode, maxDepth)
	if err != nil {
		return nil, fmt.Errorf("trace_forward: %w", err)
	}
	defer rows.Close()
	var out []models.EvidenceTraceNode
	for rows.Next() {
		n, err := scanEvidenceTraceNode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// TraceBackward is the symmetric reverse traversal: nodes reachable by
// following incoming edges (ancestors), capped at maxDepth hops.
func (s *V2SubstrateStore) TraceBackward(ctx context.Context, startNode uuid.UUID, maxDepth int) ([]models.EvidenceTraceNode, error) {
	if maxDepth <= 0 {
		return nil, fmt.Errorf("trace_backward: maxDepth must be > 0")
	}
	const q = `
		WITH RECURSIVE upstream AS (
			SELECT from_node, 1 AS depth
			  FROM evidence_trace_edges
			 WHERE to_node = $1
			UNION
			SELECT e.from_node, u.depth + 1
			  FROM evidence_trace_edges e
			  JOIN upstream u ON e.to_node = u.from_node
			 WHERE u.depth < $2
		)
		SELECT ` + evidenceTraceNodeColumns + `
		  FROM evidence_trace_nodes
		 WHERE id IN (SELECT from_node FROM upstream)`
	rows, err := s.db.QueryContext(ctx, q, startNode, maxDepth)
	if err != nil {
		return nil, fmt.Errorf("trace_backward: %w", err)
	}
	defer rows.Close()
	var out []models.EvidenceTraceNode
	for rows.Next() {
		n, err := scanEvidenceTraceNode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// Compile-time interface assertions.
var (
	_ interfaces.ResidentStore       = (*V2SubstrateStore)(nil)
	_ interfaces.PersonStore         = (*V2SubstrateStore)(nil)
	_ interfaces.RoleStore           = (*V2SubstrateStore)(nil)
	_ interfaces.MedicineUseStore    = (*V2SubstrateStore)(nil)
	_ interfaces.ObservationStore    = (*V2SubstrateStore)(nil)
	_ interfaces.EventStore          = (*V2SubstrateStore)(nil)
	_ interfaces.EvidenceTraceStore  = (*V2SubstrateStore)(nil)
)
