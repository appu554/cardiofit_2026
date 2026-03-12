package com.cardiofit.notifications;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.context.properties.ConfigurationPropertiesScan;
import org.springframework.cache.annotation.EnableCaching;
import org.springframework.kafka.annotation.EnableKafka;
import org.springframework.scheduling.annotation.EnableAsync;
import org.springframework.scheduling.annotation.EnableScheduling;

/**
 * CardioFit Notification Service Application
 *
 * Multi-channel notification delivery service with:
 * - Kafka consumer for composed-alerts topic
 * - SMS (Twilio), Email (SendGrid), Push (Firebase), Pager delivery
 * - Alert fatigue mitigation (rate limiting, deduplication, bundling)
 * - User preference management
 * - On-call schedule integration
 * - Delivery tracking and monitoring
 */
@SpringBootApplication
@EnableKafka
@EnableCaching
@EnableAsync
@EnableScheduling
@ConfigurationPropertiesScan
public class NotificationServiceApplication {

    public static void main(String[] args) {
        SpringApplication.run(NotificationServiceApplication.class, args);
    }
}
