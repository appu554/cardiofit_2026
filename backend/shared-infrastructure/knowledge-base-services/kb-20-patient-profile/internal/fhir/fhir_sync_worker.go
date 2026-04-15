package fhir

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"kb-patient-profile/internal/models"
	"kb-patient-profile/internal/services"
)

// SyncWorker polls the Google FHIR Store for new/updated resources and
// upserts them into KB-20's PostgreSQL tables. Runs as a background goroutine.
type SyncWorker struct {
	client         *FHIRClient
	kb7            *KB7Client
	db             *gorm.DB
	logger         *zap.Logger
	eventBus       *services.EventBus
	ckmRecompute   *services.CKMRecomputationService // Phase 7 P7-B
	interval       time.Duration
	lastSynced     time.Time
	cancel         context.CancelFunc
	done           chan struct{}
}

// NewSyncWorker creates a FHIR→KB-20 sync worker.
// Phase 7 P7-B: ckmRecompute is optional — pass nil in bootstrap paths
// where CKMRecomputationService isn't wired yet. When nil, the worker
// falls back to event-only behaviour and the CKM trigger gap remains
// open (production must always wire this).
func NewSyncWorker(
	client *FHIRClient,
	kb7 *KB7Client,
	db *gorm.DB,
	logger *zap.Logger,
	eventBus *services.EventBus,
	ckmRecompute *services.CKMRecomputationService,
) *SyncWorker {
	return &SyncWorker{
		client:       client,
		kb7:          kb7,
		db:           db,
		logger:       logger,
		eventBus:     eventBus,
		ckmRecompute: ckmRecompute,
		interval:     5 * time.Minute,
		lastSynced:   time.Now().UTC().Add(-30 * 24 * time.Hour), // initial: look back 30 days
		done:         make(chan struct{}),
	}
}

// Start launches the background sync goroutine.
func (w *SyncWorker) Start(ctx context.Context) {
	pollCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	go func() {
		defer close(w.done)
		w.logger.Info("FHIR sync worker started", zap.Duration("interval", w.interval))

		// Run immediately once, then on interval
		w.syncAll()

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-pollCtx.Done():
				w.logger.Info("FHIR sync worker shutting down")
				return
			case <-ticker.C:
				w.syncAll()
			}
		}
	}()
}

// Stop gracefully shuts down the sync worker.
func (w *SyncWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
		<-w.done
	}
}

// syncAll fetches all resource types updated since last sync.
func (w *SyncWorker) syncAll() {
	since := w.lastSynced
	now := time.Now().UTC()

	w.syncPatients(since)
	w.syncObservations(since)
	w.syncMedicationRequests(since)

	w.lastSynced = now
}

func (w *SyncWorker) syncPatients(since time.Time) {
	patients, err := w.client.SearchPatients(since)
	if err != nil {
		w.logger.Error("Failed to fetch FHIR Patients", zap.Error(err))
		return
	}

	for _, patient := range patients {
		profile := FHIRPatientToProfile(patient)
		if profile.PatientID == "" {
			w.logSync("Patient", extractString(patient, "id"), "SKIPPED", "no patient_id identifier")
			continue
		}

		// Upsert by patient_id
		var existing models.PatientProfile
		result := w.db.Where("patient_id = ?", profile.PatientID).First(&existing)
		if result.Error == nil {
			// Update FHIR reference
			w.db.Model(&existing).Updates(map[string]interface{}{
				"fhir_patient_id": profile.FHIRPatientID,
				"sex":             profile.Sex,
				"age":             profile.Age,
			})
			w.logSync("Patient", profile.FHIRPatientID, "UPDATED", "")
		} else {
			if err := w.db.Create(profile).Error; err != nil {
				w.logSync("Patient", profile.FHIRPatientID, "SKIPPED", err.Error())
				w.logger.Error("Failed to create patient from FHIR",
					zap.String("fhir_id", profile.FHIRPatientID),
					zap.Error(err))
				continue
			}
			w.logSync("Patient", profile.FHIRPatientID, "CREATED", "")
		}
	}

	if len(patients) > 0 {
		w.logger.Info("FHIR Patient sync completed", zap.Int("count", len(patients)))
	}
}

