package com.cardiofit.flink.operators;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.time.Instant;
import java.time.ZoneOffset;
import java.time.ZonedDateTime;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("Module 9: Daily Timer Registration")
class Module9DailyTimerTest {

    @Test
    @DisplayName("10:00 UTC -> next tick is today 23:59 UTC")
    void morningSchedulesToday() {
        // 2025-04-02 10:00:00 UTC
        ZonedDateTime tenAm = ZonedDateTime.of(2025, 4, 2, 10, 0, 0, 0, ZoneOffset.UTC);
        long result = Module9_EngagementMonitor.computeNextDailyTick(tenAm.toInstant().toEpochMilli());

        ZonedDateTime expected = ZonedDateTime.of(2025, 4, 2, 23, 59, 0, 0, ZoneOffset.UTC);
        assertEquals(expected.toInstant().toEpochMilli(), result);
    }

    @Test
    @DisplayName("23:59 UTC -> next tick is tomorrow 23:59 UTC")
    void atTickTimeSchedulesTomorrow() {
        // 2025-04-02 23:59:00 UTC (exactly at tick time)
        ZonedDateTime tickTime = ZonedDateTime.of(2025, 4, 2, 23, 59, 0, 0, ZoneOffset.UTC);
        long result = Module9_EngagementMonitor.computeNextDailyTick(tickTime.toInstant().toEpochMilli());

        ZonedDateTime expected = ZonedDateTime.of(2025, 4, 3, 23, 59, 0, 0, ZoneOffset.UTC);
        assertEquals(expected.toInstant().toEpochMilli(), result);
    }

    @Test
    @DisplayName("00:00 UTC -> next tick is today 23:59 UTC")
    void midnightSchedulesToday() {
        // 2025-04-02 00:00:00 UTC
        ZonedDateTime midnight = ZonedDateTime.of(2025, 4, 2, 0, 0, 0, 0, ZoneOffset.UTC);
        long result = Module9_EngagementMonitor.computeNextDailyTick(midnight.toInstant().toEpochMilli());

        ZonedDateTime expected = ZonedDateTime.of(2025, 4, 2, 23, 59, 0, 0, ZoneOffset.UTC);
        assertEquals(expected.toInstant().toEpochMilli(), result);
    }

    @Test
    @DisplayName("23:59:30 UTC (past tick) -> next tick is tomorrow 23:59 UTC")
    void pastTickSchedulesTomorrow() {
        // 2025-04-02 23:59:30 UTC (30 seconds after tick)
        ZonedDateTime pastTick = ZonedDateTime.of(2025, 4, 2, 23, 59, 30, 0, ZoneOffset.UTC);
        long result = Module9_EngagementMonitor.computeNextDailyTick(pastTick.toInstant().toEpochMilli());

        ZonedDateTime expected = ZonedDateTime.of(2025, 4, 3, 23, 59, 0, 0, ZoneOffset.UTC);
        assertEquals(expected.toInstant().toEpochMilli(), result);
    }

    @Test
    @DisplayName("Computed tick is always in the future")
    void tickIsAlwaysFuture() {
        long now = System.currentTimeMillis();
        long tick = Module9_EngagementMonitor.computeNextDailyTick(now);
        assertTrue(tick > now, "Next tick must be in the future");
    }

    @Test
    @DisplayName("Consecutive tick computations produce 24-hour intervals")
    void consecutiveTicksAre24hApart() {
        ZonedDateTime start = ZonedDateTime.of(2025, 4, 2, 10, 0, 0, 0, ZoneOffset.UTC);
        long firstTick = Module9_EngagementMonitor.computeNextDailyTick(start.toInstant().toEpochMilli());
        long secondTick = Module9_EngagementMonitor.computeNextDailyTick(firstTick);

        long dayMs = 86_400_000L;
        assertEquals(dayMs, secondTick - firstTick,
            "Consecutive ticks should be exactly 24 hours apart");
    }
}
