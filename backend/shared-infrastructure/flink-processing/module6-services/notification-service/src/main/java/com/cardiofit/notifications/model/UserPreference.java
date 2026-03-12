package com.cardiofit.notifications.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.util.List;
import java.util.Map;

/**
 * User notification preferences
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class UserPreference {

    @JsonProperty("user_id")
    private String userId;

    @JsonProperty("email")
    private String email;

    @JsonProperty("phone_number")
    private String phoneNumber;

    @JsonProperty("fcm_token")
    private String fcmToken;

    @JsonProperty("enabled_channels")
    private List<NotificationChannel> enabledChannels;

    @JsonProperty("severity_thresholds")
    private Map<ComposedAlert.AlertSeverity, List<NotificationChannel>> severityThresholds;

    @JsonProperty("quiet_hours")
    private QuietHours quietHours;

    @JsonProperty("on_call_schedule")
    private OnCallSchedule onCallSchedule;

    @JsonProperty("alert_bundling_enabled")
    private boolean alertBundlingEnabled;

    @JsonProperty("bundling_window_minutes")
    private int bundlingWindowMinutes;

    public enum NotificationChannel {
        EMAIL,
        SMS,
        PUSH,
        PAGER
    }

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class QuietHours {
        @JsonProperty("enabled")
        private boolean enabled;

        @JsonProperty("start_hour")
        private int startHour;

        @JsonProperty("end_hour")
        private int endHour;

        @JsonProperty("override_critical")
        private boolean overrideCritical;
    }

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class OnCallSchedule {
        @JsonProperty("on_call")
        private boolean onCall;

        @JsonProperty("shift_start")
        private String shiftStart;

        @JsonProperty("shift_end")
        private String shiftEnd;

        @JsonProperty("escalation_user_id")
        private String escalationUserId;
    }
}
