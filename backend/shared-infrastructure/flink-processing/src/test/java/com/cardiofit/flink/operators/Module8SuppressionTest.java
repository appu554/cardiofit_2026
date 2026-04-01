package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module8SuppressionTest {

    @Test
    void firstAlert_neverSuppressed() {
        ComorbidityState state = new ComorbidityState("P-SUP");
        CIDAlert alert = CIDAlert.create(CIDRuleId.CID_06, "P-SUP",
            "test", List.of("drug-a"), "action", null);
        boolean suppressed = Module8SuppressionManager.shouldSuppress(alert, state, System.currentTimeMillis());
        assertFalse(suppressed, "First alert for a rule should never be suppressed");
    }

    @Test
    void duplicateWithin72Hours_suppressed() {
        ComorbidityState state = new ComorbidityState("P-SUP");
        CIDAlert alert = CIDAlert.create(CIDRuleId.CID_06, "P-SUP",
            "test", List.of("drug-a"), "action", null);

        long now = System.currentTimeMillis();
        // Record first emission
        state.recordSuppression(alert.getSuppressionKey(), now);

        // 24 hours later, same alert
        boolean suppressed = Module8SuppressionManager.shouldSuppress(
            alert, state, now + 24 * 60 * 60 * 1000L);
        assertTrue(suppressed, "Same alert within 72h should be suppressed");
    }

    @Test
    void duplicateAfter72Hours_notSuppressed() {
        ComorbidityState state = new ComorbidityState("P-SUP");
        CIDAlert alert = CIDAlert.create(CIDRuleId.CID_06, "P-SUP",
            "test", List.of("drug-a"), "action", null);

        long now = System.currentTimeMillis();
        state.recordSuppression(alert.getSuppressionKey(), now);

        // 73 hours later
        boolean suppressed = Module8SuppressionManager.shouldSuppress(
            alert, state, now + 73 * 60 * 60 * 1000L);
        assertFalse(suppressed, "Same alert after 72h should NOT be suppressed");
    }

    @Test
    void haltAlerts_neverSuppressed() {
        ComorbidityState state = new ComorbidityState("P-HALT");
        CIDAlert alert = CIDAlert.create(CIDRuleId.CID_01, "P-HALT",
            "test", List.of("drug-a"), "action", null);

        long now = System.currentTimeMillis();
        state.recordSuppression(alert.getSuppressionKey(), now);

        // Same HALT alert 1 hour later
        boolean suppressed = Module8SuppressionManager.shouldSuppress(
            alert, state, now + 60 * 60 * 1000L);
        assertFalse(suppressed, "HALT alerts should NEVER be suppressed");
    }

    @Test
    void differentMedications_notSuppressed() {
        ComorbidityState state = new ComorbidityState("P-DIFF");
        CIDAlert alert1 = CIDAlert.create(CIDRuleId.CID_15, "P-DIFF",
            "test", List.of("empagliflozin", "ibuprofen"), "action", null);
        CIDAlert alert2 = CIDAlert.create(CIDRuleId.CID_15, "P-DIFF",
            "test", List.of("empagliflozin", "naproxen"), "action", null);

        long now = System.currentTimeMillis();
        state.recordSuppression(alert1.getSuppressionKey(), now);

        // Same rule but different medication combo
        boolean suppressed = Module8SuppressionManager.shouldSuppress(
            alert2, state, now + 1000L);
        assertFalse(suppressed,
            "Same rule but different medication combination should NOT be suppressed");
    }
}
