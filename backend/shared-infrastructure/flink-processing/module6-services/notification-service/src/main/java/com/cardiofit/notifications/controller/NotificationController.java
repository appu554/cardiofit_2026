package com.cardiofit.notifications.controller;

import com.cardiofit.notifications.model.NotificationDelivery;
import com.cardiofit.notifications.model.UserPreference;
import com.cardiofit.notifications.service.AlertFatigueTracker;
import com.cardiofit.notifications.service.DeliveryService;
import com.cardiofit.notifications.service.NotificationRouter;
import com.cardiofit.notifications.service.UserPreferenceService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.HashMap;
import java.util.Map;

/**
 * REST API for notification management
 */
@RestController
@RequestMapping("/api/v1/notifications")
@Slf4j
@RequiredArgsConstructor
public class NotificationController {

    private final NotificationRouter notificationRouter;
    private final UserPreferenceService preferenceService;
    private final DeliveryService deliveryService;
    private final AlertFatigueTracker fatigueTracker;

    /**
     * Health check endpoint
     */
    @GetMapping("/health")
    public ResponseEntity<Map<String, Object>> health() {
        Map<String, Object> health = new HashMap<>();
        health.put("status", "UP");
        health.put("service", "notification-service");
        health.put("timestamp", System.currentTimeMillis());
        return ResponseEntity.ok(health);
    }

    /**
     * Get notification statistics
     */
    @GetMapping("/stats")
    public ResponseEntity<NotificationRouter.NotificationStats> getStats() {
        return ResponseEntity.ok(notificationRouter.getStats());
    }

    /**
     * Get user preferences
     */
    @GetMapping("/preferences/{userId}")
    public ResponseEntity<UserPreference> getUserPreferences(@PathVariable String userId) {
        UserPreference prefs = preferenceService.getUserPreferences(userId);
        return ResponseEntity.ok(prefs);
    }

    /**
     * Update user preferences
     */
    @PutMapping("/preferences/{userId}")
    public ResponseEntity<Map<String, String>> updateUserPreferences(
            @PathVariable String userId,
            @RequestBody UserPreference preferences) {

        preferences.setUserId(userId);
        preferenceService.saveUserPreferences(preferences);

        Map<String, String> response = new HashMap<>();
        response.put("message", "Preferences updated successfully");
        response.put("userId", userId);

        return ResponseEntity.ok(response);
    }

    /**
     * Get rate limit status for user
     */
    @GetMapping("/rate-limit/{userId}")
    public ResponseEntity<AlertFatigueTracker.RateLimitStats> getRateLimitStats(
            @PathVariable String userId) {

        AlertFatigueTracker.RateLimitStats stats = fatigueTracker.getRateLimitStats(userId);
        return ResponseEntity.ok(stats);
    }

    /**
     * Reset rate limit for user (admin operation)
     */
    @PostMapping("/rate-limit/{userId}/reset")
    public ResponseEntity<Map<String, String>> resetRateLimit(@PathVariable String userId) {
        fatigueTracker.resetRateLimit(userId);

        Map<String, String> response = new HashMap<>();
        response.put("message", "Rate limit reset successfully");
        response.put("userId", userId);

        return ResponseEntity.ok(response);
    }

    /**
     * Get delivery record
     */
    @GetMapping("/delivery/{deliveryId}")
    public ResponseEntity<NotificationDelivery> getDeliveryRecord(@PathVariable String deliveryId) {
        NotificationDelivery delivery = deliveryService.getDeliveryRecord(deliveryId);

        if (delivery == null) {
            return ResponseEntity.notFound().build();
        }

        return ResponseEntity.ok(delivery);
    }

    /**
     * Check if user is on call
     */
    @GetMapping("/on-call/{userId}")
    public ResponseEntity<Map<String, Boolean>> checkOnCallStatus(@PathVariable String userId) {
        boolean onCall = preferenceService.isOnCall(userId);

        Map<String, Boolean> response = new HashMap<>();
        response.put("onCall", onCall);

        return ResponseEntity.ok(response);
    }

    /**
     * Exception handler
     */
    @ExceptionHandler(Exception.class)
    public ResponseEntity<Map<String, String>> handleException(Exception e) {
        log.error("API error", e);

        Map<String, String> error = new HashMap<>();
        error.put("error", e.getMessage());
        error.put("timestamp", String.valueOf(System.currentTimeMillis()));

        return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR).body(error);
    }
}
