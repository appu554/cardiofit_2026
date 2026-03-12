package com.cardiofit.flink.cds.population;

import java.time.LocalDate;
import java.time.LocalDateTime;
import java.util.*;
import java.util.stream.Collectors;

/**
 * Phase 8 Module 4 - Population Health Module
 *
 * Service for population health analytics, cohort building, care gap detection,
 * and quality measure calculation.
 *
 * @author CardioFit Clinical Intelligence Team
 * @version 1.0.0
 * @since Phase 8
 */
public class PopulationHealthService {

    /**
     * Build a patient cohort based on criteria
     */
    public PatientCohort buildCohort(String cohortName,
                                     PatientCohort.CohortType cohortType,
                                     List<PatientCohort.CriteriaRule> inclusionCriteria,
                                     List<PatientCohort.CriteriaRule> exclusionCriteria) {
        PatientCohort cohort = new PatientCohort(cohortName, cohortType);

        // Add criteria
        if (inclusionCriteria != null) {
            inclusionCriteria.forEach(cohort::addInclusionCriteria);
        }
        if (exclusionCriteria != null) {
            exclusionCriteria.forEach(cohort::addExclusionCriteria);
        }

        cohort.setCreatedAt(LocalDateTime.now());
        return cohort;
    }

    /**
     * Stratify cohort by risk level
     */
    public Map<PatientCohort.RiskLevel, List<String>> stratifyCohortByRisk(
            PatientCohort cohort,
            Map<String, Double> patientRiskScores) {

        Map<PatientCohort.RiskLevel, List<String>> stratification = new HashMap<>();

        // Initialize lists for each risk level
        for (PatientCohort.RiskLevel level : PatientCohort.RiskLevel.values()) {
            stratification.put(level, new ArrayList<>());
        }

        // Stratify patients
        for (String patientId : cohort.getPatientIds()) {
            Double riskScore = patientRiskScores.get(patientId);
            if (riskScore != null) {
                PatientCohort.RiskLevel level = categorizeRiskScore(riskScore);
                stratification.get(level).add(patientId);
            }
        }

        // Update cohort risk distribution
        for (Map.Entry<PatientCohort.RiskLevel, List<String>> entry : stratification.entrySet()) {
            cohort.updateRiskDistribution(entry.getKey(), entry.getValue().size());
        }

        // Calculate average risk score
        double avgRisk = patientRiskScores.values().stream()
            .mapToDouble(Double::doubleValue)
            .average()
            .orElse(0.0);
        cohort.setAverageRiskScore(avgRisk);

        return stratification;
    }

    /**
     * Categorize risk score into risk level
     */
    private PatientCohort.RiskLevel categorizeRiskScore(double riskScore) {
        if (riskScore < 0.2) return PatientCohort.RiskLevel.VERY_LOW;
        if (riskScore < 0.4) return PatientCohort.RiskLevel.LOW;
        if (riskScore < 0.6) return PatientCohort.RiskLevel.MODERATE;
        if (riskScore < 0.8) return PatientCohort.RiskLevel.HIGH;
        return PatientCohort.RiskLevel.VERY_HIGH;
    }

    /**
     * Identify care gaps for a patient
     */
    public List<CareGap> identifyCareGaps(String patientId,
                                          Map<String, Object> patientData) {
        List<CareGap> gaps = new ArrayList<>();

        // Preventive screening gaps
        gaps.addAll(identifyPreventiveScreeningGaps(patientId, patientData));

        // Chronic disease monitoring gaps
        gaps.addAll(identifyChronicDiseaseGaps(patientId, patientData));

        // Medication adherence gaps
        gaps.addAll(identifyMedicationAdherenceGaps(patientId, patientData));

        // Immunization gaps
        gaps.addAll(identifyImmunizationGaps(patientId, patientData));

        // Calculate days overdue for each gap
        gaps.forEach(CareGap::calculateDaysOverdue);

        return gaps;
    }