func (w *SyncWorker) syncObservations(since time.Time) {
	observations, err := w.client.SearchObservationsSince(since)
	if err != nil {
		w.logger.Error("Failed to fetch FHIR Observations", zap.Error(err))
		return
	}

	for _, obs := range observations {
		lab := FHIRObservationToLab(obs, w.kb7)
		if lab.PatientID == "" || lab.LabType == "" {
			w.logSync("Observation", extractString(obs, "id"), "SKIPPED", "missing patient_id or lab_type")
			continue
		}

		// Resolve FHIR Patient reference to KB-20 patient_id
		lab.PatientID = w.resolvePatientID(lab.PatientID)

		// Enrich with LOINC from KB-7 if not already set
		if lab.LOINCCode == "" {
			if concept, err := w.kb7.ResolveLOINC(lab.LabType); err == nil {
				lab.LOINCCode = concept.Code
			}
		}

		// Check for existing by FHIR ID
		var existing models.LabEntry
		result := w.db.Where("fhir_observation_id = ?", lab.FHIRObservationID).First(&existing)
		if result.Error == nil {
			w.logSync("Observation", lab.FHIRObservationID, "SKIPPED", "already exists")
			continue
		}

		w.db.Create(lab)
		labValueF64, _ := lab.Value.Float64()
		w.eventBus.Publish(models.EventLabResult, lab.PatientID, models.LabResultPayload{
			LabType:          lab.LabType,
			Value:            labValueF64,
			Unit:             lab.Unit,
			MeasuredAt:       lab.MeasuredAt.Format(time.RFC3339),
			Source:           "FHIR_SYNC",
			ValidationStatus: "ACCEPTED",
			IsDerived:        false,
		})

		// Phase 7 P7-B: CKM stage recomputation trigger. A newly-persisted
		// LVEF / NT-proBNP / CAC observation can shift a patient across a
		// Stage 4 substage boundary (LVEF≤40 → 4c, CAC>0 → 4a) — invoke
		// the recomputation service so CKMTransitionPublisher fires if
		// the stage changes. The publisher no-ops on unchanged stage, so
		// per-LOINC gating is defensive against wasted event traffic but
		// not strictly required for correctness.
		if w.ckmRecompute != nil && models.IsCKMStagingRelevant(lab.LabType) {
			if _, err := w.ckmRecompute.RecomputeAndPublish(lab.PatientID, lab.FHIRObservationID); err != nil {
				w.logger.Warn("CKM recomputation failed after observation sync",
					zap.String("patient_id", lab.PatientID),
					zap.String("lab_type", lab.LabType),
					zap.String("fhir_observation_id", lab.FHIRObservationID),
					zap.Error(err))
			}
		}

		w.logSync("Observation", lab.FHIRObservationID, "CREATED", "")
	}

	if len(observations) > 0 {
		w.logger.Info("FHIR Observation sync completed", zap.Int("count", len(observations)))
	}
}

