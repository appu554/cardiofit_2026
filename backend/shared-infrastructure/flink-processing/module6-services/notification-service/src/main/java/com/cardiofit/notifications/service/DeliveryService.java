package com.cardiofit.notifications.service;

import com.cardiofit.notifications.model.ComposedAlert;
import com.cardiofit.notifications.model.NotificationDelivery;
import com.cardiofit.notifications.model.UserPreference;
import com.google.firebase.messaging.*;
import com.sendgrid.Method;
import com.sendgrid.Request;
import com.sendgrid.Response;
import com.sendgrid.SendGrid;
import com.sendgrid.helpers.mail.Mail;
import com.sendgrid.helpers.mail.objects.Content;
import com.sendgrid.helpers.mail.objects.Email;
import com.twilio.rest.api.v2010.account.Message;
import com.twilio.type.PhoneNumber;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.scheduling.annotation.Async;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.time.Instant;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;

/**
 * Multi-channel notification delivery service
 */
@Service
@Slf4j
@RequiredArgsConstructor
public class DeliveryService {

    private final SendGrid sendGridClient;
    private final RedisTemplate<String, Object> redisTemplate;
    private final UserPreferenceService preferenceService;

    @Value("${twilio.phone-number}")
    private String twilioPhoneNumber;

    @Value("${sendgrid.from-email}")
    private String sendgridFromEmail;

    private static final String DELIVERY_KEY_PREFIX = "delivery:";
    private static final int MAX_RETRY_ATTEMPTS = 3;

    /**
     * Send notification through specified channel
     */
    @Async
    public void sendNotification(
            String userId,
            ComposedAlert alert,
            UserPreference.NotificationChannel channel) {

        NotificationDelivery delivery = NotificationDelivery.builder()
                .deliveryId(UUID.randomUUID().toString())
                .alertId(alert.getAlertId())
                .userId(userId)
                .channel(channel)
                .status(NotificationDelivery.DeliveryStatus.PENDING)
                .sentAt(Instant.now())
                .retryCount(0)
                .build();

        try {
            switch (channel) {
                case EMAIL -> sendEmail(userId, alert, delivery);
                case SMS -> sendSms(userId, alert, delivery);
                case PUSH -> sendPushNotification(userId, alert, delivery);
                case PAGER -> sendPager(userId, alert, delivery);
            }

            delivery.setStatus(NotificationDelivery.DeliveryStatus.SENT);
            delivery.setDeliveredAt(Instant.now());
            log.info("Notification sent successfully: {} via {}", delivery.getDeliveryId(), channel);

        } catch (Exception e) {
            log.error("Failed to send notification via {}: {}", channel, e.getMessage(), e);
            delivery.setStatus(NotificationDelivery.DeliveryStatus.FAILED);
            delivery.setErrorMessage(e.getMessage());

            // Retry logic
            if (delivery.getRetryCount() < MAX_RETRY_ATTEMPTS) {
                scheduleRetry(userId, alert, channel, delivery);
            }
        }

        // Store delivery record
        storeDeliveryRecord(delivery);
    }

    /**
     * Send bundled notifications
     */
    @Async
    public void sendBundledNotification(
            String userId,
            List<ComposedAlert> alerts,
            UserPreference.NotificationChannel channel) {

        log.info("Sending bundled notification with {} alerts via {}", alerts.size(), channel);

        // Create summary alert
        ComposedAlert bundleAlert = createBundleAlert(alerts);

        // Send as single notification
        sendNotification(userId, bundleAlert, channel);
    }

    /**
     * Send email notification
     */
    private void sendEmail(String userId, ComposedAlert alert, NotificationDelivery delivery) throws Exception {
        UserPreference prefs = preferenceService.getUserPreferences(userId);

        if (prefs.getEmail() == null || prefs.getEmail().isEmpty()) {
            throw new IllegalStateException("User email not configured");
        }

        Email from = new Email(sendgridFromEmail, "CardioFit Clinical Alerts");
        Email to = new Email(prefs.getEmail());
        String subject = formatEmailSubject(alert);
        Content content = new Content("text/html", formatEmailBody(alert));

        Mail mail = new Mail(from, subject, to, content);

        Request request = new Request();
        request.setMethod(Method.POST);
        request.setEndpoint("mail/send");
        request.setBody(mail.build());

        Response response = sendGridClient.api(request);

        if (response.getStatusCode() >= 200 && response.getStatusCode() < 300) {
            delivery.setProviderMessageId(response.getHeaders().get("X-Message-Id"));
        } else {
            throw new RuntimeException("SendGrid API error: " + response.getStatusCode());
        }
    }

