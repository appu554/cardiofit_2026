package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.models.protocol.SimplifiedProtocol;

import java.util.*;

/**
 * Test data factory for Module 3 CDS tests.
 */
public class Module3TestBuilder {

    public static EnrichedPatientContext hypertensiveDiabeticPatient(String patientId) {
        PatientContextState state = new PatientContextState(patientId);

        state.getLatestVitals().put("systolicbloodpressure", 155);
        state.getLatestVitals().put("diastolicbloodpressure", 95);
        state.getLatestVitals().put("heartrate", 82);
        state.getLatestVitals().put("oxygensaturation", 97);
        state.getLatestVitals().put("temperature", 37.0);
        state.getLatestVitals().put("respiratoryrate", 16);

        // Labs — use no-arg constructor + setters
        LabResult hba1c = new LabResult();
        hba1c.setLabCode("4548-4");
        hba1c.setValue(8.2);
        hba1c.setLabType("HbA1c");
        hba1c.setUnit("%");
        state.getRecentLabs().put("4548-4", hba1c);

        LabResult creatinine = new LabResult();
        creatinine.setLabCode("2160-0");
        creatinine.setValue(1.4);
        creatinine.setLabType("Creatinine");
        creatinine.setUnit("mg/dL");
        state.getRecentLabs().put("2160-0", creatinine);

        // Medication
        Medication telmi = new Medication();
        telmi.setCode("83367");
        telmi.setName("Telmisartan");
        telmi.setDosage("40mg");
        state.getActiveMedications().put("83367", telmi);

        // Chronic conditions
        Condition dm2 = new Condition();
        dm2.setCode("E11");
        dm2.setDisplay("Type 2 diabetes mellitus");
        Condition htn = new Condition();
        htn.setCode("I10");
        htn.setDisplay("Essential hypertension");
        state.setChronicConditions(Arrays.asList(dm2, htn));

        state.setNews2Score(3);
        state.setQsofaScore(0);
        state.setCombinedAcuityScore(3.2);

        PatientDemographics demo = new PatientDemographics();
        demo.setAge(58);
        demo.setGender("male");
        state.setDemographics(demo);

        state.setAllergies(Arrays.asList("Penicillin"));

        EnrichedPatientContext epc = new EnrichedPatientContext(patientId, state);
        epc.setEventType("VITAL_SIGN");
        epc.setEventTime(System.currentTimeMillis());
        epc.setDataTier("TIER_3_SMBG");
        return epc;
    }

    public static EnrichedPatientContext cgmPatient(String patientId) {
        EnrichedPatientContext epc = hypertensiveDiabeticPatient(patientId);
        epc.setDataTier("TIER_1_CGM");
        epc.getPatientState().getLatestVitals().put("glucose_cgm", 142);
        epc.getPatientState().getLatestVitals().put("glucose_trend", "RISING");
        return epc;
    }

    public static EnrichedPatientContext sepsisPatient(String patientId) {
        PatientContextState state = new PatientContextState(patientId);
        state.getLatestVitals().put("systolicbloodpressure", 88);
        state.getLatestVitals().put("heartrate", 112);
        state.getLatestVitals().put("temperature", 39.2);
        state.getLatestVitals().put("respiratoryrate", 24);
        state.getLatestVitals().put("oxygensaturation", 91);

        LabResult lactate = new LabResult();
        lactate.setLabCode("32693-4");
        lactate.setValue(4.2);
        lactate.setLabType("Lactate");
        lactate.setUnit("mmol/L");
        state.getRecentLabs().put("32693-4", lactate);

        state.setNews2Score(9);
        state.setQsofaScore(2);
        state.setCombinedAcuityScore(8.5);

        EnrichedPatientContext epc = new EnrichedPatientContext(patientId, state);
        epc.setEventType("VITAL_SIGN");
        epc.setEventTime(System.currentTimeMillis());
        epc.setDataTier("TIER_3_SMBG");
        return epc;
    }

