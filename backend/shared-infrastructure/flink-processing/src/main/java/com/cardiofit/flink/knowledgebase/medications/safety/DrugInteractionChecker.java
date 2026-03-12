package com.cardiofit.flink.knowledgebase.medications.safety;

import com.cardiofit.flink.knowledgebase.medications.model.Medication;
import com.cardiofit.flink.knowledgebase.medications.loader.MedicationDatabaseLoader;
import com.cardiofit.flink.models.PatientContext;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;
import java.util.stream.Collectors;

/**
 * Drug-Drug Interaction Checker.
 *
 * Checks for clinically significant drug interactions between medications
 * in a patient's medication list. Supports three severity levels:
 * - MAJOR: Life-threatening, requires immediate intervention
 * - MODERATE: Clinically significant, requires monitoring
 * - MINOR: Limited clinical impact, documentation only
 *
 * @author CardioFit Module 3 Team
 * @version 1.0
 * @since 2025-10-24
 */
public class DrugInteractionChecker {
    private static final Logger logger = LoggerFactory.getLogger(DrugInteractionChecker.class);

    private final MedicationDatabaseLoader medicationLoader;

    // Drug interaction database (in production, load from YAML files)
    private final Map<String, List<DrugInteraction>> interactionDatabase;

    public DrugInteractionChecker() {
        this.medicationLoader = MedicationDatabaseLoader.getInstance();
        this.interactionDatabase = new HashMap<>();
        loadInteractionDatabase();
    }

    /**
     * Load drug interaction database from resources.
     * In production, would load from knowledge-base/drug-interactions/ YAML files.
     */
    private void loadInteractionDatabase() {
        logger.info("Loading drug interaction database...");

        // TODO: In production, load from YAML files in resources/knowledge-base/drug-interactions/
        // For now, interactions are referenced by medication.majorInteractions, etc.

        logger.info("Drug interaction database loaded");
    }

    /**
     * Check for interaction between two medications.
     *
     * @param medicationId1 First medication ID
     * @param medicationId2 Second medication ID
     * @return InteractionResult with severity and details, or null if no interaction
     */
    public InteractionResult checkInteraction(String medicationId1, String medicationId2) {
        Medication med1 = medicationLoader.getMedication(medicationId1);
        Medication med2 = medicationLoader.getMedication(medicationId2);

        if (med1 == null || med2 == null) {
            logger.warn("Medication not found: {} or {}", medicationId1, medicationId2);
            return null;
        }

        // Check major interactions
        if (hasInteraction(med1.getMajorInteractions(), medicationId2) ||
            hasInteraction(med2.getMajorInteractions(), medicationId1)) {

            logger.warn("MAJOR interaction detected: {} <-> {}",
                med1.getGenericName(), med2.getGenericName());

            return InteractionResult.builder()
                .medication1Id(medicationId1)
                .medication1Name(med1.getGenericName())
                .medication2Id(medicationId2)
                .medication2Name(med2.getGenericName())
                .severity(InteractionSeverity.MAJOR)
                .clinicalEffect("Life-threatening interaction - immediate intervention required")
                .management("Consider alternative medication or intensive monitoring")
                .build();
        }

        // Check moderate interactions
        if (hasInteraction(med1.getModerateInteractions(), medicationId2) ||
            hasInteraction(med2.getModerateInteractions(), medicationId1)) {

            logger.info("MODERATE interaction detected: {} <-> {}",
                med1.getGenericName(), med2.getGenericName());

            return InteractionResult.builder()
                .medication1Id(medicationId1)
                .medication1Name(med1.getGenericName())
                .medication2Id(medicationId2)
                .medication2Name(med2.getGenericName())
                .severity(InteractionSeverity.MODERATE)
                .clinicalEffect("Clinically significant interaction")
                .management("Monitor patient closely, consider dose adjustment")
                .build();
        }

        // Check minor interactions
        if (hasInteraction(med1.getMinorInteractions(), medicationId2) ||
            hasInteraction(med2.getMinorInteractions(), medicationId1)) {

            logger.debug("MINOR interaction detected: {} <-> {}",
                med1.getGenericName(), med2.getGenericName());

            return InteractionResult.builder()
                .medication1Id(medicationId1)
                .medication1Name(med1.getGenericName())
                .medication2Id(medicationId2)
                .medication2Name(med2.getGenericName())
                .severity(InteractionSeverity.MINOR)
                .clinicalEffect("Limited clinical impact")
                .management("Document interaction, no action typically required")
                .build();
        }

        // No interaction found
        return null;
    }

    /**
     * Check if interaction list contains medication ID.
     */
    private boolean hasInteraction(List<String> interactionList, String medicationId) {
        return interactionList != null && interactionList.contains(medicationId);
    }