    /**
     * Identify preventive screening gaps
     */
    private List<CareGap> identifyPreventiveScreeningGaps(String patientId,
                                                          Map<String, Object> patientData) {
        List<CareGap> gaps = new ArrayList<>();

        // Example: Colonoscopy screening
        Object lastColonoscopy = patientData.get("last_colonoscopy_date");
        Integer age = (Integer) patientData.get("age");

        if (age != null && age >= 50 && age <= 75) {
            if (lastColonoscopy == null ||
                isOverdue((LocalDate) lastColonoscopy, 3650)) { // 10 years
                CareGap gap = new CareGap(
                    patientId,
                    CareGap.GapType.PREVENTIVE_SCREENING,
                    "Colorectal Cancer Screening"
                );
                gap.setCategory(CareGap.GapCategory.PREVENTIVE);
                gap.setDescription("Patient is due for colonoscopy screening");
                gap.setClinicalReason("Colorectal cancer screening reduces mortality");
                gap.setRecommendedAction("Schedule colonoscopy");
                gap.setSeverity(CareGap.GapSeverity.MODERATE);
                gap.setPriority(6);
                gap.setGuidelineReference("USPSTF 2021 Recommendations");
                gap.setQualityMeasureId("HEDIS COL");
                gap.setImpactsQualityMeasure(true);
                gap.setRelatedProcedure("45378"); // CPT for colonoscopy

                if (lastColonoscopy != null) {
                    gap.setDueDate(((LocalDate) lastColonoscopy).plusYears(10));
                } else {
                    gap.setDueDate(LocalDate.now());
                }

                gaps.add(gap);
            }
        }

        // Example: Mammography screening
        Object lastMammogram = patientData.get("last_mammogram_date");
        String gender = (String) patientData.get("gender");

        if ("F".equals(gender) && age != null && age >= 50 && age <= 74) {
            if (lastMammogram == null ||
                isOverdue((LocalDate) lastMammogram, 730)) { // 2 years
                CareGap gap = new CareGap(
                    patientId,
                    CareGap.GapType.PREVENTIVE_SCREENING,
                    "Breast Cancer Screening"
                );
                gap.setCategory(CareGap.GapCategory.PREVENTIVE);
                gap.setDescription("Patient is due for mammography");
                gap.setClinicalReason("Early detection improves breast cancer outcomes");
                gap.setRecommendedAction("Schedule bilateral mammography");
                gap.setSeverity(CareGap.GapSeverity.MODERATE);
                gap.setPriority(7);
                gap.setGuidelineReference("USPSTF 2016 Recommendations");
                gap.setQualityMeasureId("HEDIS BCS");
                gap.setImpactsQualityMeasure(true);
                gap.setRelatedProcedure("77067"); // CPT for mammography

                if (lastMammogram != null) {
                    gap.setDueDate(((LocalDate) lastMammogram).plusYears(2));
                } else {
                    gap.setDueDate(LocalDate.now());
                }

                gaps.add(gap);
            }
        }

        return gaps;
    }

    /**
     * Identify chronic disease monitoring gaps
     */
    private List<CareGap> identifyChronicDiseaseGaps(String patientId,
                                                     Map<String, Object> patientData) {
        List<CareGap> gaps = new ArrayList<>();

        Boolean hasDiabetes = (Boolean) patientData.get("has_diabetes");
        Object lastHbA1c = patientData.get("last_hba1c_date");

        if (Boolean.TRUE.equals(hasDiabetes)) {
            if (lastHbA1c == null ||
                isOverdue((LocalDate) lastHbA1c, 180)) { // 6 months
                CareGap gap = new CareGap(
                    patientId,
                    CareGap.GapType.CHRONIC_DISEASE_MONITORING,
                    "Diabetes HbA1c Testing"
                );
                gap.setCategory(CareGap.GapCategory.CHRONIC_MANAGEMENT);
                gap.setDescription("HbA1c testing overdue for diabetes management");
                gap.setClinicalReason("Regular HbA1c monitoring essential for diabetes control");
                gap.setRecommendedAction("Order HbA1c test");
                gap.setSeverity(CareGap.GapSeverity.HIGH);
                gap.setPriority(8);
                gap.setGuidelineReference("ADA 2023 Standards of Care");
                gap.setQualityMeasureId("HEDIS CDC-HbA1c");
                gap.setImpactsQualityMeasure(true);
                gap.setRelatedCondition("E11.9"); // ICD-10 for Type 2 diabetes
                gap.setRelatedLab("4548-4"); // LOINC for HbA1c
                gap.setUrgent(true);

                if (lastHbA1c != null) {
                    gap.setDueDate(((LocalDate) lastHbA1c).plusMonths(6));
                } else {
                    gap.setDueDate(LocalDate.now());
                }

                gaps.add(gap);
            }
        }

        return gaps;
    }

