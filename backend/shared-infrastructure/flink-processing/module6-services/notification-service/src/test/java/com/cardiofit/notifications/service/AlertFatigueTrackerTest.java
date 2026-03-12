package com.cardiofit.notifications.service;

import com.cardiofit.notifications.model.ComposedAlert;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.core.ValueOperations;

import java.time.Instant;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class AlertFatigueTrackerTest {

    @Mock
    private RedisTemplate<String, Object> redisTemplate;

    @Mock
    private ValueOperations<String, Object> valueOperations;

    private AlertFatigueTracker fatigueTracker;

    @BeforeEach
    void setUp() {
        when(redisTemplate.opsForValue()).thenReturn(valueOperations);
        fatigueTracker = new AlertFatigueTracker(redisTemplate);
    }

    @Test
    void testAllowAlert_CriticalAlertBypassesRateLimit() {
        // Given
        String userId = "user123";
        ComposedAlert alert = createTestAlert(ComposedAlert.AlertSeverity.CRITICAL);

        // When
        boolean allowed = fatigueTracker.allowAlert(userId, alert);

        // Then
        assertTrue(allowed, "Critical alerts should bypass rate limiting");
    }

    @Test
    void testAllowAlert_RateLimitEnforced() {
        // Given
        String userId = "user123";
        ComposedAlert alert = createTestAlert(ComposedAlert.AlertSeverity.HIGH);

        // When - Send 20 alerts (should all pass)
        for (int i = 0; i < 20; i++) {
            assertTrue(fatigueTracker.allowAlert(userId, alert));
        }

        // Then - 21st alert should be rate limited
        assertFalse(fatigueTracker.allowAlert(userId, alert));
    }

    @Test
    void testIsDuplicate_FirstAlertNotDuplicate() {
        // Given
        String userId = "user123";
        ComposedAlert alert = createTestAlert(ComposedAlert.AlertSeverity.HIGH);
        when(valueOperations.setIfAbsent(anyString(), any(), any())).thenReturn(true);

        // When
        boolean isDuplicate = fatigueTracker.isDuplicate(userId, alert);

        // Then
        assertFalse(isDuplicate);
    }

    @Test
    void testIsDuplicate_SubsequentAlertIsDuplicate() {
        // Given
        String userId = "user123";
        ComposedAlert alert = createTestAlert(ComposedAlert.AlertSeverity.HIGH);
        when(valueOperations.setIfAbsent(anyString(), any(), any())).thenReturn(false);

        // When
        boolean isDuplicate = fatigueTracker.isDuplicate(userId, alert);

        // Then
        assertTrue(isDuplicate);
    }

    @Test
    void testGetRateLimitStats_NewUser() {
        // Given
        String userId = "newuser";

        // When
        AlertFatigueTracker.RateLimitStats stats = fatigueTracker.getRateLimitStats(userId);

        // Then
        assertEquals(20, stats.getLimit());
        assertEquals(0, stats.getConsumed());
        assertEquals(20, stats.getRemaining());
    }

    @Test
    void testResetRateLimit_ClearsLimits() {
        // Given
        String userId = "user123";

        // When
        fatigueTracker.resetRateLimit(userId);

        // Then
        verify(redisTemplate).delete(contains(userId));
    }

    @Test
    void testShouldBundle_CriticalAlertsNeverBundled() {
        // Given
        String userId = "user123";
        ComposedAlert alert = createTestAlert(ComposedAlert.AlertSeverity.CRITICAL);

        // When
        AlertFatigueTracker.BundlingDecision decision = fatigueTracker.shouldBundle(userId, alert);

        // Then
        assertFalse(decision.shouldWait());
        assertNull(decision.getBundledAlerts());
    }

    private ComposedAlert createTestAlert(ComposedAlert.AlertSeverity severity) {
        return ComposedAlert.builder()
                .alertId("test-alert-" + System.currentTimeMillis())
                .patientId("patient123")
                .patientName("John Doe")
                .alertType("VITAL_SIGN_ALERT")
                .severity(severity)
                .title("Test Alert")
                .message("This is a test alert")
                .timestamp(Instant.now())
                .build();
    }
}
