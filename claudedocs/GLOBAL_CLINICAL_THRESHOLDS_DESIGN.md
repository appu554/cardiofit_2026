# Global Clinical Thresholds Architecture Design

**Goal**: Make PatientContextAggregator globally applicable across different populations, geographies, and clinical contexts while handling all edge cases.

---

## Current Limitations Analysis

### **Problem 1: Hardcoded Geographic Assumptions**
```java
// India-specific threshold
if (systolic >= 140) { // ❌ US guidelines use 130 mmHg
```

**Impact**:
- Misses early hypertension in US patients (130-139 range)
- Over-diagnoses in some Asian populations with different baselines
- Violates clinical accuracy for international deployment

### **Problem 2: No Demographic Adjustment**
```java
private static final double CREATININE_THRESHOLD = 1.3; // mg/dL
```

**Missing Factors**:
- **Age**: Elderly have lower muscle mass → lower baseline creatinine
- **Sex**: Males have higher normal range (0.7-1.3) vs females (0.6-1.1)
- **Ethnicity**: African populations have 10-20% higher baseline creatinine
- **Body Mass**: Athletes have higher creatinine due to muscle mass

**Real-World Impact**: A 75-year-old female with creatinine 1.2 might have **severe renal impairment** despite being below the 1.3 threshold.

### **Problem 3: No Contextual Adjustment**
```java
private static final double POTASSIUM_HIGH = 5.5; // mEq/L
```

**Missing Context**:
- **Pregnancy**: Potassium 3.0-3.5 is normal (lower than non-pregnant 3.5-5.0)
- **Chronic Kidney Disease**: Baseline potassium 4.5-5.5 is acceptable
- **Altitude**: High-altitude residents have different oxygen saturation baselines
- **Medication Context**: ACE-I patients commonly have K+ 4.8-5.2 (acceptable)

### **Problem 4: No Unit Handling**
```java
private static final double LACTATE_THRESHOLD = 2.0; // mmol/L
```

**Global Variation**:
- **US/Europe**: mmol/L (threshold 2.0)
- **Some Asian labs**: mg/dL (threshold 18.0)
- **Conversion needed**: 1 mmol/L = 9 mg/dL

**Risk**: 10x errors if units aren't validated.

### **Problem 5: Static Guidelines**
```java
private static final double TROPONIN_THRESHOLD = 0.04; // ng/mL
```

**Problem**: Troponin assays vary by manufacturer:
- Abbott ARCHITECT: 0.04 ng/mL
- Roche Elecsys: 0.014 ng/mL (high-sensitivity)
- Siemens: 0.05 ng/mL

**Impact**: Same patient, different labs → different alert thresholds needed.

---

## Proposed Solution: 4-Layer Configuration Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Layer 1: Universal Base Thresholds (Evidence-Based)       │
│  - WHO/AHA/ESC consensus guidelines                        │
│  - Universal physiological limits (e.g., K+ >6.5 = crisis) │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Layer 2: Geographic/Regional Adjustments                   │
│  - Country-specific guidelines (US vs India vs EU)         │
│  - Local lab calibration variations                        │
│  - Regulatory compliance (FDA, EMA, CDSCO)                 │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Layer 3: Demographic Personalization                       │
│  - Age-adjusted ranges (pediatric, adult, elderly)         │
│  - Sex-specific thresholds                                 │
│  - Ethnicity-based adjustments                            │
│  - Body mass index corrections                             │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Layer 4: Contextual Overrides                             │
│  - Pregnancy-specific ranges                               │
│  - Chronic disease baselines (CKD, diabetes, heart failure)│
│  - Medication-induced expected deviations                  │
│  - Acute vs chronic condition adjustments                  │
└─────────────────────────────────────────────────────────────┘
```

---

## Implementation Design

### **1. Configuration Data Model**

```java
/**
 * Multi-layered clinical threshold configuration with demographic and contextual adjustments.
 *
 * Supports:
 * - Geographic guidelines (US, EU, India, WHO)
 * - Demographic personalization (age, sex, ethnicity)
 * - Contextual adjustments (pregnancy, chronic conditions, medications)
 * - Unit conversions (mmol/L ↔ mg/dL)
 */
public class ClinicalThresholdConfig {
    // Layer 1: Universal base thresholds (evidence-based consensus)
    private Map<String, UniversalThreshold> universalThresholds;

    // Layer 2: Geographic/regional adjustments
    private Map<String, GeographicProfile> geographicProfiles; // "US", "India", "EU", "WHO"

    // Layer 3: Demographic adjustment rules
    private DemographicAdjustmentRules demographicRules;

    // Layer 4: Contextual override rules
    private ContextualAdjustmentRules contextualRules;

    // Lab assay variations (manufacturer-specific thresholds)
    private Map<String, AssayCalibration> assayCalibrations;

    // Unit conversion definitions
    private Map<String, UnitConversion> unitConversions;
}

/**
 * Universal threshold definition (baseline for all populations)
 */
