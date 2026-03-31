package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

/**
 * Structured notification request emitted to clinical-notifications.v1.
 * Module 6 does NOT send notifications directly — a separate service handles delivery.
 */
public class NotificationRequest implements Serializable {
    private static final long serialVersionUID = 1L;

    public enum Channel { SMS, FCM_PUSH, EMAIL, PHONE_FALLBACK, DASHBOARD_ONLY }

    @JsonProperty("notification_id") private String notificationId;
    @JsonProperty("alert_id") private String alertId;
    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("channel") private Channel channel;
    @JsonProperty("tier") private ActionTier tier;
    @JsonProperty("title") private String title;
    @JsonProperty("body") private String body;
    @JsonProperty("data") private Map<String, String> data = new HashMap<>();
    @JsonProperty("created_at") private long createdAt = System.currentTimeMillis();
    @JsonProperty("priority") private int priority;
    @JsonProperty("requires_acknowledgment") private boolean requiresAcknowledgment;

    public NotificationRequest() {}

    public String getNotificationId() { return notificationId; }
    public void setNotificationId(String notificationId) { this.notificationId = notificationId; }
    public String getAlertId() { return alertId; }
    public void setAlertId(String alertId) { this.alertId = alertId; }
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public Channel getChannel() { return channel; }
    public void setChannel(Channel channel) { this.channel = channel; }
    public ActionTier getTier() { return tier; }
    public void setTier(ActionTier tier) { this.tier = tier; }
    public String getTitle() { return title; }
    public void setTitle(String title) { this.title = title; }
    public String getBody() { return body; }
    public void setBody(String body) { this.body = body; }
    public Map<String, String> getData() { return data; }
    public void setData(Map<String, String> data) { this.data = data; }
    public long getCreatedAt() { return createdAt; }
    public void setCreatedAt(long createdAt) { this.createdAt = createdAt; }
    public int getPriority() { return priority; }
    public void setPriority(int priority) { this.priority = priority; }
    public boolean isRequiresAcknowledgment() { return requiresAcknowledgment; }
    public void setRequiresAcknowledgment(boolean requiresAcknowledgment) { this.requiresAcknowledgment = requiresAcknowledgment; }
}
