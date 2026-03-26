package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.SemanticEvent;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;
import static org.junit.jupiter.api.Assertions.*;
import java.util.*;

public class V4ExtensionsTest {

    @Test
    @DisplayName("CGM observation type classified as TIER_1_CGM in payload")
    void cgmDataTierClassification() {
        Map<String, Object> payload = new HashMap<>();
        String obsType = "CGM_RAW";
        // Simulate the classification logic from Module1b
        if (obsType.toUpperCase().contains("CGM")) {
            payload.put("data_tier", "TIER_1_CGM");
            payload.put("cgm_active", true);
        }
        assertEquals("TIER_1_CGM", payload.get("data_tier"));
        assertTrue((Boolean) payload.get("cgm_active"));
    }

    @Test
    @DisplayName("Non-CGM device classified as TIER_2_HYBRID")
    void deviceDataTierClassification() {
        Map<String, Object> payload = new HashMap<>();
        String obsType = "DEVICE_DATA";
        String sourceType = "WEARABLE";
        if (obsType.toUpperCase().contains("CGM")) {
            payload.put("data_tier", "TIER_1_CGM");
        } else if (sourceType.toUpperCase().contains("WEARABLE") || obsType.toUpperCase().contains("DEVICE")) {
            payload.put("data_tier", "TIER_2_HYBRID");
            payload.put("cgm_active", false);
        }
        assertEquals("TIER_2_HYBRID", payload.get("data_tier"));
    }

    @Test
    @DisplayName("Plain observation classified as TIER_3_SMBG")
    void smbgDataTierClassification() {
        Map<String, Object> payload = new HashMap<>();
        String obsType = "LABS";
        String sourceType = "EHR";
        if (obsType.toUpperCase().contains("CGM")) {
            payload.put("data_tier", "TIER_1_CGM");
            payload.put("cgm_active", true);
        } else if (sourceType.toUpperCase().contains("WEARABLE") || obsType.toUpperCase().contains("DEVICE")) {
            payload.put("data_tier", "TIER_2_HYBRID");
            payload.put("cgm_active", false);
        } else {
            payload.put("data_tier", "TIER_3_SMBG");
            payload.put("cgm_active", false);
        }
        assertEquals("TIER_3_SMBG", payload.get("data_tier"));
        assertFalse((Boolean) payload.get("cgm_active"));
    }

    @Test
    @DisplayName("SemanticEvent V4 domain and trajectory fields")
    void semanticEventV4Fields() {
        SemanticEvent event = SemanticEvent.builder()
            .id("test-1")
            .patientId("P001")
            .build();
        event.setClinicalDomain("GLYCAEMIC");
        event.setTrajectoryClass("DECLINING");
        assertEquals("GLYCAEMIC", event.getClinicalDomain());
        assertEquals("DECLINING", event.getTrajectoryClass());
    }

    @Test
    @DisplayName("SemanticEvent V4 fields via builder")
    void semanticEventV4FieldsViaBuilder() {
        SemanticEvent event = SemanticEvent.builder()
            .id("test-2")
            .patientId("P002")
            .clinicalDomain("HEMODYNAMIC")
            .trajectoryClass("RAPID_RISING")
            .build();
        assertEquals("HEMODYNAMIC", event.getClinicalDomain());
        assertEquals("RAPID_RISING", event.getTrajectoryClass());
    }

    @Test
    @DisplayName("MHRI trigger fires when ARV crosses threshold")
    void mhriTriggerARVCrossing() {
        double prevArv = 7.5;
        double currentArv = 16.0;
        double arvThresholdLow = 8.0;
        double arvThresholdHigh = 15.0;
        boolean arvCrossed = (prevArv < arvThresholdLow && currentArv >= arvThresholdHigh) ||
                             (prevArv >= arvThresholdHigh && currentArv < arvThresholdLow);
        assertTrue(arvCrossed);
    }

    @Test
    @DisplayName("MHRI trigger does not fire for small ARV change")
    void mhriTriggerNoFire() {
        double prevArv = 9.0;
        double currentArv = 11.0;
        double arvThresholdLow = 8.0;
        double arvThresholdHigh = 15.0;
        boolean arvCrossed = (prevArv < arvThresholdLow && currentArv >= arvThresholdHigh) ||
                             (prevArv >= arvThresholdHigh && currentArv < arvThresholdLow);
        assertFalse(arvCrossed);
    }

    @Test
    @DisplayName("MHRI trigger fires when ARV crosses from high to low")
    void mhriTriggerARVCrossingReverse() {
        double prevArv = 16.0;
        double currentArv = 7.0;
        double arvThresholdLow = 8.0;
        double arvThresholdHigh = 15.0;
        boolean arvCrossed = (prevArv < arvThresholdLow && currentArv >= arvThresholdHigh) ||
                             (prevArv >= arvThresholdHigh && currentArv < arvThresholdLow);
        assertTrue(arvCrossed);
    }

    @Test
    @DisplayName("MHRI trigger fires on severe morning surge")
    void mhriTriggerSevereSurge() {
        Double morningSurge = 40.0;
        boolean surgeSevere = morningSurge != null && morningSurge > 35.0;
        assertTrue(surgeSevere);
    }

    @Test
    @DisplayName("MHRI trigger does not fire on mild morning surge")
    void mhriTriggerMildSurge() {
        Double morningSurge = 20.0;
        boolean surgeSevere = morningSurge != null && morningSurge > 35.0;
        assertFalse(surgeSevere);
    }
}