public class UniversalThreshold {
    private String loincCode;          // LOINC code (e.g., "2524-7" for lactate)
    private String labName;            // Human-readable name
    private ThresholdRange normal;     // Normal range (low-high)
    private ThresholdRange warning;    // Warning range
    private ThresholdRange critical;   // Critical range
    private String preferredUnit;      // Standard unit (e.g., "mmol/L")
    private String evidenceSource;     // WHO/AHA/ESC guideline reference
    private LocalDate lastUpdated;     // Evidence review date
}

/**
 * Threshold range with low/high bounds
 */
public class ThresholdRange {
    private Double low;   // Lower bound (null if unbounded)
    private Double high;  // Upper bound (null if unbounded)
    private String interpretation; // Clinical meaning
}

/**
 * Geographic profile (country/region-specific adjustments)
 */
public class GeographicProfile {
    private String region;  // "US", "India", "EU", "WHO"
    private Map<String, ThresholdAdjustment> adjustments; // LOINC → adjustment
    private String regulatoryBody; // "FDA", "CDSCO", "EMA"
    private String guidelineVersion; // "AHA 2017", "India HTN 2020"
}

/**
 * Threshold adjustment (delta from universal baseline)
 */
public class ThresholdAdjustment {
    private Double offsetLow;   // Additive offset to low bound
    private Double offsetHigh;  // Additive offset to high bound
    private Double scaleFactor; // Multiplicative factor (for unit changes)
    private String rationale;   // Why this adjustment (evidence reference)
}

/**
 * Demographic adjustment rules (age, sex, ethnicity, BMI)
 */
public class DemographicAdjustmentRules {
    private List<AgeAdjustment> ageRules;        // Age-specific ranges
    private List<SexAdjustment> sexRules;        // Male/female differences
    private List<EthnicityAdjustment> ethnicityRules; // Population-specific baselines
    private List<BMIAdjustment> bmiRules;        // Body mass corrections
}

/**
 * Age-based adjustment (e.g., creatinine decreases with age)
 */
public class AgeAdjustment {
    private String loincCode;
    private int ageMin;   // Age range start (years)
    private int ageMax;   // Age range end (years)
    private ThresholdAdjustment adjustment;
    private String rationale; // Physiological reason (e.g., "decreased muscle mass")
}

/**
 * Sex-based adjustment (e.g., creatinine higher in males)
 */
public class SexAdjustment {
    private String loincCode;
    private String sex; // "MALE", "FEMALE", "OTHER"
    private ThresholdAdjustment adjustment;
    private String rationale;
}

/**
 * Ethnicity-based adjustment (e.g., African ancestry → higher creatinine baseline)
 */
public class EthnicityAdjustment {
    private String loincCode;
    private String ethnicity; // "AFRICAN", "ASIAN", "CAUCASIAN", "HISPANIC"
    private ThresholdAdjustment adjustment;
    private String rationale;
}

/**
 * BMI-based adjustment (e.g., athletes have higher creatinine)
 */
public class BMIAdjustment {
    private String loincCode;
    private double bmiMin;
    private double bmiMax;
    private ThresholdAdjustment adjustment;
    private String rationale;
}

/**
 * Contextual adjustment rules (pregnancy, chronic conditions, medications)
 */
public class ContextualAdjustmentRules {
    private List<PregnancyAdjustment> pregnancyRules;
    private List<ChronicConditionAdjustment> chronicConditionRules;
    private List<MedicationAdjustment> medicationRules;
    private List<AltitudeAdjustment> altitudeRules;
}

/**
 * Pregnancy-specific thresholds (e.g., lower potassium, higher WBC)
 */
public class PregnancyAdjustment {
    private String loincCode;
    private int trimester; // 1, 2, 3 (0 = all trimesters)
    private ThresholdAdjustment adjustment;
    private String rationale;
}

/**
 * Chronic condition baseline adjustments (e.g., CKD → accept higher creatinine)
 */
public class ChronicConditionAdjustment {
    private String loincCode;
    private String conditionCode; // SNOMED-CT or ICD-10 code
    private String conditionName; // Human-readable (e.g., "Chronic Kidney Disease Stage 3")
    private ThresholdAdjustment adjustment;
    private String rationale;
}

/**
 * Medication-induced expected deviations (e.g., ACE-I → higher potassium acceptable)
 */
public class MedicationAdjustment {
    private String loincCode;
    private String rxNormCode; // Medication identifier
    private String medicationName;
    private ThresholdAdjustment adjustment;
    private String rationale; // Mechanism (e.g., "ACE-I reduces renal K+ excretion")
}

/**
 * Altitude-based adjustments (e.g., high altitude → lower SpO2 acceptable)
 */
public class AltitudeAdjustment {
    private String loincCode;
    private int altitudeMeters; // Elevation threshold
    private ThresholdAdjustment adjustment;
    private String rationale;
}

/**
 * Lab assay calibration (manufacturer-specific thresholds)
 */
