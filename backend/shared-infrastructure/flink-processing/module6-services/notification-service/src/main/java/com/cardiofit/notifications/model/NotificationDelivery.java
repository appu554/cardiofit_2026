package com.cardiofit.notifications.model;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.time.Instant;

/**
 * Notification delivery tracking record
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class NotificationDelivery {

    @JsonProperty("delivery_id")
    private String deliveryId;

    @JsonProperty("alert_id")
    private String alertId;

    @JsonProperty("user_id")
    private String userId;

    @JsonProperty("channel")
    private UserPreference.NotificationChannel channel;

    @JsonProperty("status")
    private DeliveryStatus status;

    @JsonProperty("sent_at")
    @JsonFormat(shape = JsonFormat.Shape.STRING, pattern = "yyyy-MM-dd'T'HH:mm:ss'Z'", timezone = "UTC")
    private Instant sentAt;

    @JsonProperty("delivered_at")
    @JsonFormat(shape = JsonFormat.Shape.STRING, pattern = "yyyy-MM-dd'T'HH:mm:ss'Z'", timezone = "UTC")
    private Instant deliveredAt;

    @JsonProperty("read_at")
    @JsonFormat(shape = JsonFormat.Shape.STRING, pattern = "yyyy-MM-dd'T'HH:mm:ss'Z'", timezone = "UTC")
    private Instant readAt;

    @JsonProperty("error_message")
    private String errorMessage;

    @JsonProperty("retry_count")
    private int retryCount;

    @JsonProperty("provider_message_id")
    private String providerMessageId;

    public enum DeliveryStatus {
        PENDING,
        SENT,
        DELIVERED,
        READ,
        FAILED,
        RATE_LIMITED,
        SUPPRESSED,
        BUNDLED
    }
}
