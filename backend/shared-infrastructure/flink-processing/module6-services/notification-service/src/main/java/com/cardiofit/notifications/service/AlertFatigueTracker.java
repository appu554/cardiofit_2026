package com.cardiofit.notifications.service;

import com.cardiofit.notifications.model.ComposedAlert;
import com.cardiofit.notifications.model.NotificationDelivery;
import io.github.bucket4j.Bandwidth;
import io.github.bucket4j.Bucket;
import io.github.bucket4j.Refill;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.TimeUnit;

/**
 * Alert fatigue mitigation through rate limiting, deduplication, and bundling
 */
@Service
@Slf4j
@RequiredArgsConstructor
public class AlertFatigueTracker {

    private final RedisTemplate<String, Object> redisTemplate;

    // In-memory rate limiters per user (backed by Redis for distributed scenario)
    private final Map<String, Bucket> rateLimitBuckets = new ConcurrentHashMap<>();

    // Configuration
    private static final int MAX_ALERTS_PER_HOUR = 20;
    private static final int DEDUPLICATION_WINDOW_MINUTES = 5;
    private static final int BUNDLING_WINDOW_MINUTES = 10;
    private static final String RATE_LIMIT_KEY_PREFIX = "ratelimit:";
    private static final String DEDUP_KEY_PREFIX = "dedup:";
    private static final String BUNDLE_KEY_PREFIX = "bundle:";

    /**
     * Check if alert should be sent based on rate limiting
     */
    public boolean allowAlert(String userId, ComposedAlert alert) {
        // Critical alerts bypass rate limiting
        if (alert.getSeverity() == ComposedAlert.AlertSeverity.CRITICAL) {
            log.info("Critical alert bypasses rate limiting for user: {}", userId);
            return true;
        }

        Bucket bucket = getRateLimitBucket(userId);
        if (bucket.tryConsume(1)) {
            return true;
        }

        log.warn("Rate limit exceeded for user: {}, alert: {}", userId, alert.getAlertId());
        recordRateLimitEvent(userId, alert);
        return false;
    }

    /**
     * Check if alert is a duplicate within the deduplication window
     */
    public boolean isDuplicate(String userId, ComposedAlert alert) {
        String dedupKey = buildDeduplicationKey(userId, alert);
        Boolean isDuplicate = redisTemplate.opsForValue().setIfAbsent(
                dedupKey,
                alert.getAlertId(),
                Duration.ofMinutes(DEDUPLICATION_WINDOW_MINUTES)
        );

        if (isDuplicate != null && !isDuplicate) {
            log.info("Duplicate alert suppressed for user: {}, alert: {}", userId, alert.getAlertId());
            recordSuppressionEvent(userId, alert, "DUPLICATE");
            return true;
        }

        return false;
    }

    /**
     * Check if alert should be bundled
     */
    public BundlingDecision shouldBundle(String userId, ComposedAlert alert) {
        String bundleKey = BUNDLE_KEY_PREFIX + userId;

        // Critical alerts are never bundled
        if (alert.getSeverity() == ComposedAlert.AlertSeverity.CRITICAL) {
            return new BundlingDecision(false, null);
        }

        List<Object> bundledAlerts = redisTemplate.opsForList().range(bundleKey, 0, -1);

        if (bundledAlerts == null || bundledAlerts.isEmpty()) {
            // Start new bundle
            redisTemplate.opsForList().rightPush(bundleKey, alert);
            redisTemplate.expire(bundleKey, Duration.ofMinutes(BUNDLING_WINDOW_MINUTES));
            return new BundlingDecision(true, null);
        }

        // Check bundle size
        if (bundledAlerts.size() >= 5) {
            // Bundle is full, send it
            List<ComposedAlert> alerts = new ArrayList<>();
            for (Object obj : bundledAlerts) {
                if (obj instanceof ComposedAlert) {
                    alerts.add((ComposedAlert) obj);
                }
            }
            alerts.add(alert);

            // Clear bundle
            redisTemplate.delete(bundleKey);

            return new BundlingDecision(false, alerts);
        }

        // Add to bundle
        redisTemplate.opsForList().rightPush(bundleKey, alert);
        return new BundlingDecision(true, null);
    }