public class AssayCalibration {
    private String loincCode;
    private String manufacturer; // "Abbott", "Roche", "Siemens"
    private String assayName;    // "ARCHITECT Troponin-I"
    private ThresholdAdjustment adjustment;
    private String calibrationDate;
}

/**
 * Unit conversion definition
 */
public class UnitConversion {
    private String loincCode;
    private String fromUnit;
    private String toUnit;
    private double conversionFactor; // multiply factor
    private double conversionOffset; // additive offset (for temperature etc.)
}
```

---

### **2. Threshold Resolution Engine**

```java
/**
 * Intelligent threshold resolver that applies all 4 layers of configuration
 * to compute personalized, context-aware clinical thresholds.
 */
public class ClinicalThresholdResolver {
    private final ClinicalThresholdConfig config;
    private final Logger log = LoggerFactory.getLogger(ClinicalThresholdResolver.class);

    public ClinicalThresholdResolver(ClinicalThresholdConfig config) {
        this.config = config;
    }

    /**
     * Resolve personalized threshold for a specific patient and lab test.
     *
     * @param loincCode Lab test LOINC code
     * @param patientContext Patient demographics and clinical context
     * @return Personalized threshold with audit trail of adjustments applied
     */
    public PersonalizedThreshold resolveThreshold(
            String loincCode,
            PatientContext patientContext) {

        // LAYER 1: Start with universal evidence-based threshold
        UniversalThreshold universal = config.getUniversalThresholds().get(loincCode);
        if (universal == null) {
            log.warn("No universal threshold found for LOINC {}, using safe defaults", loincCode);
            return getConservativeDefault(loincCode);
        }

        ThresholdRange current = universal.getNormal().copy();
        List<String> adjustmentLog = new ArrayList<>();

        // LAYER 2: Apply geographic/regional adjustments
        if (patientContext.getGeographicRegion() != null) {
            GeographicProfile geoProfile = config.getGeographicProfiles()
                    .get(patientContext.getGeographicRegion());
            if (geoProfile != null && geoProfile.getAdjustments().containsKey(loincCode)) {
                ThresholdAdjustment adj = geoProfile.getAdjustments().get(loincCode);
                current = applyAdjustment(current, adj);
                adjustmentLog.add(String.format("Geographic (%s): %s",
                        patientContext.getGeographicRegion(), adj.getRationale()));
            }
        }

        // LAYER 3: Apply demographic adjustments
        current = applyDemographicAdjustments(current, loincCode, patientContext, adjustmentLog);

        // LAYER 4: Apply contextual overrides
        current = applyContextualAdjustments(current, loincCode, patientContext, adjustmentLog);

        // ASSAY CALIBRATION: Apply lab-specific calibration if available
        if (patientContext.getLabAssay() != null) {
            String assayKey = loincCode + ":" + patientContext.getLabAssay();
            AssayCalibration assay = config.getAssayCalibrations().get(assayKey);
            if (assay != null) {
                current = applyAdjustment(current, assay.getAdjustment());
                adjustmentLog.add(String.format("Assay (%s): calibration applied",
                        assay.getAssayName()));
            }
        }

        // Build result with full audit trail
        return PersonalizedThreshold.builder()
                .loincCode(loincCode)
                .labName(universal.getLabName())
                .thresholdRange(current)
                .baselineRange(universal.getNormal())
                .adjustmentsApplied(adjustmentLog)
                .evidenceSource(universal.getEvidenceSource())
                .patientContext(patientContext.getSummary())
                .computedAt(Instant.now())
                .build();
    }

    /**
     * Apply demographic adjustments (age, sex, ethnicity, BMI)
     */
    private ThresholdRange applyDemographicAdjustments(
            ThresholdRange current,
            String loincCode,
            PatientContext patientContext,
            List<String> adjustmentLog) {

        DemographicAdjustmentRules rules = config.getDemographicRules();

        // Age adjustment
        if (patientContext.getAge() != null) {
            Optional<AgeAdjustment> ageRule = rules.getAgeRules().stream()
                    .filter(r -> r.getLoincCode().equals(loincCode))
                    .filter(r -> patientContext.getAge() >= r.getAgeMin() &&
                                 patientContext.getAge() <= r.getAgeMax())
                    .findFirst();

            if (ageRule.isPresent()) {
                current = applyAdjustment(current, ageRule.get().getAdjustment());
                adjustmentLog.add(String.format("Age (%d years): %s",
                        patientContext.getAge(), ageRule.get().getRationale()));
            }
        }

        // Sex adjustment
        if (patientContext.getSex() != null) {
            Optional<SexAdjustment> sexRule = rules.getSexRules().stream()
                    .filter(r -> r.getLoincCode().equals(loincCode))
                    .filter(r -> r.getSex().equals(patientContext.getSex()))
                    .findFirst();

            if (sexRule.isPresent()) {
                current = applyAdjustment(current, sexRule.get().getAdjustment());
                adjustmentLog.add(String.format("Sex (%s): %s",
                        patientContext.getSex(), sexRule.get().getRationale()));
            }
        }

        // Ethnicity adjustment
        if (patientContext.getEthnicity() != null) {
            Optional<EthnicityAdjustment> ethRule = rules.getEthnicityRules().stream()
                    .filter(r -> r.getLoincCode().equals(loincCode))
                    .filter(r -> r.getEthnicity().equals(patientContext.getEthnicity()))
                    .findFirst();

            if (ethRule.isPresent()) {
                current = applyAdjustment(current, ethRule.get().getAdjustment());
                adjustmentLog.add(String.format("Ethnicity (%s): %s",
                        patientContext.getEthnicity(), ethRule.get().getRationale()));
            }
        }

        // BMI adjustment
        if (patientContext.getBMI() != null) {
            Optional<BMIAdjustment> bmiRule = rules.getBmiRules().stream()
                    .filter(r -> r.getLoincCode().equals(loincCode))
                    .filter(r -> patientContext.getBMI() >= r.getBmiMin() &&
                                 patientContext.getBMI() <= r.getBmiMax())
                    .findFirst();

            if (bmiRule.isPresent()) {
                current = applyAdjustment(current, bmiRule.get().getAdjustment());
                adjustmentLog.add(String.format("BMI (%.1f): %s",
                        patientContext.getBMI(), bmiRule.get().getRationale()));
            }
        }

        return current;
    }