func (w *SyncWorker) syncMedicationRequests(since time.Time) {
	requests, err := w.client.SearchMedicationRequestsSince(since)
	if err != nil {
		w.logger.Error("Failed to fetch FHIR MedicationRequests", zap.Error(err))
		return
	}

	for _, req := range requests {
		state := FHIRMedicationRequestToState(req)
		if state.PatientID == "" || state.DrugName == "" {
			w.logSync("MedicationRequest", extractString(req, "id"), "SKIPPED", "missing patient_id or drug_name")
			continue
		}

		// Resolve FHIR Patient reference to KB-20 patient_id
		state.PatientID = w.resolvePatientID(state.PatientID)

		// Check for existing by FHIR ID.
		//
		// This branch handles three distinct clinical events on a single
		// existing MedicationRequest: dose change (state.IsActive true,
		// previously active), drug stop (state.IsActive false, previously
		// active), and re-start (state.IsActive true, previously inactive).
		// All three are pharmacologically significant and must fire the
		// stability override — stopping amlodipine shifts BP just as much
		// as starting it, so the dwell bypass applies uniformly.
		//
		// The ChangeType label distinguishes STOP from UPDATE so downstream
		// consumers (KB-19 protocol orchestrator, KB-23 card generator) can
		// react to the clinical semantics, not just the database operation.
		var existing models.MedicationState
		result := w.db.Where("fhir_medication_request_id = ?", state.FHIRMedicationRequestID).First(&existing)
		if result.Error == nil {
			// Persist the status transition.
			w.db.Model(&existing).Updates(map[string]interface{}{
				"is_active": state.IsActive,
				"atc_code":  state.ATCCode,
			})

			// Phase 5 P5-2/P5-5: stamp the change timestamp + drug class on
			// the patient profile regardless of whether this is a dose
			// change, a stop, or a re-start — all three are events the
			// stability engine should honour. stampMedicationChange is
			// unconditional by design; do not add is_active gating here.
			w.stampMedicationChange(state.PatientID, state.DrugClass)

			// Label the event by clinical semantics, not database op.
			changeType := "UPDATE"
			if existing.IsActive && !state.IsActive {
				changeType = "STOP"
			} else if !existing.IsActive && state.IsActive {
				changeType = "RESTART"
			}
			w.eventBus.Publish(models.EventMedicationChange, state.PatientID, models.MedicationChangePayload{
				ChangeType: changeType,
				DrugName:   state.DrugName,
				DrugClass:  state.DrugClass,
				NewDoseMg:  state.DoseMg.String(),
			})
			w.logSync("MedicationRequest", state.FHIRMedicationRequestID, "UPDATED", "")
			continue
		}

		w.db.Create(state)
		w.stampMedicationChange(state.PatientID, state.DrugClass)
		w.eventBus.Publish(models.EventMedicationChange, state.PatientID, models.MedicationChangePayload{
			ChangeType: "ADD",
			DrugName:   state.DrugName,
			DrugClass:  state.DrugClass,
			NewDoseMg:  state.DoseMg.String(),
		})
		w.logSync("MedicationRequest", state.FHIRMedicationRequestID, "CREATED", "")
	}

	if len(requests) > 0 {
		w.logger.Info("FHIR MedicationRequest sync completed", zap.Int("count", len(requests)))
	}
}

// resolvePatientID maps a FHIR Patient resource ID to the KB-20 patient_id.
// FHIR observations reference "Patient/<fhir-uuid>", but KB-20 stores the
// identifier value (e.g., "FHIR-TEST-001") as patient_id. This lookup ensures
// all synced resources use the same patient_id.
func (w *SyncWorker) resolvePatientID(fhirPatientID string) string {
	var profile models.PatientProfile
	result := w.db.Where("fhir_patient_id = ?", fhirPatientID).First(&profile)
	if result.Error == nil && profile.PatientID != "" {
		return profile.PatientID
	}
	// If no match found, keep the FHIR ID as-is (patient may not be synced yet)
	return fhirPatientID
}

func (w *SyncWorker) logSync(resourceType, fhirID, action, errMsg string) {
	entry := models.FHIRSyncLog{
		ResourceType: resourceType,
		FHIRID:       fhirID,
		Action:       action,
		SyncedAt:     time.Now().UTC(),
		Error:        errMsg,
	}
	w.db.Create(&entry)
}

// stampMedicationChange records the most recent antihypertensive medication
// event on the patient profile. Read by KB-26's BP context stability engine
// (Phase 5 P5-2 + P5-5) to bypass the phenotype dwell window when a recent
// prescription change would otherwise be suppressed, with a PK-aware window
// sized by the drug class. Non-fatal: if the update fails, the event still
// publishes and KB-26 falls back to no-override behaviour (safe default).
func (w *SyncWorker) stampMedicationChange(patientID, drugClass string) {
	if patientID == "" {
		return
	}
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"last_medication_change_at":    now,
		"last_medication_change_class": drugClass,
	}
	if err := w.db.Model(&models.PatientProfile{}).
		Where("patient_id = ?", patientID).
		Updates(updates).Error; err != nil {
		w.logger.Warn("failed to stamp medication change on patient profile",
			zap.String("patient_id", patientID),
			zap.String("drug_class", drugClass),
			zap.Error(err))
	}
}
