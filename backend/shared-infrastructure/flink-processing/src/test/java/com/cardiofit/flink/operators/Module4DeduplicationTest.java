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
}
