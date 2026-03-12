package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * Patient state snapshot stored in Flink keyed state.
 *
 * This class represents the complete patient context maintained in Flink's state backend
 * with a 7-day TTL for readmission correlation. It follows the three-tier state management
 * pattern: Hot (Flink State) → Warm (FHIR) → Graph (Neo4j).
 *
 * State Evolution Pattern:
 * - First-time patient: Initialize empty or hydrate from FHIR/Neo4j
 * - Progressive enrichment: Update with each event type
 * - Encounter closure: Flush to FHIR store, maintain in Flink for 7 days
 *
 * @see Module2_ContextAssembly Architecture specification (C01_10)
 */
public class PatientSnapshot implements Serializable {
    private static final long serialVersionUID = 1L;

    // ============================================================
    // PATIENT IDENTIFICATION
    // ============================================================

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("mrn")
    private String mrn; // Medical Record Number

    // ============================================================
    // DEMOGRAPHICS (from FHIR Patient resource)
    // ============================================================

    @JsonProperty("demographics")
    private PatientDemographics demographics;

    @JsonProperty("firstName")
    private String firstName;

    @JsonProperty("lastName")
    private String lastName;

    @JsonProperty("dateOfBirth")
    private String dateOfBirth; // ISO 8601 format

    @JsonProperty("gender")
    private String gender; // FHIR AdministrativeGender: male, female, other, unknown

    @JsonProperty("age")
    private Integer age;

    // ============================================================
    // CLINICAL DATA (from FHIR resources)
    // ============================================================

    @JsonProperty("activeConditions")
    private List<Condition> activeConditions = new ArrayList<>();

    @JsonProperty("activeMedications")
    private List<Medication> activeMedications = new ArrayList<>();

    @JsonProperty("allergies")
    private List<String> allergies = new ArrayList<>();

    // ============================================================
    // RECENT HISTORY (circular buffers)
    // ============================================================

    @JsonProperty("vitalsHistory")
    private VitalsHistory vitalsHistory = new VitalsHistory(10); // Last 10 vitals

    @JsonProperty("labHistory")
    private LabHistory labHistory = new LabHistory(20); // Last 20 labs

    // ============================================================
    // RISK SCORES (calculated from vitals/labs/conditions)
    // ============================================================

    @JsonProperty("sepsisScore")
    private Double sepsisScore;

    @JsonProperty("deteriorationScore")
    private Double deteriorationScore;

    @JsonProperty("readmissionRisk")
    private Double readmissionRisk;

    // ============================================================
    // ENCOUNTER CONTEXT (current encounter)
    // ============================================================

    @JsonProperty("encounterContext")
    private EncounterContext encounterContext;

    @JsonProperty("location")
    private String location; // Current location (e.g., "ICU-2", "ER", "Ward-4A")

    @JsonProperty("admissionTime")
    private Long admissionTime; // Epoch milliseconds

    // ============================================================
    // SOCIAL DETERMINANTS (V2 schema addition)
    // ============================================================

    @JsonProperty("socialDeterminants")
    private SocialDeterminants socialDeterminants;

    // ============================================================
    // GRAPH DATA (from Neo4j)
    // ============================================================

    @JsonProperty("careTeam")
    private List<String> careTeam = new ArrayList<>();

    @JsonProperty("riskCohorts")
    private List<String> riskCohorts = new ArrayList<>(); // e.g., "CHF", "Diabetes"

    // ============================================================
    // STATE METADATA
    // ============================================================

    @JsonProperty("lastUpdated")
    private long lastUpdated; // Epoch milliseconds

    @JsonProperty("stateVersion")
    private int stateVersion; // Incremented with each update

    @JsonProperty("firstSeen")
    private long firstSeen; // When state was first created

    @JsonProperty("isNewPatient")
    private boolean isNewPatient; // True if 404 from FHIR API

    // ============================================================
    // CONSTRUCTORS
    // ============================================================

    /**
     * Default constructor for deserialization.
     */
    public PatientSnapshot() {
        this.lastUpdated = System.currentTimeMillis();
        this.firstSeen = this.lastUpdated;
        this.stateVersion = 0;
    }

    /**
     * Constructor with patient ID.
     */
    public PatientSnapshot(String patientId) {
        this();
        this.patientId = patientId;
    }

    // ============================================================
    // FACTORY METHODS (per architecture spec)
    // ============================================================

