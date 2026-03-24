package com.cardiofit.flink.thresholds;

import java.io.Serializable;

/**
 * Centralized POJO holding all clinical threshold groups used across Flink operators.
 *
 * Replaces 50+ hardcoded threshold constants scattered across NEWS2Calculator,
 * ClinicalScoreCalculator, SmartAlertGenerator, RiskScoreCalculator, EnhancedRiskIndicators,
 * and AlertPrioritizer.
 *
 * Data sources:
 * - vitals:      KB-4  /v1/thresholds/vitals
 * - labs:        KB-20 /api/v1/thresholds/labs
 * - news2/mews:  KB-4  /v1/thresholds/early-warning-scores
 * - riskScoring: KB-23 /api/v1/config/risk-scoring
 * - highRisk:    KB-1  /v1/high-risk/categories
 *
 * Tier 3 fallback: {@link #hardcodedDefaults()} returns the exact values currently
 * hardcoded in each operator, guaranteeing zero regression if KB services are unavailable.
 */
public class ClinicalThresholdSet implements Serializable {
    private static final long serialVersionUID = 1L;

    private VitalThresholds vitals;
    private LabThresholds labs;
    private NEWS2Params news2;
    private MEWSParams mews;
    private RiskScoringConfig riskScoring;
    private HighRiskCategories highRisk;
    private String version;
    private long loadedAtEpochMs;

    // ---- Getters / Setters ----

    public VitalThresholds getVitals() { return vitals; }
    public void setVitals(VitalThresholds vitals) { this.vitals = vitals; }

    public LabThresholds getLabs() { return labs; }
    public void setLabs(LabThresholds labs) { this.labs = labs; }

    public NEWS2Params getNews2() { return news2; }
    public void setNews2(NEWS2Params news2) { this.news2 = news2; }

    public MEWSParams getMews() { return mews; }
    public void setMews(MEWSParams mews) { this.mews = mews; }

    public RiskScoringConfig getRiskScoring() { return riskScoring; }
    public void setRiskScoring(RiskScoringConfig riskScoring) { this.riskScoring = riskScoring; }

    public HighRiskCategories getHighRisk() { return highRisk; }
    public void setHighRisk(HighRiskCategories highRisk) { this.highRisk = highRisk; }

    public String getVersion() { return version; }
    public void setVersion(String version) { this.version = version; }

    public long getLoadedAtEpochMs() { return loadedAtEpochMs; }
    public void setLoadedAtEpochMs(long loadedAtEpochMs) { this.loadedAtEpochMs = loadedAtEpochMs; }

    // ========================================================================
    // Tier 3 fallback -- MUST match the values currently hardcoded in operators
    // ========================================================================

    /**
     * Returns the exact threshold values currently hardcoded across Flink operators.
     * This is the Tier 3 (last-resort) fallback that guarantees zero clinical regression
     * when KB services are unreachable and both L1 and L2 caches are empty.
     */
    public static ClinicalThresholdSet hardcodedDefaults() {
        ClinicalThresholdSet set = new ClinicalThresholdSet();
        set.setVersion("hardcoded-v1.0.0");
        set.setLoadedAtEpochMs(System.currentTimeMillis());
        set.setVitals(VitalThresholds.defaults());
        set.setLabs(LabThresholds.defaults());
        set.setNews2(NEWS2Params.defaults());
        set.setMews(MEWSParams.defaults());
        set.setRiskScoring(RiskScoringConfig.defaults());
        set.setHighRisk(HighRiskCategories.defaults());
        return set;
    }

    // ========================================================================
    // Inner classes -- one per threshold group
    // ========================================================================

    /**
     * Vital sign thresholds used by EnhancedRiskIndicators, SmartAlertGenerator,
     * and RiskScoreCalculator.
     */
    public static class VitalThresholds implements Serializable {
        private static final long serialVersionUID = 1L;

        // Heart rate (bpm) -- from EnhancedRiskIndicators
        private int hrBradycardiaSevere;   // 40
        private int hrBradycardiaModerate; // 50
        private int hrBradycardiaMild;     // 60
        private int hrTachycardiaMild;     // 100
        private int hrTachycardiaModerate; // 110
        private int hrTachycardiaSevere;   // 120
        private int hrCriticalHigh;        // 150  (RiskScoreCalculator)

        // Systolic blood pressure (mmHg) -- from EnhancedRiskIndicators
        private int sbpHypotensionCritical; // 70  (RiskScoreCalculator)
        private int sbpHypotension;         // 90  (RiskScoreCalculator)
        private int sbpNormalHigh;          // 140 (RiskScoreCalculator)
        private int sbpStage1;              // 130 (EnhancedRiskIndicators)
        private int sbpStage2;              // 140 (EnhancedRiskIndicators)
        private int sbpCrisis;              // 180

        // Diastolic blood pressure (mmHg)
        private int dbpStage1;  // 80
        private int dbpStage2;  // 90
        private int dbpCrisis;  // 120

