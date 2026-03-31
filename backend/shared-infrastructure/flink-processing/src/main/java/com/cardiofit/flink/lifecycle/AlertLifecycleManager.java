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

    public static ClinicalAlert createAlert(String patientId, ActionTier tier,
                                             String clinicalCategory, String sourceModule) {
        ClinicalAlert alert = new ClinicalAlert();
        alert.setAlertId(UUID.randomUUID().toString());
        alert.setPatientId(patientId);
        alert.setTier(tier);
        alert.setClinicalCategory(clinicalCategory);
        alert.setSourceModule(sourceModule);
        alert.setState(AlertState.ACTIVE);
        alert.setCreatedAt(System.currentTimeMillis());

        // SLA deadline
        long sla = tier.getSlaMs();
        alert.setSlaDeadlineMs(sla > 0 ? alert.getCreatedAt() + sla : -1L);

        return alert;
    }

    /**
     * Check alert fatigue: cap at 10 alerts per 24-hour window per patient.
     * @return true if fatigued (should suppress), false if OK to emit
     */
    public static boolean checkAlertFatigue(PatientAlertState state) {
        long now = System.currentTimeMillis();
        if (now - state.getAlertWindowStart() > 24 * 60 * 60 * 1000L) {
            state.setAlertsInLast24Hours(0);
            state.setAlertWindowStart(now);
        }
        if (state.getAlertsInLast24Hours() >= MAX_ALERTS_PER_24H) {
            LOG.warn("Alert fatigue threshold for patient {} — {} alerts in 24h. Suppressing.",
                state.getPatientId(), state.getAlertsInLast24Hours());
            return true;
        }
        state.setAlertsInLast24Hours(state.getAlertsInLast24Hours() + 1);
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