    /**
     * Create empty patient snapshot for new patients (404 from FHIR).
     *
     * This method is called when:
     * - Google FHIR API returns 404 (patient not found)
     * - Both FHIR and Neo4j return no data
     * - Async lookup times out (500ms timeout)
     *
     * @param patientId The patient identifier
     * @return Empty patient snapshot with minimal data
     */
    public static PatientSnapshot createEmpty(String patientId) {
        PatientSnapshot snapshot = new PatientSnapshot(patientId);
        snapshot.isNewPatient = true;
        snapshot.vitalsHistory = new VitalsHistory(10);
        snapshot.labHistory = new LabHistory(20);
        snapshot.activeConditions = new ArrayList<>();
        snapshot.activeMedications = new ArrayList<>();
        snapshot.allergies = new ArrayList<>();
        snapshot.careTeam = new ArrayList<>();
        snapshot.riskCohorts = new ArrayList<>();
        return snapshot;
    }

    /**
     * Hydrate patient snapshot from FHIR and Neo4j data.
     *
     * This method is called when async lookups successfully retrieve patient data
     * from Google Healthcare FHIR API and Neo4j within the 500ms timeout.
     *
     * @param patientId The patient identifier
     * @param fhirPatient FHIR Patient resource data
     * @param conditions List of active conditions from FHIR
     * @param medications List of active medications from FHIR
     * @param graphData Neo4j graph data (care network, cohorts)
     * @return Hydrated patient snapshot with full historical context
     */
    public static PatientSnapshot hydrateFromHistory(
            String patientId,
            FHIRPatientData fhirPatient,
            List<Condition> conditions,
            List<Medication> medications,
            GraphData graphData) {

        PatientSnapshot snapshot = new PatientSnapshot(patientId);
        snapshot.isNewPatient = false;

        // Demographics from FHIR Patient resource
        if (fhirPatient != null) {
            snapshot.firstName = fhirPatient.getFirstName();
            snapshot.lastName = fhirPatient.getLastName();
            snapshot.dateOfBirth = fhirPatient.getDateOfBirth();
            snapshot.gender = fhirPatient.getGender();
            snapshot.age = fhirPatient.getAge();
            snapshot.mrn = fhirPatient.getMrn();
            snapshot.allergies = fhirPatient.getAllergies();

            // Create PatientDemographics wrapper object for JSON serialization
            snapshot.demographics = new PatientDemographics(
                fhirPatient.getFirstName(),
                fhirPatient.getLastName(),
                fhirPatient.getDateOfBirth(),
                fhirPatient.getGender(),
                fhirPatient.getAge(),
                fhirPatient.getMrn()
            );
        }

        // Clinical data from FHIR
        if (conditions != null) {
            snapshot.activeConditions = new ArrayList<>(conditions);
        }
        if (medications != null) {
            snapshot.activeMedications = new ArrayList<>(medications);
        }

        // Graph data from Neo4j
        if (graphData != null) {
            snapshot.careTeam = graphData.getCareTeam();
            snapshot.riskCohorts = graphData.getRiskCohorts();
        }

        // Initialize empty history buffers (will be populated by events)
        snapshot.vitalsHistory = new VitalsHistory(10);
        snapshot.labHistory = new LabHistory(20);

        return snapshot;
    }

    // ============================================================
    // PROGRESSIVE ENRICHMENT (per architecture spec)
    // ============================================================

    /**
     * Update snapshot with new event data.
     *
     * This implements the progressive enrichment pattern where state evolves
     * with each event type. Different event types update different parts of the state.
     *
     * @param event The canonical event to process
     */
    public void updateWithEvent(CanonicalEvent event) {
        Map<String, Object> payload = event.getPayload();

        switch (event.getEventType().toString()) {
            case "VITAL_SIGNS":
                updateVitals(payload);
                break;

            case "MEDICATION":
                updateMedications(payload);
                break;

            case "LAB_RESULT":
                updateLabs(payload);
                break;

            case "OBSERVATION":
                updateObservations(payload);
                break;

            case "ENCOUNTER":
                updateEncounter(payload);
                break;

            default:
                // Unknown event type - no state update
                break;
        }

        // Update metadata
        this.lastUpdated = System.currentTimeMillis();
        this.stateVersion++;
    }

