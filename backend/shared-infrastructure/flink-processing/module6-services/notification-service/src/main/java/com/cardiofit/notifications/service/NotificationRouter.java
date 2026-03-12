package com.cardiofit.notifications.service;

import com.cardiofit.notifications.model.ComposedAlert;
import com.cardiofit.notifications.model.UserPreference;
import io.micrometer.core.instrument.Counter;
import io.micrometer.core.instrument.MeterRegistry;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.kafka.support.Acknowledgment;
import org.springframework.kafka.support.KafkaHeaders;
import org.springframework.messaging.handler.annotation.Header;
import org.springframework.messaging.handler.annotation.Payload;
import org.springframework.stereotype.Service;

import java.util.List;
import java.util.Optional;

/**
 * Main notification routing service
 * Consumes alerts from Kafka and routes to appropriate delivery channels
 */
@Service
@Slf4j
@RequiredArgsConstructor
public class NotificationRouter {

    private final AlertFatigueTracker fatigueTracker;
    private final UserPreferenceService preferenceService;
    private final DeliveryService deliveryService;
    private final MeterRegistry meterRegistry;

    private Counter alertsReceivedCounter;
    private Counter alertsProcessedCounter;
    private Counter alertsRateLimitedCounter;
    private Counter alertsSuppressedCounter;
    private Counter alertsBundledCounter;

    /**
     * Initialize metrics
     */
    public NotificationRouter(
            AlertFatigueTracker fatigueTracker,
            UserPreferenceService preferenceService,
            DeliveryService deliveryService,
            MeterRegistry meterRegistry) {

        this.fatigueTracker = fatigueTracker;
        this.preferenceService = preferenceService;
        this.deliveryService = deliveryService;
        this.meterRegistry = meterRegistry;

        // Initialize counters
        this.alertsReceivedCounter = Counter.builder("alerts.received")
                .description("Total alerts received from Kafka")
                .register(meterRegistry);

        this.alertsProcessedCounter = Counter.builder("alerts.processed")
                .description("Total alerts successfully processed")
                .register(meterRegistry);

        this.alertsRateLimitedCounter = Counter.builder("alerts.rate_limited")
                .description("Total alerts blocked by rate limiting")
                .register(meterRegistry);

        this.alertsSuppressedCounter = Counter.builder("alerts.suppressed")
                .description("Total alerts suppressed as duplicates")
                .register(meterRegistry);

        this.alertsBundledCounter = Counter.builder("alerts.bundled")
                .description("Total alerts added to bundles")
                .register(meterRegistry);
    }

    /**
     * Main Kafka listener for composed alerts
     */
    @KafkaListener(
            topics = "${kafka.topics.composed-alerts}",
            groupId = "${spring.kafka.consumer.group-id}",
            containerFactory = "kafkaListenerContainerFactory"
    )
    public void processAlert(
            @Payload ComposedAlert alert,
            @Header(KafkaHeaders.RECEIVED_PARTITION) int partition,
            @Header(KafkaHeaders.OFFSET) long offset,
            Acknowledgment acknowledgment) {

        log.info("Received alert: {} from partition: {}, offset: {}",
                alert.getAlertId(), partition, offset);

        alertsReceivedCounter.increment();

        try {
            // Get users assigned to this alert
            List<String> assignedUsers = alert.getAssignedTo();

            if (assignedUsers == null || assignedUsers.isEmpty()) {
                log.warn("No users assigned to alert: {}", alert.getAlertId());
                acknowledgment.acknowledge();
                return;
            }

            // Process for each assigned user
            for (String userId : assignedUsers) {
                processAlertForUser(userId, alert);
            }

            alertsProcessedCounter.increment();
            acknowledgment.acknowledge();

        } catch (Exception e) {
            log.error("Error processing alert: {}", alert.getAlertId(), e);
            // Don't acknowledge - message will be retried
        }
    }

