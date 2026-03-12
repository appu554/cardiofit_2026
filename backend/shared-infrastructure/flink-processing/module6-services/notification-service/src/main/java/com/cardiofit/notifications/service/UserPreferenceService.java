package com.cardiofit.notifications.service;

import com.cardiofit.notifications.model.ComposedAlert;
import com.cardiofit.notifications.model.UserPreference;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.cache.annotation.Cacheable;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;

import java.time.LocalTime;
import java.util.*;

/**
 * User preference management service
 */
@Service
@Slf4j
@RequiredArgsConstructor
public class UserPreferenceService {

    private final RedisTemplate<String, Object> redisTemplate;

    private static final String PREFERENCE_KEY_PREFIX = "preferences:";

    /**
     * Get user preferences (cached)
     */
    @Cacheable(value = "userPreferences", key = "#userId")
    public UserPreference getUserPreferences(String userId) {
        String key = PREFERENCE_KEY_PREFIX + userId;
        Object prefs = redisTemplate.opsForValue().get(key);

        if (prefs instanceof UserPreference) {
            return (UserPreference) prefs;
        }

        // Return default preferences
        return createDefaultPreferences(userId);
    }

    /**
     * Save user preferences
     */
    public void saveUserPreferences(UserPreference preference) {
        String key = PREFERENCE_KEY_PREFIX + preference.getUserId();
        redisTemplate.opsForValue().set(key, preference);
        log.info("Saved preferences for user: {}", preference.getUserId());
    }

    /**
     * Get enabled notification channels for user and alert severity
     */
    public List<UserPreference.NotificationChannel> getEnabledChannels(
            String userId,
            ComposedAlert.AlertSeverity severity) {

        UserPreference prefs = getUserPreferences(userId);

        // Check severity-specific thresholds
        if (prefs.getSeverityThresholds() != null &&
                prefs.getSeverityThresholds().containsKey(severity)) {
            return prefs.getSeverityThresholds().get(severity);
        }

        // Return default enabled channels
        return prefs.getEnabledChannels();
    }

    /**
     * Check if user is in quiet hours
     */
    public boolean isInQuietHours(String userId, ComposedAlert alert) {
        UserPreference prefs = getUserPreferences(userId);
        UserPreference.QuietHours quietHours = prefs.getQuietHours();

        if (quietHours == null || !quietHours.isEnabled()) {
            return false;
        }

        // Critical alerts can override quiet hours
        if (alert.getSeverity() == ComposedAlert.AlertSeverity.CRITICAL &&
                quietHours.isOverrideCritical()) {
            return false;
        }

        LocalTime now = LocalTime.now();
        LocalTime start = LocalTime.of(quietHours.getStartHour(), 0);
        LocalTime end = LocalTime.of(quietHours.getEndHour(), 0);

        // Handle overnight quiet hours
        if (start.isBefore(end)) {
            return now.isAfter(start) && now.isBefore(end);
        } else {
            return now.isAfter(start) || now.isBefore(end);
        }
    }

    /**
     * Check if user is on call
     */
    public boolean isOnCall(String userId) {
        UserPreference prefs = getUserPreferences(userId);
        UserPreference.OnCallSchedule schedule = prefs.getOnCallSchedule();

        return schedule != null && schedule.isOnCall();
    }

    /**
     * Get escalation user if primary user is not on call
     */
    public Optional<String> getEscalationUser(String userId) {
        UserPreference prefs = getUserPreferences(userId);
        UserPreference.OnCallSchedule schedule = prefs.getOnCallSchedule();

        if (schedule != null && schedule.getEscalationUserId() != null) {
            return Optional.of(schedule.getEscalationUserId());
        }

        return Optional.empty();
    }

    /**
     * Check if alert bundling is enabled for user
     */
    public boolean isBundlingEnabled(String userId) {
        UserPreference prefs = getUserPreferences(userId);
        return prefs.isAlertBundlingEnabled();
    }

    /**
     * Get bundling window in minutes
     */
    public int getBundlingWindowMinutes(String userId) {
        UserPreference prefs = getUserPreferences(userId);
        return prefs.getBundlingWindowMinutes() > 0 ? prefs.getBundlingWindowMinutes() : 10;
    }

    /**
     * Create default preferences for new user
     */
    private UserPreference createDefaultPreferences(String userId) {
        return UserPreference.builder()
                .userId(userId)
                .enabledChannels(Arrays.asList(
                        UserPreference.NotificationChannel.EMAIL,
                        UserPreference.NotificationChannel.PUSH
                ))
                .severityThresholds(createDefaultSeverityThresholds())
                .quietHours(UserPreference.QuietHours.builder()
                        .enabled(false)
                        .startHour(22)
                        .endHour(7)
                        .overrideCritical(true)
                        .build())
                .onCallSchedule(UserPreference.OnCallSchedule.builder()
                        .onCall(true)
                        .build())
                .alertBundlingEnabled(true)
                .bundlingWindowMinutes(10)
                .build();
    }

    /**
     * Create default severity thresholds
     */
    private Map<ComposedAlert.AlertSeverity, List<UserPreference.NotificationChannel>> createDefaultSeverityThresholds() {
        Map<ComposedAlert.AlertSeverity, List<UserPreference.NotificationChannel>> thresholds = new HashMap<>();

        thresholds.put(ComposedAlert.AlertSeverity.CRITICAL, Arrays.asList(
                UserPreference.NotificationChannel.SMS,
                UserPreference.NotificationChannel.PUSH,
                UserPreference.NotificationChannel.PAGER,
                UserPreference.NotificationChannel.EMAIL
        ));

        thresholds.put(ComposedAlert.AlertSeverity.HIGH, Arrays.asList(
                UserPreference.NotificationChannel.SMS,
                UserPreference.NotificationChannel.PUSH,
                UserPreference.NotificationChannel.EMAIL
        ));

        thresholds.put(ComposedAlert.AlertSeverity.MEDIUM, Arrays.asList(
                UserPreference.NotificationChannel.PUSH,
                UserPreference.NotificationChannel.EMAIL
        ));

        thresholds.put(ComposedAlert.AlertSeverity.LOW, Arrays.asList(
                UserPreference.NotificationChannel.EMAIL
        ));

        thresholds.put(ComposedAlert.AlertSeverity.INFO, Arrays.asList(
                UserPreference.NotificationChannel.EMAIL
        ));

        return thresholds;
    }
}