    /**
     * Identify medication adherence gaps
     */
    private List<CareGap> identifyMedicationAdherenceGaps(String patientId,
                                                          Map<String, Object> patientData) {
        List<CareGap> gaps = new ArrayList<>();

        Boolean hasHypertension = (Boolean) patientData.get("has_hypertension");
        Double medicationAdherence = (Double) patientData.get("bp_med_adherence"); // PDC (Proportion of Days Covered)

        if (Boolean.TRUE.equals(hasHypertension) &&
            (medicationAdherence == null || medicationAdherence < 0.8)) {
            CareGap gap = new CareGap(
                patientId,
                CareGap.GapType.MEDICATION_ADHERENCE,
                "Hypertension Medication Non-Adherence"
            );
            gap.setCategory(CareGap.GapCategory.MEDICATION);
            gap.setDescription(String.format("Blood pressure medication adherence: %.0f%% (target ≥80%%)",
                medicationAdherence != null ? medicationAdherence * 100 : 0));
            gap.setClinicalReason("Poor adherence increases cardiovascular risk");
            gap.setRecommendedAction("Patient outreach for adherence counseling");
            gap.setSeverity(CareGap.GapSeverity.HIGH);
            gap.setPriority(9);
            gap.setGuidelineReference("AHA/ACC Hypertension Guidelines");
            gap.setQualityMeasureId("HEDIS SAA");
            gap.setImpactsQualityMeasure(true);
            gap.setRelatedCondition("I10"); // ICD-10 for hypertension
            gap.setRelatedMedication("ACE Inhibitors / ARBs");
            gap.setDueDate(LocalDate.now().plusDays(7));

            gaps.add(gap);
        }

        return gaps;
    }

    /**
     * Identify immunization gaps
     */
    private List<CareGap> identifyImmunizationGaps(String patientId,
                                                   Map<String, Object> patientData) {
        List<CareGap> gaps = new ArrayList<>();

        Integer age = (Integer) patientData.get("age");
        Object lastFluShot = patientData.get("last_flu_vaccine_date");

        // Annual flu vaccine
        if (age != null && age >= 6) {
            if (lastFluShot == null ||
                ((LocalDate) lastFluShot).getYear() < LocalDate.now().getYear()) {
                CareGap gap = new CareGap(
                    patientId,
                    CareGap.GapType.IMMUNIZATION,
                    "Annual Influenza Vaccination"
                );
                gap.setCategory(CareGap.GapCategory.PREVENTIVE);
                gap.setDescription("Patient is due for annual flu vaccine");
                gap.setClinicalReason("Flu vaccination reduces morbidity and mortality");
                gap.setRecommendedAction("Administer seasonal influenza vaccine");
                gap.setSeverity(CareGap.GapSeverity.MODERATE);
                gap.setPriority(5);
                gap.setGuidelineReference("CDC ACIP Recommendations");
                gap.setQualityMeasureId("IMA");
                gap.setImpactsQualityMeasure(true);
                gap.setDueDate(LocalDate.of(LocalDate.now().getYear(), 10, 31)); // October 31st deadline

                gaps.add(gap);
            }
        }

        return gaps;
    }

    /**
     * Check if a date is overdue
     */
    private boolean isOverdue(LocalDate lastDate, int maxDays) {
        if (lastDate == null) return true;
        LocalDate dueDate = lastDate.plusDays(maxDays);
        return LocalDate.now().isAfter(dueDate);
    }

    /**
     * Calculate quality measure
     */
    public void calculateQualityMeasure(QualityMeasure measure,
                                        PatientCohort cohort,
                                        Map<String, Boolean> patientCompliance) {
        // Count numerator (patients meeting goal)
        int numerator = 0;
        for (Map.Entry<String, Boolean> entry : patientCompliance.entrySet()) {
            if (cohort.getPatientIds().contains(entry.getKey()) &&
                Boolean.TRUE.equals(entry.getValue())) {
                numerator++;
            }
        }

        // Denominator is cohort size (simplified - would need exclusions/exceptions)
        int denominator = cohort.getTotalPatients();

        measure.setNumeratorCount(numerator);
        measure.setDenominatorCount(denominator);
        measure.calculateComplianceRate();
    }

    /**
     * Get high-priority care gaps
     */
    public List<CareGap> getHighPriorityCareGaps(List<CareGap> allGaps) {
        return allGaps.stream()
            .filter(gap -> gap.getSeverity() == CareGap.GapSeverity.HIGH ||
                          gap.getSeverity() == CareGap.GapSeverity.CRITICAL ||
                          gap.isUrgent())
            .sorted(Comparator.comparingInt(CareGap::getPriority).reversed())
            .collect(Collectors.toList());
    }

    /**
     * Get overdue care gaps
     */
    public List<CareGap> getOverdueCareGaps(List<CareGap> allGaps) {
        return allGaps.stream()
            .filter(CareGap::isOverdue)
            .sorted(Comparator.comparingInt(CareGap::getDaysOverdue).reversed())
            .collect(Collectors.toList());
    }

    /**
     * Get care gaps by type
     */
    public Map<CareGap.GapType, List<CareGap>> groupCareGapsByType(List<CareGap> allGaps) {
        return allGaps.stream()
            .collect(Collectors.groupingBy(CareGap::getGapType));
    }

    /**
     * Calculate care gap closure rate
     */
    public double calculateCareGapClosureRate(List<CareGap> allGaps) {
        if (allGaps.isEmpty()) return 0.0;

        long closedGaps = allGaps.stream()
            .filter(gap -> !gap.isOpen())
            .count();

        return (double) closedGaps / allGaps.size() * 100.0;
    }