        // SpO2 (%) -- from SmartAlertGenerator
        private int spo2Critical;       // 88 (RiskScoreCalculator)
        private int spo2AlertCritical;  // 90 (SmartAlertGenerator)
        private int spo2AlertLow;       // 92 (SmartAlertGenerator)
        private int spo2NormalLow;      // 95 (RiskScoreCalculator)

        // Respiratory rate (/min) -- from SmartAlertGenerator + RiskScoreCalculator
        private int rrCriticalLow;   // 8
        private int rrNormalLow;     // 12
        private int rrNormalHigh;    // 20
        private int rrCriticalHigh;  // 30

        // Temperature (Celsius) -- from SmartAlertGenerator + RiskScoreCalculator
        private double tempCriticalLow;    // 35.0
        private double tempNormalLow;      // 36.1
        private double tempNormalHigh;     // 37.8
        private double tempCriticalHigh;   // 39.0
        private double tempHighFever;      // 39.5 (SmartAlertGenerator)

        // Freshness thresholds (ms)
        private long freshThresholdMs;  // 4 hours
        private long staleThresholdMs;  // 24 hours

        public static VitalThresholds defaults() {
            VitalThresholds v = new VitalThresholds();
            // Heart rate
            v.hrBradycardiaSevere = 40;
            v.hrBradycardiaModerate = 50;
            v.hrBradycardiaMild = 60;
            v.hrTachycardiaMild = 100;
            v.hrTachycardiaModerate = 110;
            v.hrTachycardiaSevere = 120;
            v.hrCriticalHigh = 150;
            // SBP
            v.sbpHypotensionCritical = 70;
            v.sbpHypotension = 90;
            v.sbpNormalHigh = 140;
            v.sbpStage1 = 130;
            v.sbpStage2 = 140;
            v.sbpCrisis = 180;
            // DBP
            v.dbpStage1 = 80;
            v.dbpStage2 = 90;
            v.dbpCrisis = 120;
            // SpO2
            v.spo2Critical = 88;
            v.spo2AlertCritical = 90;
            v.spo2AlertLow = 92;
            v.spo2NormalLow = 95;
            // RR
            v.rrCriticalLow = 8;
            v.rrNormalLow = 12;
            v.rrNormalHigh = 20;
            v.rrCriticalHigh = 30;
            // Temperature
            v.tempCriticalLow = 35.0;
            v.tempNormalLow = 36.1;
            v.tempNormalHigh = 37.8;
            v.tempCriticalHigh = 39.0;
            v.tempHighFever = 39.5;
            // Freshness
            v.freshThresholdMs = 4L * 60 * 60 * 1000;  // 4 hours
            v.staleThresholdMs = 24L * 60 * 60 * 1000;  // 24 hours
            return v;
        }

