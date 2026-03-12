package com.cardiofit.notifications.config;

import com.google.auth.oauth2.GoogleCredentials;
import com.google.firebase.FirebaseApp;
import com.google.firebase.FirebaseOptions;
import com.sendgrid.SendGrid;
import com.twilio.Twilio;
import jakarta.annotation.PostConstruct;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.io.FileInputStream;
import java.io.IOException;

/**
 * Configuration for notification providers (Twilio, SendGrid, Firebase)
 */
@Configuration
@Slf4j
public class NotificationProviderConfig {

    @Value("${twilio.account-sid}")
    private String twilioAccountSid;

    @Value("${twilio.auth-token}")
    private String twilioAuthToken;

    @Value("${twilio.phone-number}")
    private String twilioPhoneNumber;

    @Value("${sendgrid.api-key}")
    private String sendgridApiKey;

    @Value("${sendgrid.from-email}")
    private String sendgridFromEmail;

    @Value("${firebase.credentials-path:firebase-credentials.json}")
    private String firebaseCredentialsPath;

    @Value("${firebase.enabled:true}")
    private boolean firebaseEnabled;

    @PostConstruct
    public void initializeTwilio() {
        try {
            Twilio.init(twilioAccountSid, twilioAuthToken);
            log.info("Twilio initialized successfully");
        } catch (Exception e) {
            log.error("Failed to initialize Twilio", e);
        }
    }

    @PostConstruct
    public void initializeFirebase() {
        if (!firebaseEnabled) {
            log.info("Firebase is disabled");
            return;
        }

        try {
            if (FirebaseApp.getApps().isEmpty()) {
                FirebaseOptions options = FirebaseOptions.builder()
                        .setCredentials(GoogleCredentials.fromStream(
                                new FileInputStream(firebaseCredentialsPath)))
                        .build();
                FirebaseApp.initializeApp(options);
                log.info("Firebase initialized successfully");
            }
        } catch (IOException e) {
            log.error("Failed to initialize Firebase", e);
        }
    }

    @Bean
    public SendGrid sendGridClient() {
        return new SendGrid(sendgridApiKey);
    }

    @Bean
    public String twilioPhoneNumber() {
        return twilioPhoneNumber;
    }

    @Bean
    public String sendgridFromEmail() {
        return sendgridFromEmail;
    }
}