    /**
     * Send SMS notification
     */
    private void sendSms(String userId, ComposedAlert alert, NotificationDelivery delivery) throws Exception {
        UserPreference prefs = preferenceService.getUserPreferences(userId);

        if (prefs.getPhoneNumber() == null || prefs.getPhoneNumber().isEmpty()) {
            throw new IllegalStateException("User phone number not configured");
        }

        String messageBody = formatSmsBody(alert);

        Message message = Message.creator(
                new PhoneNumber(prefs.getPhoneNumber()),
                new PhoneNumber(twilioPhoneNumber),
                messageBody
        ).create();

        delivery.setProviderMessageId(message.getSid());
        log.info("SMS sent via Twilio: {}", message.getSid());
    }

    /**
     * Send push notification via Firebase
     */
    private void sendPushNotification(String userId, ComposedAlert alert, NotificationDelivery delivery) throws Exception {
        UserPreference prefs = preferenceService.getUserPreferences(userId);

        if (prefs.getFcmToken() == null || prefs.getFcmToken().isEmpty()) {
            throw new IllegalStateException("User FCM token not configured");
        }

        Map<String, String> data = new HashMap<>();
        data.put("alertId", alert.getAlertId());
        data.put("patientId", alert.getPatientId());
        data.put("severity", alert.getSeverity().name());
        data.put("timestamp", alert.getTimestamp().toString());

        Notification notification = Notification.builder()
                .setTitle(formatPushTitle(alert))
                .setBody(formatPushBody(alert))
                .build();

        Message message = Message.builder()
                .setToken(prefs.getFcmToken())
                .setNotification(notification)
                .putAllData(data)
                .setAndroidConfig(AndroidConfig.builder()
                        .setPriority(alert.getSeverity() == ComposedAlert.AlertSeverity.CRITICAL ?
                                AndroidConfig.Priority.HIGH : AndroidConfig.Priority.NORMAL)
                        .build())
                .setApnsConfig(ApnsConfig.builder()
                        .setAps(Aps.builder()
                                .setSound("default")
                                .build())
                        .build())
                .build();

        String messageId = FirebaseMessaging.getInstance().send(message);
        delivery.setProviderMessageId(messageId);
        log.info("Push notification sent via Firebase: {}", messageId);
    }

    /**
     * Send pager notification (via SMS gateway)
     */
    private void sendPager(String userId, ComposedAlert alert, NotificationDelivery delivery) throws Exception {
        // Pager typically uses SMS gateway with specific format
        UserPreference prefs = preferenceService.getUserPreferences(userId);

        if (prefs.getPhoneNumber() == null) {
            throw new IllegalStateException("User pager number not configured");
        }

        // Format for pager: numeric codes + brief message
        String pagerMessage = formatPagerMessage(alert);

        Message message = Message.creator(
                new PhoneNumber(prefs.getPhoneNumber()),
                new PhoneNumber(twilioPhoneNumber),
                pagerMessage
        ).create();

        delivery.setProviderMessageId(message.getSid());
        log.info("Pager notification sent: {}", message.getSid());
    }

    /**
     * Format email subject
     */
    private String formatEmailSubject(ComposedAlert alert) {
        return String.format("[%s] %s - %s",
                alert.getSeverity(),
                alert.getPatientName(),
                alert.getTitle());
    }

    /**
     * Format email body
     */
    private String formatEmailBody(ComposedAlert alert) {
        StringBuilder html = new StringBuilder();
        html.append("<html><body>");
        html.append("<h2 style='color: ").append(getSeverityColor(alert.getSeverity())).append("'>");
        html.append(alert.getTitle()).append("</h2>");
        html.append("<p><strong>Patient:</strong> ").append(alert.getPatientName()).append("</p>");
        html.append("<p><strong>Severity:</strong> ").append(alert.getSeverity()).append("</p>");
        html.append("<p><strong>Time:</strong> ").append(alert.getTimestamp()).append("</p>");
        html.append("<p>").append(alert.getMessage()).append("</p>");

        if (alert.getRecommendedActions() != null && !alert.getRecommendedActions().isEmpty()) {
            html.append("<h3>Recommended Actions:</h3><ul>");
            for (String action : alert.getRecommendedActions()) {
                html.append("<li>").append(action).append("</li>");
            }
            html.append("</ul>");
        }

        if (alert.getClinicalContext() != null) {
            html.append("<h3>Clinical Context:</h3>");
            html.append("<p><strong>Location:</strong> ")
                    .append(alert.getClinicalContext().getLocation()).append("</p>");
        }

        html.append("<p><em>Alert ID: ").append(alert.getAlertId()).append("</em></p>");
        html.append("</body></html>");
        return html.toString();
    }

