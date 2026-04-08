package com.cardiofit.flink.lifecycle;

import com.cardiofit.flink.models.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.UUID;

/**
 * Alert lifecycle operations: creation, SLA management, escalation, fatigue protection.
 * Stateless utility — state is managed by the Flink operator.
 */
public final class AlertLifecycleManager {
    private static final Logger LOG = LoggerFactory.getLogger(AlertLifecycleManager.class);
    private static final int MAX_ALERTS_PER_24H = 10;

    private AlertLifecycleManager() {}

    /**
     * Create alert using event time for consistency (not wall clock).
     */
    public static ClinicalAlert createAlert(String patientId, ActionTier tier,
                                             String clinicalCategory, String sourceModule,
                                             long eventTime) {
        ClinicalAlert alert = new ClinicalAlert();
        alert.setAlertId(UUID.randomUUID().toString());
        alert.setPatientId(patientId);
        alert.setTier(tier);
        alert.setClinicalCategory(clinicalCategory);
        alert.setSourceModule(sourceModule);
        alert.setState(AlertState.ACTIVE);
        alert.setCreatedAt(eventTime);

        // SLA deadline based on event time
        long sla = tier.getSlaMs();
        alert.setSlaDeadlineMs(sla > 0 ? eventTime + sla : -1L);

        return alert;
    }

    /** Backward-compatible overload using wall clock (for tests/legacy callers). */
    public static ClinicalAlert createAlert(String patientId, ActionTier tier,
                                             String clinicalCategory, String sourceModule) {
        return createAlert(patientId, tier, clinicalCategory, sourceModule, System.currentTimeMillis());
    }

    /**
     * Check alert fatigue: cap at 10 alerts per 24-hour window per patient.
     * HALT-tier alerts are NEVER suppressed — patient safety overrides fatigue protection.
     * The counter still increments for monitoring/reporting purposes.
     *
     * @param state  per-patient alert state
     * @param tier   the action tier of the incoming alert
     * @return true if fatigued (should suppress), false if OK to emit
     */
    public static boolean checkAlertFatigue(PatientAlertState state, ActionTier tier) {
        long now = System.currentTimeMillis();
        if (now - state.getAlertWindowStart() > 24 * 60 * 60 * 1000L) {
            state.setAlertsInLast24Hours(0);
            state.setAlertWindowStart(now);
        }
        state.setAlertsInLast24Hours(state.getAlertsInLast24Hours() + 1);

        // HALT alerts ALWAYS pass — immediate patient safety cannot be suppressed
        if (tier == ActionTier.HALT) {
            return false;
        }

        if (state.getAlertsInLast24Hours() > MAX_ALERTS_PER_24H) {
            LOG.warn("Alert fatigue threshold for patient {} — {} alerts in 24h. Suppressing {} alert.",
                state.getPatientId(), state.getAlertsInLast24Hours(), tier);
            return true;
        }
        return false;
    }

    public static void escalate(ClinicalAlert alert) {
        alert.setState(AlertState.ESCALATED);
        alert.setEscalatedAt(System.currentTimeMillis());
        alert.setEscalationLevel(alert.getEscalationLevel() + 1);

        String escalateTo = switch (alert.getEscalationLevel()) {
            case 1 -> "CARE_COORDINATOR";
            case 2 -> "CLINICAL_SUPERVISOR";
            default -> "DEPARTMENT_HEAD";
        };
        alert.setAssignedTo(escalateTo);
    }
}