        // Getters and setters
        public int getHrBradycardiaSevere() { return hrBradycardiaSevere; }
        public void setHrBradycardiaSevere(int v) { this.hrBradycardiaSevere = v; }
        public int getHrBradycardiaModerate() { return hrBradycardiaModerate; }
        public void setHrBradycardiaModerate(int v) { this.hrBradycardiaModerate = v; }
        public int getHrBradycardiaMild() { return hrBradycardiaMild; }
        public void setHrBradycardiaMild(int v) { this.hrBradycardiaMild = v; }
        public int getHrTachycardiaMild() { return hrTachycardiaMild; }
        public void setHrTachycardiaMild(int v) { this.hrTachycardiaMild = v; }
        public int getHrTachycardiaModerate() { return hrTachycardiaModerate; }
        public void setHrTachycardiaModerate(int v) { this.hrTachycardiaModerate = v; }
        public int getHrTachycardiaSevere() { return hrTachycardiaSevere; }
        public void setHrTachycardiaSevere(int v) { this.hrTachycardiaSevere = v; }
        public int getHrCriticalHigh() { return hrCriticalHigh; }
        public void setHrCriticalHigh(int v) { this.hrCriticalHigh = v; }
        public int getSbpHypotensionCritical() { return sbpHypotensionCritical; }
        public void setSbpHypotensionCritical(int v) { this.sbpHypotensionCritical = v; }
        public int getSbpHypotension() { return sbpHypotension; }
        public void setSbpHypotension(int v) { this.sbpHypotension = v; }
        public int getSbpNormalHigh() { return sbpNormalHigh; }
        public void setSbpNormalHigh(int v) { this.sbpNormalHigh = v; }
        public int getSbpStage1() { return sbpStage1; }
        public void setSbpStage1(int v) { this.sbpStage1 = v; }
        public int getSbpStage2() { return sbpStage2; }
        public void setSbpStage2(int v) { this.sbpStage2 = v; }
        public int getSbpCrisis() { return sbpCrisis; }
        public void setSbpCrisis(int v) { this.sbpCrisis = v; }
        public int getDbpStage1() { return dbpStage1; }
        public void setDbpStage1(int v) { this.dbpStage1 = v; }
        public int getDbpStage2() { return dbpStage2; }
        public void setDbpStage2(int v) { this.dbpStage2 = v; }
        public int getDbpCrisis() { return dbpCrisis; }
        public void setDbpCrisis(int v) { this.dbpCrisis = v; }
        public int getSpo2Critical() { return spo2Critical; }
        public void setSpo2Critical(int v) { this.spo2Critical = v; }
        public int getSpo2AlertCritical() { return spo2AlertCritical; }
        public void setSpo2AlertCritical(int v) { this.spo2AlertCritical = v; }
        public int getSpo2AlertLow() { return spo2AlertLow; }
        public void setSpo2AlertLow(int v) { this.spo2AlertLow = v; }
        public int getSpo2NormalLow() { return spo2NormalLow; }
        public void setSpo2NormalLow(int v) { this.spo2NormalLow = v; }
        public int getRrCriticalLow() { return rrCriticalLow; }
        public void setRrCriticalLow(int v) { this.rrCriticalLow = v; }
        public int getRrNormalLow() { return rrNormalLow; }
        public void setRrNormalLow(int v) { this.rrNormalLow = v; }
        public int getRrNormalHigh() { return rrNormalHigh; }
        public void setRrNormalHigh(int v) { this.rrNormalHigh = v; }
        public int getRrCriticalHigh() { return rrCriticalHigh; }
        public void setRrCriticalHigh(int v) { this.rrCriticalHigh = v; }
        public double getTempCriticalLow() { return tempCriticalLow; }
        public void setTempCriticalLow(double v) { this.tempCriticalLow = v; }
        public double getTempNormalLow() { return tempNormalLow; }
        public void setTempNormalLow(double v) { this.tempNormalLow = v; }
        public double getTempNormalHigh() { return tempNormalHigh; }
        public void setTempNormalHigh(double v) { this.tempNormalHigh = v; }
        public double getTempCriticalHigh() { return tempCriticalHigh; }
        public void setTempCriticalHigh(double v) { this.tempCriticalHigh = v; }
        public double getTempHighFever() { return tempHighFever; }
        public void setTempHighFever(double v) { this.tempHighFever = v; }
        public long getFreshThresholdMs() { return freshThresholdMs; }
        public void setFreshThresholdMs(long v) { this.freshThresholdMs = v; }
        public long getStaleThresholdMs() { return staleThresholdMs; }
        public void setStaleThresholdMs(long v) { this.staleThresholdMs = v; }
    }

    /**
     * Lab thresholds used by RiskScoreCalculator and ClinicalScoreCalculator.
     */
    public static class LabThresholds implements Serializable {
        private static final long serialVersionUID = 1L;

        // Potassium (mEq/L) -- from RiskScoreCalculator
        private double potassiumNormalLow;    // 3.5
        private double potassiumNormalHigh;   // 5.0
        private double potassiumAlertHigh;    // 5.5 (from spec)
        private double potassiumCriticalLow;  // 2.5
        private double potassiumHaltHigh;     // 6.0

        // Creatinine (mg/dL) -- from RiskScoreCalculator + ClinicalScoreCalculator
        private double creatinineNormal;       // 1.2
        private double creatinineAbnormal;     // 1.5 (ClinicalScoreCalculator metabolic acuity)
        private double creatinineAKIStage3;    // 3.0

        // Glucose (mg/dL)
        private double glucoseNormalLow;   // 70
        private double glucoseNormalHigh;  // 140
        private double glucoseCriticalHigh; // 400
        private double glucoseFasting;      // 100 (metabolic syndrome)

        // Lactate (mmol/L)
        private double lactateNormal;    // 2.0
        private double lactateCritical;  // 4.0

        // Troponin (ng/mL)
        private double troponinNormal;    // 0.04
        private double troponinCritical;  // 0.5

        // WBC (K/uL)
        private double wbcNormalLow;    // 4.0
        private double wbcNormalHigh;   // 11.0
        private double wbcCriticalLow;  // 4.0
        private double wbcCriticalHigh; // 15.0

        // HbA1c (%)
        private double hba1cPrediabetes;  // 5.7
        private double hba1cDiabetes;     // 6.5

        // Lipids
        private double triglyceridesHigh;  // 150 mg/dL
        private double hdlLowMale;         // 40 mg/dL
        private double hdlLowFemale;       // 50 mg/dL

