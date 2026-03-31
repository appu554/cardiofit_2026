package com.cardiofit.flink.routing;

import com.cardiofit.flink.models.*;

import java.util.List;

/**
 * Notification channel selection based on alert tier.
 * Module 6 emits NotificationRequests — a separate service handles delivery.
 */
public final class NotificationRouter {

    private NotificationRouter() {}

    public static List<NotificationRequest.Channel> getChannels(ClinicalAlert alert) {
        return switch (alert.getTier()) {
            case HALT -> List.of(
                NotificationRequest.Channel.SMS,
                NotificationRequest.Channel.FCM_PUSH,
                NotificationRequest.Channel.PHONE_FALLBACK
            );
            case PAUSE -> List.of(
                NotificationRequest.Channel.FCM_PUSH,
                NotificationRequest.Channel.EMAIL
            );
            case SOFT_FLAG -> List.of(NotificationRequest.Channel.DASHBOARD_ONLY);
            case ROUTINE -> List.of(NotificationRequest.Channel.DASHBOARD_ONLY);
        };
    }
}