    public static SimplifiedProtocol sepsisProtocol() {
        SimplifiedProtocol p = new SimplifiedProtocol();
        p.setProtocolId("SEPSIS-BUNDLE-V2");
        p.setName("Sepsis 3-Hour Bundle");
        p.setVersion("2.0");
        p.setCategory("SEPSIS");
        p.setSpecialty("Critical Care");
        p.setBaseConfidence(0.90);
        p.setActivationThreshold(0.70);
        p.setTriggerParameters(Arrays.asList("qsofaScore", "lactate", "temperature"));
        Map<String, Double> thresholds = new HashMap<>();
        thresholds.put("qsofaScore", 2.0);
        thresholds.put("lactate", 2.0);
        thresholds.put("temperature", 38.3);
        p.setTriggerThresholds(thresholds);
        return p;
    }

    public static SimplifiedProtocol hypertensionProtocol() {
        SimplifiedProtocol p = new SimplifiedProtocol();
        p.setProtocolId("HTN-MGMT-V3");
        p.setName("Hypertension Management Protocol");
        p.setVersion("3.0");
        p.setCategory("CARDIOLOGY");
        p.setSpecialty("Cardiology");
        p.setBaseConfidence(0.85);
        p.setActivationThreshold(0.65);
        p.setTriggerParameters(Arrays.asList("systolicbloodpressure", "diastolicbloodpressure"));
        Map<String, Double> thresholds = new HashMap<>();
        thresholds.put("systolicbloodpressure", 140.0);
        thresholds.put("diastolicbloodpressure", 90.0);
        p.setTriggerThresholds(thresholds);
        return p;
    }

    /**
     * Patient on metformin with impaired renal function (eGFR ~58).
     * Creatinine=1.4, age=58, male → CKD-EPI eGFR ≈ 58 mL/min (<60 threshold).
     * Metformin IS in RENALLY_CLEARED_MEDS, so Phase 6 should flag it.
     */
    public static EnrichedPatientContext renalImpairedMetforminPatient(String patientId) {
        PatientContextState state = new PatientContextState(patientId);

        state.getLatestVitals().put("systolicbloodpressure", 145);
        state.getLatestVitals().put("diastolicbloodpressure", 88);
        state.getLatestVitals().put("heartrate", 78);

        LabResult creatinine = new LabResult();
        creatinine.setLabCode("2160-0");
        creatinine.setValue(1.4);
        creatinine.setLabType("Creatinine");
        creatinine.setUnit("mg/dL");
        state.getRecentLabs().put("2160-0", creatinine);

        LabResult hba1c = new LabResult();
        hba1c.setLabCode("4548-4");
        hba1c.setValue(7.8);
        hba1c.setLabType("HbA1c");
        hba1c.setUnit("%");
        state.getRecentLabs().put("4548-4", hba1c);

        // Metformin — renally cleared, requires dose adjustment when eGFR <60
        Medication metformin = new Medication();
        metformin.setCode("6809");
        metformin.setName("Metformin");
        metformin.setDosage("1000mg");
        state.getActiveMedications().put("6809", metformin);

        PatientDemographics demo = new PatientDemographics();
        demo.setAge(58);
        demo.setGender("male");
        state.setDemographics(demo);

        EnrichedPatientContext epc = new EnrichedPatientContext(patientId, state);
        epc.setEventType("VITAL_SIGN");
        epc.setEventTime(System.currentTimeMillis());
        epc.setDataTier("TIER_3_SMBG");
        return epc;
    }

    /**
     * Protocol with no trigger thresholds — simulates legacy CDC protocols
     * that have category/name but no structured trigger criteria.
     */
    public static SimplifiedProtocol emptyThresholdProtocol() {
        SimplifiedProtocol p = new SimplifiedProtocol();
        p.setProtocolId("LEGACY-GENERIC-V1");
        p.setName("Generic Clinical Protocol");
        p.setVersion("1.0");
        p.setCategory("CLINICAL");
        p.setSpecialty("General");
        // Deliberately NO triggerThresholds set — uses empty map from constructor
        return p;
    }

    public static Map<String, SimplifiedProtocol> defaultProtocolMap() {
        Map<String, SimplifiedProtocol> map = new HashMap<>();
        SimplifiedProtocol sepsis = sepsisProtocol();
        SimplifiedProtocol htn = hypertensionProtocol();
        map.put(sepsis.getProtocolId(), sepsis);
        map.put(htn.getProtocolId(), htn);
        return map;
    }
}
