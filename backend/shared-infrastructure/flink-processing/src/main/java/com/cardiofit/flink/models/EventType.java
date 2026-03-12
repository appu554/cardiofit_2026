package com.cardiofit.flink.models;

/**
 * Enumeration of canonical event types in the CardioFit healthcare system
 */
public enum EventType {

    // Patient-related events
    ADMISSION,              // Alias for PATIENT_ADMISSION (test convenience)
    PATIENT_ADMISSION,
    PATIENT_DISCHARGE,
    PATIENT_TRANSFER,
    PATIENT_UPDATE,

    // Clinical observation events
    VITAL_SIGN,
    VITAL_SIGNS,        // Added for ClinicalPathwayAdherenceFunction
    LAB_RESULT,
    DIAGNOSTIC_RESULT,
    IMAGING_RESULT,
    DIAGNOSIS_ADDED,    // Added for ClinicalPathwayAdherenceFunction

    // Medication events
    MEDICATION,            // Alias for MEDICATION_ORDERED (test convenience)
    MEDICATION_ORDERED,
    MEDICATION_PRESCRIBED,  // Added for ClinicalPathwayAdherenceFunction
    MEDICATION_ADMINISTERED,
    MEDICATION_DISCONTINUED,
    MEDICATION_MISSED,

    // Procedure events
    PROCEDURE_SCHEDULED,
    PROCEDURE_STARTED,
    PROCEDURE_COMPLETED,
    PROCEDURE_CANCELLED,

    // Encounter events
    ENCOUNTER_START,
    ENCOUNTER_END,
    ENCOUNTER_UPDATE,

    // Device events
    DEVICE_READING,
    DEVICE_ALARM,
    DEVICE_MALFUNCTION,
    DEVICE_MAINTENANCE,

    // Safety events
    ADVERSE_EVENT,
    ALLERGY_ALERT,
    DRUG_INTERACTION,
    CONTRAINDICATION,

    // Workflow events
    TASK_CREATED,
    TASK_ASSIGNED,
    TASK_COMPLETED,
    ALERT_GENERATED,

    // System events
    SYSTEM_ERROR,
    INTEGRATION_FAILURE,
    VALIDATION_ERROR,

    // Unknown or unmapped events
    UNKNOWN;

    /**
     * Map a raw event type string to canonical EventType
     */
    public static EventType fromString(String rawType) {
        if (rawType == null || rawType.trim().isEmpty()) {
            return UNKNOWN;
        }

        String normalized = rawType.toUpperCase().replace("-", "_").replace(" ", "_");

        // Handle common variations
        switch (normalized) {
            case "PATIENT.ADMISSION":
            case "PATIENT_ADMITTED":
            case "ADMISSION":
                return PATIENT_ADMISSION;

            case "PATIENT.DISCHARGE":
            case "PATIENT_DISCHARGED":
            case "DISCHARGE":
                return PATIENT_DISCHARGE;

            case "VITAL":
            case "VITALS":
            case "VITAL_SIGNS":
                return VITAL_SIGN;

            case "LAB":
            case "LABORATORY":
            case "LAB_RESULT":
                return LAB_RESULT;

            // Generic "MEDICATION" should default to MEDICATION_ORDERED
            case "MEDICATION":
                return MEDICATION_ORDERED;

            case "MEDICATION.ADMINISTERED":
            case "MED_ADMIN":
            case "DRUG_ADMINISTERED":
                return MEDICATION_ADMINISTERED;

            case "MEDICATION.ORDERED":
            case "MED_ORDER":
            case "PRESCRIPTION":
                return MEDICATION_ORDERED;

            // Generic "OBSERVATION" should map to LAB_RESULT
            case "OBSERVATION":
                return LAB_RESULT;

            case "DEVICE.READING":
            case "MONITOR_DATA":
            case "SENSOR_DATA":
                return DEVICE_READING;

            case "DEVICE.ALARM":
            case "MONITOR_ALARM":
            case "SENSOR_ALARM":
                return DEVICE_ALARM;

            default:
                try {
                    return EventType.valueOf(normalized);
                } catch (IllegalArgumentException e) {
                    return UNKNOWN;
                }
        }
    }

    /**
     * Check if this event type is clinical (patient-care related)
     */
    public boolean isClinical() {
        switch (this) {
            case PATIENT_ADMISSION:
            case PATIENT_DISCHARGE:
            case PATIENT_TRANSFER:
            case VITAL_SIGN:
            case LAB_RESULT:
            case DIAGNOSTIC_RESULT:
            case MEDICATION_ORDERED:
            case MEDICATION_ADMINISTERED:
            case PROCEDURE_SCHEDULED:
            case PROCEDURE_COMPLETED:
            case ADVERSE_EVENT:
            case ALLERGY_ALERT:
            case DRUG_INTERACTION:
                return true;
            default:
                return false;
        }
    }

    /**
     * Check if this event type is critical (requires immediate attention)
     */
    public boolean isCritical() {
        switch (this) {
            case DEVICE_ALARM:
            case ADVERSE_EVENT:
            case ALLERGY_ALERT:
            case DRUG_INTERACTION:
            case CONTRAINDICATION:
            case DEVICE_MALFUNCTION:
                return true;
            default:
                return false;
        }
    }

    /**
     * Check if this event type is related to medication
     */
    public boolean isMedicationRelated() {
        switch (this) {
            case MEDICATION_ORDERED:
            case MEDICATION_ADMINISTERED:
            case MEDICATION_DISCONTINUED:
            case MEDICATION_MISSED:
            case DRUG_INTERACTION:
            case ALLERGY_ALERT:
                return true;
            default:
                return false;
        }
    }

    /**
     * Get the priority level for this event type
     */
    public Priority getPriority() {
        if (isCritical()) {
            return Priority.CRITICAL;
        } else if (isClinical()) {
            return Priority.HIGH;
        } else {
            return Priority.NORMAL;
        }
    }

    public enum Priority {
        CRITICAL(1),
        HIGH(2),
        NORMAL(3),
        LOW(4);

        private final int level;

        Priority(int level) {
            this.level = level;
        }

        public int getLevel() {
            return level;
        }
    }
}