    /**
     * Update vitals history with new vital signs.
     */
    private void updateVitals(Map<String, Object> payload) {
        VitalSign vital = VitalSign.fromPayload(payload);
        if (vital != null) {
            this.vitalsHistory.add(vital);
            // Recalculate risk scores based on new vitals
            updateRiskScores();
        }
    }

    /**
     * Update active medications list.
     */
    private void updateMedications(Map<String, Object> payload) {
        String medicationName = (String) payload.get("medication_name");
        String action = (String) payload.get("action"); // "start", "stop", "modify"

        if ("stop".equals(action)) {
            // Remove from active medications
            activeMedications.removeIf(med -> med.getName().equals(medicationName));
        } else if ("start".equals(action)) {
            // Add new medication
            Medication med = Medication.fromPayload(payload);
            if (med != null) {
                activeMedications.add(med);
            }
        }
    }

    /**
     * Update lab history with new lab results.
     */
    private void updateLabs(Map<String, Object> payload) {
        LabResult lab = LabResult.fromPayload(payload);
        if (lab != null) {
            this.labHistory.add(lab);
            // Recalculate risk scores based on new labs
            updateRiskScores();
        }
    }

    /**
     * Update clinical observations and conditions.
     */
    private void updateObservations(Map<String, Object> payload) {
        String conditionCode = (String) payload.get("condition_code");
        String status = (String) payload.get("status"); // "active", "resolved"

        if ("resolved".equals(status)) {
            // Remove from active conditions
            activeConditions.removeIf(cond -> cond.getCode().equals(conditionCode));
        } else if ("active".equals(status)) {
            // Add new condition
            Condition cond = Condition.fromPayload(payload);
            if (cond != null && !activeConditions.contains(cond)) {
                activeConditions.add(cond);
            }
        }
    }

    /**
     * Update encounter context (admission, transfer, discharge).
     */
    private void updateEncounter(Map<String, Object> payload) {
        String encounterType = (String) payload.get("encounter_type");

        if ("admission".equals(encounterType)) {
            // Start new encounter
            this.encounterContext = EncounterContext.fromPayload(payload);
        } else if ("transfer".equals(encounterType)) {
            // Update department/room
            if (this.encounterContext != null) {
                this.encounterContext.updateDepartment(
                    (String) payload.get("department"),
                    (String) payload.get("room")
                );
            }
        } else if ("discharge".equals(encounterType)) {
            // Note: Encounter context kept until state flush
            if (this.encounterContext != null) {
                this.encounterContext.setDischargeTime(System.currentTimeMillis());
            }
        }
    }

    /**
     * Recalculate risk scores based on current vitals, labs, and conditions.
     *
     * This is a simplified version. Production would use actual clinical algorithms.
     */
    private void updateRiskScores() {
        // Simplified risk scoring logic
        // TODO: Implement actual clinical risk algorithms

        // Sepsis score based on recent vitals
        List<VitalSign> recentVitals = vitalsHistory.getRecent(3);
        double sepsisRisk = calculateSepsisRisk(recentVitals);
        this.sepsisScore = sepsisRisk;

        // Deterioration score based on trends
        this.deteriorationScore = calculateDeteriorationRisk(recentVitals);

        // Readmission risk based on conditions and history
        this.readmissionRisk = calculateReadmissionRisk();
    }

    private double calculateSepsisRisk(List<VitalSign> vitals) {
        // Simplified: Check for abnormal vitals
        if (vitals.isEmpty()) return 0.0;

        double risk = 0.0;
        VitalSign latest = vitals.get(vitals.size() - 1);

        // Check heart rate > 100 (tachycardia)
        if (latest.getHeartRate() != null && latest.getHeartRate() > 100) {
            risk += 0.3;
        }

        // Check temperature > 100.4°F (fever)
        if (latest.getTemperature() != null && latest.getTemperature() > 100.4) {
            risk += 0.4;
        }

        // Check respiratory rate > 20 (tachypnea)
        if (latest.getRespiratoryRate() != null && latest.getRespiratoryRate() > 20) {
            risk += 0.3;
        }

        return Math.min(risk, 1.0); // Cap at 1.0
    }

    private double calculateDeteriorationRisk(List<VitalSign> vitals) {
        // Simplified: Check for worsening trends
        // TODO: Implement NEWS2 or MEWS scoring
        return 0.0;
    }

    private double calculateReadmissionRisk() {
        // Simplified: Based on number of active conditions
        int conditionCount = activeConditions.size();
        return Math.min(conditionCount * 0.15, 1.0);
    }