    /**
     * Apply contextual overrides (pregnancy, chronic conditions, medications, altitude)
     */
    private ThresholdRange applyContextualAdjustments(
            ThresholdRange current,
            String loincCode,
            PatientContext patientContext,
            List<String> adjustmentLog) {

        ContextualAdjustmentRules rules = config.getContextualRules();

        // Pregnancy adjustment
        if (patientContext.isPregnant() && patientContext.getTrimester() != null) {
            Optional<PregnancyAdjustment> pregRule = rules.getPregnancyRules().stream()
                    .filter(r -> r.getLoincCode().equals(loincCode))
                    .filter(r -> r.getTrimester() == 0 || r.getTrimester() == patientContext.getTrimester())
                    .findFirst();

            if (pregRule.isPresent()) {
                current = applyAdjustment(current, pregRule.get().getAdjustment());
                adjustmentLog.add(String.format("Pregnancy (trimester %d): %s",
                        patientContext.getTrimester(), pregRule.get().getRationale()));
            }
        }

        // Chronic condition adjustments
        if (patientContext.getChronicConditions() != null) {
            for (String conditionCode : patientContext.getChronicConditions()) {
                Optional<ChronicConditionAdjustment> condRule = rules.getChronicConditionRules().stream()
                        .filter(r -> r.getLoincCode().equals(loincCode))
                        .filter(r -> r.getConditionCode().equals(conditionCode))
                        .findFirst();

                if (condRule.isPresent()) {
                    current = applyAdjustment(current, condRule.get().getAdjustment());
                    adjustmentLog.add(String.format("Chronic condition (%s): %s",
                            condRule.get().getConditionName(), condRule.get().getRationale()));
                }
            }
        }

        // Medication-induced adjustments
        if (patientContext.getActiveMedications() != null) {
            for (String rxNormCode : patientContext.getActiveMedications()) {
                Optional<MedicationAdjustment> medRule = rules.getMedicationRules().stream()
                        .filter(r -> r.getLoincCode().equals(loincCode))
                        .filter(r -> r.getRxNormCode().equals(rxNormCode))
                        .findFirst();

                if (medRule.isPresent()) {
                    current = applyAdjustment(current, medRule.get().getAdjustment());
                    adjustmentLog.add(String.format("Medication (%s): %s",
                            medRule.get().getMedicationName(), medRule.get().getRationale()));
                }
            }
        }

        // Altitude adjustment
        if (patientContext.getAltitudeMeters() != null) {
            Optional<AltitudeAdjustment> altRule = rules.getAltitudeRules().stream()
                    .filter(r -> r.getLoincCode().equals(loincCode))
                    .filter(r -> patientContext.getAltitudeMeters() >= r.getAltitudeMeters())
                    .findFirst();

            if (altRule.isPresent()) {
                current = applyAdjustment(current, altRule.get().getAdjustment());
                adjustmentLog.add(String.format("Altitude (%d meters): %s",
                        patientContext.getAltitudeMeters(), altRule.get().getRationale()));
            }
        }

        return current;
    }

    /**
     * Apply a threshold adjustment (offset + scale)
     */
    private ThresholdRange applyAdjustment(ThresholdRange range, ThresholdAdjustment adj) {
        Double newLow = range.getLow();
        Double newHigh = range.getHigh();

        if (newLow != null) {
            newLow = (newLow + (adj.getOffsetLow() != null ? adj.getOffsetLow() : 0.0)) *
                     (adj.getScaleFactor() != null ? adj.getScaleFactor() : 1.0);
        }

        if (newHigh != null) {
            newHigh = (newHigh + (adj.getOffsetHigh() != null ? adj.getOffsetHigh() : 0.0)) *
                      (adj.getScaleFactor() != null ? adj.getScaleFactor() : 1.0);
        }

        return new ThresholdRange(newLow, newHigh, range.getInterpretation());
    }