    /**
     * Check patient's medication list for all interactions.
     *
     * @param medicationIds List of medication IDs in patient's medication list
     * @return List of all interactions found
     */
    public List<InteractionResult> checkPatientMedications(List<String> medicationIds) {
        if (medicationIds == null || medicationIds.size() < 2) {
            return Collections.emptyList();
        }

        logger.info("Checking interactions for {} medications", medicationIds.size());

        List<InteractionResult> interactions = new ArrayList<>();

        // Check all pairs
        for (int i = 0; i < medicationIds.size(); i++) {
            for (int j = i + 1; j < medicationIds.size(); j++) {
                InteractionResult interaction = checkInteraction(
                    medicationIds.get(i),
                    medicationIds.get(j));

                if (interaction != null) {
                    interactions.add(interaction);
                }
            }
        }

        // Sort by severity (MAJOR first)
        interactions.sort(Comparator.comparing(InteractionResult::getSeverity));

        logger.info("Found {} interactions ({} major, {} moderate, {} minor)",
            interactions.size(),
            interactions.stream().filter(i -> i.getSeverity() == InteractionSeverity.MAJOR).count(),
            interactions.stream().filter(i -> i.getSeverity() == InteractionSeverity.MODERATE).count(),
            interactions.stream().filter(i -> i.getSeverity() == InteractionSeverity.MINOR).count());

        return interactions;
    }

    /**
     * Check if adding new medication creates interactions.
     *
     * @param newMedicationId New medication to add
     * @param currentMedicationIds Current medication list
     * @return List of interactions with new medication
     */
    public List<InteractionResult> checkNewMedication(
            String newMedicationId,
            List<String> currentMedicationIds) {

        if (currentMedicationIds == null || currentMedicationIds.isEmpty()) {
            return Collections.emptyList();
        }

        logger.info("Checking interactions for new medication {} against {} existing medications",
            newMedicationId, currentMedicationIds.size());

        List<InteractionResult> interactions = new ArrayList<>();

        for (String existingMedicationId : currentMedicationIds) {
            InteractionResult interaction = checkInteraction(
                newMedicationId,
                existingMedicationId);

            if (interaction != null) {
                interactions.add(interaction);
            }
        }

        return interactions;
    }

    /**
     * Get count of major interactions in medication list.
     *
     * @param medicationIds Medication list
     * @return Count of major interactions
     */
    public int getMajorInteractionCount(List<String> medicationIds) {
        List<InteractionResult> interactions = checkPatientMedications(medicationIds);
        return (int) interactions.stream()
            .filter(i -> i.getSeverity() == InteractionSeverity.MAJOR)
            .count();
    }

    // ================================================================
    // SUPPORTING CLASSES
    // ================================================================

    /**
     * Drug interaction result.
     */
    public static class InteractionResult {
        private String medication1Id;
        private String medication1Name;
        private String medication2Id;
        private String medication2Name;
        private InteractionSeverity severity;
        private String mechanism;
        private String clinicalEffect;
        private String management;
        private List<String> evidenceReferences;

        public static InteractionResultBuilder builder() {
            return new InteractionResultBuilder();
        }

        // Getters
        public String getMedication1Id() { return medication1Id; }
        public String getMedication1Name() { return medication1Name; }
        public String getMedication2Id() { return medication2Id; }
        public String getMedication2Name() { return medication2Name; }
        public InteractionSeverity getSeverity() { return severity; }
        public String getMechanism() { return mechanism; }
        public String getClinicalEffect() { return clinicalEffect; }
        public String getManagement() { return management; }
        public List<String> getEvidenceReferences() { return evidenceReferences; }

        public static class InteractionResultBuilder {
            private String medication1Id;
            private String medication1Name;
            private String medication2Id;
            private String medication2Name;
            private InteractionSeverity severity;
            private String mechanism;
            private String clinicalEffect;
            private String management;
            private List<String> evidenceReferences;

            public InteractionResultBuilder medication1Id(String medication1Id) {
                this.medication1Id = medication1Id;
                return this;
            }
            public InteractionResultBuilder medication1Name(String medication1Name) {
                this.medication1Name = medication1Name;
                return this;
            }
            public InteractionResultBuilder medication2Id(String medication2Id) {
                this.medication2Id = medication2Id;
                return this;
            }
            public InteractionResultBuilder medication2Name(String medication2Name) {
                this.medication2Name = medication2Name;
                return this;
            }
            public InteractionResultBuilder severity(InteractionSeverity severity) {
                this.severity = severity;
                return this;
            }
            public InteractionResultBuilder mechanism(String mechanism) {
                this.mechanism = mechanism;
                return this;
            }
            public InteractionResultBuilder clinicalEffect(String clinicalEffect) {
                this.clinicalEffect = clinicalEffect;
                return this;
            }
            public InteractionResultBuilder management(String management) {
                this.management = management;
                return this;
            }
            public InteractionResultBuilder evidenceReferences(List<String> evidenceReferences) {
                this.evidenceReferences = evidenceReferences;
                return this;
            }

