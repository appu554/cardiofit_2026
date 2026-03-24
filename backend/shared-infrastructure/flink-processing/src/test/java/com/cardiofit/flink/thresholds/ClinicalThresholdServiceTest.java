package com.cardiofit.flink.thresholds;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for {@link ClinicalThresholdService} and {@link ClinicalThresholdSet}.
 *
 * Validates that:
 * 1. getThresholds() returns hardcoded defaults when no KB is available
 * 2. Hardcoded default values exactly match the constants in existing Flink operators
 * 3. Three-tier resolution degrades gracefully
 */
class ClinicalThresholdServiceTest {

    private ClinicalThresholdService service;

    @BeforeEach
    void setUp() {
        service = new ClinicalThresholdService();
        service.initialize();
        // Do NOT call loadInitialThresholds -- KB endpoints are not running in unit tests
    }

    // ========================================================================
    // Tier 3 fallback: empty cache returns hardcoded defaults
    // ========================================================================

    @Test
    void getThresholds_returnsDefaults_whenCacheEmpty() {
        ClinicalThresholdSet thresholds = service.getThresholds();

        assertNotNull(thresholds, "Thresholds must never be null");
        assertNotNull(thresholds.getVitals());
        assertNotNull(thresholds.getLabs());
        assertNotNull(thresholds.getNews2());
        assertNotNull(thresholds.getMews());
        assertNotNull(thresholds.getRiskScoring());
        assertNotNull(thresholds.getHighRisk());
        assertTrue(thresholds.getVersion().startsWith("hardcoded"),
                "Default version should start with 'hardcoded'");
    }

    // ========================================================================
    // VitalThresholds -- match EnhancedRiskIndicators + SmartAlertGenerator
    // ========================================================================

    @Test
    void vitalDefaults_matchEnhancedRiskIndicators() {
        ClinicalThresholdSet.VitalThresholds v = ClinicalThresholdSet.hardcodedDefaults().getVitals();

        // EnhancedRiskIndicators constants
        assertEquals(40, v.getHrBradycardiaSevere(), "BRADYCARDIA_SEVERE");
        assertEquals(50, v.getHrBradycardiaModerate(), "BRADYCARDIA_MODERATE");
        assertEquals(60, v.getHrBradycardiaMild(), "BRADYCARDIA_MILD");
        assertEquals(100, v.getHrTachycardiaMild(), "TACHYCARDIA_MILD");
        assertEquals(110, v.getHrTachycardiaModerate(), "TACHYCARDIA_MODERATE");
        assertEquals(120, v.getHrTachycardiaSevere(), "TACHYCARDIA_SEVERE");

        // Blood pressure stages
        assertEquals(130, v.getSbpStage1(), "HTN_STAGE1_SYSTOLIC");
        assertEquals(140, v.getSbpStage2(), "HTN_STAGE2_SYSTOLIC");
        assertEquals(180, v.getSbpCrisis(), "HTN_CRISIS_SYSTOLIC");
        assertEquals(80, v.getDbpStage1(), "HTN_STAGE1_DIASTOLIC");
        assertEquals(90, v.getDbpStage2(), "HTN_STAGE2_DIASTOLIC");
        assertEquals(120, v.getDbpCrisis(), "HTN_CRISIS_DIASTOLIC");
    }

    @Test
    void vitalDefaults_matchSmartAlertGenerator() {
        ClinicalThresholdSet.VitalThresholds v = ClinicalThresholdSet.hardcodedDefaults().getVitals();

        // SmartAlertGenerator vital thresholds
        assertEquals(90, v.getSpo2AlertCritical(), "SpO2 critical alert <90");
        assertEquals(92, v.getSpo2AlertLow(), "SpO2 low alert <92");
        assertEquals(35.0, v.getTempCriticalLow(), 0.01, "Hypothermia <35.0");
        assertEquals(39.5, v.getTempHighFever(), 0.01, "High fever >=39.5");
        assertEquals(8, v.getRrCriticalLow(), "RR critical low");
        assertEquals(30, v.getRrCriticalHigh(), "RR critical high");
    }

    @Test
    void vitalDefaults_matchRiskScoreCalculator() {
        ClinicalThresholdSet.VitalThresholds v = ClinicalThresholdSet.hardcodedDefaults().getVitals();

        // RiskScoreCalculator ranges
        assertEquals(150, v.getHrCriticalHigh(), "HR critical high (RiskScoreCalculator)");
        assertEquals(70, v.getSbpHypotensionCritical(), "SBP critical low (RiskScoreCalculator)");
        assertEquals(88, v.getSpo2Critical(), "SpO2 critical (RiskScoreCalculator)");
        assertEquals(95, v.getSpo2NormalLow(), "SpO2 normal low (RiskScoreCalculator)");
    }

