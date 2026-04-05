package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module13TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

class Module13DataAbsenceTimerTest {

    // --- Test 1: 7 days no data → DATA_ABSENCE_WARNING detected ---
    @Test
    void sevenDaysNoData_warningDetected() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        long now = Module13TestBuilder.BASE_TIME + 8 * Module13TestBuilder.DAY_MS;

        Module13DataCompletenessMonitor.Result result =
                Module13DataCompletenessMonitor.evaluate(state, now);

        assertFalse(result.getDataGapFlags().isEmpty(),
                "Should have data gap flags after 8 days");
    }

    // --- Test 2: 14 days no data → DATA_ABSENCE_CRITICAL ---
    @Test
    void fourteenDaysNoData_criticalDetected() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        long now = Module13TestBuilder.BASE_TIME + 20 * Module13TestBuilder.DAY_MS;

        Module13DataCompletenessMonitor.Result result =
                Module13DataCompletenessMonitor.evaluate(state, now);

        assertTrue(result.isDataAbsenceCritical());
    }

    // --- Test 3: Partial data (some modules active) → WARNING not CRITICAL ---
    @Test
    void partialData_warningNotCritical() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        long now = Module13TestBuilder.BASE_TIME + 10 * Module13TestBuilder.DAY_MS;
        state.recordModuleSeen("module7", now - Module13TestBuilder.DAY_MS);
        state.recordModuleSeen("enriched", now - Module13TestBuilder.DAY_MS);

        Module13DataCompletenessMonitor.Result result =
                Module13DataCompletenessMonitor.evaluate(state, now);

        assertFalse(result.isDataAbsenceCritical(),
                "Should not be CRITICAL when some modules are active");
        assertFalse(result.getDataGapFlags().isEmpty(),
                "Should still flag stale modules");
    }

    // --- Test 4: Fresh data arrival resets gap flags ---
    @Test
    void freshDataArrival_resetsGapFlags() {
        ClinicalStateSummary state = Module13TestBuilder.stateWithBaselines("p1");
        long now = Module13TestBuilder.BASE_TIME + 2 * Module13TestBuilder.DAY_MS;
        for (String module : new String[]{"module7", "module9", "module10b", "module11b", "enriched"}) {
            state.recordModuleSeen(module, now - Module13TestBuilder.HOUR_MS);
        }

        Module13DataCompletenessMonitor.Result result =
                Module13DataCompletenessMonitor.evaluate(state, now);

        assertFalse(result.isDataAbsenceCritical());
        assertTrue(result.getCompositeScore() > 0.6);
    }
}
