package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module6DeduplicationTest {

    private Module6CrossModuleDedup dedup;

    @BeforeEach
    void setUp() {
        dedup = new Module6CrossModuleDedup();
    }

    @Test
    void firstAlert_alwaysEmitted() {
        long now = System.currentTimeMillis();
        assertTrue(dedup.shouldEmit("P001", ActionTier.HALT, "SEPSIS", now));
    }

    @Test
    void duplicateWithinHaltWindow_suppressed() {
        long now = System.currentTimeMillis();
        dedup.shouldEmit("P001", ActionTier.HALT, "SEPSIS", now);
        assertFalse(dedup.shouldEmit("P001", ActionTier.HALT, "SEPSIS", now + 60_000),
            "Same HALT within 5 min should be suppressed");
    }

    @Test
    void duplicateAfterHaltWindow_emitted() {
        long now = System.currentTimeMillis();
        dedup.shouldEmit("P001", ActionTier.HALT, "SEPSIS", now);
        assertTrue(dedup.shouldEmit("P001", ActionTier.HALT, "SEPSIS", now + 6 * 60_000),
            "Same HALT after 5 min window should emit");
    }

    @Test
    void duplicateWithinPauseWindow_suppressed() {
        long now = System.currentTimeMillis();
        dedup.shouldEmit("P001", ActionTier.PAUSE, "AKI", now);
        assertFalse(dedup.shouldEmit("P001", ActionTier.PAUSE, "AKI", now + 10 * 60_000),
            "Same PAUSE within 30 min should be suppressed");
    }

    @Test
    void differentClinicalCategory_notSuppressed() {
        long now = System.currentTimeMillis();
        dedup.shouldEmit("P001", ActionTier.HALT, "SEPSIS", now);
        assertTrue(dedup.shouldEmit("P001", ActionTier.HALT, "HYPERKALEMIA", now + 1000),
            "Different clinical category should not be suppressed");
    }

    @Test
    void differentPatient_notSuppressed() {
        long now = System.currentTimeMillis();
        dedup.shouldEmit("P001", ActionTier.HALT, "SEPSIS", now);
        assertTrue(dedup.shouldEmit("P002", ActionTier.HALT, "SEPSIS", now + 1000),
            "Different patient should not be suppressed");
    }

    @Test
    void softFlagWindow_60minutes() {
        long now = System.currentTimeMillis();
        dedup.shouldEmit("P001", ActionTier.SOFT_FLAG, "TREND", now);
        assertFalse(dedup.shouldEmit("P001", ActionTier.SOFT_FLAG, "TREND", now + 30 * 60_000),
            "Same SOFT_FLAG within 60 min should be suppressed");
        assertTrue(dedup.shouldEmit("P001", ActionTier.SOFT_FLAG, "TREND", now + 61 * 60_000),
            "Same SOFT_FLAG after 60 min should emit");
    }

    @Test
    void routineEvents_neverSuppressed() {
        long now = System.currentTimeMillis();
        dedup.shouldEmit("P001", ActionTier.ROUTINE, "CDS_GENERAL", now);
        assertTrue(dedup.shouldEmit("P001", ActionTier.ROUTINE, "CDS_GENERAL", now + 1000),
            "ROUTINE events should never be suppressed");
    }
}
