package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.routing.NotificationRouter;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module6NotificationRoutingTest {

    @Test
    void haltAlert_getsSmsAndFcmAndPhoneFallback() {
        ClinicalAlert alert = new ClinicalAlert();
        alert.setTier(ActionTier.HALT);
        List<NotificationRequest.Channel> channels = NotificationRouter.getChannels(alert);
        assertEquals(3, channels.size());
        assertTrue(channels.contains(NotificationRequest.Channel.SMS));
        assertTrue(channels.contains(NotificationRequest.Channel.FCM_PUSH));
        assertTrue(channels.contains(NotificationRequest.Channel.PHONE_FALLBACK));
    }

    @Test
    void pauseAlert_getsFcmAndEmail() {
        ClinicalAlert alert = new ClinicalAlert();
        alert.setTier(ActionTier.PAUSE);
        List<NotificationRequest.Channel> channels = NotificationRouter.getChannels(alert);
        assertEquals(2, channels.size());
        assertTrue(channels.contains(NotificationRequest.Channel.FCM_PUSH));
        assertTrue(channels.contains(NotificationRequest.Channel.EMAIL));
    }

    @Test
    void softFlagAlert_getsDashboardOnly() {
        ClinicalAlert alert = new ClinicalAlert();
        alert.setTier(ActionTier.SOFT_FLAG);
        List<NotificationRequest.Channel> channels = NotificationRouter.getChannels(alert);
        assertEquals(1, channels.size());
        assertTrue(channels.contains(NotificationRequest.Channel.DASHBOARD_ONLY));
    }

    @Test
    void routineAlert_getsDashboardOnly() {
        ClinicalAlert alert = new ClinicalAlert();
        alert.setTier(ActionTier.ROUTINE);
        List<NotificationRequest.Channel> channels = NotificationRouter.getChannels(alert);
        assertEquals(1, channels.size());
        assertTrue(channels.contains(NotificationRequest.Channel.DASHBOARD_ONLY));
    }
}
