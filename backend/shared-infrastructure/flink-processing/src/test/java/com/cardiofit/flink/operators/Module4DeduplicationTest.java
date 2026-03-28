package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module4TestBuilder;
import com.cardiofit.flink.functions.PatternDeduplicationFunction;
import com.cardiofit.flink.models.PatternEvent;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for PatternDeduplicationFunction improvements:
 * 1. Severity escalation passthrough (HIGH→CRITICAL emitted, not merged)
 * 2. Same-severity within window suppressed
 * 3. Different pattern types never suppressed
 */
public class Module4DeduplicationTest {

    @Test
    void severityEscalation_shouldBeDetectable() {
        assertTrue(PatternDeduplicationFunction.isSeverityEscalation("HIGH", "CRITICAL"),
            "HIGH→CRITICAL is an escalation");
        assertTrue(PatternDeduplicationFunction.isSeverityEscalation("MODERATE", "HIGH"),
            "MODERATE→HIGH is an escalation");
        assertFalse(PatternDeduplicationFunction.isSeverityEscalation("CRITICAL", "CRITICAL"),
            "Same severity is NOT an escalation");
        assertFalse(PatternDeduplicationFunction.isSeverityEscalation("HIGH", "MODERATE"),
            "HIGH→MODERATE is a de-escalation, NOT an escalation");
        assertFalse(PatternDeduplicationFunction.isSeverityEscalation("HIGH", "HIGH"),
            "Same severity is NOT an escalation");
    }

    @Test
    void severityIndex_ordersCorrectly() {
        assertTrue(PatternDeduplicationFunction.severityIndex("CRITICAL") >
                   PatternDeduplicationFunction.severityIndex("HIGH"));
        assertTrue(PatternDeduplicationFunction.severityIndex("HIGH") >
                   PatternDeduplicationFunction.severityIndex("MODERATE"));
        assertTrue(PatternDeduplicationFunction.severityIndex("MODERATE") >
                   PatternDeduplicationFunction.severityIndex("LOW"));
        assertEquals(0, PatternDeduplicationFunction.severityIndex("UNKNOWN"));
    }

    @Test
    void patternKey_includesTypeOnly_notSeverity() {
        PatternEvent highPattern = Module4TestBuilder.deteriorationPattern("P1", "HIGH", 0.85);
        PatternEvent criticalPattern = Module4TestBuilder.deteriorationPattern("P1", "CRITICAL", 0.95);

        String key1 = PatternDeduplicationFunction.computePatternKey(highPattern);
        String key2 = PatternDeduplicationFunction.computePatternKey(criticalPattern);

        assertEquals(key1, key2,
            "Same pattern type should have same dedup key regardless of severity");
        assertEquals("CLINICAL_DETERIORATION", key1);
    }

    @Test
    void escalationScenario_fullSeverityLadder() {
        // Verify the complete escalation ladder: LOW → MODERATE → HIGH → CRITICAL
        // Each step should be detected as escalation; reverse should not
        String[] levels = {"LOW", "MODERATE", "HIGH", "CRITICAL"};

        for (int i = 0; i < levels.length; i++) {
            for (int j = 0; j < levels.length; j++) {
                boolean expected = j > i; // escalation only if new > old
                assertEquals(expected,
                    PatternDeduplicationFunction.isSeverityEscalation(levels[i], levels[j]),
                    levels[i] + " → " + levels[j] + " should " + (expected ? "" : "NOT ") + "be escalation");
            }
        }
    }

    @Test
    void patternKey_differentPatientsSameType_sameKey() {
        // Dedup key is type-only, not patient-specific (patient keying happens at Flink level)
        PatternEvent p1 = Module4TestBuilder.deteriorationPattern("PAT-A", "HIGH", 0.8);
        PatternEvent p2 = Module4TestBuilder.deteriorationPattern("PAT-B", "HIGH", 0.8);

        assertEquals(
            PatternDeduplicationFunction.computePatternKey(p1),
            PatternDeduplicationFunction.computePatternKey(p2),
            "Pattern key should be type-only, patient keying is done by Flink");
    }

    @Test
    void severityIndex_unknownValues_returnZero() {
        assertEquals(0, PatternDeduplicationFunction.severityIndex(null));
        assertEquals(0, PatternDeduplicationFunction.severityIndex("UNKNOWN"));
        assertEquals(0, PatternDeduplicationFunction.severityIndex(""));
        assertEquals(0, PatternDeduplicationFunction.severityIndex("invalid"));
    }
}
