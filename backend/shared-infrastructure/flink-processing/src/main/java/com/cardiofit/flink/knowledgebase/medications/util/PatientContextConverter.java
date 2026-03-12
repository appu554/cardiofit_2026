package com.cardiofit.flink.knowledgebase.medications.util;

import com.cardiofit.flink.models.PatientContext;
import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.PatientContextState;

/**
 * Utility class to convert between PatientContext and EnrichedPatientContext.
 *
 * This converter bridges the gap between test expectations (PatientContext)
 * and actual implementation requirements (EnrichedPatientContext).
 *
 * @author CardioFit Module 3 Team
 * @version 1.0
 * @since 2025-10-25
 */
public class PatientContextConverter {

    /**
     * Convert PatientContext to EnrichedPatientContext.
     *
     * Creates a minimal EnrichedPatientContext wrapper around PatientContext data.
     * Used primarily for testing and backward compatibility.
     *
     * NOTE: This is a simplified converter for test compatibility.
     * PatientContextState has different structure than PatientContext,
     * so we create minimal enriched context for testing purposes.
     *
     * @param patientContext The patient context to convert
     * @return EnrichedPatientContext with minimal patient state
     */
    public static EnrichedPatientContext toEnriched(PatientContext patientContext) {
        if (patientContext == null) {
            return null;
        }

        // Create minimal patient state (PatientContextState structure differs)
        PatientContextState state = new PatientContextState();
        state.setPatientId(patientContext.getPatientId());
        state.setEventCount(patientContext.getEventCount());

        // Create enriched context
        EnrichedPatientContext enriched = new EnrichedPatientContext(
            patientContext.getPatientId(),
            state
        );

        // Set metadata
        enriched.setEventTime(patientContext.getLastEventTime());
        enriched.setEncounterId(patientContext.getCurrentEncounterId());

        return enriched;
    }

    /**
     * Extract PatientContext from EnrichedPatientContext.
     *
     * Creates a PatientContext from the patient state within EnrichedPatientContext.
     * Used for backward compatibility with older APIs.
     *
     * NOTE: This is a simplified converter for test compatibility.
     *
     * @param enriched The enriched patient context
     * @return PatientContext with basic data extracted
     */
    public static PatientContext fromEnriched(EnrichedPatientContext enriched) {
        if (enriched == null || enriched.getPatientState() == null) {
            return null;
        }

        PatientContext context = new PatientContext();
        PatientContextState state = enriched.getPatientState();

        // Copy basic fields
        context.setPatientId(enriched.getPatientId());
        context.setCurrentEncounterId(enriched.getEncounterId());
        context.setLastEventTime(enriched.getEventTime());
        context.setEventCount((int) state.getEventCount()); // Cast long to int for compatibility

        return context;
    }

    private PatientContextConverter() {
        // Private constructor to prevent instantiation
    }
}