    // ========================================================================
    // NEWS2Params -- match NEWS2Calculator scoring bands
    // ========================================================================

    @Test
    void news2Defaults_matchNEWS2Calculator() {
        ClinicalThresholdSet.NEWS2Params n = ClinicalThresholdSet.hardcodedDefaults().getNews2();

        // RR bands: <=8 -> 3, 9-11 -> 1, 12-20 -> 0, 21-24 -> 2, >=25 -> 3
        assertEquals(8, n.getRrScore3Low());
        assertEquals(11, n.getRrScore1Low());
        assertEquals(20, n.getRrScore0High());
        assertEquals(24, n.getRrScore2High());

        // SpO2 Scale 1: <=91 -> 3, 92-93 -> 2, 94-95 -> 1, >=96 -> 0
        assertEquals(91, n.getSpo2Scale1Score3());
        assertEquals(93, n.getSpo2Scale1Score2());
        assertEquals(95, n.getSpo2Scale1Score1());

        // SBP: <=90 -> 3, 91-100 -> 2, 101-110 -> 1, 111-219 -> 0, >=220 -> 3
        assertEquals(90, n.getSbpScore3Low());
        assertEquals(100, n.getSbpScore2Low());
        assertEquals(110, n.getSbpScore1Low());
        assertEquals(219, n.getSbpScore0High());

        // HR: <=40 -> 3, 41-50 -> 1, 51-90 -> 0, 91-110 -> 1, 111-130 -> 2, >=131 -> 3
        assertEquals(40, n.getHrScore3Low());
        assertEquals(50, n.getHrScore1Low());
        assertEquals(90, n.getHrScore0High());
        assertEquals(110, n.getHrScore1High());
        assertEquals(130, n.getHrScore2High());

        // Temperature: <=35.0 -> 3, 35.1-36.0 -> 1, 36.1-38.0 -> 0, 38.1-39.0 -> 1, >=39.1 -> 2
        assertEquals(35.0, n.getTempScore3Low(), 0.01);
        assertEquals(36.0, n.getTempScore1Low(), 0.01);
        assertEquals(38.0, n.getTempScore0High(), 0.01);
        assertEquals(39.0, n.getTempScore1High(), 0.01);

        // Supplemental oxygen
        assertEquals(2, n.getSupplementalOxygenScore());

        // Risk level thresholds
        assertEquals(5, n.getMediumThreshold());
        assertEquals(7, n.getHighThreshold());
    }

    // ========================================================================
    // LabThresholds -- match RiskScoreCalculator + ClinicalScoreCalculator
    // ========================================================================

    @Test
    void labDefaults_matchRiskScoreCalculator() {
        ClinicalThresholdSet.LabThresholds l = ClinicalThresholdSet.hardcodedDefaults().getLabs();

        // Potassium
        assertEquals(3.5, l.getPotassiumNormalLow(), 0.01);
        assertEquals(5.0, l.getPotassiumNormalHigh(), 0.01);
        assertEquals(2.5, l.getPotassiumCriticalLow(), 0.01);
        assertEquals(6.0, l.getPotassiumHaltHigh(), 0.01);

        // Creatinine
        assertEquals(1.2, l.getCreatinineNormal(), 0.01);
        assertEquals(1.5, l.getCreatinineAbnormal(), 0.01);
        assertEquals(3.0, l.getCreatinineAKIStage3(), 0.01);

        // Glucose
        assertEquals(70.0, l.getGlucoseNormalLow(), 0.01);
        assertEquals(140.0, l.getGlucoseNormalHigh(), 0.01);
        assertEquals(400.0, l.getGlucoseCriticalHigh(), 0.01);

        // Lactate
        assertEquals(2.0, l.getLactateNormal(), 0.01);
        assertEquals(4.0, l.getLactateCritical(), 0.01);

        // Troponin
        assertEquals(0.04, l.getTroponinNormal(), 0.001);
        assertEquals(0.5, l.getTroponinCritical(), 0.01);

        // WBC
        assertEquals(4.0, l.getWbcCriticalLow(), 0.01);
        assertEquals(15.0, l.getWbcCriticalHigh(), 0.01);
    }

