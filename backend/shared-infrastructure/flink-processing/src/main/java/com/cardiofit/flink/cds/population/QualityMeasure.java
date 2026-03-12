package com.cardiofit.flink.cds.population;

import java.io.Serializable;
import java.time.LocalDate;
import java.time.LocalDateTime;
import java.util.*;

/**
 * Phase 8 Module 4 - Population Health Module
 *
 * Represents a clinical quality measure for population health reporting.
 * Implements HEDIS, CMS, and other quality measure specifications.
 *
 * @author CardioFit Clinical Intelligence Team
 * @version 1.0.0
 * @since Phase 8
 */
public class QualityMeasure implements Serializable {
    private static final long serialVersionUID = 1L;

    // Core Identification
    private String measureId;
    private String measureName;
    private String description;
    private MeasureType measureType;
    private MeasureSource source;

    // Specification
    private String version;                     // e.g., "HEDIS 2023"
    private String nqfNumber;                   // National Quality Forum ID
    private String cmsId;                       // CMS measure ID
    private String hedisCode;                   // HEDIS measure code

    // Measure Definition
    private List<MeasureCriterion> numeratorCriteria;
    private List<MeasureCriterion> denominatorCriteria;
    private List<MeasureCriterion> exclusionCriteria;
    private List<MeasureCriterion> exceptionCriteria;

    // Performance Period
    private LocalDate measurementStartDate;
    private LocalDate measurementEndDate;
    private int measurementPeriodDays;          // e.g., 365 for annual

    // Results
    private int numeratorCount;                 // Patients meeting goal
    private int denominatorCount;               // Eligible patients
    private int exclusionCount;                 // Excluded patients
    private int exceptionCount;                 // Exception patients
    private double complianceRate;              // Numerator / (Denominator - Exclusions - Exceptions)

    // Targets and Benchmarks
    private double targetRate;                  // Internal target
    private double nationalBenchmark;           // National average
    private double topDecileBenchmark;          // Top 10% performance
    private PerformanceLevel performanceLevel;

    // Stratification
    private Map<String, Double> stratificationResults; // Age group, gender, etc.

    // Clinical Context
    private String clinicalDomain;              // "Diabetes", "Cardiovascular", etc.
    private String relatedGuideline;
    private boolean isCoreSet;                  // CMS Core Set measure
    private boolean isStarRating;               // Medicare Star Rating measure

    // Metadata
    private LocalDateTime lastCalculated;
    private String calculatedBy;
    private boolean isActive;

    /**
     * Quality measure types
     */
    public enum MeasureType {
        PROCESS,            // Process measure (e.g., HbA1c tested)
        OUTCOME,            // Outcome measure (e.g., HbA1c <8%)
        STRUCTURE,          // Structure measure (e.g., EMR capability)
        PATIENT_EXPERIENCE, // Patient satisfaction
        EFFICIENCY,         // Cost-effectiveness
        COMPOSITE           // Multiple measures combined
    }

    /**
     * Quality measure sources
     */
    public enum MeasureSource {
        HEDIS,              // Healthcare Effectiveness Data and Information Set
        CMS,                // Centers for Medicare & Medicaid Services
        NQF,                // National Quality Forum
        TJC,                // The Joint Commission
        NCQA,               // National Committee for Quality Assurance
        CUSTOM              // Organization-specific
    }

    /**
     * Performance levels
     */
    public enum PerformanceLevel {
        NEEDS_IMPROVEMENT(1, "Below target", "red"),
        MEETS_TARGET(2, "Meets internal target", "yellow"),
        EXCEEDS_TARGET(3, "Exceeds target", "green"),
        TOP_DECILE(4, "Top 10% nationally", "blue");

        private final int rank;
        private final String description;
        private final String color;

        PerformanceLevel(int rank, String description, String color) {
            this.rank = rank;
            this.description = description;
            this.color = color;
        }

        public int getRank() { return rank; }
        public String getDescription() { return description; }
        public String getColor() { return color; }
    }

    /**
     * Measure criterion (for numerator/denominator/exclusion/exception)
     */
    public static class MeasureCriterion implements Serializable {
        private static final long serialVersionUID = 1L;

        private String criterionId;
        private String description;
        private CriterionType criterionType;
        private String codeSystem;             // ICD-10, LOINC, CPT, RxNorm
        private List<String> codes;            // Value set codes
        private String operator;               // >, <, =, BETWEEN, IN
        private Object value;
        private Object secondValue;            // For BETWEEN
        private String timeframe;              // "past 12 months", "current year"

        public enum CriterionType {
            DIAGNOSIS,          // ICD-10
            PROCEDURE,          // CPT
            LAB_VALUE,          // LOINC
            MEDICATION,         // RxNorm
            VITAL_SIGN,
            AGE_RANGE,
            GENDER,
            ENCOUNTER_TYPE,
            OBSERVATION,
            CUSTOM
        }

        public MeasureCriterion() {
            this.codes = new ArrayList<>();
        }

        public MeasureCriterion(CriterionType type, String codeSystem) {
            this();
            this.criterionType = type;
            this.codeSystem = codeSystem;
        }

