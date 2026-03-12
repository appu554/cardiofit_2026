package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.Objects;

/**
 * Risk Indicators Model for Clinical Alerting System.
 *
 * Provides structured boolean flags and trend indicators for Complex Event Processing (CEP)
 * pattern matching instead of manual vital sign extraction. This model supports multiple
 * clinical scoring systems including qSOFA, SIRS, NEWS2, and custom deterioration indices.
 *
 * <p><b>Clinical Scoring System Support:</b></p>
 * <ul>
 *   <li><b>qSOFA (Quick Sequential Organ Failure Assessment):</b> Uses respiratory rate,
 *       altered mentation, systolic blood pressure to predict sepsis mortality</li>
 *   <li><b>SIRS (Systemic Inflammatory Response Syndrome):</b> Uses temperature, heart rate,
 *       respiratory rate, and WBC count for infection screening</li>
 *   <li><b>NEWS2 (National Early Warning Score 2):</b> Uses vital signs to detect clinical
 *       deterioration in acute care settings</li>
 * </ul>
 *
 * @see <a href="https://www.mdcalc.com/qsofa-quick-sofa-score-sepsis">qSOFA Calculator</a>
 * @see <a href="https://www.mdcalc.com/sirs-sepsis-septic-shock-criteria">SIRS Criteria</a>
 * @see <a href="https://www.rcplondon.ac.uk/projects/outputs/national-early-warning-score-news-2">NEWS2 Score</a>
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class RiskIndicators implements Serializable {
    private static final long serialVersionUID = 1L;

    // ========================================================================================
    // VITAL SIGN CONCERNS - Boolean flags based on clinical threshold values
    // ========================================================================================

    /**
     * Tachycardia indicator (HR > 100 bpm).
     * Clinical Significance: May indicate fever, pain, anxiety, hypovolemia, or cardiac arrhythmia.
     * Used in SIRS criteria (HR > 90) and NEWS2 scoring.
     */
    @JsonProperty("tachycardia")
    private boolean tachycardia;

    /**
     * Bradycardia indicator (HR < 60 bpm).
     * Clinical Significance: May indicate cardiac conduction abnormalities, beta-blocker effect,
     * or increased intracranial pressure. Concerning in unstable patients.
     */
    @JsonProperty("bradycardia")
    private boolean bradycardia;

    /**
     * Hypotension indicator (SBP < 90 mmHg).
     * Clinical Significance: Critical for qSOFA scoring (≤ 100 mmHg). Indicates inadequate
     * tissue perfusion, shock states, or medication effects.
     */
    @JsonProperty("hypotension")
    private boolean hypotension;

    /**
     * Hypertension indicator (SBP > 180 mmHg).
     * Clinical Significance: Hypertensive emergency threshold. Risk for stroke, acute coronary
     * syndrome, or aortic dissection.
     */
    @JsonProperty("hypertension")
    private boolean hypertension;

    /**
     * Fever indicator (Temperature > 38.3°C / 101°F).
     * Clinical Significance: Part of SIRS criteria (> 38°C). Indicates infection, inflammation,
     * or drug fever. Higher threshold (38.3°C) reduces false positives.
     */
    @JsonProperty("fever")
    private boolean fever;

    /**
     * Hypothermia indicator (Temperature < 36.0°C / 96.8°F).
     * Clinical Significance: Part of SIRS criteria. May indicate sepsis (especially in elderly),
     * environmental exposure, or endocrine dysfunction.
     */
    @JsonProperty("hypothermia")
    private boolean hypothermia;

    /**
     * Hypoxia indicator (SpO2 < 92%).
     * Clinical Significance: Indicates inadequate oxygenation. NEWS2 uses 93-94% as trigger
     * threshold. Critical for respiratory failure assessment.
     */
    @JsonProperty("hypoxia")
    private boolean hypoxia;

    /**
     * Tachypnea indicator (RR > 22 breaths/min).
     * Clinical Significance: qSOFA criterion (≥ 22). SIRS uses > 20. Early sign of respiratory
     * distress, metabolic acidosis, or sepsis.
     */
    @JsonProperty("tachypnea")
    private boolean tachypnea;

    /**
     * Bradypnea indicator (RR < 12 breaths/min).
     * Clinical Significance: May indicate respiratory depression from opioids/sedatives,
     * neurological impairment, or fatigue preceding respiratory arrest.
     */
    @JsonProperty("bradypnea")
    private boolean bradypnea;

    // ========================================================================================
    // LAB ABNORMALITIES - Boolean flags for critical laboratory values
    // ========================================================================================

    /**
     * Elevated lactate indicator (> 2.0 mmol/L).
     * Clinical Significance: Indicates tissue hypoperfusion and anaerobic metabolism.
     * Key sepsis marker. Normal range: 0.5-2.0 mmol/L.
     */
    @JsonProperty("elevatedLactate")
    private boolean elevatedLactate;

    /**
     * Severely elevated lactate indicator (> 4.0 mmol/L).
     * Clinical Significance: Septic shock criterion. Associated with high mortality.
     * Requires immediate intervention.
     */
    @JsonProperty("severelyElevatedLactate")
    private boolean severelyElevatedLactate;

    /**
     * Elevated creatinine indicator (> 1.5x baseline).
     * Clinical Significance: KDIGO AKI (Acute Kidney Injury) Stage 1 criterion. Indicates
     * renal dysfunction from sepsis, nephrotoxins, or hypovolemia.
     */
    @JsonProperty("elevatedCreatinine")
    private boolean elevatedCreatinine;

    /**
     * Leukocytosis indicator (WBC > 12,000/mm³).
     * Clinical Significance: SIRS criterion. Indicates infection, inflammation, or stress response.
     * May also indicate leukemia or medication effect.
     */
    @JsonProperty("leukocytosis")
    private boolean leukocytosis;

    /**
     * Leukopenia indicator (WBC < 4,000/mm³).
     * Clinical Significance: SIRS criterion. May indicate overwhelming sepsis (especially with
     * bandemia), bone marrow suppression, or viral infection.
     */
    @JsonProperty("leukopenia")
    private boolean leukopenia;

    /**
     * Thrombocytopenia indicator (Platelets < 100,000/mm³).
     * Clinical Significance: Moderate thrombocytopenia. Risk for bleeding, may indicate DIC,
     * sepsis, or medication effect. Normal range: 150,000-400,000/mm³.
     */
    @JsonProperty("thrombocytopenia")
    private boolean thrombocytopenia;

    /**
     * Elevated troponin indicator (> 0.04 ng/mL).
     * Clinical Significance: Positive troponin indicating myocardial injury. Used for acute
     * coronary syndrome diagnosis. High-sensitivity assays have lower thresholds.
     */
    @JsonProperty("elevatedTroponin")
    private boolean elevatedTroponin;

    /**
     * Elevated BNP indicator (> 400 pg/mL).
     * Clinical Significance: B-type Natriuretic Peptide elevation indicates heart failure,
     * volume overload, or cardiac stress. Used for heart failure diagnosis and prognosis.
     */
    @JsonProperty("elevatedBNP")
    private boolean elevatedBNP;

    /**
     * Elevated CK-MB indicator (> 25 U/L).
     * Clinical Significance: Creatine Kinase-MB elevation indicates myocardial injury.
     * More specific than total CK for cardiac muscle damage.
     */
    @JsonProperty("elevatedCKMB")
    private boolean elevatedCKMB;

    /**
     * Hyperkalemia indicator (K+ > 5.5 mEq/L).
     * Clinical Significance: Elevated potassium increases risk of fatal cardiac arrhythmias.
     * Requires immediate ECG monitoring and treatment. Common with renal failure, ACE-I/ARBs,
     * K-sparing diuretics, or tissue breakdown.
     */
    @JsonProperty("hyperkalemia")
    private boolean hyperkalemia;

    /**
     * Hypokalemia indicator (K+ < 3.5 mEq/L).
     * Clinical Significance: Low potassium increases arrhythmia risk (especially with digoxin),
     * muscle weakness, and respiratory compromise. Common with diuretics, vomiting, diarrhea.
     */
    @JsonProperty("hypokalemia")
    private boolean hypokalemia;

    /**
     * Hypernatremia indicator (Na+ > 145 mEq/L).
     * Clinical Significance: Elevated sodium indicates dehydration, diabetes insipidus, or
     * excessive sodium intake. Can cause altered mental status, seizures if severe (>160).
     */
    @JsonProperty("hypernatremia")
    private boolean hypernatremia;

    /**
     * Hyponatremia indicator (Na+ < 135 mEq/L).
     * Clinical Significance: Low sodium can cause confusion, seizures, coma if severe (<120).
     * Common with diuretics, SIADH, heart failure, cirrhosis. Rapid correction dangerous.
     */
    @JsonProperty("hyponatremia")
    private boolean hyponatremia;

    // ========================================================================================
    // MEDICATION CONCERNS - Boolean flags for high-risk medication status
    // ========================================================================================

    /**
     * Vasopressor therapy indicator.
     * Clinical Significance: Indicates shock state requiring hemodynamic support. Increases
     * risk for arrhythmias, tissue ischemia, and requires intensive monitoring.
     */
    @JsonProperty("onVasopressors")
    private boolean onVasopressors;

    /**
     * Anticoagulation therapy indicator.
     * Clinical Significance: Increased bleeding risk, especially with procedures, trauma, or
     * thrombocytopenia. Requires monitoring of coagulation parameters.
     */
    @JsonProperty("onAnticoagulation")
    private boolean onAnticoagulation;

    /**
     * Nephrotoxic medication indicator.
     * Clinical Significance: Medications like vancomycin, aminoglycosides, NSAIDs, ACE inhibitors
     * increase AKI risk. Requires renal function monitoring and dose adjustment.
     */
    @JsonProperty("onNephrotoxicMeds")
    private boolean onNephrotoxicMeds;

    /**
     * Recent medication change indicator (within 24 hours).
     * Clinical Significance: New medications may cause adverse reactions, drug interactions,
     * or therapeutic gaps. Requires close monitoring for effectiveness and side effects.
     */
    @JsonProperty("recentMedicationChange")
    private boolean recentMedicationChange;

    // ========================================================================================
    // CLINICAL CONTEXT - Boolean flags for patient-specific risk factors
    // ========================================================================================

    /**
     * ICU location indicator.
     * Clinical Significance: Higher acuity patients with complex medical conditions. Greater
     * risk for nosocomial infections, delirium, and critical illness complications.
     */
    @JsonProperty("inICU")
    private boolean inICU;

    /**
     * Diabetes mellitus indicator.
     * Clinical Significance: Increased infection risk, impaired wound healing, cardiovascular
     * disease risk. Requires glucose monitoring and management.
     */
    @JsonProperty("hasDiabetes")
    private boolean hasDiabetes;

    /**
     * Chronic Kidney Disease indicator.
     * Clinical Significance: Baseline renal impairment affects medication dosing, contrast
     * procedures, and increases risk for acute-on-chronic kidney injury.
     */
    @JsonProperty("hasChronicKidneyDisease")
    private boolean hasChronicKidneyDisease;

    /**
     * Heart Failure indicator.
     * Clinical Significance: Volume management critical. Risk for decompensation with infections,
     * arrhythmias, or medication non-compliance. May affect hemodynamic interpretation.
     */
    @JsonProperty("hasHeartFailure")
    private boolean hasHeartFailure;

    /**
     * Post-operative status indicator (within 48 hours).
     * Clinical Significance: Increased risk for bleeding, infection, atelectasis, thromboembolism.
     * Pain and anesthesia effects may mask clinical deterioration.
     */
    @JsonProperty("postOperative")
    private boolean postOperative;

    // ========================================================================================
    // CARDIOVASCULAR RISK INDICATORS - For India CVD Prevention Project
    // ========================================================================================

    /**
     * Elevated Total Cholesterol (> 200 mg/dL).
     * Clinical Significance: Major risk factor for atherosclerotic cardiovascular disease (ASCVD).
     * India-specific: High prevalence in urban populations with metabolic syndrome.
     */
    @JsonProperty("elevatedTotalCholesterol")
    private boolean elevatedTotalCholesterol;

    /**
     * Low HDL Cholesterol (< 35 mg/dL for Indian population).
     * Clinical Significance: Independent risk factor for coronary artery disease.
     * India-specific: Lower threshold than Western guidelines (< 40 mg/dL).
     */
    @JsonProperty("lowHDL")
    private boolean lowHDL;

    /**
     * High Triglycerides (> 150 mg/dL).
     * Clinical Significance: Associated with metabolic syndrome and increased CVD risk.
     * Often elevated in South Asian populations even with normal BMI.
     */
    @JsonProperty("highTriglycerides")
    private boolean highTriglycerides;

    /**
     * Metabolic Syndrome composite indicator.
     * Clinical Significance: Cluster of conditions increasing heart disease, stroke, diabetes risk.
     * Criteria: 3+ of (abdominal obesity, high BP, high glucose, high TG, low HDL).
     */
    @JsonProperty("metabolicSyndrome")
    private boolean metabolicSyndrome;

    /**
     * Therapy failure for antihypertensive medication.
     * Clinical Significance: Persistent BP elevation despite established medication (>4 weeks).
     * Indicates need for medication adjustment or adherence evaluation.
     */
    @JsonProperty("antihypertensiveTherapyFailure")
    private boolean antihypertensiveTherapyFailure;

    /**
     * Elevated LDL Cholesterol (> 130 mg/dL).
     * Clinical Significance: Primary target for statin therapy in CVD prevention.
     * India-specific: Aggressive targets (< 70 mg/dL) for high-risk patients.
     */
    @JsonProperty("elevatedLDL")
    private boolean elevatedLDL;

    // ========================================================================================
    // TREND INDICATORS - Enum-based directional trends from sliding window analysis
    // ========================================================================================

    /**
     * Heart rate trend over time (computed from sliding windows).
     * Clinical Significance: Trending deterioration more predictive than single values.
     * Increasing trend may precede clinical decompensation.
     */
    @JsonProperty("heartRateTrend")
    private TrendDirection heartRateTrend;

    /**
     * Blood pressure trend over time (computed from sliding windows).
     * Clinical Significance: Progressive hypotension indicates worsening shock.
     * Progressive hypertension may indicate pain or catecholamine surge.
     */
    @JsonProperty("bloodPressureTrend")
    private TrendDirection bloodPressureTrend;

    /**
     * Oxygen saturation trend over time (computed from sliding windows).
     * Clinical Significance: Declining SpO2 trend indicates worsening respiratory status
     * requiring intervention escalation.
     */
    @JsonProperty("oxygenSaturationTrend")
    private TrendDirection oxygenSaturationTrend;

    /**
     * Temperature trend over time (computed from sliding windows).
     * Clinical Significance: Rising temperature may indicate infection progression.
     * Falling temperature in sepsis may indicate hypothermia or improvement.
     */
    @JsonProperty("temperatureTrend")
    private TrendDirection temperatureTrend;

    // ========================================================================================
    // METADATA - Tracking and quality assurance fields
    // ========================================================================================

    /**
     * Timestamp of last indicator update (epoch milliseconds).
     * Used to determine indicator freshness and validity for decision making.
     */
    @JsonProperty("lastUpdated")
    private long lastUpdated;

    /**
     * Confidence score (0.0 - 1.0) based on data completeness.
     * Calculated from percentage of populated indicators. Low confidence scores may indicate
     * incomplete data affecting clinical decision accuracy.
     */
    @JsonProperty("confidenceScore")
    private double confidenceScore;

    // ========================================================================================
    // CONSTRUCTORS
    // ========================================================================================

    /**
     * Default constructor for serialization frameworks.
     */
    public RiskIndicators() {
        this.lastUpdated = System.currentTimeMillis();
        this.confidenceScore = 0.0;
        this.heartRateTrend = TrendDirection.STABLE;
        this.bloodPressureTrend = TrendDirection.STABLE;
        this.oxygenSaturationTrend = TrendDirection.STABLE;
        this.temperatureTrend = TrendDirection.STABLE;
    }

    /**
     * Private constructor for Builder pattern.
     */
    private RiskIndicators(Builder builder) {
        this.tachycardia = builder.tachycardia;
        this.bradycardia = builder.bradycardia;
        this.hypotension = builder.hypotension;
        this.hypertension = builder.hypertension;
        this.fever = builder.fever;
        this.hypothermia = builder.hypothermia;
        this.hypoxia = builder.hypoxia;
        this.tachypnea = builder.tachypnea;
        this.bradypnea = builder.bradypnea;
        this.elevatedLactate = builder.elevatedLactate;
        this.severelyElevatedLactate = builder.severelyElevatedLactate;
        this.elevatedCreatinine = builder.elevatedCreatinine;
        this.leukocytosis = builder.leukocytosis;
        this.leukopenia = builder.leukopenia;
        this.thrombocytopenia = builder.thrombocytopenia;
        this.elevatedTroponin = builder.elevatedTroponin;
        this.elevatedBNP = builder.elevatedBNP;
        this.elevatedCKMB = builder.elevatedCKMB;
        this.hyperkalemia = builder.hyperkalemia;
        this.hypokalemia = builder.hypokalemia;
        this.hypernatremia = builder.hypernatremia;
        this.hyponatremia = builder.hyponatremia;
        this.onVasopressors = builder.onVasopressors;
        this.onAnticoagulation = builder.onAnticoagulation;
        this.onNephrotoxicMeds = builder.onNephrotoxicMeds;
        this.recentMedicationChange = builder.recentMedicationChange;
        this.inICU = builder.inICU;
        this.hasDiabetes = builder.hasDiabetes;
        this.hasChronicKidneyDisease = builder.hasChronicKidneyDisease;
        this.hasHeartFailure = builder.hasHeartFailure;
        this.postOperative = builder.postOperative;
        this.heartRateTrend = builder.heartRateTrend;
        this.bloodPressureTrend = builder.bloodPressureTrend;
        this.oxygenSaturationTrend = builder.oxygenSaturationTrend;
        this.temperatureTrend = builder.temperatureTrend;
        this.lastUpdated = builder.lastUpdated;
        this.confidenceScore = validateConfidenceScore(builder.confidenceScore);
    }

    /**
     * Validates confidence score is within valid range [0.0, 1.0].
     * @param score The confidence score to validate
     * @return Validated score (clamped to valid range)
     */
    private double validateConfidenceScore(double score) {
        if (score < 0.0) return 0.0;
        if (score > 1.0) return 1.0;
        return score;
    }

    // ========================================================================================
    // CLINICAL SCORING METHODS
    // ========================================================================================

    /**
     * Calculates qSOFA score (Quick Sequential Organ Failure Assessment).
     * Used for rapid sepsis screening in non-ICU settings.
     *
     * Scoring criteria (1 point each):
     * - Respiratory rate ≥ 22/min (tachypnea)
     * - Altered mentation (would need additional indicator)
     * - Systolic BP ≤ 100 mmHg (hypotension)
     *
     * Score ≥ 2 indicates high risk for poor outcomes.
     *
     * @return qSOFA score (0-2, max 3 with altered mentation if added)
     */
    public int calculateQSOFA() {
        int score = 0;
        if (tachypnea) score++;
        if (hypotension) score++;
        // Note: Altered mentation would require additional GCS or consciousness indicator
        return score;
    }

    /**
     * Calculates SIRS criteria count (Systemic Inflammatory Response Syndrome).
     * Used for infection and sepsis screening.
     *
     * Criteria (1 point each):
     * - Temperature > 38°C or < 36°C
     * - Heart rate > 90 bpm (using tachycardia as proxy > 100)
     * - Respiratory rate > 20/min (using tachypnea ≥ 22)
     * - WBC > 12,000 or < 4,000/mm³
     *
     * ≥ 2 criteria indicates SIRS (with infection = sepsis).
     *
     * @return SIRS criteria count (0-4)
     */
    public int calculateSIRS() {
        int score = 0;
        if (fever || hypothermia) score++;
        if (tachycardia) score++; // Proxy for HR > 90
        if (tachypnea) score++;   // Proxy for RR > 20
        if (leukocytosis || leukopenia) score++;
        return score;
    }

    /**
     * Checks if patient has indicators suggesting septic shock.
     * Septic Shock = Sepsis + Persistent hypotension + Lactate > 2 mmol/L despite fluids.
     *
     * @return true if septic shock indicators present
     */
    public boolean hasSepticShockIndicators() {
        return hypotension && elevatedLactate && calculateSIRS() >= 2;
    }

    /**
     * Checks if patient has severe sepsis indicators.
     * Severe Sepsis = Sepsis + Organ dysfunction.
     *
     * @return true if severe sepsis indicators present
     */
    public boolean hasSevereSepsisIndicators() {
        return calculateSIRS() >= 2 &&
               (elevatedLactate || elevatedCreatinine || hypotension || severelyElevatedLactate);
    }

    // ========================================================================================
    // BUILDER PATTERN
    // ========================================================================================

    /**
     * Creates a new Builder instance for constructing RiskIndicators.
     * @return new Builder instance
     */
    public static Builder builder() {
        return new Builder();
    }

    /**
     * Builder class for constructing RiskIndicators with fluent API.
     */
    public static class Builder {
        private boolean tachycardia;
        private boolean bradycardia;
        private boolean hypotension;
        private boolean hypertension;
        private boolean fever;
        private boolean hypothermia;
        private boolean hypoxia;
        private boolean tachypnea;
        private boolean bradypnea;
        private boolean elevatedLactate;
        private boolean severelyElevatedLactate;
        private boolean elevatedCreatinine;
        private boolean leukocytosis;
        private boolean leukopenia;
        private boolean thrombocytopenia;
        private boolean elevatedTroponin;
        private boolean elevatedBNP;
        private boolean elevatedCKMB;
        private boolean hyperkalemia;
        private boolean hypokalemia;
        private boolean hypernatremia;
        private boolean hyponatremia;
        private boolean onVasopressors;
        private boolean onAnticoagulation;
        private boolean onNephrotoxicMeds;
        private boolean recentMedicationChange;
        private boolean inICU;
        private boolean hasDiabetes;
        private boolean hasChronicKidneyDisease;
        private boolean hasHeartFailure;
        private boolean postOperative;
        private TrendDirection heartRateTrend = TrendDirection.STABLE;
        private TrendDirection bloodPressureTrend = TrendDirection.STABLE;
        private TrendDirection oxygenSaturationTrend = TrendDirection.STABLE;
        private TrendDirection temperatureTrend = TrendDirection.STABLE;
        private long lastUpdated = System.currentTimeMillis();
        private double confidenceScore = 0.0;

        public Builder tachycardia(boolean tachycardia) {
            this.tachycardia = tachycardia;
            return this;
        }

        public Builder bradycardia(boolean bradycardia) {
            this.bradycardia = bradycardia;
            return this;
        }

        public Builder hypotension(boolean hypotension) {
            this.hypotension = hypotension;
            return this;
        }

        public Builder hypertension(boolean hypertension) {
            this.hypertension = hypertension;
            return this;
        }

        public Builder fever(boolean fever) {
            this.fever = fever;
            return this;
        }

        public Builder hypothermia(boolean hypothermia) {
            this.hypothermia = hypothermia;
            return this;
        }

        public Builder hypoxia(boolean hypoxia) {
            this.hypoxia = hypoxia;
            return this;
        }

        public Builder tachypnea(boolean tachypnea) {
            this.tachypnea = tachypnea;
            return this;
        }

        public Builder bradypnea(boolean bradypnea) {
            this.bradypnea = bradypnea;
            return this;
        }

        public Builder elevatedLactate(boolean elevatedLactate) {
            this.elevatedLactate = elevatedLactate;
            return this;
        }

        public Builder severelyElevatedLactate(boolean severelyElevatedLactate) {
            this.severelyElevatedLactate = severelyElevatedLactate;
            return this;
        }

        public Builder elevatedCreatinine(boolean elevatedCreatinine) {
            this.elevatedCreatinine = elevatedCreatinine;
            return this;
        }

        public Builder leukocytosis(boolean leukocytosis) {
            this.leukocytosis = leukocytosis;
            return this;
        }

        public Builder leukopenia(boolean leukopenia) {
            this.leukopenia = leukopenia;
            return this;
        }

        public Builder thrombocytopenia(boolean thrombocytopenia) {
            this.thrombocytopenia = thrombocytopenia;
            return this;
        }

        public Builder elevatedTroponin(boolean elevatedTroponin) {
            this.elevatedTroponin = elevatedTroponin;
            return this;
        }

        public Builder elevatedBNP(boolean elevatedBNP) {
            this.elevatedBNP = elevatedBNP;
            return this;
        }

        public Builder elevatedCKMB(boolean elevatedCKMB) {
            this.elevatedCKMB = elevatedCKMB;
            return this;
        }

        public Builder hyperkalemia(boolean hyperkalemia) {
            this.hyperkalemia = hyperkalemia;
            return this;
        }

        public Builder hypokalemia(boolean hypokalemia) {
            this.hypokalemia = hypokalemia;
            return this;
        }

        public Builder hypernatremia(boolean hypernatremia) {
            this.hypernatremia = hypernatremia;
            return this;
        }

        public Builder hyponatremia(boolean hyponatremia) {
            this.hyponatremia = hyponatremia;
            return this;
        }

        public Builder onVasopressors(boolean onVasopressors) {
            this.onVasopressors = onVasopressors;
            return this;
        }

        public Builder onAnticoagulation(boolean onAnticoagulation) {
            this.onAnticoagulation = onAnticoagulation;
            return this;
        }

        public Builder onNephrotoxicMeds(boolean onNephrotoxicMeds) {
            this.onNephrotoxicMeds = onNephrotoxicMeds;
            return this;
        }

        public Builder recentMedicationChange(boolean recentMedicationChange) {
            this.recentMedicationChange = recentMedicationChange;
            return this;
        }

        public Builder inICU(boolean inICU) {
            this.inICU = inICU;
            return this;
        }

        public Builder hasDiabetes(boolean hasDiabetes) {
            this.hasDiabetes = hasDiabetes;
            return this;
        }

        public Builder hasChronicKidneyDisease(boolean hasChronicKidneyDisease) {
            this.hasChronicKidneyDisease = hasChronicKidneyDisease;
            return this;
        }

        public Builder hasHeartFailure(boolean hasHeartFailure) {
            this.hasHeartFailure = hasHeartFailure;
            return this;
        }

        public Builder postOperative(boolean postOperative) {
            this.postOperative = postOperative;
            return this;
        }

        public Builder heartRateTrend(TrendDirection heartRateTrend) {
            this.heartRateTrend = heartRateTrend;
            return this;
        }

        public Builder bloodPressureTrend(TrendDirection bloodPressureTrend) {
            this.bloodPressureTrend = bloodPressureTrend;
            return this;
        }

        public Builder oxygenSaturationTrend(TrendDirection oxygenSaturationTrend) {
            this.oxygenSaturationTrend = oxygenSaturationTrend;
            return this;
        }

        public Builder temperatureTrend(TrendDirection temperatureTrend) {
            this.temperatureTrend = temperatureTrend;
            return this;
        }

        public Builder lastUpdated(long lastUpdated) {
            this.lastUpdated = lastUpdated;
            return this;
        }

        public Builder confidenceScore(double confidenceScore) {
            this.confidenceScore = confidenceScore;
            return this;
        }

        /**
         * Builds the RiskIndicators instance with validation.
         * @return new RiskIndicators instance
         */
        public RiskIndicators build() {
            return new RiskIndicators(this);
        }
    }

    // ========================================================================================
    // GETTERS AND SETTERS
    // ========================================================================================

    public boolean isTachycardia() { return tachycardia; }
    public void setTachycardia(boolean tachycardia) { this.tachycardia = tachycardia; }

    public boolean isBradycardia() { return bradycardia; }
    public void setBradycardia(boolean bradycardia) { this.bradycardia = bradycardia; }

    public boolean isHypotension() { return hypotension; }
    public void setHypotension(boolean hypotension) { this.hypotension = hypotension; }

    public boolean isHypertension() { return hypertension; }
    public void setHypertension(boolean hypertension) { this.hypertension = hypertension; }

    public boolean isFever() { return fever; }
    public void setFever(boolean fever) { this.fever = fever; }

    public boolean isHypothermia() { return hypothermia; }
    public void setHypothermia(boolean hypothermia) { this.hypothermia = hypothermia; }

    public boolean isHypoxia() { return hypoxia; }
    public void setHypoxia(boolean hypoxia) { this.hypoxia = hypoxia; }

    public boolean isTachypnea() { return tachypnea; }
    public void setTachypnea(boolean tachypnea) { this.tachypnea = tachypnea; }

    public boolean isBradypnea() { return bradypnea; }
    public void setBradypnea(boolean bradypnea) { this.bradypnea = bradypnea; }

    public boolean isElevatedLactate() { return elevatedLactate; }
    public void setElevatedLactate(boolean elevatedLactate) { this.elevatedLactate = elevatedLactate; }

    public boolean isSeverelyElevatedLactate() { return severelyElevatedLactate; }
    public void setSeverelyElevatedLactate(boolean severelyElevatedLactate) {
        this.severelyElevatedLactate = severelyElevatedLactate;
    }

    public boolean isElevatedCreatinine() { return elevatedCreatinine; }
    public void setElevatedCreatinine(boolean elevatedCreatinine) {
        this.elevatedCreatinine = elevatedCreatinine;
    }

    public boolean isLeukocytosis() { return leukocytosis; }
    public void setLeukocytosis(boolean leukocytosis) { this.leukocytosis = leukocytosis; }

    public boolean isLeukopenia() { return leukopenia; }
    public void setLeukopenia(boolean leukopenia) { this.leukopenia = leukopenia; }

    public boolean isThrombocytopenia() { return thrombocytopenia; }
    public void setThrombocytopenia(boolean thrombocytopenia) {
        this.thrombocytopenia = thrombocytopenia;
    }

    public boolean isElevatedTroponin() { return elevatedTroponin; }
    public void setElevatedTroponin(boolean elevatedTroponin) {
        this.elevatedTroponin = elevatedTroponin;
    }

    public boolean isElevatedBNP() { return elevatedBNP; }
    public void setElevatedBNP(boolean elevatedBNP) {
        this.elevatedBNP = elevatedBNP;
    }

    public boolean isElevatedCKMB() { return elevatedCKMB; }
    public void setElevatedCKMB(boolean elevatedCKMB) {
        this.elevatedCKMB = elevatedCKMB;
    }

    public boolean isHyperkalemia() { return hyperkalemia; }
    public void setHyperkalemia(boolean hyperkalemia) {
        this.hyperkalemia = hyperkalemia;
    }

    public boolean isHypokalemia() { return hypokalemia; }
    public void setHypokalemia(boolean hypokalemia) {
        this.hypokalemia = hypokalemia;
    }

    public boolean isHypernatremia() { return hypernatremia; }
    public void setHypernatremia(boolean hypernatremia) {
        this.hypernatremia = hypernatremia;
    }

    public boolean isHyponatremia() { return hyponatremia; }
    public void setHyponatremia(boolean hyponatremia) {
        this.hyponatremia = hyponatremia;
    }

    public boolean isOnVasopressors() { return onVasopressors; }
    public void setOnVasopressors(boolean onVasopressors) { this.onVasopressors = onVasopressors; }

    public boolean isOnAnticoagulation() { return onAnticoagulation; }
    public void setOnAnticoagulation(boolean onAnticoagulation) {
        this.onAnticoagulation = onAnticoagulation;
    }

    public boolean isOnNephrotoxicMeds() { return onNephrotoxicMeds; }
    public void setOnNephrotoxicMeds(boolean onNephrotoxicMeds) {
        this.onNephrotoxicMeds = onNephrotoxicMeds;
    }

    public boolean isRecentMedicationChange() { return recentMedicationChange; }
    public void setRecentMedicationChange(boolean recentMedicationChange) {
        this.recentMedicationChange = recentMedicationChange;
    }

    public boolean isInICU() { return inICU; }
    public void setInICU(boolean inICU) { this.inICU = inICU; }

    public boolean isHasDiabetes() { return hasDiabetes; }
    public void setHasDiabetes(boolean hasDiabetes) { this.hasDiabetes = hasDiabetes; }

    public boolean isHasChronicKidneyDisease() { return hasChronicKidneyDisease; }
    public void setHasChronicKidneyDisease(boolean hasChronicKidneyDisease) {
        this.hasChronicKidneyDisease = hasChronicKidneyDisease;
    }

    public boolean isHasHeartFailure() { return hasHeartFailure; }
    public void setHasHeartFailure(boolean hasHeartFailure) { this.hasHeartFailure = hasHeartFailure; }

    public boolean isPostOperative() { return postOperative; }
    public void setPostOperative(boolean postOperative) { this.postOperative = postOperative; }

    // CVD Risk Indicators getters/setters
    public boolean isElevatedTotalCholesterol() { return elevatedTotalCholesterol; }
    public void setElevatedTotalCholesterol(boolean elevatedTotalCholesterol) { this.elevatedTotalCholesterol = elevatedTotalCholesterol; }

    public boolean isLowHDL() { return lowHDL; }
    public void setLowHDL(boolean lowHDL) { this.lowHDL = lowHDL; }

    public boolean isHighTriglycerides() { return highTriglycerides; }
    public void setHighTriglycerides(boolean highTriglycerides) { this.highTriglycerides = highTriglycerides; }

    public boolean isMetabolicSyndrome() { return metabolicSyndrome; }
    public void setMetabolicSyndrome(boolean metabolicSyndrome) { this.metabolicSyndrome = metabolicSyndrome; }

    public boolean isAntihypertensiveTherapyFailure() { return antihypertensiveTherapyFailure; }
    public void setAntihypertensiveTherapyFailure(boolean antihypertensiveTherapyFailure) { this.antihypertensiveTherapyFailure = antihypertensiveTherapyFailure; }

    public boolean isElevatedLDL() { return elevatedLDL; }
    public void setElevatedLDL(boolean elevatedLDL) { this.elevatedLDL = elevatedLDL; }

    public TrendDirection getHeartRateTrend() { return heartRateTrend; }
    public void setHeartRateTrend(TrendDirection heartRateTrend) {
        this.heartRateTrend = heartRateTrend;
    }

    public TrendDirection getBloodPressureTrend() { return bloodPressureTrend; }
    public void setBloodPressureTrend(TrendDirection bloodPressureTrend) {
        this.bloodPressureTrend = bloodPressureTrend;
    }

    public TrendDirection getOxygenSaturationTrend() { return oxygenSaturationTrend; }
    public void setOxygenSaturationTrend(TrendDirection oxygenSaturationTrend) {
        this.oxygenSaturationTrend = oxygenSaturationTrend;
    }

    public TrendDirection getTemperatureTrend() { return temperatureTrend; }
    public void setTemperatureTrend(TrendDirection temperatureTrend) {
        this.temperatureTrend = temperatureTrend;
    }

    public long getLastUpdated() { return lastUpdated; }
    public void setLastUpdated(long lastUpdated) { this.lastUpdated = lastUpdated; }

    public double getConfidenceScore() { return confidenceScore; }
    public void setConfidenceScore(double confidenceScore) {
        this.confidenceScore = validateConfidenceScore(confidenceScore);
    }

    // ========================================================================================
    // ADDITIONAL CLINICAL RISK INDICATORS (for Module 3 CDS compatibility)
    // ========================================================================================

    @JsonProperty("sepsisRisk")
    private boolean sepsisRisk; // Infection suspected

    @JsonProperty("immunocompromised")
    private boolean immunocompromised; // Immunosuppressed state

    public boolean getSepsisRisk() {
        return sepsisRisk;
    }

    public void setSepsisRisk(boolean sepsisRisk) {
        this.sepsisRisk = sepsisRisk;
    }

    public boolean isImmunocompromised() {
        return immunocompromised;
    }

    public void setImmunocompromised(boolean immunocompromised) {
        this.immunocompromised = immunocompromised;
    }

    // ========================================================================================
    // UTILITY METHODS
    // ========================================================================================

    /**
     * Returns a detailed string representation for debugging.
     * Groups indicators by category for readability.
     */
    @Override
    public String toString() {
        return "RiskIndicators{" +
                "\n  Vitals: {tachycardia=" + tachycardia +
                ", bradycardia=" + bradycardia +
                ", hypotension=" + hypotension +
                ", hypertension=" + hypertension +
                ", fever=" + fever +
                ", hypothermia=" + hypothermia +
                ", hypoxia=" + hypoxia +
                ", tachypnea=" + tachypnea +
                ", bradypnea=" + bradypnea + "}" +
                "\n  Labs: {elevatedLactate=" + elevatedLactate +
                ", severelyElevatedLactate=" + severelyElevatedLactate +
                ", elevatedCreatinine=" + elevatedCreatinine +
                ", leukocytosis=" + leukocytosis +
                ", leukopenia=" + leukopenia +
                ", thrombocytopenia=" + thrombocytopenia +
                ", elevatedTroponin=" + elevatedTroponin +
                ", elevatedBNP=" + elevatedBNP +
                ", elevatedCKMB=" + elevatedCKMB +
                ", hyperkalemia=" + hyperkalemia +
                ", hypokalemia=" + hypokalemia +
                ", hypernatremia=" + hypernatremia +
                ", hyponatremia=" + hyponatremia + "}" +
                "\n  Medications: {vasopressors=" + onVasopressors +
                ", anticoagulation=" + onAnticoagulation +
                ", nephrotoxic=" + onNephrotoxicMeds +
                ", recentChange=" + recentMedicationChange + "}" +
                "\n  Context: {ICU=" + inICU +
                ", diabetes=" + hasDiabetes +
                ", CKD=" + hasChronicKidneyDisease +
                ", HF=" + hasHeartFailure +
                ", postOp=" + postOperative + "}" +
                "\n  Trends: {HR=" + heartRateTrend +
                ", BP=" + bloodPressureTrend +
                ", SpO2=" + oxygenSaturationTrend +
                ", Temp=" + temperatureTrend + "}" +
                "\n  Scores: {qSOFA=" + calculateQSOFA() +
                ", SIRS=" + calculateSIRS() +
                ", septicShock=" + hasSepticShockIndicators() +
                ", severeSepsis=" + hasSevereSepsisIndicators() + "}" +
                "\n  Meta: {updated=" + lastUpdated +
                ", confidence=" + String.format("%.2f", confidenceScore) + "}" +
                "\n}";
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        RiskIndicators that = (RiskIndicators) o;
        return tachycardia == that.tachycardia &&
                bradycardia == that.bradycardia &&
                hypotension == that.hypotension &&
                hypertension == that.hypertension &&
                fever == that.fever &&
                hypothermia == that.hypothermia &&
                hypoxia == that.hypoxia &&
                tachypnea == that.tachypnea &&
                bradypnea == that.bradypnea &&
                elevatedLactate == that.elevatedLactate &&
                severelyElevatedLactate == that.severelyElevatedLactate &&
                elevatedCreatinine == that.elevatedCreatinine &&
                leukocytosis == that.leukocytosis &&
                leukopenia == that.leukopenia &&
                thrombocytopenia == that.thrombocytopenia &&
                elevatedTroponin == that.elevatedTroponin &&
                elevatedBNP == that.elevatedBNP &&
                elevatedCKMB == that.elevatedCKMB &&
                hyperkalemia == that.hyperkalemia &&
                hypokalemia == that.hypokalemia &&
                hypernatremia == that.hypernatremia &&
                hyponatremia == that.hyponatremia &&
                onVasopressors == that.onVasopressors &&
                onAnticoagulation == that.onAnticoagulation &&
                onNephrotoxicMeds == that.onNephrotoxicMeds &&
                recentMedicationChange == that.recentMedicationChange &&
                inICU == that.inICU &&
                hasDiabetes == that.hasDiabetes &&
                hasChronicKidneyDisease == that.hasChronicKidneyDisease &&
                hasHeartFailure == that.hasHeartFailure &&
                postOperative == that.postOperative &&
                lastUpdated == that.lastUpdated &&
                Double.compare(that.confidenceScore, confidenceScore) == 0 &&
                heartRateTrend == that.heartRateTrend &&
                bloodPressureTrend == that.bloodPressureTrend &&
                oxygenSaturationTrend == that.oxygenSaturationTrend &&
                temperatureTrend == that.temperatureTrend;
    }

    @Override
    public int hashCode() {
        return Objects.hash(tachycardia, bradycardia, hypotension, hypertension, fever, hypothermia,
                hypoxia, tachypnea, bradypnea, elevatedLactate, severelyElevatedLactate,
                elevatedCreatinine, leukocytosis, leukopenia, thrombocytopenia, elevatedTroponin,
                elevatedBNP, elevatedCKMB, hyperkalemia, hypokalemia, hypernatremia, hyponatremia,
                onVasopressors, onAnticoagulation, onNephrotoxicMeds, recentMedicationChange,
                inICU, hasDiabetes, hasChronicKidneyDisease, hasHeartFailure, postOperative,
                heartRateTrend, bloodPressureTrend, oxygenSaturationTrend, temperatureTrend,
                lastUpdated, confidenceScore);
    }

}