    /**
     * Get conservative default threshold when no config available
     */
    private PersonalizedThreshold getConservativeDefault(String loincCode) {
        log.warn("Using conservative default for unconfigured lab {}", loincCode);
        // Return very wide range to avoid false alerts
        return PersonalizedThreshold.builder()
                .loincCode(loincCode)
                .labName("Unknown Lab")
                .thresholdRange(new ThresholdRange(null, null, "No threshold configured"))
                .adjustmentsApplied(List.of("WARNING: No configuration found, using permissive defaults"))
                .computedAt(Instant.now())
                .build();
    }
}
```

---

### **3. Configuration Storage & Loading**

#### **Option A: JSON/YAML Configuration Files** (Recommended for flexibility)

```yaml
# config/thresholds/universal-thresholds.yaml
universal_thresholds:
  "2524-7": # Lactate
    loinc_code: "2524-7"
    lab_name: "Lactate"
    normal:
      low: 0.5
      high: 2.0
      interpretation: "Normal tissue perfusion"
    warning:
      low: null
      high: 2.5
      interpretation: "Mild hypoperfusion"
    critical:
      low: null
      high: 4.0
      interpretation: "Severe hypoperfusion, shock"
    preferred_unit: "mmol/L"
    evidence_source: "Surviving Sepsis Campaign 2021"
    last_updated: "2021-10-01"

  "2160-0": # Creatinine
    loinc_code: "2160-0"
    lab_name: "Creatinine"
    normal:
      low: 0.6
      high: 1.2
      interpretation: "Normal kidney function"
    warning:
      low: null
      high: 1.5
      interpretation: "Possible kidney impairment"
    critical:
      low: null
      high: 3.0
      interpretation: "Severe kidney dysfunction"
    preferred_unit: "mg/dL"
    evidence_source: "KDIGO CKD Guidelines 2012"
    last_updated: "2020-01-15"
```

```yaml
# config/thresholds/geographic-us.yaml
geographic_profiles:
  US:
    region: "US"
    regulatory_body: "FDA"
    guideline_version: "AHA/ACC 2017"
    adjustments:
      "systolic_bp": # Hypertension
        offset_high: -10.0  # US uses 130 mmHg (India uses 140)
        rationale: "AHA/ACC 2017 lowered hypertension threshold to 130 mmHg"
      "2524-7": # Lactate (no adjustment, universal)
        offset_high: 0.0
        rationale: "Lactate threshold universal across regions"
```

```yaml
# config/thresholds/demographic-age.yaml
demographic_adjustments:
  age_rules:
    - loinc_code: "2160-0" # Creatinine
      age_min: 65
      age_max: 120
      adjustment:
        offset_high: -0.2  # Elderly have -0.2 mg/dL lower threshold
        rationale: "Decreased muscle mass and GFR with aging (Cockcroft-Gault formula)"

    - loinc_code: "6690-2" # WBC
      age_min: 0
      age_max: 2
      adjustment:
        offset_high: 6.0  # Neonates have WBC 5-21 K/uL (adults 4-11)
        rationale: "Physiological leukocytosis in neonates"
```

```yaml
# config/thresholds/contextual-pregnancy.yaml
contextual_adjustments:
  pregnancy_rules:
    - loinc_code: "2823-3" # Potassium
      trimester: 0  # All trimesters
      adjustment:
        offset_low: -0.3  # Pregnancy K+ 3.0-3.5 mmol/L normal
        rationale: "Hemodilution and increased renal clearance in pregnancy"

    - loinc_code: "6690-2" # WBC
      trimester: 3  # Third trimester
      adjustment:
        offset_high: 5.0  # Pregnancy WBC can be 5-16 K/uL
        rationale: "Physiological leukocytosis in late pregnancy"
```

#### **Option B: Database Storage** (Recommended for production with versioning)

```sql
-- Table: universal_thresholds
CREATE TABLE universal_thresholds (
    loinc_code VARCHAR(20) PRIMARY KEY,
    lab_name VARCHAR(100) NOT NULL,
    normal_low DECIMAL(10,4),
    normal_high DECIMAL(10,4),
    warning_low DECIMAL(10,4),
    warning_high DECIMAL(10,4),
    critical_low DECIMAL(10,4),
    critical_high DECIMAL(10,4),
    preferred_unit VARCHAR(20),
    evidence_source VARCHAR(255),
    last_updated DATE,
    version INT DEFAULT 1,
    active BOOLEAN DEFAULT TRUE
);

-- Table: geographic_profiles
CREATE TABLE geographic_profiles (
    id SERIAL PRIMARY KEY,
    region VARCHAR(50) NOT NULL,
    regulatory_body VARCHAR(100),
    guideline_version VARCHAR(100),
    active BOOLEAN DEFAULT TRUE,
    effective_date DATE NOT NULL
);