        // Getters and setters
        public String getCriterionId() { return criterionId; }
        public void setCriterionId(String criterionId) { this.criterionId = criterionId; }
        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }
        public CriterionType getCriterionType() { return criterionType; }
        public void setCriterionType(CriterionType criterionType) { this.criterionType = criterionType; }
        public String getCodeSystem() { return codeSystem; }
        public void setCodeSystem(String codeSystem) { this.codeSystem = codeSystem; }
        public List<String> getCodes() { return codes; }
        public void setCodes(List<String> codes) { this.codes = codes; }
        public void addCode(String code) { this.codes.add(code); }
        public String getOperator() { return operator; }
        public void setOperator(String operator) { this.operator = operator; }
        public Object getValue() { return value; }
        public void setValue(Object value) { this.value = value; }
        public Object getSecondValue() { return secondValue; }
        public void setSecondValue(Object secondValue) { this.secondValue = secondValue; }
        public String getTimeframe() { return timeframe; }
        public void setTimeframe(String timeframe) { this.timeframe = timeframe; }
    }

    // Constructors
    public QualityMeasure() {
        this.measureId = generateMeasureId();
        this.numeratorCriteria = new ArrayList<>();
        this.denominatorCriteria = new ArrayList<>();
        this.exclusionCriteria = new ArrayList<>();
        this.exceptionCriteria = new ArrayList<>();
        this.stratificationResults = new HashMap<>();
        this.lastCalculated = LocalDateTime.now();
        this.isActive = true;
    }

    public QualityMeasure(String measureName, MeasureType measureType, MeasureSource source) {
        this();
        this.measureName = measureName;
        this.measureType = measureType;
        this.source = source;
    }

    private String generateMeasureId() {
        return "QM-" + System.currentTimeMillis();
    }

    /**
     * Calculate compliance rate
     */
    public void calculateComplianceRate() {
        int adjustedDenominator = denominatorCount - exclusionCount - exceptionCount;
        if (adjustedDenominator > 0) {
            this.complianceRate = (double) numeratorCount / adjustedDenominator * 100.0;
        } else {
            this.complianceRate = 0.0;
        }
        this.lastCalculated = LocalDateTime.now();

        // Determine performance level
        determinePerformanceLevel();
    }

    /**
     * Determine performance level based on compliance rate
     */
    private void determinePerformanceLevel() {
        if (topDecileBenchmark > 0 && complianceRate >= topDecileBenchmark) {
            this.performanceLevel = PerformanceLevel.TOP_DECILE;
        } else if (targetRate > 0 && complianceRate >= targetRate + 5) {
            this.performanceLevel = PerformanceLevel.EXCEEDS_TARGET;
        } else if (targetRate > 0 && complianceRate >= targetRate) {
            this.performanceLevel = PerformanceLevel.MEETS_TARGET;
        } else {
            this.performanceLevel = PerformanceLevel.NEEDS_IMPROVEMENT;
        }
    }

    /**
     * Add numerator criterion
     */
    public void addNumeratorCriterion(MeasureCriterion criterion) {
        this.numeratorCriteria.add(criterion);
    }

    /**
     * Add denominator criterion
     */
    public void addDenominatorCriterion(MeasureCriterion criterion) {
        this.denominatorCriteria.add(criterion);
    }

    /**
     * Add exclusion criterion
     */
    public void addExclusionCriterion(MeasureCriterion criterion) {
        this.exclusionCriteria.add(criterion);
    }

    /**
     * Add exception criterion
     */
    public void addExceptionCriterion(MeasureCriterion criterion) {
        this.exceptionCriteria.add(criterion);
    }

    /**
     * Add stratification result
     */
    public void addStratification(String stratum, double rate) {
        this.stratificationResults.put(stratum, rate);
    }

    /**
     * Get gap from target
     */
    public double getGapFromTarget() {
        if (targetRate > 0) {
            return targetRate - complianceRate;
        }
        return 0.0;
    }

    /**
     * Check if measure is passing
     */
    public boolean isPassing() {
        return targetRate > 0 && complianceRate >= targetRate;
    }

    // Getters and Setters
    public String getMeasureId() {
        return measureId;
    }

    public void setMeasureId(String measureId) {
        this.measureId = measureId;
    }

    public String getMeasureName() {
        return measureName;
    }

    public void setMeasureName(String measureName) {
        this.measureName = measureName;
    }

    public String getDescription() {
        return description;
    }

    public void setDescription(String description) {
        this.description = description;
    }

    public MeasureType getMeasureType() {
        return measureType;
    }

    public void setMeasureType(MeasureType measureType) {
        this.measureType = measureType;
    }

    public MeasureSource getSource() {
        return source;
    }

    public void setSource(MeasureSource source) {
        this.source = source;
    }

    public String getVersion() {
        return version;
    }

    public void setVersion(String version) {
        this.version = version;
    }

    public String getNqfNumber() {
        return nqfNumber;
    }

    public void setNqfNumber(String nqfNumber) {
        this.nqfNumber = nqfNumber;
    }

    public String getCmsId() {
        return cmsId;
    }

    public void setCmsId(String cmsId) {
        this.cmsId = cmsId;
    }

    public String getHedisCode() {
        return hedisCode;
    }

    public void setHedisCode(String hedisCode) {
        this.hedisCode = hedisCode;
    }

    public List<MeasureCriterion> getNumeratorCriteria() {
        return numeratorCriteria;
    }

    public void setNumeratorCriteria(List<MeasureCriterion> numeratorCriteria) {
        this.numeratorCriteria = numeratorCriteria;
    }

    public List<MeasureCriterion> getDenominatorCriteria() {
        return denominatorCriteria;
    }

    public void setDenominatorCriteria(List<MeasureCriterion> denominatorCriteria) {
        this.denominatorCriteria = denominatorCriteria;
    }

    public List<MeasureCriterion> getExclusionCriteria() {
        return exclusionCriteria;
    }

    public void setExclusionCriteria(List<MeasureCriterion> exclusionCriteria) {
        this.exclusionCriteria = exclusionCriteria;
    }

    public List<MeasureCriterion> getExceptionCriteria() {
        return exceptionCriteria;
    }

    public void setExceptionCriteria(List<MeasureCriterion> exceptionCriteria) {
        this.exceptionCriteria = exceptionCriteria;
    }

    public LocalDate getMeasurementStartDate() {
        return measurementStartDate;
    }

    public void setMeasurementStartDate(LocalDate measurementStartDate) {
        this.measurementStartDate = measurementStartDate;
    }

    public LocalDate getMeasurementEndDate() {
        return measurementEndDate;
    }

    public void setMeasurementEndDate(LocalDate measurementEndDate) {
        this.measurementEndDate = measurementEndDate;
    }

    public int getMeasurementPeriodDays() {
        return measurementPeriodDays;
    }

    public void setMeasurementPeriodDays(int measurementPeriodDays) {
        this.measurementPeriodDays = measurementPeriodDays;
    }

    public int getNumeratorCount() {
        return numeratorCount;
    }

    public void setNumeratorCount(int numeratorCount) {
        this.numeratorCount = numeratorCount;
    }

    public int getDenominatorCount() {
        return denominatorCount;
    }

    public void setDenominatorCount(int denominatorCount) {
        this.denominatorCount = denominatorCount;
    }

    public int getExclusionCount() {
        return exclusionCount;
    }

    public void setExclusionCount(int exclusionCount) {
        this.exclusionCount = exclusionCount;
    }

    public int getExceptionCount() {
        return exceptionCount;
    }

    public void setExceptionCount(int exceptionCount) {
        this.exceptionCount = exceptionCount;
    }

    public double getComplianceRate() {
        return complianceRate;
    }

    public void setComplianceRate(double complianceRate) {
        this.complianceRate = complianceRate;
    }

    public double getTargetRate() {
        return targetRate;
    }

    public void setTargetRate(double targetRate) {
        this.targetRate = targetRate;
    }

    public double getNationalBenchmark() {
        return nationalBenchmark;
    }

    public void setNationalBenchmark(double nationalBenchmark) {
        this.nationalBenchmark = nationalBenchmark;
    }

    public double getTopDecileBenchmark() {
        return topDecileBenchmark;
    }

    public void setTopDecileBenchmark(double topDecileBenchmark) {
        this.topDecileBenchmark = topDecileBenchmark;
    }

    public PerformanceLevel getPerformanceLevel() {
        return performanceLevel;
    }

    public void setPerformanceLevel(PerformanceLevel performanceLevel) {
        this.performanceLevel = performanceLevel;
    }

    public Map<String, Double> getStratificationResults() {
        return stratificationResults;
    }

    public void setStratificationResults(Map<String, Double> stratificationResults) {
        this.stratificationResults = stratificationResults;
    }

    public String getClinicalDomain() {
        return clinicalDomain;
    }

    public void setClinicalDomain(String clinicalDomain) {
        this.clinicalDomain = clinicalDomain;
    }

    public String getRelatedGuideline() {
        return relatedGuideline;
    }

    public void setRelatedGuideline(String relatedGuideline) {
        this.relatedGuideline = relatedGuideline;
    }

    public boolean isCoreSet() {
        return isCoreSet;
    }

    public void setCoreSet(boolean coreSet) {
        isCoreSet = coreSet;
    }

    public boolean isStarRating() {
        return isStarRating;
    }

    public void setStarRating(boolean starRating) {
        isStarRating = starRating;
    }

    public LocalDateTime getLastCalculated() {
        return lastCalculated;
    }

    public void setLastCalculated(LocalDateTime lastCalculated) {
        this.lastCalculated = lastCalculated;
    }

    public String getCalculatedBy() {
        return calculatedBy;
    }

    public void setCalculatedBy(String calculatedBy) {
        this.calculatedBy = calculatedBy;
    }

    public boolean isActive() {
        return isActive;
    }

    public void setActive(boolean active) {
        isActive = active;
    }

    @Override
    public String toString() {
        return String.format("QualityMeasure{id='%s', name='%s', type=%s, compliance=%.1f%%, target=%.1f%%, level=%s}",
            measureId, measureName, measureType, complianceRate, targetRate, performanceLevel);
    }
}