        public static LabThresholds defaults() {
            LabThresholds l = new LabThresholds();
            l.potassiumNormalLow = 3.5;
            l.potassiumNormalHigh = 5.0;
            l.potassiumAlertHigh = 5.5;
            l.potassiumCriticalLow = 2.5;
            l.potassiumHaltHigh = 6.0;
            l.creatinineNormal = 1.2;
            l.creatinineAbnormal = 1.5;
            l.creatinineAKIStage3 = 3.0;
            l.glucoseNormalLow = 70.0;
            l.glucoseNormalHigh = 140.0;
            l.glucoseCriticalHigh = 400.0;
            l.glucoseFasting = 100.0;
            l.lactateNormal = 2.0;
            l.lactateCritical = 4.0;
            l.troponinNormal = 0.04;
            l.troponinCritical = 0.5;
            l.wbcNormalLow = 4.0;
            l.wbcNormalHigh = 11.0;
            l.wbcCriticalLow = 4.0;
            l.wbcCriticalHigh = 15.0;
            l.hba1cPrediabetes = 5.7;
            l.hba1cDiabetes = 6.5;
            l.triglyceridesHigh = 150.0;
            l.hdlLowMale = 40.0;
            l.hdlLowFemale = 50.0;
            return l;
        }

        // Getters and setters
        public double getPotassiumNormalLow() { return potassiumNormalLow; }
        public void setPotassiumNormalLow(double v) { this.potassiumNormalLow = v; }
        public double getPotassiumNormalHigh() { return potassiumNormalHigh; }
        public void setPotassiumNormalHigh(double v) { this.potassiumNormalHigh = v; }
        public double getPotassiumAlertHigh() { return potassiumAlertHigh; }
        public void setPotassiumAlertHigh(double v) { this.potassiumAlertHigh = v; }
        public double getPotassiumCriticalLow() { return potassiumCriticalLow; }
        public void setPotassiumCriticalLow(double v) { this.potassiumCriticalLow = v; }
        public double getPotassiumHaltHigh() { return potassiumHaltHigh; }
        public void setPotassiumHaltHigh(double v) { this.potassiumHaltHigh = v; }
        public double getCreatinineNormal() { return creatinineNormal; }
        public void setCreatinineNormal(double v) { this.creatinineNormal = v; }
        public double getCreatinineAbnormal() { return creatinineAbnormal; }
        public void setCreatinineAbnormal(double v) { this.creatinineAbnormal = v; }
        public double getCreatinineAKIStage3() { return creatinineAKIStage3; }
        public void setCreatinineAKIStage3(double v) { this.creatinineAKIStage3 = v; }
        public double getGlucoseNormalLow() { return glucoseNormalLow; }
        public void setGlucoseNormalLow(double v) { this.glucoseNormalLow = v; }
        public double getGlucoseNormalHigh() { return glucoseNormalHigh; }
        public void setGlucoseNormalHigh(double v) { this.glucoseNormalHigh = v; }
        public double getGlucoseCriticalHigh() { return glucoseCriticalHigh; }
        public void setGlucoseCriticalHigh(double v) { this.glucoseCriticalHigh = v; }
        public double getGlucoseFasting() { return glucoseFasting; }
        public void setGlucoseFasting(double v) { this.glucoseFasting = v; }
        public double getLactateNormal() { return lactateNormal; }
        public void setLactateNormal(double v) { this.lactateNormal = v; }
        public double getLactateCritical() { return lactateCritical; }
        public void setLactateCritical(double v) { this.lactateCritical = v; }
        public double getTroponinNormal() { return troponinNormal; }
        public void setTroponinNormal(double v) { this.troponinNormal = v; }
        public double getTroponinCritical() { return troponinCritical; }
        public void setTroponinCritical(double v) { this.troponinCritical = v; }
        public double getWbcNormalLow() { return wbcNormalLow; }
        public void setWbcNormalLow(double v) { this.wbcNormalLow = v; }
        public double getWbcNormalHigh() { return wbcNormalHigh; }
        public void setWbcNormalHigh(double v) { this.wbcNormalHigh = v; }
        public double getWbcCriticalLow() { return wbcCriticalLow; }
        public void setWbcCriticalLow(double v) { this.wbcCriticalLow = v; }
        public double getWbcCriticalHigh() { return wbcCriticalHigh; }
        public void setWbcCriticalHigh(double v) { this.wbcCriticalHigh = v; }
        public double getHba1cPrediabetes() { return hba1cPrediabetes; }
        public void setHba1cPrediabetes(double v) { this.hba1cPrediabetes = v; }
        public double getHba1cDiabetes() { return hba1cDiabetes; }
        public void setHba1cDiabetes(double v) { this.hba1cDiabetes = v; }
        public double getTriglyceridesHigh() { return triglyceridesHigh; }
        public void setTriglyceridesHigh(double v) { this.triglyceridesHigh = v; }
        public double getHdlLowMale() { return hdlLowMale; }
        public void setHdlLowMale(double v) { this.hdlLowMale = v; }
        public double getHdlLowFemale() { return hdlLowFemale; }
        public void setHdlLowFemale(double v) { this.hdlLowFemale = v; }
    }

    /**
     * NEWS2 scoring parameters -- from NEWS2Calculator.
     *
     * Each band array encodes {upperBound, score} pairs evaluated top-to-bottom.
     * For simplicity the defaults method encodes the exact band boundaries from
     * NEWS2Calculator as discrete fields rather than arrays, keeping the POJO
     * straightforward for JSON (de)serialization from KB-4.
     */
    public static class NEWS2Params implements Serializable {
        private static final long serialVersionUID = 1L;