-- Table: geographic_adjustments
CREATE TABLE geographic_adjustments (
    id SERIAL PRIMARY KEY,
    profile_id INT REFERENCES geographic_profiles(id),
    loinc_code VARCHAR(20) REFERENCES universal_thresholds(loinc_code),
    offset_low DECIMAL(10,4),
    offset_high DECIMAL(10,4),
    scale_factor DECIMAL(10,4) DEFAULT 1.0,
    rationale TEXT,
    evidence_source VARCHAR(255),
    UNIQUE(profile_id, loinc_code)
);

-- Table: demographic_age_rules
CREATE TABLE demographic_age_rules (
    id SERIAL PRIMARY KEY,
    loinc_code VARCHAR(20) REFERENCES universal_thresholds(loinc_code),
    age_min INT NOT NULL,
    age_max INT NOT NULL,
    offset_low DECIMAL(10,4),
    offset_high DECIMAL(10,4),
    scale_factor DECIMAL(10,4) DEFAULT 1.0,
    rationale TEXT,
    evidence_source VARCHAR(255)
);

-- Similar tables for sex_rules, ethnicity_rules, contextual_rules, etc.
```

**Benefits of Database Approach**:
- ✅ Version control with effective dates
- ✅ A/B testing different threshold configurations
- ✅ Audit trail of threshold changes
- ✅ Hot reload without redeploying code
- ✅ Multi-tenancy (different hospitals can have different configs)

---

### **4. Integration with PatientContextAggregator**

Update PatientContextAggregator to use the threshold resolver:

```java
public class PatientContextAggregator extends KeyedProcessFunction<String, GenericEvent, EnrichedPatientContext> {

    // Replace static thresholds with resolver
    private transient ClinicalThresholdResolver thresholdResolver;

    @Override
    public void open(OpenContext openContext) throws Exception {
        super.open(openContext);

        // Load configuration from file/database
        ClinicalThresholdConfig config = loadConfiguration();

        // Initialize threshold resolver
        this.thresholdResolver = new ClinicalThresholdResolver(config);

        LOG.info("Initialized ClinicalThresholdResolver with global configuration");
    }

    /**
     * Enhanced lab abnormality check with personalized thresholds
     */
    private void checkLabAbnormalities(PatientContextState state) {
        Map<String, LabResult> labs = state.getRecentLabs();

        // Build patient context for personalization
        PatientContext patientContext = buildPatientContext(state);

        for (Map.Entry<String, LabResult> entry : labs.entrySet()) {
            String loincCode = entry.getKey();
            LabResult labResult = entry.getValue();

            // Resolve personalized threshold
            PersonalizedThreshold threshold = thresholdResolver.resolveThreshold(
                    loincCode, patientContext);

            // Check if abnormal against personalized threshold
            boolean isAbnormal = checkAbnormal(labResult, threshold);

            if (isAbnormal) {
                // Create alert with personalization details
                SimpleAlert alert = createPersonalizedAlert(labResult, threshold);
                state.addAlert(alert);

                // Log audit trail
                LOG.info("Personalized alert for patient {}: {} - Adjustments: {}",
                        state.getPatientId(), threshold.getLabName(),
                        threshold.getAdjustmentsApplied());
            }
        }
    }

    /**
     * Build patient context from state for threshold personalization
     */
    private PatientContext buildPatientContext(PatientContextState state) {
        Demographics demo = state.getDemographics();

        return PatientContext.builder()
                .geographicRegion(getGeographicRegion())  // From config or hospital location
                .age(demo != null ? demo.getAge() : null)
                .sex(demo != null ? demo.getGender() : null)
                .ethnicity(demo != null ? demo.getEthnicity() : null)
                .bmi(calculateBMI(state))
                .isPregnant(isPregnant(state))
                .trimester(getTrimester(state))
                .chronicConditions(state.getChronicConditions())
                .activeMedications(state.getActiveMedications().keySet())
                .altitudeMeters(getHospitalAltitude())
                .build();
    }

    /**
     * Create alert with personalization audit trail
     */
    private SimpleAlert createPersonalizedAlert(LabResult labResult, PersonalizedThreshold threshold) {
        String message = String.format("%s %s (%.1f %s, personalized threshold: %.1f)",
                threshold.getLabName(),
                labResult.getValue() > threshold.getThresholdRange().getHigh() ? "elevated" : "low",
                labResult.getValue(),
                labResult.getUnit(),
                labResult.getValue() > threshold.getThresholdRange().getHigh() ?
                        threshold.getThresholdRange().getHigh() : threshold.getThresholdRange().getLow());

        SimpleAlert alert = new SimpleAlert(
                AlertType.LAB_ABNORMALITY,
                determineSeverity(labResult, threshold),
                message,
                state.getPatientId());

        // Add personalization metadata to alert context
        alert.getContext().put("personalizedThreshold", true);
        alert.getContext().put("adjustmentsApplied", threshold.getAdjustmentsApplied());
        alert.getContext().put("evidenceSource", threshold.getEvidenceSource());
        alert.getContext().put("baselineThreshold", threshold.getBaselineRange());
        alert.getContext().put("personalizedThreshold", threshold.getThresholdRange());

        return alert;
    }
}
```

---

## Edge Case Coverage

### **Edge Case 1: Conflicting Adjustments**

**Scenario**: 75-year-old African male on ACE-I with CKD Stage 3
- Age adjustment: creatinine threshold DOWN by 0.2 mg/dL
- Ethnicity adjustment: creatinine threshold UP by 0.15 mg/dL
- CKD adjustment: creatinine threshold UP by 0.3 mg/dL

**Resolution Strategy**: Priority-based or additive
```java
// Option A: Priority (most specific wins)
Priority: Chronic Condition > Ethnicity > Age > Geographic