    // ============================================================
    // EVENT CREATION (convert state to enriched event)
    // ============================================================

    /**
     * Create enriched event by merging base event with patient context.
     *
     * @param baseEvent The canonical event from Module 1
     * @return Enriched event with patient context fields
     */
    public EnrichedEvent toEnrichedEvent(CanonicalEvent baseEvent) {
        EnrichedEvent enriched = new EnrichedEvent();

        // Copy base event fields
        enriched.setId(baseEvent.getId());
        enriched.setPatientId(baseEvent.getPatientId());
        enriched.setEncounterId(baseEvent.getEncounterId());
        enriched.setEventType(baseEvent.getEventType());
        enriched.setEventTime(baseEvent.getEventTime());
        enriched.setSourceSystem(baseEvent.getSourceSystem());
        enriched.setPayload(baseEvent.getPayload());
        enriched.setProcessingTime(System.currentTimeMillis());

        // Create PatientContext with snapshot data
        PatientContext context = new PatientContext();
        context.setPatientId(this.patientId);

        // Create demographics object
        PatientContext.PatientDemographics demographics = new PatientContext.PatientDemographics();
        demographics.setAge(this.age != null ? this.age : 0);
        demographics.setGender(this.gender);
        context.setDemographics(demographics);

        // Set context on enriched event
        enriched.setPatientContext(context);

        // Set enrichment version
        enriched.setEnrichmentVersion("2.0-snapshot");

        return enriched;
    }

    // ============================================================
    // GETTERS AND SETTERS
    // ============================================================

    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

    public String getMrn() {
        return mrn;
    }

    public void setMrn(String mrn) {
        this.mrn = mrn;
    }

    public String getFirstName() {
        return firstName;
    }

    public void setFirstName(String firstName) {
        this.firstName = firstName;
    }

    public String getLastName() {
        return lastName;
    }

    public void setLastName(String lastName) {
        this.lastName = lastName;
    }

    public String getDateOfBirth() {
        return dateOfBirth;
    }

    public void setDateOfBirth(String dateOfBirth) {
        this.dateOfBirth = dateOfBirth;
    }

    public String getGender() {
        return gender;
    }

    public void setGender(String gender) {
        this.gender = gender;
    }

    public Integer getAge() {
        return age;
    }

    public void setAge(Integer age) {
        this.age = age;
    }

    public List<Condition> getActiveConditions() {
        return activeConditions;
    }

    public void setActiveConditions(List<Condition> activeConditions) {
        this.activeConditions = activeConditions;
    }

    public List<Medication> getActiveMedications() {
        return activeMedications;
    }

    public void setActiveMedications(List<Medication> activeMedications) {
        this.activeMedications = activeMedications;
    }

    public List<String> getAllergies() {
        return allergies;
    }

    public void setAllergies(List<String> allergies) {
        this.allergies = allergies;
    }

    public VitalsHistory getVitalsHistory() {
        return vitalsHistory;
    }

    public void setVitalsHistory(VitalsHistory vitalsHistory) {
        this.vitalsHistory = vitalsHistory;
    }

    public LabHistory getLabHistory() {
        return labHistory;
    }

    public void setLabHistory(LabHistory labHistory) {
        this.labHistory = labHistory;
    }

    public Double getSepsisScore() {
        return sepsisScore;
    }

    public void setSepsisScore(Double sepsisScore) {
        this.sepsisScore = sepsisScore;
    }

    public Double getDeteriorationScore() {
        return deteriorationScore;
    }

    public void setDeteriorationScore(Double deteriorationScore) {
        this.deteriorationScore = deteriorationScore;
    }

    public Double getReadmissionRisk() {
        return readmissionRisk;
    }

    public void setReadmissionRisk(Double readmissionRisk) {
        this.readmissionRisk = readmissionRisk;
    }

    public EncounterContext getEncounterContext() {
        return encounterContext;
    }

    public void setEncounterContext(EncounterContext encounterContext) {
        this.encounterContext = encounterContext;
    }

    public List<String> getCareTeam() {
        return careTeam;
    }

    public void setCareTeam(List<String> careTeam) {
        this.careTeam = careTeam;
    }

    public List<String> getRiskCohorts() {
        return riskCohorts;
    }

    public void setRiskCohorts(List<String> riskCohorts) {
        this.riskCohorts = riskCohorts;
    }