        // Respiratory Rate bands
        private int rrScore3Low;    // <=8
        private int rrScore1Low;    // 9-11
        private int rrScore0High;   // 12-20
        private int rrScore2High;   // 21-24
        // >=25 => score 3

        // SpO2 Scale 1 (on air)
        private int spo2Scale1Score3; // <=91
        private int spo2Scale1Score2; // 92-93
        private int spo2Scale1Score1; // 94-95
        // >=96 => score 0

        // SpO2 Scale 2 (on oxygen) -- from ClinicalScoreCalculator
        private int spo2Scale2Score3Low;  // <=92
        private int spo2Scale2Score3High; // >=97
        private int spo2Scale2Score2Low;  // 93-94
        private int spo2Scale2Score1Low;  // 95-96

        // Systolic BP
        private int sbpScore3Low;   // <=90
        private int sbpScore2Low;   // 91-100
        private int sbpScore1Low;   // 101-110
        private int sbpScore0High;  // 111-219
        // >=220 => score 3

        // Heart Rate
        private int hrScore3Low;    // <=40
        private int hrScore1Low;    // 41-50
        private int hrScore0High;   // 51-90
        private int hrScore1High;   // 91-110
        private int hrScore2High;   // 111-130
        // >=131 => score 3

        // Temperature (Celsius)
        private double tempScore3Low;   // <=35.0
        private double tempScore1Low;   // 35.1-36.0
        private double tempScore0High;  // 36.1-38.0
        private double tempScore1High;  // 38.1-39.0
        // >=39.1 => score 2

        // Consciousness: Alert=0, V/P/U=3 (no configurable thresholds)

        // Supplemental oxygen score
        private int supplementalOxygenScore; // 2

        // Risk level thresholds
        private int lowMediumThreshold;  // individual score of 3
        private int mediumThreshold;     // total 5
        private int highThreshold;       // total 7

        public static NEWS2Params defaults() {
            NEWS2Params p = new NEWS2Params();
            // RR
            p.rrScore3Low = 8;
            p.rrScore1Low = 11;  // 9-11
            p.rrScore0High = 20; // 12-20
            p.rrScore2High = 24; // 21-24
            // SpO2 Scale 1
            p.spo2Scale1Score3 = 91;
            p.spo2Scale1Score2 = 93;
            p.spo2Scale1Score1 = 95;
            // SpO2 Scale 2
            p.spo2Scale2Score3Low = 92;
            p.spo2Scale2Score3High = 97;
            p.spo2Scale2Score2Low = 94;  // 93-94
            p.spo2Scale2Score1Low = 96;  // 95-96
            // SBP
            p.sbpScore3Low = 90;
            p.sbpScore2Low = 100;
            p.sbpScore1Low = 110;
            p.sbpScore0High = 219;
            // HR
            p.hrScore3Low = 40;
            p.hrScore1Low = 50;
            p.hrScore0High = 90;
            p.hrScore1High = 110;
            p.hrScore2High = 130;
            // Temperature
            p.tempScore3Low = 35.0;
            p.tempScore1Low = 36.0;
            p.tempScore0High = 38.0;
            p.tempScore1High = 39.0;
            // Supplemental oxygen
            p.supplementalOxygenScore = 2;
            // Risk levels
            p.lowMediumThreshold = 3;
            p.mediumThreshold = 5;
            p.highThreshold = 7;
            return p;
        }