    /**
     * Process alert for a specific user
     */
    private void processAlertForUser(String userId, ComposedAlert alert) {
        log.debug("Processing alert {} for user {}", alert.getAlertId(), userId);

        // Step 1: Check if user is in quiet hours
        if (preferenceService.isInQuietHours(userId, alert)) {
            log.info("User {} is in quiet hours, skipping alert: {}", userId, alert.getAlertId());
            return;
        }

        // Step 2: Check on-call status
        if (!preferenceService.isOnCall(userId)) {
            log.info("User {} is not on call, checking escalation", userId);
            Optional<String> escalationUser = preferenceService.getEscalationUser(userId);
            if (escalationUser.isPresent()) {
                log.info("Escalating to user: {}", escalationUser.get());
                processAlertForUser(escalationUser.get(), alert);
                return;
            }
        }

        // Step 3: Check for duplicate alerts
        if (fatigueTracker.isDuplicate(userId, alert)) {
            log.info("Duplicate alert suppressed for user {}: {}", userId, alert.getAlertId());
            alertsSuppressedCounter.increment();
            return;
        }

        // Step 4: Check rate limiting
        if (!fatigueTracker.allowAlert(userId, alert)) {
            log.warn("Rate limit exceeded for user {}: {}", userId, alert.getAlertId());
            alertsRateLimitedCounter.increment();
            return;
        }

        // Step 5: Check bundling
        if (preferenceService.isBundlingEnabled(userId)) {
            AlertFatigueTracker.BundlingDecision bundling = fatigueTracker.shouldBundle(userId, alert);

            if (bundling.shouldWait()) {
                log.info("Alert {} added to bundle for user {}", alert.getAlertId(), userId);
                alertsBundledCounter.increment();
                return;
            }

            if (bundling.getBundledAlerts() != null) {
                log.info("Sending bundled alerts for user {}", userId);
                sendBundledAlerts(userId, bundling.getBundledAlerts());
                return;
            }
        }

        // Step 6: Route to delivery channels
        routeToChannels(userId, alert);
    }

    /**
     * Route alert to appropriate delivery channels
     */
    private void routeToChannels(String userId, ComposedAlert alert) {
        List<UserPreference.NotificationChannel> channels =
                preferenceService.getEnabledChannels(userId, alert.getSeverity());

        log.info("Routing alert {} to {} channels for user {}",
                alert.getAlertId(), channels.size(), userId);

        for (UserPreference.NotificationChannel channel : channels) {
            try {
                deliveryService.sendNotification(userId, alert, channel);
                recordChannelMetric(channel, "success");
            } catch (Exception e) {
                log.error("Failed to send via {}: {}", channel, e.getMessage());
                recordChannelMetric(channel, "failure");
            }
        }
    }

    /**
     * Send bundled alerts
     */
    private void sendBundledAlerts(String userId, List<ComposedAlert> alerts) {
        List<UserPreference.NotificationChannel> channels =
                preferenceService.getEnabledChannels(userId, ComposedAlert.AlertSeverity.MEDIUM);

        log.info("Sending bundle of {} alerts to user {} via {} channels",
                alerts.size(), userId, channels.size());

        for (UserPreference.NotificationChannel channel : channels) {
            try {
                deliveryService.sendBundledNotification(userId, alerts, channel);
                recordChannelMetric(channel, "bundle_success");
            } catch (Exception e) {
                log.error("Failed to send bundled notification via {}: {}", channel, e.getMessage());
                recordChannelMetric(channel, "bundle_failure");
            }
        }
    }

    /**
     * Record channel-specific metrics
     */
    private void recordChannelMetric(UserPreference.NotificationChannel channel, String status) {
        Counter.builder("notifications.channel")
                .tag("channel", channel.name())
                .tag("status", status)
                .description("Notification delivery by channel and status")
                .register(meterRegistry)
                .increment();
    }

    /**
     * Get processing statistics
     */
    public NotificationStats getStats() {
        return NotificationStats.builder()
                .alertsReceived((long) alertsReceivedCounter.count())
                .alertsProcessed((long) alertsProcessedCounter.count())
                .alertsRateLimited((long) alertsRateLimitedCounter.count())
                .alertsSuppressed((long) alertsSuppressedCounter.count())
                .alertsBundled((long) alertsBundledCounter.count())
                .build();
    }

    /**
     * Notification statistics
     */
    @lombok.Data
    @lombok.Builder
    public static class NotificationStats {
        private Long alertsReceived;
        private Long alertsProcessed;
        private Long alertsRateLimited;
        private Long alertsSuppressed;
        private Long alertsBundled;
    }
}