    public long getLastUpdated() {
        return lastUpdated;
    }

    public void setLastUpdated(long lastUpdated) {
        this.lastUpdated = lastUpdated;
    }

    public int getStateVersion() {
        return stateVersion;
    }

    public void setStateVersion(int stateVersion) {
        this.stateVersion = stateVersion;
    }

    public long getFirstSeen() {
        return firstSeen;
    }

    public void setFirstSeen(long firstSeen) {
        this.firstSeen = firstSeen;
    }

    public boolean isNewPatient() {
        return isNewPatient;
    }

    public void setNewPatient(boolean newPatient) {
        isNewPatient = newPatient;
    }

    public PatientDemographics getDemographics() {
        return demographics;
    }

    public void setDemographics(PatientDemographics demographics) {
        this.demographics = demographics;
    }

    public String getLocation() {
        return location;
    }

    public void setLocation(String location) {
        this.location = location;
    }

    public Long getAdmissionTime() {
        return admissionTime;
    }

    public void setAdmissionTime(Long admissionTime) {
        this.admissionTime = admissionTime;
    }

    public SocialDeterminants getSocialDeterminants() {
        return socialDeterminants;
    }

    public void setSocialDeterminants(SocialDeterminants socialDeterminants) {
        this.socialDeterminants = socialDeterminants;
    }

    // ============================================================
    // CLINICAL BASELINE METHODS (for Module 2 enrichment)
    // ============================================================

    /**
     * Get baseline creatinine value for acute kidney injury detection.
     * Returns the baseline value from recent lab history or null if not available.
     *
     * @return Baseline creatinine in mg/dL, or null if unavailable
     */
    public Double getBaselineCreatinine() {
        if (labHistory == null || labHistory.isEmpty()) {
            return null;
        }

        // Look for creatinine in recent labs (last 7 days)
        List<LabResult> recentLabs = labHistory.getRecent(20);
        for (LabResult lab : recentLabs) {
            if (lab.getLabCode().equalsIgnoreCase("creatinine") ||
                lab.getLabCode().equalsIgnoreCase("CREA")) {
                return lab.getValue();
            }
        }

        return null; // No baseline available
    }

    /**
     * Get timestamp of last surgical procedure.
     * Used for post-operative monitoring and risk scoring.
     *
     * @return Epoch milliseconds of last surgery, or null if no surgery recorded
     */
    public Long getLastSurgeryTime() {
        if (encounterContext == null) {
            return null;
        }

        // Check encounter context for surgery-related procedures
        // This would be populated from FHIR Procedure resources
        // For now, return null as this requires procedure history integration
        return null; // TODO: Implement procedure history tracking
    }

    // Alias methods for compatibility with V1/V2 migration code
    public List<String> getConditions() {
        // Convert List<Condition> to List<String> for backward compatibility
        List<String> conditionCodes = new ArrayList<>();
        for (Condition condition : activeConditions) {
            conditionCodes.add(condition.getCode());
        }
        return conditionCodes;
    }

    public void setConditions(List<String> conditions) {
        // Convert List<String> to List<Condition> for backward compatibility
        this.activeConditions = new ArrayList<>();
        for (String code : conditions) {
            Condition condition = new Condition();
            condition.setCode(code);
            this.activeConditions.add(condition);
        }
    }

    public List<String> getLabResults() {
        // Convert LabHistory to List<String> for backward compatibility
        if (labHistory == null) {
            return new ArrayList<>();
        }
        List<LabResult> labs = labHistory.getRecent(20);
        List<String> labStrings = new ArrayList<>();
        for (LabResult lab : labs) {
            labStrings.add(lab.toString());
        }
        return labStrings;
    }

    public void setLabResults(List<String> labResults) {
        // Initialize lab history if needed
        if (this.labHistory == null) {
            this.labHistory = new LabHistory(20);
        }
        // Note: This is a simplified implementation
        // Full implementation would parse the strings back to LabResult objects
    }

    @Override
    public String toString() {
        return "PatientSnapshot{" +
                "patientId='" + patientId + '\'' +
                ", name='" + firstName + " " + lastName + '\'' +
                ", age=" + age +
                ", conditions=" + activeConditions.size() +
                ", medications=" + activeMedications.size() +
                ", stateVersion=" + stateVersion +
                ", isNew=" + isNewPatient +
                '}';
    }
}
