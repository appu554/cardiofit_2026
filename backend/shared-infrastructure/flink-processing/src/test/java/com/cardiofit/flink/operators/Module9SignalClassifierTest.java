package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module9TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("Module 9: Signal Classifier (8-channel DD#8 reconciled)")
class Module9SignalClassifierTest {

    private static final String PID = "test-patient-001";
    private static final long NOW = System.currentTimeMillis();

    @Test
    @DisplayName("LAB_RESULT with lab_type=glucose -> GLUCOSE_MONITORING")
    void glucoseLabResult() {
        CanonicalEvent event = Module9TestBuilder.glucoseLabResult(PID, NOW);
        assertEquals(SignalType.GLUCOSE_MONITORING, Module9SignalClassifier.classify(event));
    }

    @Test
    @DisplayName("LAB_RESULT with lab_type=fbg -> GLUCOSE_MONITORING")
    void fbgLabResult() {
        CanonicalEvent event = new CanonicalEvent();
        event.setPatientId(PID);
        event.setEventType(EventType.LAB_RESULT);
        event.setEventTime(NOW);
        Map<String, Object> payload = new HashMap<>();
        payload.put("lab_type", "FBG");
        event.setPayload(payload);
        assertEquals(SignalType.GLUCOSE_MONITORING, Module9SignalClassifier.classify(event));
    }

    @Test
    @DisplayName("LAB_RESULT with lab_type=creatinine -> null (physician-ordered)")
    void nonGlucoseLabResult() {
        CanonicalEvent event = Module9TestBuilder.nonGlucoseLabResult(PID, NOW);
        assertNull(Module9SignalClassifier.classify(event));
    }

    @Test
    @DisplayName("DEVICE_READING with data_tier=TIER_1_CGM -> GLUCOSE_MONITORING")
    void cgmDeviceReading() {
        CanonicalEvent event = Module9TestBuilder.cgmDeviceReading(PID, NOW);
        assertEquals(SignalType.GLUCOSE_MONITORING, Module9SignalClassifier.classify(event));
    }

    @Test
    @DisplayName("MEDICATION_ADMINISTERED -> MEDICATION_ADHERENCE")
    void medicationAdministered() {
        CanonicalEvent event = Module9TestBuilder.medicationAdministered(PID, NOW);
        assertEquals(SignalType.MEDICATION_ADHERENCE, Module9SignalClassifier.classify(event));
    }

    @Test
    @DisplayName("MEDICATION_MISSED -> null (missed dose is NOT engagement)")
    void medicationMissedIsNotEngagement() {
        CanonicalEvent event = Module9TestBuilder.medicationMissed(PID, NOW);
        assertNull(Module9SignalClassifier.classify(event));
    }

    @Test
    @DisplayName("VITAL_SIGN with systolic_bp -> BP_MEASUREMENT")
    void bpReading() {
        CanonicalEvent event = Module9TestBuilder.bpReading(PID, NOW, 130.0, 85.0);
        assertEquals(SignalType.BP_MEASUREMENT, Module9SignalClassifier.classify(event));
    }

    @Test
    @DisplayName("VITAL_SIGN with weight only -> WEIGHT_MEASUREMENT")
    void weightOnly() {
        CanonicalEvent event = Module9TestBuilder.weightReading(PID, NOW);
        assertEquals(SignalType.WEIGHT_MEASUREMENT, Module9SignalClassifier.classify(event));
    }

    @Test
    @DisplayName("VITAL_SIGN with both systolic_bp and weight -> BP_MEASUREMENT (priority)")
    void bpTakesPriorityOverWeight() {
        CanonicalEvent event = new CanonicalEvent();
        event.setPatientId(PID);
        event.setEventType(EventType.VITAL_SIGN);
        event.setEventTime(NOW);
        Map<String, Object> payload = new HashMap<>();
        payload.put("systolic_bp", 130.0);
        payload.put("weight", 75.5);
        event.setPayload(payload);
        assertEquals(SignalType.BP_MEASUREMENT, Module9SignalClassifier.classify(event));
    }

    @Test
    @DisplayName("VITAL_SIGN without BP or weight (HR only) -> null")
    void heartRateOnly() {
        CanonicalEvent event = new CanonicalEvent();
        event.setPatientId(PID);
        event.setEventType(EventType.VITAL_SIGN);
        event.setEventTime(NOW);
        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", 72);
        event.setPayload(payload);
        assertNull(Module9SignalClassifier.classify(event));
    }

    @Test
    @DisplayName("PATIENT_REPORTED with report_type=MEAL_LOG -> MEAL_LOGGING")
    void mealLog() {
        CanonicalEvent event = Module9TestBuilder.mealLog(PID, NOW);
        assertEquals(SignalType.MEAL_LOGGING, Module9SignalClassifier.classify(event));
    }

    @Test
    @DisplayName("PATIENT_REPORTED with report_type=APP_SESSION -> APP_SESSION")
    void appSession() {
        CanonicalEvent event = Module9TestBuilder.appSession(PID, NOW);
        assertEquals(SignalType.APP_SESSION, Module9SignalClassifier.classify(event));
    }

    @Test
    @DisplayName("PATIENT_REPORTED with report_type=GOAL_COMPLETED -> GOAL_COMPLETION")
    void goalCompleted() {
        CanonicalEvent event = Module9TestBuilder.goalCompleted(PID, NOW);
        assertEquals(SignalType.GOAL_COMPLETION, Module9SignalClassifier.classify(event));
    }

    @Test
    @DisplayName("PATIENT_REPORTED with symptom_type (no report_type) -> null")
    void symptomReport() {
        CanonicalEvent event = Module9TestBuilder.symptomReport(PID, NOW);
        assertNull(Module9SignalClassifier.classify(event));
    }

    @Test
    @DisplayName("ENCOUNTER_START -> APPOINTMENT_ATTENDANCE")
    void encounterStart() {
        CanonicalEvent event = Module9TestBuilder.appointmentAttended(PID, NOW);
        assertEquals(SignalType.APPOINTMENT_ATTENDANCE, Module9SignalClassifier.classify(event));
    }

    @Test
    @DisplayName("DEVICE_READING with systolic_bp -> BP_MEASUREMENT")
    void deviceBpReading() {
        CanonicalEvent event = new CanonicalEvent();
        event.setPatientId(PID);
        event.setEventType(EventType.DEVICE_READING);
        event.setEventTime(NOW);
        Map<String, Object> payload = new HashMap<>();
        payload.put("systolic_bp", 125.0);
        event.setPayload(payload);
        assertEquals(SignalType.BP_MEASUREMENT, Module9SignalClassifier.classify(event));
    }

    @Test
    @DisplayName("null event -> null")
    void nullEvent() {
        assertNull(Module9SignalClassifier.classify(null));
    }

    @Test
    @DisplayName("Event with null payload -> null")
    void nullPayload() {
        CanonicalEvent event = new CanonicalEvent();
        event.setPatientId(PID);
        event.setEventType(EventType.VITAL_SIGN);
        event.setEventTime(NOW);
        assertNull(Module9SignalClassifier.classify(event));
    }
}
