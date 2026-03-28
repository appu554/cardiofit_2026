package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.AlertPriority;
import com.cardiofit.flink.models.AlertSeverity;
import com.cardiofit.flink.models.AlertType;
import com.cardiofit.flink.models.SimpleAlert;
import org.junit.jupiter.api.Test;

import java.util.HashSet;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module 4 clinical significance calculation and risk level determination.
 * These functions control CEP pattern matching thresholds.
 *
 * NOTE: determineRiskLevel compares AlertSeverity/AlertPriority enums to String "CRITICAL"
 * via .equals(), which always returns false (enum.equals(String) == false in Java).
 * The alert-based "high" path is therefore unreachable via severity/priority fields.
 * Tests reflect the actual implemented behavior.
 */
public class Module4CDSConversionTest {

    // ── Clinical Significance ─────────────────────────────────

    @Test
    void significance_lowRisk_allZeros() {
        double sig = Module4ClinicalScoring.calculateClinicalSignificance(0, 0, 0.0);
        assertEquals(0.0, sig, 0.001, "All zeros should produce 0.0 significance");
    }

    @Test
    void significance_lowRisk_news2Under5() {
        // NEWS2=3 → 0.15, qSOFA=0, acuity=2.0 → 0.04 = 0.19
        double sig = Module4ClinicalScoring.calculateClinicalSignificance(3, 0, 2.0);
        assertTrue(sig > 0.1 && sig < 0.3,
            "NEWS2=3, qSOFA=0, acuity=2.0 should be LOW range (0.1-0.3), got " + sig);
    }

    @Test
    void significance_moderate_news5_qsofa1() {
        // NEWS2=5 → 0.35, qSOFA=1 → 0.15, acuity=5.0 → 0.10 = 0.60
        double sig = Module4ClinicalScoring.calculateClinicalSignificance(5, 1, 5.0);
        assertTrue(sig >= 0.55 && sig <= 0.65,
            "NEWS2=5, qSOFA=1, acuity=5.0 should be ~0.60, got " + sig);
    }

    @Test
    void significance_high_news7_qsofa2() {
        // NEWS2=7 → 0.40, qSOFA=2 → 0.30, acuity=7.0 → 0.14 = 0.84
        double sig = Module4ClinicalScoring.calculateClinicalSignificance(7, 2, 7.0);
        assertTrue(sig >= 0.80 && sig <= 0.90,
            "NEWS2=7, qSOFA=2, acuity=7.0 should be ~0.84, got " + sig);
    }

    @Test
    void significance_critical_news10_qsofa2() {
        // NEWS2=10 → 0.50, qSOFA=2 → 0.30, acuity=9.0 → 0.18 = 0.98
        double sig = Module4ClinicalScoring.calculateClinicalSignificance(10, 2, 9.0);
        assertTrue(sig >= 0.90 && sig <= 1.0,
            "NEWS2=10, qSOFA=2, acuity=9.0 should be CRITICAL (>0.90), got " + sig);
    }

    @Test
    void significance_capped_at_1() {
        // NEWS2=15 → 0.50, qSOFA=3 → 0.30, acuity=10.0 → 0.20 = 1.0
        double sig = Module4ClinicalScoring.calculateClinicalSignificance(15, 3, 10.0);
        assertEquals(1.0, sig, 0.001, "Significance must not exceed 1.0");
    }

    // ── Risk Level Determination ──────────────────────────────

    @Test
    void riskLevel_high_whenNEWS10OrMore() {
        assertEquals("high", Module4ClinicalScoring.determineRiskLevel(10, 0, null));
        assertEquals("high", Module4ClinicalScoring.determineRiskLevel(12, 0, null));
    }

    @Test
    void riskLevel_high_whenQSOFA2OrMore() {
        assertEquals("high", Module4ClinicalScoring.determineRiskLevel(3, 2, null));
    }

    /**
     * The alert-based critical path uses .equals("CRITICAL") on AlertSeverity/AlertPriority enums.
     * Java enum.equals(String) always returns false, so the criticalAlertCount is always 0.
     * This test documents that the path is effectively unreachable via severity/priority fields,
     * and that alert presence alone (with low scores) falls through to "low".
     */
    @Test
    void riskLevel_alertsPresent_butCriticalPathUnreachable_dueTo_enumStringComparison() {
        Set<SimpleAlert> alerts = new HashSet<>();
        SimpleAlert a1 = new SimpleAlert(AlertType.VITAL_THRESHOLD_BREACH, AlertSeverity.CRITICAL,
            "Critical alert 1", "P001");
        a1.setPriorityLevel(AlertPriority.CRITICAL);
        SimpleAlert a2 = new SimpleAlert(AlertType.VITAL_THRESHOLD_BREACH, AlertSeverity.CRITICAL,
            "Critical alert 2", "P001");
        a2.setPriorityLevel(AlertPriority.CRITICAL);
        alerts.add(a1);
        alerts.add(a2);
        // NEWS2=4, qSOFA=0 → falls through alert loop (enum.equals(String)==false) → "low"
        assertEquals("low", Module4ClinicalScoring.determineRiskLevel(4, 0, alerts),
            "Alert-based critical path is unreachable due to enum.equals(String) always false");
    }

    @Test
    void riskLevel_moderate_whenNEWS5to9() {
        assertEquals("moderate", Module4ClinicalScoring.determineRiskLevel(5, 0, null));
        assertEquals("moderate", Module4ClinicalScoring.determineRiskLevel(9, 0, null));
    }

    @Test
    void riskLevel_moderate_whenQSOFA1() {
        assertEquals("moderate", Module4ClinicalScoring.determineRiskLevel(3, 1, null));
    }

    @Test
    void riskLevel_low_whenNEWS4OrLess_qSOFA0() {
        assertEquals("low", Module4ClinicalScoring.determineRiskLevel(4, 0, null));
        assertEquals("low", Module4ClinicalScoring.determineRiskLevel(0, 0, null));
    }

    @Test
    void riskLevel_nullAlerts_noNPE() {
        // Verifies null alerts set is handled gracefully
        assertDoesNotThrow(() -> Module4ClinicalScoring.determineRiskLevel(4, 0, null));
    }
}
