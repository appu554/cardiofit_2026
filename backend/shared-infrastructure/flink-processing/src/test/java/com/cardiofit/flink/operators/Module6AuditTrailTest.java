package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module6TestBuilder;
import com.cardiofit.flink.lifecycle.AlertLifecycleManager;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module6AuditTrailTest {

    @Test
    void alertCreated_auditHasAllRequiredFields() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "SEPSIS", "MODULE_3_CDS");
        ClinicalEvent event = Module6TestBuilder.sepsisHaltEvent("P001");

        AuditRecord audit = AuditRecord.alertCreated(alert, event);

        assertNotNull(audit.getAuditId(), "auditId required");
        assertEquals("ALERT_CREATED", audit.getEventType());
        assertEquals("P001", audit.getPatientId());
        assertEquals(ActionTier.HALT, audit.getTier());
        assertEquals("SEPSIS", audit.getClinicalCategory());
        assertEquals(alert.getAlertId(), audit.getAlertId());
        assertTrue(audit.getTimestamp() > 0, "timestamp required");
    }

    @Test
    void auditRecord_containsSourceModule() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.PAUSE, "AKI", "MODULE_5_ML");
        ClinicalEvent event = Module6TestBuilder.deteriorationPrediction("P001", 0.50);

        AuditRecord audit = AuditRecord.alertCreated(alert, event);

        assertEquals("MODULE_5_ML", audit.getSourceModule());
    }

    @Test
    void auditRecord_provenance_isNotNull() {
        ClinicalAlert alert = AlertLifecycleManager.createAlert(
            "P001", ActionTier.HALT, "HYPERKALEMIA", "MODULE_3_CDS");
        ClinicalEvent event = Module6TestBuilder.hyperkalemiaHaltEvent("P001");

        AuditRecord audit = AuditRecord.alertCreated(alert, event);

        assertNotNull(audit.getClinicalData(), "clinicalData map must not be null");
        assertNotNull(audit.getInputSnapshot(), "inputSnapshot must not be null");
    }
}