    /**
     * Get population health summary
     */
    public PopulationHealthSummary generatePopulationHealthSummary(PatientCohort cohort,
                                                                    List<CareGap> careGaps,
                                                                    List<QualityMeasure> qualityMeasures) {
        PopulationHealthSummary summary = new PopulationHealthSummary();
        summary.setCohortId(cohort.getCohortId());
        summary.setCohortName(cohort.getCohortName());
        summary.setTotalPatients(cohort.getTotalPatients());
        summary.setAverageRiskScore(cohort.getAverageRiskScore());
        summary.setHighRiskPatients(cohort.getHighRiskPatientCount());

        // Care gap statistics
        summary.setTotalCareGaps(careGaps.size());
        summary.setOpenCareGaps((int) careGaps.stream().filter(CareGap::isOpen).count());
        summary.setOverdueCareGaps((int) careGaps.stream().filter(CareGap::isOverdue).count());
        summary.setHighPriorityCareGaps(getHighPriorityCareGaps(careGaps).size());

        // Quality measure statistics
        if (!qualityMeasures.isEmpty()) {
            double avgCompliance = qualityMeasures.stream()
                .mapToDouble(QualityMeasure::getComplianceRate)
                .average()
                .orElse(0.0);
            summary.setAverageQualityCompliance(avgCompliance);

            long passingMeasures = qualityMeasures.stream()
                .filter(QualityMeasure::isPassing)
                .count();
            summary.setPassingQualityMeasures((int) passingMeasures);
            summary.setTotalQualityMeasures(qualityMeasures.size());
        }

        summary.setGeneratedAt(LocalDateTime.now());
        return summary;
    }

    /**
     * Population health summary report
     */
    public static class PopulationHealthSummary {
        private String cohortId;
        private String cohortName;
        private int totalPatients;
        private double averageRiskScore;
        private int highRiskPatients;
        private int totalCareGaps;
        private int openCareGaps;
        private int overdueCareGaps;
        private int highPriorityCareGaps;
        private double averageQualityCompliance;
        private int passingQualityMeasures;
        private int totalQualityMeasures;
        private LocalDateTime generatedAt;

        // Getters and setters
        public String getCohortId() { return cohortId; }
        public void setCohortId(String cohortId) { this.cohortId = cohortId; }
        public String getCohortName() { return cohortName; }
        public void setCohortName(String cohortName) { this.cohortName = cohortName; }
        public int getTotalPatients() { return totalPatients; }
        public void setTotalPatients(int totalPatients) { this.totalPatients = totalPatients; }
        public double getAverageRiskScore() { return averageRiskScore; }
        public void setAverageRiskScore(double averageRiskScore) { this.averageRiskScore = averageRiskScore; }
        public int getHighRiskPatients() { return highRiskPatients; }
        public void setHighRiskPatients(int highRiskPatients) { this.highRiskPatients = highRiskPatients; }
        public int getTotalCareGaps() { return totalCareGaps; }
        public void setTotalCareGaps(int totalCareGaps) { this.totalCareGaps = totalCareGaps; }
        public int getOpenCareGaps() { return openCareGaps; }
        public void setOpenCareGaps(int openCareGaps) { this.openCareGaps = openCareGaps; }
        public int getOverdueCareGaps() { return overdueCareGaps; }
        public void setOverdueCareGaps(int overdueCareGaps) { this.overdueCareGaps = overdueCareGaps; }
        public int getHighPriorityCareGaps() { return highPriorityCareGaps; }
        public void setHighPriorityCareGaps(int highPriorityCareGaps) { this.highPriorityCareGaps = highPriorityCareGaps; }
        public double getAverageQualityCompliance() { return averageQualityCompliance; }
        public void setAverageQualityCompliance(double averageQualityCompliance) { this.averageQualityCompliance = averageQualityCompliance; }
        public int getPassingQualityMeasures() { return passingQualityMeasures; }
        public void setPassingQualityMeasures(int passingQualityMeasures) { this.passingQualityMeasures = passingQualityMeasures; }
        public int getTotalQualityMeasures() { return totalQualityMeasures; }
        public void setTotalQualityMeasures(int totalQualityMeasures) { this.totalQualityMeasures = totalQualityMeasures; }
        public LocalDateTime getGeneratedAt() { return generatedAt; }
        public void setGeneratedAt(LocalDateTime generatedAt) { this.generatedAt = generatedAt; }

        @Override
        public String toString() {
            return String.format("PopulationHealthSummary{cohort='%s', patients=%d, avgRisk=%.2f, gaps=%d (open=%d, overdue=%d), quality=%.1f%%}",
                cohortName, totalPatients, averageRiskScore, totalCareGaps, openCareGaps, overdueCareGaps, averageQualityCompliance);
        }
    }
}