        // Getters and setters
        public int getRrScore3Low() { return rrScore3Low; }
        public void setRrScore3Low(int v) { this.rrScore3Low = v; }
        public int getRrScore1Low() { return rrScore1Low; }
        public void setRrScore1Low(int v) { this.rrScore1Low = v; }
        public int getRrScore0High() { return rrScore0High; }
        public void setRrScore0High(int v) { this.rrScore0High = v; }
        public int getRrScore2High() { return rrScore2High; }
        public void setRrScore2High(int v) { this.rrScore2High = v; }
        public int getSpo2Scale1Score3() { return spo2Scale1Score3; }
        public void setSpo2Scale1Score3(int v) { this.spo2Scale1Score3 = v; }
        public int getSpo2Scale1Score2() { return spo2Scale1Score2; }
        public void setSpo2Scale1Score2(int v) { this.spo2Scale1Score2 = v; }
        public int getSpo2Scale1Score1() { return spo2Scale1Score1; }
        public void setSpo2Scale1Score1(int v) { this.spo2Scale1Score1 = v; }
        public int getSpo2Scale2Score3Low() { return spo2Scale2Score3Low; }
        public void setSpo2Scale2Score3Low(int v) { this.spo2Scale2Score3Low = v; }
        public int getSpo2Scale2Score3High() { return spo2Scale2Score3High; }
        public void setSpo2Scale2Score3High(int v) { this.spo2Scale2Score3High = v; }
        public int getSpo2Scale2Score2Low() { return spo2Scale2Score2Low; }
        public void setSpo2Scale2Score2Low(int v) { this.spo2Scale2Score2Low = v; }
        public int getSpo2Scale2Score1Low() { return spo2Scale2Score1Low; }
        public void setSpo2Scale2Score1Low(int v) { this.spo2Scale2Score1Low = v; }
        public int getSbpScore3Low() { return sbpScore3Low; }
        public void setSbpScore3Low(int v) { this.sbpScore3Low = v; }
        public int getSbpScore2Low() { return sbpScore2Low; }
        public void setSbpScore2Low(int v) { this.sbpScore2Low = v; }
        public int getSbpScore1Low() { return sbpScore1Low; }
        public void setSbpScore1Low(int v) { this.sbpScore1Low = v; }
        public int getSbpScore0High() { return sbpScore0High; }
        public void setSbpScore0High(int v) { this.sbpScore0High = v; }
        public int getHrScore3Low() { return hrScore3Low; }
        public void setHrScore3Low(int v) { this.hrScore3Low = v; }
        public int getHrScore1Low() { return hrScore1Low; }
        public void setHrScore1Low(int v) { this.hrScore1Low = v; }
        public int getHrScore0High() { return hrScore0High; }
        public void setHrScore0High(int v) { this.hrScore0High = v; }
        public int getHrScore1High() { return hrScore1High; }
        public void setHrScore1High(int v) { this.hrScore1High = v; }
        public int getHrScore2High() { return hrScore2High; }
        public void setHrScore2High(int v) { this.hrScore2High = v; }
        public double getTempScore3Low() { return tempScore3Low; }
        public void setTempScore3Low(double v) { this.tempScore3Low = v; }
        public double getTempScore1Low() { return tempScore1Low; }
        public void setTempScore1Low(double v) { this.tempScore1Low = v; }
        public double getTempScore0High() { return tempScore0High; }
        public void setTempScore0High(double v) { this.tempScore0High = v; }
        public double getTempScore1High() { return tempScore1High; }
        public void setTempScore1High(double v) { this.tempScore1High = v; }
        public int getSupplementalOxygenScore() { return supplementalOxygenScore; }
        public void setSupplementalOxygenScore(int v) { this.supplementalOxygenScore = v; }
        public int getLowMediumThreshold() { return lowMediumThreshold; }
        public void setLowMediumThreshold(int v) { this.lowMediumThreshold = v; }
        public int getMediumThreshold() { return mediumThreshold; }
        public void setMediumThreshold(int v) { this.mediumThreshold = v; }
        public int getHighThreshold() { return highThreshold; }
        public void setHighThreshold(int v) { this.highThreshold = v; }
    }

    /**
     * MEWS (Modified Early Warning Score) parameters.
     * Placeholder for future KB-4 integration -- currently not computed in Flink
     * but included for completeness since the spec requires it.
     */
    public static class MEWSParams implements Serializable {
        private static final long serialVersionUID = 1L;

        // MEWS uses similar bands to NEWS2 but with different cut-points
        private int sbpScore3Low;   // <=70
        private int sbpScore2Low;   // 71-80
        private int sbpScore1Low;   // 81-100
        private int sbpScore0High;  // 101-199
        // >=200 => score 2

        private int hrScore2Low;    // <40
        private int hrScore1Low;    // 40-50
        private int hrScore0High;   // 51-100
        private int hrScore1High;   // 101-110
        private int hrScore2High;   // 111-129
        // >=130 => score 3

        private int rrScore2Low;    // <9
        private int rrScore0High;   // 9-14
        private int rrScore1High;   // 15-20
        private int rrScore2High;   // 21-29
        // >=30 => score 3

        private double tempScore2Low;   // <35.0
        private double tempScore0High;  // 35.0-38.4
        // >=38.5 => score 2

        public static MEWSParams defaults() {
            MEWSParams m = new MEWSParams();
            m.sbpScore3Low = 70;
            m.sbpScore2Low = 80;
            m.sbpScore1Low = 100;
            m.sbpScore0High = 199;
            m.hrScore2Low = 40;
            m.hrScore1Low = 50;
            m.hrScore0High = 100;
            m.hrScore1High = 110;
            m.hrScore2High = 129;
            m.rrScore2Low = 9;
            m.rrScore0High = 14;
            m.rrScore1High = 20;
            m.rrScore2High = 29;
            m.tempScore2Low = 35.0;
            m.tempScore0High = 38.4;
            return m;
        }

