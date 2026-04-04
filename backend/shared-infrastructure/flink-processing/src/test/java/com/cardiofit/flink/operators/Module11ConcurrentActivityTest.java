package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module11TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.HashMap;
import static org.junit.jupiter.api.Assertions.*;

class Module11ConcurrentActivityTest {

    private static final long BASE = Module11TestBuilder.BASE_TIME;
    private static final long MIN = 60_000L;

    @Test
    void twoActivitiesWithin30min_bothFlagged() {
        ActivityCorrelationState state = Module11TestBuilder.emptyState("P1");
        state.openSession("act-1", BASE, 30 * MIN, new HashMap<>());
        state.openSession("act-2", BASE + 20 * MIN, 30 * MIN, new HashMap<>());

        assertTrue(state.getActiveSessions().get("act-1").concurrent);
        assertTrue(state.getActiveSessions().get("act-2").concurrent);
    }

    @Test
    void twoActivitiesOver30minApart_notFlagged() {
        ActivityCorrelationState state = Module11TestBuilder.emptyState("P1");
        state.openSession("act-1", BASE, 30 * MIN, new HashMap<>());
        state.openSession("act-2", BASE + 45 * MIN, 30 * MIN, new HashMap<>());

        assertFalse(state.getActiveSessions().get("act-1").concurrent);
        assertFalse(state.getActiveSessions().get("act-2").concurrent);
    }

    @Test
    void hrReadings_feedBothConcurrentSessions() {
        ActivityCorrelationState state = Module11TestBuilder.emptyState("P1");
        state.openSession("act-1", BASE, 60 * MIN, new HashMap<>());
        state.openSession("act-2", BASE + 20 * MIN, 60 * MIN, new HashMap<>());

        // HR reading at 30 min — within both windows
        state.addHRReading(BASE + 30 * MIN, 155.0, "WEARABLE");

        assertEquals(1, state.getActiveSessions().get("act-1").hrWindow.size());
        assertEquals(1, state.getActiveSessions().get("act-2").hrWindow.size());
    }
}
