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
	client     *FHIRClient
	kb7        *KB7Client
	db         *gorm.DB
	logger     *zap.Logger
	eventBus   *services.EventBus
	interval   time.Duration
	lastSynced time.Time
	cancel     context.CancelFunc
	done       chan struct{}
}

// NewSyncWorker creates a FHIR→KB-20 sync worker.
func NewSyncWorker(client *FHIRClient, kb7 *KB7Client, db *gorm.DB, logger *zap.Logger, eventBus *services.EventBus) *SyncWorker {
	return &SyncWorker{
		client:     client,
		kb7:        kb7,
		db:         db,
		logger:     logger,
		eventBus:   eventBus,
		interval:   5 * time.Minute,
		lastSynced: time.Now().UTC().Add(-30 * 24 * time.Hour), // initial: look back 30 days
		done:       make(chan struct{}),
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

		// Check for existing by FHIR ID
		var existing models.MedicationState
		result := w.db.Where("fhir_medication_request_id = ?", state.FHIRMedicationRequestID).First(&existing)
		if result.Error == nil {
			// Update status
			w.db.Model(&existing).Updates(map[string]interface{}{
				"is_active": state.IsActive,
				"atc_code":  state.ATCCode,
			})
			w.eventBus.Publish(models.EventMedicationChange, state.PatientID, models.MedicationChangePayload{
				ChangeType: "UPDATE",
				DrugName:   state.DrugName,
				DrugClass:  state.DrugClass,
				NewDoseMg:  state.DoseMg.String(),
			})
			w.logSync("MedicationRequest", state.FHIRMedicationRequestID, "UPDATED", "")
			continue
		}

		w.db.Create(state)
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