    /**
     * Format SMS body (160 char limit)
     */
    private String formatSmsBody(ComposedAlert alert) {
        return String.format("[%s] %s: %s - %s",
                alert.getSeverity(),
                alert.getPatientName(),
                alert.getTitle(),
                alert.getMessage().substring(0, Math.min(80, alert.getMessage().length())));
    }

    /**
     * Format push notification title
     */
    private String formatPushTitle(ComposedAlert alert) {
        return String.format("%s Alert - %s", alert.getSeverity(), alert.getPatientName());
    }

    /**
     * Format push notification body
     */
    private String formatPushBody(ComposedAlert alert) {
        return alert.getTitle() + ": " + alert.getMessage();
    }

    /**
     * Format pager message
     */
    private String formatPagerMessage(ComposedAlert alert) {
        return String.format("CODE:%s PT:%s %s",
                getSeverityCode(alert.getSeverity()),
                alert.getPatientId().substring(0, 8),
                alert.getTitle().substring(0, Math.min(50, alert.getTitle().length())));
    }

    /**
     * Get severity color for HTML
     */
    private String getSeverityColor(ComposedAlert.AlertSeverity severity) {
        return switch (severity) {
            case CRITICAL -> "#DC143C";
            case HIGH -> "#FF6347";
            case MEDIUM -> "#FFA500";
            case LOW -> "#FFD700";
            case INFO -> "#4682B4";
        };
    }

    /**
     * Get numeric severity code for pager
     */
    private String getSeverityCode(ComposedAlert.AlertSeverity severity) {
        return switch (severity) {
            case CRITICAL -> "999";
            case HIGH -> "911";
            case MEDIUM -> "711";
            case LOW -> "511";
            case INFO -> "111";
        };
    }

    /**
     * Create bundled alert summary
     */
    private ComposedAlert createBundleAlert(List<ComposedAlert> alerts) {
        Map<ComposedAlert.AlertSeverity, Long> severityCounts = new HashMap<>();
        for (ComposedAlert alert : alerts) {
            severityCounts.merge(alert.getSeverity(), 1L, Long::sum);
        }

        ComposedAlert.AlertSeverity maxSeverity = alerts.stream()
                .map(ComposedAlert::getSeverity)
                .max(Enum::compareTo)
                .orElse(ComposedAlert.AlertSeverity.INFO);

        String title = String.format("Bundled Alert Summary - %d alerts", alerts.size());
        StringBuilder message = new StringBuilder("Alert Summary:\n");
        severityCounts.forEach((severity, count) ->
                message.append(String.format("- %s: %d\n", severity, count)));

        return ComposedAlert.builder()
                .alertId(UUID.randomUUID().toString())
                .patientId("BUNDLED")
                .patientName("Multiple Patients")
                .alertType("BUNDLED_ALERT")
                .severity(maxSeverity)
                .title(title)
                .message(message.toString())
                .timestamp(Instant.now())
                .build();
    }

    /**
     * Store delivery record in Redis
     */
    private void storeDeliveryRecord(NotificationDelivery delivery) {
        String key = DELIVERY_KEY_PREFIX + delivery.getDeliveryId();
        redisTemplate.opsForValue().set(key, delivery, Duration.ofDays(30));
    }

    /**
     * Schedule retry for failed delivery
     */
    private void scheduleRetry(
            String userId,
            ComposedAlert alert,
            UserPreference.NotificationChannel channel,
            NotificationDelivery previousDelivery) {

        int retryCount = previousDelivery.getRetryCount() + 1;
        log.info("Scheduling retry {} for delivery: {}", retryCount, previousDelivery.getDeliveryId());

        // Exponential backoff: 1min, 2min, 4min
        long delaySeconds = (long) Math.pow(2, retryCount) * 60;

        // In production, use a proper scheduler or queue
        // For now, just log the retry intent
        log.warn("Retry would be scheduled after {} seconds", delaySeconds);
    }

    /**
     * Get delivery record
     */
    public NotificationDelivery getDeliveryRecord(String deliveryId) {
        String key = DELIVERY_KEY_PREFIX + deliveryId;
        Object record = redisTemplate.opsForValue().get(key);
        return record instanceof NotificationDelivery ? (NotificationDelivery) record : null;
    }
}