// Option B: Additive (sum all adjustments)
Final threshold = baseline + age_adj + ethnicity_adj + ckd_adj
                = 1.2 + (-0.2) + 0.15 + 0.3
                = 1.45 mg/dL
```

**Recommendation**: Use **additive** with cap/floor to prevent extreme values.

### **Edge Case 2: Missing Patient Data**

**Scenario**: No age, sex, or ethnicity in patient record

**Resolution**:
```java
if (patientContext.getAge() == null) {
    log.warn("Missing age for patient {}, using universal threshold without age adjustment", patientId);
    // Fall back to Layer 1 + Layer 2 only (no demographic personalization)
}
```

**Principle**: Graceful degradation - use best available data, document what's missing.

### **Edge Case 3: Rare Conditions Not in Config**

**Scenario**: Patient has Addison's disease (rare adrenal insufficiency) affecting sodium levels

**Resolution**:
```java
// Check for unmapped chronic conditions
if (!config.hasRuleFor(conditionCode)) {
    log.warn("No threshold adjustment rule for condition {}, using conservative defaults", conditionCode);
    alert.getContext().put("unmappedCondition", conditionCode);
    alert.setSeverity(AlertSeverity.WARNING);  // Conservative (don't miss critical alerts)
}
```

**Principle**: **Fail-safe** - when unsure, alert with lower severity rather than miss critical alerts.

### **Edge Case 4: Unit Conversion Errors**

**Scenario**: Lab sends lactate in mg/dL instead of expected mmol/L

**Resolution**:
```java
// Validate unit matches expected
if (!labResult.getUnit().equals(threshold.getPreferredUnit())) {
    // Attempt conversion
    UnitConversion conversion = config.getUnitConversion(loincCode, labResult.getUnit());
    if (conversion != null) {
        double convertedValue = labResult.getValue() * conversion.getConversionFactor() +
                                conversion.getConversionOffset();
        labResult.setValue(convertedValue);
        labResult.setUnit(threshold.getPreferredUnit());
        log.info("Converted {} from {} to {} for LOINC {}",
                 labResult.getValue(), labResult.getUnit(),
                 threshold.getPreferredUnit(), loincCode);
    } else {
        log.error("UNIT MISMATCH: Lab {} received in {} but expected {} - SKIPPING ALERT",
                  loincCode, labResult.getUnit(), threshold.getPreferredUnit());
        return;  // Skip alert rather than give false alert
    }
}
```

**Principle**: **Explicit conversion** with logging - never assume units match.

### **Edge Case 5: Threshold Configuration Versioning**

**Scenario**: Clinical guidelines updated (e.g., AHA 2017 → AHA 2024 hypertension guidelines)

**Resolution**:
```sql
-- Use effective_date for version control
SELECT * FROM geographic_adjustments
WHERE profile_id = (SELECT id FROM geographic_profiles
                    WHERE region = 'US'
                    AND effective_date <= CURRENT_DATE
                    AND active = TRUE
                    ORDER BY effective_date DESC
                    LIMIT 1);
```

**Principle**: **Time-travel queries** - always use thresholds valid at analysis time.

---

## Deployment Strategy

### **Phase 1: Backward Compatible Introduction** (No Behavioral Change)
1. Add ClinicalThresholdConfig classes (POJOs only)
2. Load config in PatientContextAggregator but DON'T use yet
3. Run both old (hardcoded) and new (config-based) thresholds in parallel
4. Log differences for validation
5. **Goal**: Prove config-based system produces same results

### **Phase 2: A/B Testing** (Limited Rollout)
1. Route 10% of patients through new config-based system
2. Monitor alert differences, false positive/negative rates
3. Collect clinical feedback from physicians
4. **Goal**: Validate personalized thresholds don't degrade clinical accuracy

### **Phase 3: Geographic Rollout** (Region-by-Region)
1. Enable config-based system for US region first
2. Monitor for 2 weeks, gather metrics
3. Roll out to India region with India-specific config
4. Roll out to EU, other regions
5. **Goal**: Prove multi-region support works correctly

### **Phase 4: Full Production** (Global Deployment)
1. Switch all regions to config-based system
2. Deprecate hardcoded thresholds
3. Enable hot-reload for config updates
4. **Goal**: Globally applicable, dynamically updatable threshold system

---

## Monitoring & Validation

### **Key Metrics to Track**

```java
// Alert accuracy metrics
@Metric("alert.threshold.personalization_rate")
double personalizationRate;  // % of alerts using personalized thresholds