    // ========================================================================
    // RiskScoringConfig -- match RiskScoreCalculator weights and levels
    // ========================================================================

    @Test
    void riskScoringDefaults_matchRiskScoreCalculator() {
        ClinicalThresholdSet.RiskScoringConfig r = ClinicalThresholdSet.hardcodedDefaults().getRiskScoring();

        // Daily composite weights
        assertEquals(0.40, r.getVitalWeight(), 0.001, "Vital weight 40%");
        assertEquals(0.35, r.getLabWeight(), 0.001, "Lab weight 35%");
        assertEquals(0.25, r.getMedicationWeight(), 0.001, "Medication weight 25%");
        assertEquals(1.0, r.getVitalWeight() + r.getLabWeight() + r.getMedicationWeight(), 0.001,
                "Weights must sum to 1.0");

        // Scoring multipliers
        assertEquals(50.0, r.getAbnormalVitalMultiplier(), 0.01);
        assertEquals(100.0, r.getCriticalVitalMultiplier(), 0.01);
        assertEquals(40.0, r.getAbnormalLabMultiplier(), 0.01);
        assertEquals(120.0, r.getCriticalLabMultiplier(), 0.01);

        // Risk level boundaries
        assertEquals(24, r.getLowMaxScore());
        assertEquals(49, r.getModerateMaxScore());
        assertEquals(74, r.getHighMaxScore());
    }

    @Test
    void riskScoringDefaults_matchAcuityLevels() {
        ClinicalThresholdSet.RiskScoringConfig r = ClinicalThresholdSet.hardcodedDefaults().getRiskScoring();

        // ClinicalScoreCalculator acuity thresholds
        assertEquals(7.0, r.getAcuityCriticalThreshold(), 0.01);
        assertEquals(5.0, r.getAcuityHighThreshold(), 0.01);
        assertEquals(3.0, r.getAcuityMediumThreshold(), 0.01);
        assertEquals(0.70, r.getNews2AcuityWeight(), 0.01);
        assertEquals(0.30, r.getMetabolicAcuityWeight(), 0.01);
    }

    @Test
    void riskScoringDefaults_matchAlertPrioritizer() {
        ClinicalThresholdSet.RiskScoringConfig r = ClinicalThresholdSet.hardcodedDefaults().getRiskScoring();

        // AlertPrioritizer weights
        assertEquals(2.0, r.getClinicalSeverityWeight(), 0.01);
        assertEquals(1.5, r.getTimeSensitivityWeight(), 0.01);
        assertEquals(1.0, r.getPatientVulnerabilityWeight(), 0.01);
        assertEquals(1.5, r.getTrendingPatternWeight(), 0.01);
        assertEquals(0.5, r.getConfidenceScoreWeight(), 0.01);
        assertEquals(30.0, r.getMaxPriorityScore(), 0.01);
    }

    // ========================================================================
    // HighRiskCategories -- match ClinicalScoreCalculator metabolic acuity
    // ========================================================================

    @Test
    void highRiskDefaults_matchMetabolicAcuity() {
        ClinicalThresholdSet.HighRiskCategories h = ClinicalThresholdSet.hardcodedDefaults().getHighRisk();

        assertEquals(1.5, h.getMetabolicAcuityCKD(), 0.01);
        assertEquals(2.0, h.getMetabolicAcuityHF(), 0.01);
        assertEquals(1.0, h.getMetabolicAcuityDM(), 0.01);
        assertEquals(1.0, h.getMetabolicAcuityHTN(), 0.01);
        assertEquals(0.5, h.getMetabolicAcuityObesity(), 0.01);
        assertEquals(5.0, h.getMetabolicAcuityCap(), 0.01);
    }

    // ========================================================================
    // Service lifecycle
    // ========================================================================

    @Test
    void service_returnsDefaults_whenNotInitialized() {
        ClinicalThresholdService uninitService = new ClinicalThresholdService();
        // Do NOT call initialize()
        ClinicalThresholdSet result = uninitService.getThresholds();

        assertNotNull(result);
        assertTrue(result.getVersion().startsWith("hardcoded"));
    }

    @Test
    void service_isInitialized_afterInitialize() {
        assertTrue(service.isInitialized());
    }

    @Test
    void service_shutdownIsIdempotent() {
        service.shutdown();
        assertFalse(service.isInitialized());
        // Second call should not throw
        service.shutdown();
    }
}