        // Getters and setters
        public int getSbpScore3Low() { return sbpScore3Low; }
        public void setSbpScore3Low(int v) { this.sbpScore3Low = v; }
        public int getSbpScore2Low() { return sbpScore2Low; }
        public void setSbpScore2Low(int v) { this.sbpScore2Low = v; }
        public int getSbpScore1Low() { return sbpScore1Low; }
        public void setSbpScore1Low(int v) { this.sbpScore1Low = v; }
        public int getSbpScore0High() { return sbpScore0High; }
        public void setSbpScore0High(int v) { this.sbpScore0High = v; }
        public int getHrScore2Low() { return hrScore2Low; }
        public void setHrScore2Low(int v) { this.hrScore2Low = v; }
        public int getHrScore1Low() { return hrScore1Low; }
        public void setHrScore1Low(int v) { this.hrScore1Low = v; }
        public int getHrScore0High() { return hrScore0High; }
        public void setHrScore0High(int v) { this.hrScore0High = v; }
        public int getHrScore1High() { return hrScore1High; }
        public void setHrScore1High(int v) { this.hrScore1High = v; }
        public int getHrScore2High() { return hrScore2High; }
        public void setHrScore2High(int v) { this.hrScore2High = v; }
        public int getRrScore2Low() { return rrScore2Low; }
        public void setRrScore2Low(int v) { this.rrScore2Low = v; }
        public int getRrScore0High() { return rrScore0High; }
        public void setRrScore0High(int v) { this.rrScore0High = v; }
        public int getRrScore1High() { return rrScore1High; }
        public void setRrScore1High(int v) { this.rrScore1High = v; }
        public int getRrScore2High() { return rrScore2High; }
        public void setRrScore2High(int v) { this.rrScore2High = v; }
        public double getTempScore2Low() { return tempScore2Low; }
        public void setTempScore2Low(double v) { this.tempScore2Low = v; }
        public double getTempScore0High() { return tempScore0High; }
        public void setTempScore0High(double v) { this.tempScore0High = v; }
    }

    /**
     * Risk scoring configuration from RiskScoreCalculator.
     * Daily composite weights and risk stratification boundaries.
     */
    public static class RiskScoringConfig implements Serializable {
        private static final long serialVersionUID = 1L;

        // Daily composite weights (must sum to 1.0)
        private double vitalWeight;      // 0.40
        private double labWeight;        // 0.35
        private double medicationWeight; // 0.25

        // Vital stability scoring multipliers
        private double abnormalVitalMultiplier;  // 50
        private double criticalVitalMultiplier;  // 100

        // Lab scoring multipliers
        private double abnormalLabMultiplier;  // 40
        private double criticalLabMultiplier;  // 120

        // Risk level boundaries (composite score 0-100)
        private int lowMaxScore;       // 24
        private int moderateMaxScore;  // 49
        private int highMaxScore;      // 74
        // >=75 => CRITICAL

        // Acuity level boundaries (combined acuity from ClinicalScoreCalculator)
        private double acuityCriticalThreshold; // 7.0
        private double acuityHighThreshold;     // 5.0
        private double acuityMediumThreshold;   // 3.0

        // NEWS2 weight in combined acuity
        private double news2AcuityWeight;      // 0.70
        private double metabolicAcuityWeight;  // 0.30

        // AlertPrioritizer weights
        private double clinicalSeverityWeight;       // 2.0
        private double timeSensitivityWeight;        // 1.5
        private double patientVulnerabilityWeight;   // 1.0
        private double trendingPatternWeight;        // 1.5
        private double confidenceScoreWeight;        // 0.5
        private double maxPriorityScore;             // 30.0

        public static RiskScoringConfig defaults() {
            RiskScoringConfig c = new RiskScoringConfig();
            c.vitalWeight = 0.40;
            c.labWeight = 0.35;
            c.medicationWeight = 0.25;
            c.abnormalVitalMultiplier = 50.0;
            c.criticalVitalMultiplier = 100.0;
            c.abnormalLabMultiplier = 40.0;
            c.criticalLabMultiplier = 120.0;
            c.lowMaxScore = 24;
            c.moderateMaxScore = 49;
            c.highMaxScore = 74;
            c.acuityCriticalThreshold = 7.0;
            c.acuityHighThreshold = 5.0;
            c.acuityMediumThreshold = 3.0;
            c.news2AcuityWeight = 0.70;
            c.metabolicAcuityWeight = 0.30;
            c.clinicalSeverityWeight = 2.0;
            c.timeSensitivityWeight = 1.5;
            c.patientVulnerabilityWeight = 1.0;
            c.trendingPatternWeight = 1.5;
            c.confidenceScoreWeight = 0.5;
            c.maxPriorityScore = 30.0;
            return c;
        }

