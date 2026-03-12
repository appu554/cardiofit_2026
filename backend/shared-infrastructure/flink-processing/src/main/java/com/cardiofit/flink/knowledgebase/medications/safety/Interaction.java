package com.cardiofit.flink.knowledgebase.medications.safety;

import com.cardiofit.flink.knowledgebase.medications.safety.DrugInteractionChecker.InteractionSeverity;
import lombok.Builder;
import lombok.Data;
import java.io.Serializable;
import java.util.List;

/**
 * Drug Interaction Result
 *
 * Represents a detected drug-drug interaction with severity, mechanism,
 * clinical significance, and management recommendations.
 *
 * @author CardioFit Module 3 Team
 * @version 1.0
 * @since 2025-10-25
 */
@Data
@Builder
public class Interaction implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * Medication IDs involved in the interaction
     */
    private String medicationId1;
    private String medicationId2;

    /**
     * Medication names for display
     */
    private String medicationName1;
    private String medicationName2;

    /**
     * Interaction severity (MAJOR, MODERATE, MINOR)
     */
    private InteractionSeverity severity;

    /**
     * Description of the interaction and clinical significance
     */
    private String description;

    /**
     * Mechanism of the interaction (pharmacokinetic, pharmacodynamic, etc.)
     */
    private String mechanism;

    /**
     * Clinical management recommendations
     */
    private String management;

    /**
     * Additional clinical considerations
     */
    private String clinicalConsiderations;

    /**
     * Evidence level supporting this interaction (WELL_ESTABLISHED, THEORETICAL, etc.)
     */
    private String evidenceLevel;

    /**
     * Whether this interaction is contraindicated (absolute avoidance required)
     */
    @Builder.Default
    private boolean contraindicated = false;

    /**
     * Onset timing (IMMEDIATE, DELAYED, GRADUAL)
     */
    private String onset;

    /**
     * Documentation quality (EXCELLENT, GOOD, FAIR, POOR)
     */
    private String documentationQuality;

    /**
     * Supporting citations (PMIDs)
     */
    private List<String> citationPmids;

    /**
     * Alert priority (1 = highest, 5 = lowest)
     */
    @Builder.Default
    private Integer alertPriority = 3;

    /**
     * Whether monitoring is required
     */
    @Builder.Default
    private boolean requiresMonitoring = false;

    /**
     * Specific monitoring parameters (e.g., "INR", "Creatinine", "BP")
     */
    private List<String> monitoringParameters;

    /**
     * Get formatted interaction summary for clinical display
     */
    public String getSummary() {
        return String.format("%s ⇄ %s: %s (%s severity)",
            medicationName1 != null ? medicationName1 : medicationId1,
            medicationName2 != null ? medicationName2 : medicationId2,
            description != null ? description : "Drug interaction detected",
            severity != null ? severity : "UNKNOWN");
    }

    /**
     * Check if this is a high-priority interaction requiring immediate attention
     */
    public boolean isHighPriority() {
        return severity == InteractionSeverity.MAJOR || contraindicated ||
               (alertPriority != null && alertPriority <= 2);
    }

    /**
     * Get detailed interaction report for documentation
     */
    public String getDetailedReport() {
        StringBuilder report = new StringBuilder();
        report.append("═══ DRUG INTERACTION ALERT ═══\n");
        report.append("Medications: ").append(medicationName1).append(" ⇄ ").append(medicationName2).append("\n");
        report.append("Severity: ").append(severity).append("\n");

        if (contraindicated) {
            report.append("⚠️ CONTRAINDICATED - Avoid combination\n");
        }

        report.append("\nDescription:\n").append(description).append("\n");

        if (mechanism != null) {
            report.append("\nMechanism:\n").append(mechanism).append("\n");
        }

        if (management != null) {
            report.append("\nManagement:\n").append(management).append("\n");
        }

        if (requiresMonitoring && monitoringParameters != null && !monitoringParameters.isEmpty()) {
            report.append("\nMonitoring Required:\n");
            for (String param : monitoringParameters) {
                report.append("  • ").append(param).append("\n");
            }
        }

        if (evidenceLevel != null) {
            report.append("\nEvidence Level: ").append(evidenceLevel).append("\n");
        }

        return report.toString();
    }

    @Override
    public String toString() {
        return getSummary();
    }
}