    /**
     * Get or create rate limit bucket for user
     */
    private Bucket getRateLimitBucket(String userId) {
        return rateLimitBuckets.computeIfAbsent(userId, id -> {
            // Check Redis for existing bucket state
            String redisKey = RATE_LIMIT_KEY_PREFIX + userId;
            Long remaining = (Long) redisTemplate.opsForValue().get(redisKey);

            Bandwidth limit = Bandwidth.classic(MAX_ALERTS_PER_HOUR, Refill.intervally(
                    MAX_ALERTS_PER_HOUR,
                    Duration.ofHours(1)
            ));

            Bucket bucket = Bucket.builder()
                    .addLimit(limit)
                    .build();

            // Initialize from Redis if available
            if (remaining != null && remaining < MAX_ALERTS_PER_HOUR) {
                long consumed = MAX_ALERTS_PER_HOUR - remaining;
                for (int i = 0; i < consumed; i++) {
                    bucket.tryConsume(1);
                }
            }

            return bucket;
        });
    }

    /**
     * Build deduplication key based on alert characteristics
     */
    private String buildDeduplicationKey(String userId, ComposedAlert alert) {
        // Deduplicate based on patient, alert type, and severity
        String fingerprint = String.format("%s:%s:%s:%s",
                userId,
                alert.getPatientId(),
                alert.getAlertType(),
                alert.getSeverity()
        );
        return DEDUP_KEY_PREFIX + fingerprint;
    }

    /**
     * Record rate limit event for monitoring
     */
    private void recordRateLimitEvent(String userId, ComposedAlert alert) {
        String key = "events:ratelimit:" + userId + ":" + Instant.now().toEpochMilli();
        Map<String, Object> event = new HashMap<>();
        event.put("userId", userId);
        event.put("alertId", alert.getAlertId());
        event.put("severity", alert.getSeverity());
        event.put("timestamp", Instant.now().toString());

        redisTemplate.opsForValue().set(key, event, Duration.ofDays(7));
    }

    /**
     * Record suppression event for monitoring
     */
    private void recordSuppressionEvent(String userId, ComposedAlert alert, String reason) {
        String key = "events:suppression:" + userId + ":" + Instant.now().toEpochMilli();
        Map<String, Object> event = new HashMap<>();
        event.put("userId", userId);
        event.put("alertId", alert.getAlertId());
        event.put("reason", reason);
        event.put("timestamp", Instant.now().toString());

        redisTemplate.opsForValue().set(key, event, Duration.ofDays(7));
    }

    /**
     * Get rate limit statistics for user
     */
    public RateLimitStats getRateLimitStats(String userId) {
        Bucket bucket = rateLimitBuckets.get(userId);
        if (bucket == null) {
            return new RateLimitStats(MAX_ALERTS_PER_HOUR, 0, MAX_ALERTS_PER_HOUR);
        }

        long available = bucket.getAvailableTokens();
        long consumed = MAX_ALERTS_PER_HOUR - available;

        return new RateLimitStats(MAX_ALERTS_PER_HOUR, consumed, available);
    }

    /**
     * Reset rate limit for user (admin operation)
     */
    public void resetRateLimit(String userId) {
        rateLimitBuckets.remove(userId);
        String redisKey = RATE_LIMIT_KEY_PREFIX + userId;
        redisTemplate.delete(redisKey);
        log.info("Rate limit reset for user: {}", userId);
    }

    /**
     * Bundling decision result
     */
    public static class BundlingDecision {
        private final boolean shouldWait;
        private final List<ComposedAlert> bundledAlerts;

        public BundlingDecision(boolean shouldWait, List<ComposedAlert> bundledAlerts) {
            this.shouldWait = shouldWait;
            this.bundledAlerts = bundledAlerts;
        }

        public boolean shouldWait() {
            return shouldWait;
        }

        public List<ComposedAlert> getBundledAlerts() {
            return bundledAlerts;
        }
    }

    /**
     * Rate limit statistics
     */
    public static class RateLimitStats {
        private final long limit;
        private final long consumed;
        private final long remaining;

        public RateLimitStats(long limit, long consumed, long remaining) {
            this.limit = limit;
            this.consumed = consumed;
            this.remaining = remaining;
        }

        public long getLimit() { return limit; }
        public long getConsumed() { return consumed; }
        public long getRemaining() { return remaining; }
    }
}