        // Getters and setters
        public double getVitalWeight() { return vitalWeight; }
        public void setVitalWeight(double v) { this.vitalWeight = v; }
        public double getLabWeight() { return labWeight; }
        public void setLabWeight(double v) { this.labWeight = v; }
        public double getMedicationWeight() { return medicationWeight; }
        public void setMedicationWeight(double v) { this.medicationWeight = v; }
        public double getAbnormalVitalMultiplier() { return abnormalVitalMultiplier; }
        public void setAbnormalVitalMultiplier(double v) { this.abnormalVitalMultiplier = v; }
        public double getCriticalVitalMultiplier() { return criticalVitalMultiplier; }
        public void setCriticalVitalMultiplier(double v) { this.criticalVitalMultiplier = v; }
        public double getAbnormalLabMultiplier() { return abnormalLabMultiplier; }
        public void setAbnormalLabMultiplier(double v) { this.abnormalLabMultiplier = v; }
        public double getCriticalLabMultiplier() { return criticalLabMultiplier; }
        public void setCriticalLabMultiplier(double v) { this.criticalLabMultiplier = v; }
        public int getLowMaxScore() { return lowMaxScore; }
        public void setLowMaxScore(int v) { this.lowMaxScore = v; }
        public int getModerateMaxScore() { return moderateMaxScore; }
        public void setModerateMaxScore(int v) { this.moderateMaxScore = v; }
        public int getHighMaxScore() { return highMaxScore; }
        public void setHighMaxScore(int v) { this.highMaxScore = v; }
        public double getAcuityCriticalThreshold() { return acuityCriticalThreshold; }
        public void setAcuityCriticalThreshold(double v) { this.acuityCriticalThreshold = v; }
        public double getAcuityHighThreshold() { return acuityHighThreshold; }
        public void setAcuityHighThreshold(double v) { this.acuityHighThreshold = v; }
        public double getAcuityMediumThreshold() { return acuityMediumThreshold; }
        public void setAcuityMediumThreshold(double v) { this.acuityMediumThreshold = v; }
        public double getNews2AcuityWeight() { return news2AcuityWeight; }
        public void setNews2AcuityWeight(double v) { this.news2AcuityWeight = v; }
        public double getMetabolicAcuityWeight() { return metabolicAcuityWeight; }
        public void setMetabolicAcuityWeight(double v) { this.metabolicAcuityWeight = v; }
        public double getClinicalSeverityWeight() { return clinicalSeverityWeight; }
        public void setClinicalSeverityWeight(double v) { this.clinicalSeverityWeight = v; }
        public double getTimeSensitivityWeight() { return timeSensitivityWeight; }
        public void setTimeSensitivityWeight(double v) { this.timeSensitivityWeight = v; }
        public double getPatientVulnerabilityWeight() { return patientVulnerabilityWeight; }
        public void setPatientVulnerabilityWeight(double v) { this.patientVulnerabilityWeight = v; }
        public double getTrendingPatternWeight() { return trendingPatternWeight; }
        public void setTrendingPatternWeight(double v) { this.trendingPatternWeight = v; }
        public double getConfidenceScoreWeight() { return confidenceScoreWeight; }
        public void setConfidenceScoreWeight(double v) { this.confidenceScoreWeight = v; }
        public double getMaxPriorityScore() { return maxPriorityScore; }
        public void setMaxPriorityScore(double v) { this.maxPriorityScore = v; }
    }

    /**
     * High-risk categories from KB-1.
     * Placeholder for drug/condition categories that trigger elevated monitoring.
     */
    public static class HighRiskCategories implements Serializable {
        private static final long serialVersionUID = 1L;

        private double metabolicAcuityCKD;     // 1.5 points
        private double metabolicAcuityHF;      // 2.0 points
        private double metabolicAcuityDM;      // 1.0 points
        private double metabolicAcuityHTN;     // 1.0 points
        private double metabolicAcuityObesity; // 0.5 points
        private double metabolicAcuityCap;     // 5.0 max

        public static HighRiskCategories defaults() {
            HighRiskCategories h = new HighRiskCategories();
            h.metabolicAcuityCKD = 1.5;
            h.metabolicAcuityHF = 2.0;
            h.metabolicAcuityDM = 1.0;
            h.metabolicAcuityHTN = 1.0;
            h.metabolicAcuityObesity = 0.5;
            h.metabolicAcuityCap = 5.0;
            return h;
        }

        // Getters and setters
        public double getMetabolicAcuityCKD() { return metabolicAcuityCKD; }
        public void setMetabolicAcuityCKD(double v) { this.metabolicAcuityCKD = v; }
        public double getMetabolicAcuityHF() { return metabolicAcuityHF; }
        public void setMetabolicAcuityHF(double v) { this.metabolicAcuityHF = v; }
        public double getMetabolicAcuityDM() { return metabolicAcuityDM; }
        public void setMetabolicAcuityDM(double v) { this.metabolicAcuityDM = v; }
        public double getMetabolicAcuityHTN() { return metabolicAcuityHTN; }
        public void setMetabolicAcuityHTN(double v) { this.metabolicAcuityHTN = v; }
        public double getMetabolicAcuityObesity() { return metabolicAcuityObesity; }
        public void setMetabolicAcuityObesity(double v) { this.metabolicAcuityObesity = v; }
        public double getMetabolicAcuityCap() { return metabolicAcuityCap; }
        public void setMetabolicAcuityCap(double v) { this.metabolicAcuityCap = v; }
    }

    @Override
    public String toString() {
        return "ClinicalThresholdSet{version='" + version + "', loadedAt=" + loadedAtEpochMs + "}";
    }
}