            public InteractionResult build() {
                InteractionResult result = new InteractionResult();
                result.medication1Id = this.medication1Id;
                result.medication1Name = this.medication1Name;
                result.medication2Id = this.medication2Id;
                result.medication2Name = this.medication2Name;
                result.severity = this.severity;
                result.mechanism = this.mechanism;
                result.clinicalEffect = this.clinicalEffect;
                result.management = this.management;
                result.evidenceReferences = this.evidenceReferences;
                return result;
            }
        }
    }

    /**
     * Drug interaction severity levels.
     */
    public enum InteractionSeverity {
        MAJOR,      // Life-threatening, contraindicated
        MODERATE,   // Clinically significant
        MINOR       // Limited impact
    }

    /**
     * Drug interaction record (for loading from YAML).
     */
    public static class DrugInteraction {
        private String interactionId;
        private String medication1Id;
        private String medication2Id;
        private InteractionSeverity severity;
        private String mechanism;
        private String clinicalEffect;
        private String management;
        private List<String> evidenceReferences;

        // Getters and setters
        public String getInteractionId() { return interactionId; }
        public void setInteractionId(String interactionId) { this.interactionId = interactionId; }

        public String getMedication1Id() { return medication1Id; }
        public void setMedication1Id(String medication1Id) { this.medication1Id = medication1Id; }

        public String getMedication2Id() { return medication2Id; }
        public void setMedication2Id(String medication2Id) { this.medication2Id = medication2Id; }

        public InteractionSeverity getSeverity() { return severity; }
        public void setSeverity(InteractionSeverity severity) { this.severity = severity; }

        public String getMechanism() { return mechanism; }
        public void setMechanism(String mechanism) { this.mechanism = mechanism; }

        public String getClinicalEffect() { return clinicalEffect; }
        public void setClinicalEffect(String clinicalEffect) { this.clinicalEffect = clinicalEffect; }

        public String getManagement() { return management; }
        public void setManagement(String management) { this.management = management; }

        public List<String> getEvidenceReferences() { return evidenceReferences; }
        public void setEvidenceReferences(List<String> evidenceReferences) {
            this.evidenceReferences = evidenceReferences;
        }
    }

    // ═══════════════════════════════════════════════════════════════════════
    // TEST COMPATIBILITY METHODS
    // ═══════════════════════════════════════════════════════════════════════

    /**
     * Check interactions between two medications (test-compatible signature).
     * @param med1 First medication
     * @param med2 Second medication
     * @return List of interactions found
     */
    public List<Interaction> checkInteraction(Medication med1, Medication med2) {
        List<Interaction> interactions = new ArrayList<>();

        if (med1 == null || med2 == null) {
            return interactions;
        }

        InteractionResult result = checkInteraction(med1.getMedicationId(), med2.getMedicationId());

        if (result != null) {
            Interaction interaction = Interaction.builder()
                .medicationId1(med1.getMedicationId())
                .medicationId2(med2.getMedicationId())
                .medicationName1(med1.getName())
                .medicationName2(med2.getName())
                .severity(result.getSeverity())
                .description(result.getClinicalEffect())
                .mechanism(result.getMechanism())
                .management(result.getManagement())
                .requiresMonitoring(result.getSeverity() == InteractionSeverity.MAJOR)
                .build();
            interactions.add(interaction);
        }

        return interactions;
    }

    /**
     * Check interactions for a medication against patient's current medications.
     * @param newMedication Medication being added
     * @param patient Patient context with current medications
     * @return List of interactions found
     */
    public List<Interaction> checkInteractions(Medication newMedication, PatientContext patient) {
        List<Interaction> interactions = new ArrayList<>();

        if (newMedication == null || patient == null || patient.getCurrentMedications() == null) {
            return interactions;
        }

        // Check against each current medication
        for (String currentMed : patient.getCurrentMedications().keySet()) {
            InteractionResult result = checkInteraction(newMedication.getMedicationId(), currentMed);

            if (result != null) {
                Interaction interaction = Interaction.builder()
                    .medicationId1(newMedication.getMedicationId())
                    .medicationId2(currentMed)
                    .medicationName1(newMedication.getName())
                    .medicationName2(currentMed)
                    .severity(result.getSeverity())
                    .description(result.getClinicalEffect())
                    .mechanism(result.getMechanism())
                    .management(result.getManagement())
                    .requiresMonitoring(result.getSeverity() == InteractionSeverity.MAJOR)
                    .contraindicated(result.getSeverity() == InteractionSeverity.MAJOR)
                    .build();
                interactions.add(interaction);
            }
        }

        return interactions;
    }
}