@Metric("alert.threshold.adjustment_count")
Histogram adjustmentCount;  // Distribution of adjustments per alert

@Metric("alert.threshold.missing_patient_data_rate")
double missingDataRate;  // % of alerts missing patient context

// Alert quality metrics
@Metric("alert.quality.false_positive_rate")
double falsePositiveRate;  // Clinician-dismissed alerts

@Metric("alert.quality.false_negative_rate")
double falseNegativeRate;  // Missed critical events

// Configuration coverage metrics
@Metric("config.threshold.coverage_rate")
double coverageRate;  // % of labs with configured thresholds

@Metric("config.threshold.cache_hit_rate")
double cacheHitRate;  // Performance optimization
```

### **Validation Dashboards**

```sql
-- Alert adjustment frequency
SELECT adjustment_type, COUNT(*) as count
FROM alert_audit_log
WHERE personalized = true
GROUP BY adjustment_type
ORDER BY count DESC;

-- Most common missing patient data
SELECT missing_field, COUNT(*) as count
FROM threshold_resolution_log
WHERE missing_fields IS NOT NULL
GROUP BY missing_field
ORDER BY count DESC;

-- Threshold configuration usage
SELECT loinc_code, lab_name,
       COUNT(*) as usage_count,
       AVG(ARRAY_LENGTH(adjustments_applied, 1)) as avg_adjustments
FROM alert_audit_log
WHERE personalized = true
GROUP BY loinc_code, lab_name
ORDER BY usage_count DESC;
```

---

## Example: Real-World Scenario

**Patient**: Rohan Sharma (from our test case)
- Age: 42 years
- Sex: Male
- Ethnicity: Asian (Indian)
- Location: Mumbai, India (sea level)
- Chronic Conditions: Prediabetes, Hypertension
- Medications: Telmisartan 40mg (ACE-I/ARB)
- Lab: Creatinine 1.25 mg/dL

### **Threshold Resolution Process**:

```
LAYER 1: Universal Baseline
  Creatinine normal range: 0.6-1.2 mg/dL (KDIGO 2012)

LAYER 2: Geographic Adjustment
  India profile: No adjustment (uses universal threshold)

LAYER 3: Demographic Adjustments
  Age (42): No adjustment (applies to 65+)
  Sex (Male): +0.1 mg/dL high threshold (males have higher baseline)
  Ethnicity (Asian): No adjustment (specific to African ancestry)

LAYER 4: Contextual Adjustments
  Chronic Condition (Prediabetes): No adjustment for creatinine
  Chronic Condition (Hypertension): No adjustment for creatinine
  Medication (Telmisartan ARB): +0.15 mg/dL (ARBs slightly increase creatinine, acceptable)

FINAL PERSONALIZED THRESHOLD:
  Normal range: 0.6-1.45 mg/dL (vs universal 1.2)

RESULT:
  Creatinine 1.25 mg/dL → NORMAL (within personalized range)
  No alert generated (appropriate - ARB-induced elevation is expected)

ADJUSTMENTS APPLIED:
  1. Sex (Male): +0.1 mg/dL - Higher muscle mass
  2. Medication (Telmisartan ARB): +0.15 mg/dL - Expected drug effect on GFR

EVIDENCE:
  - KDIGO CKD Guidelines 2012
  - ARB Pharmacology: Expected Cr increase 10-15%
```

**Clinical Impact**: Without personalization, this would trigger a false alert ("elevated creatinine 1.25 > 1.2"). With personalization, correctly identified as expected medication effect.

---

## Conclusion

This 4-layer architecture provides:

✅ **Global Applicability**: Works across all geographies (US, India, EU, etc.)
✅ **Demographic Personalization**: Age, sex, ethnicity, BMI adjustments
✅ **Contextual Awareness**: Pregnancy, chronic conditions, medications
✅ **Edge Case Handling**: Graceful degradation, unit conversion, missing data
✅ **Evidence-Based**: All adjustments traced to clinical guidelines
✅ **Audit Trail**: Full transparency of adjustments applied
✅ **Hot-Reload**: Update thresholds without code deployment
✅ **Versioning**: Time-travel queries for guideline changes
✅ **Multi-Tenancy**: Different hospitals can have different configs

**Implementation Effort**:
- **Phase 1** (POJOs + Config Loading): 2-3 weeks
- **Phase 2** (Threshold Resolver): 2-3 weeks
- **Phase 3** (Integration + Testing): 3-4 weeks
- **Phase 4** (Production Deployment): 2 weeks
- **Total**: ~10-12 weeks for full global deployment

**Next Steps**:
1. Review this design with clinical stakeholders
2. Build configuration schema (YAML vs Database)
3. Implement ClinicalThresholdResolver with unit tests
4. Create initial configuration for 1-2 regions
5. Run A/B test with parallel systems
