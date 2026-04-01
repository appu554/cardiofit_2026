package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.*;
import java.util.*;

/**
 * Test data factory for Module 8 CID Engine tests.
 * Provides patient comorbidity state scenarios.
 *
 * Drug class conventions (must match Module8 evaluator constants):
 *   ACEI, ARB, SGLT2I, THIAZIDE, LOOP_DIURETIC, BETA_BLOCKER,
 *   INSULIN, SU (sulfonylurea), GLP1RA, STATIN, FINERENONE,
 *   CCB, NSAID, CORTICOSTEROID, FLUDROCORTISONE
 */
public class Module8TestBuilder {

    private static long daysAgo(int days) {
        return System.currentTimeMillis() - (long) days * 86_400_000L;
    }

    public static ComorbidityState tripleWhammyPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        state.addMedication("chlorthalidone", "THIAZIDE", 12.5);
        state.addToRollingBuffer("weight", 75.0, daysAgo(7));
        state.addToRollingBuffer("weight", 72.0, System.currentTimeMillis());
        state.setEGFR14dAgo(60.0);
        state.setEGFR14dAgoTimestamp(daysAgo(14));
        state.setEGFRCurrent(48.0);
        state.setEGFRCurrentTimestamp(System.currentTimeMillis());
        return state;
    }

    public static ComorbidityState tripleWhammyNoPrecipitant(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        state.addMedication("chlorthalidone", "THIAZIDE", 12.5);
        state.addToRollingBuffer("weight", 74.5, daysAgo(7));
        state.addToRollingBuffer("weight", 74.0, System.currentTimeMillis());
        state.setEGFR14dAgo(60.0);
        state.setEGFR14dAgoTimestamp(daysAgo(14));
        state.setEGFRCurrent(58.0);
        state.setEGFRCurrentTimestamp(System.currentTimeMillis());
        return state;
    }

    public static ComorbidityState hyperkalemiaPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("finerenone", "FINERENONE", 20.0);
        state.setPreviousPotassium(5.1); // K+ rising: 5.1 → 5.5
        state.updateLab("potassium", 5.5);
        return state;
    }

    public static ComorbidityState hypoMaskingPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("insulin glargine", "INSULIN", 30.0);
        state.addMedication("metoprolol", "BETA_BLOCKER", 100.0);
        state.setLatestGlucose(55.0);
        state.setSymptomReportedHypoglycemia(false);
        return state;
    }

    public static ComorbidityState hypoWithSymptoms(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("insulin glargine", "INSULIN", 30.0);
        state.addMedication("metoprolol", "BETA_BLOCKER", 100.0);
        state.setLatestGlucose(55.0);
        state.setSymptomReportedHypoglycemia(true);
        return state;
    }

    public static ComorbidityState euglycemicDKAPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("dapagliflozin", "SGLT2I", 10.0);
        state.setSymptomReportedNauseaVomiting(true);
        state.setLatestGlucose(140.0);
        return state;
    }

    public static ComorbidityState severeHypotensionPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("amlodipine", "CCB", 10.0);
        state.addMedication("chlorthalidone", "THIAZIDE", 12.5);
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        state.setLatestSBP(92.0);
        return state;
    }

    public static ComorbidityState thiazideGlucosePatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("hydrochlorothiazide", "THIAZIDE", 25.0);
        state.addToRollingBuffer("fbg", 110.0, daysAgo(14));
        state.addToRollingBuffer("fbg", 128.0, System.currentTimeMillis());
        return state;
    }

    public static ComorbidityState eGFRDeclinePatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("telmisartan", "ARB", 80.0);
        state.setEGFRBaseline(65.0);
        state.setEGFRCurrent(45.0);
        state.setEGFRBaselineTimestamp(System.currentTimeMillis() - 56 * 86400000L);
        state.setEGFRCurrentTimestamp(System.currentTimeMillis());
        return state;
    }

    public static ComorbidityState statinMyopathyPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("atorvastatin", "STATIN", 40.0);
        state.setSymptomReportedMusclePain(true);
        return state;
    }

    public static ComorbidityState glp1raGIPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("semaglutide", "GLP1RA", 0.5);
        state.setSymptomReportedNauseaVomiting(true);
        state.setSymptomNauseaOnsetTimestamp(daysAgo(3));
        state.addToRollingBuffer("weight", 80.0, daysAgo(7));
        state.addToRollingBuffer("weight", 78.0, System.currentTimeMillis());
        return state;
    }

    public static ComorbidityState concurrentDeteriorationPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("metformin", "METFORMIN", 1000.0);
        state.addMedication("amlodipine", "CCB", 5.0);
        state.addToRollingBuffer("fbg", 135.0, daysAgo(14));
        state.addToRollingBuffer("fbg", 165.0, System.currentTimeMillis());
        state.addToRollingBuffer("sbp", 138.0, daysAgo(14));
        state.addToRollingBuffer("sbp", 152.0, System.currentTimeMillis());
        state.setLastMedicationChangeTimestamp(daysAgo(30));
        return state;
    }

    public static ComorbidityState elderlyIntensiveBPPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.setAge(78);
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("amlodipine", "CCB", 10.0);
        state.setEGFRCurrent(42.0);
        state.setFallsHistory(true);
        return state;
    }

    public static ComorbidityState metforminEGFRPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("metformin", "METFORMIN", 2000.0);
        state.setEGFRCurrent(32.0);
        state.setEGFRBaseline(45.0);
        state.setEGFRBaselineTimestamp(System.currentTimeMillis() - 180 * 86400000L);
        state.setEGFRCurrentTimestamp(System.currentTimeMillis());
        return state;
    }

    public static ComorbidityState sglt2iNsaidPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        state.addMedication("ibuprofen", "NSAID", 400.0);
        return state;
    }

    public static ComorbidityState polypharmacyPatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("metformin", "METFORMIN", 1000.0);
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        state.addMedication("semaglutide", "GLP1RA", 1.0);
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("amlodipine", "CCB", 10.0);
        state.addMedication("atorvastatin", "STATIN", 40.0);
        state.addMedication("aspirin", "ANTIPLATELET", 75.0);
        state.addMedication("omeprazole", "PPI", 20.0);
        state.addMedication("levothyroxine", "THYROID", 100.0);
        return state;
    }

    public static ComorbidityState safePatient(String patientId) {
        ComorbidityState state = new ComorbidityState(patientId);
        state.addMedication("metformin", "METFORMIN", 1000.0);
        state.addMedication("atorvastatin", "STATIN", 20.0);
        state.setAge(55);
        state.setLatestSBP(128.0);
        state.setLatestGlucose(108.0);
        state.setEGFRCurrent(82.0);
        state.updateLab("potassium", 4.2);
        state.addToRollingBuffer("fbg", 110.0, daysAgo(14));
        state.addToRollingBuffer("fbg", 112.0, System.currentTimeMillis());
        state.addToRollingBuffer("sbp", 129.0, daysAgo(14));
        state.addToRollingBuffer("sbp", 130.0, System.currentTimeMillis());
        return state;
    }
}
