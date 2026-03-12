package com.cardiofit.flink.cds.population;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;

import java.time.LocalDate;
import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for PopulationHealthService
 * Phase 8 Module 4 - Population Health Module
 *
 * Test Coverage:
 * - Cohort building with criteria
 * - Risk stratification logic
 * - Care gap detection (all 4 types)
 * - Quality measure calculation
 * - Aggregate analysis methods
 * - Population health summary generation
 *
 * @author CardioFit Testing Team
 * @version 1.0.0
 * @since Phase 8
 */
@DisplayName("PopulationHealthService Tests")
class PopulationHealthServiceTest {

    private PopulationHealthService service;

    @BeforeEach
    void setUp() {
        service = new PopulationHealthService();
    }

    @Nested
    @DisplayName("Cohort Building Tests")
    class CohortBuildingTests {

        @Test
        @DisplayName("Should build cohort with inclusion criteria")
        void testBuildCohortWithInclusion() {
            List<PatientCohort.CriteriaRule> inclusionCriteria = new ArrayList<>();

            PatientCohort.CriteriaRule diagnosisRule = new PatientCohort.CriteriaRule(
                PatientCohort.CriteriaRule.CriteriaType.DIAGNOSIS,
                "ICD-10",
                "=",
                "E11.9"
            );
            diagnosisRule.setDescription("Type 2 Diabetes");
            inclusionCriteria.add(diagnosisRule);

            PatientCohort cohort = service.buildCohort(
                "Diabetes Management Cohort",
                PatientCohort.CohortType.DISEASE_BASED,
                inclusionCriteria,
                null
            );

            assertNotNull(cohort);
            assertEquals("Diabetes Management Cohort", cohort.getCohortName());
            assertEquals(PatientCohort.CohortType.DISEASE_BASED, cohort.getCohortType());
            assertEquals(1, cohort.getInclusionCriteria().size());
            assertEquals("Type 2 Diabetes", cohort.getInclusionCriteria().get(0).getDescription());
        }

        @Test
        @DisplayName("Should build cohort with exclusion criteria")
        void testBuildCohortWithExclusion() {
            List<PatientCohort.CriteriaRule> exclusionCriteria = new ArrayList<>();

            PatientCohort.CriteriaRule ageRule = new PatientCohort.CriteriaRule(
                PatientCohort.CriteriaRule.CriteriaType.AGE,
                "age",
                "<",
                18
            );
            ageRule.setDescription("Exclude pediatric");
            exclusionCriteria.add(ageRule);

            PatientCohort cohort = service.buildCohort(
                "Adult Hypertension Cohort",
                PatientCohort.CohortType.DISEASE_BASED,
                null,
                exclusionCriteria
            );

            assertNotNull(cohort);
            assertEquals(1, cohort.getExclusionCriteria().size());
            assertEquals("Exclude pediatric", cohort.getExclusionCriteria().get(0).getDescription());
        }

        @Test
        @DisplayName("Should build cohort with both inclusion and exclusion criteria")
        void testBuildCohortWithBothCriteria() {
            List<PatientCohort.CriteriaRule> inclusionCriteria = Arrays.asList(
                new PatientCohort.CriteriaRule(
                    PatientCohort.CriteriaRule.CriteriaType.DIAGNOSIS,
                    "ICD-10",
                    "=",
                    "I10"
                )
            );

            List<PatientCohort.CriteriaRule> exclusionCriteria = Arrays.asList(
                new PatientCohort.CriteriaRule(
                    PatientCohort.CriteriaRule.CriteriaType.AGE,
                    "age",
                    "<",
                    18
                )
            );

            PatientCohort cohort = service.buildCohort(
                "Adult Hypertension",
                PatientCohort.CohortType.DISEASE_BASED,
                inclusionCriteria,
                exclusionCriteria
            );

            assertEquals(1, cohort.getInclusionCriteria().size());
            assertEquals(1, cohort.getExclusionCriteria().size());
        }

        @Test
        @DisplayName("Should build cohort with null criteria lists")
        void testBuildCohortWithNullCriteria() {
            PatientCohort cohort = service.buildCohort(
                "Test Cohort",
                PatientCohort.CohortType.CUSTOM,
                null,
                null
            );

            assertNotNull(cohort);
            assertEquals(0, cohort.getInclusionCriteria().size());
            assertEquals(0, cohort.getExclusionCriteria().size());
        }
    }

    @Nested
    @DisplayName("Risk Stratification Tests")
    class RiskStratificationTests {

        @Test
        @DisplayName("Should stratify cohort by risk scores")
        void testStratifyCohortByRisk() {
            PatientCohort cohort = new PatientCohort("Test Cohort", PatientCohort.CohortType.RISK_BASED);
            cohort.addPatient("PATIENT-001");
            cohort.addPatient("PATIENT-002");
            cohort.addPatient("PATIENT-003");
            cohort.addPatient("PATIENT-004");
            cohort.addPatient("PATIENT-005");

            Map<String, Double> patientRiskScores = new HashMap<>();
            patientRiskScores.put("PATIENT-001", 0.15); // VERY_LOW
            patientRiskScores.put("PATIENT-002", 0.35); // LOW
            patientRiskScores.put("PATIENT-003", 0.55); // MODERATE
            patientRiskScores.put("PATIENT-004", 0.75); // HIGH
            patientRiskScores.put("PATIENT-005", 0.95); // VERY_HIGH

            Map<PatientCohort.RiskLevel, List<String>> stratification =
                service.stratifyCohortByRisk(cohort, patientRiskScores);

            assertEquals(1, stratification.get(PatientCohort.RiskLevel.VERY_LOW).size());
            assertTrue(stratification.get(PatientCohort.RiskLevel.VERY_LOW).contains("PATIENT-001"));

            assertEquals(1, stratification.get(PatientCohort.RiskLevel.LOW).size());
            assertTrue(stratification.get(PatientCohort.RiskLevel.LOW).contains("PATIENT-002"));

            assertEquals(1, stratification.get(PatientCohort.RiskLevel.MODERATE).size());
            assertEquals(1, stratification.get(PatientCohort.RiskLevel.HIGH).size());
            assertEquals(1, stratification.get(PatientCohort.RiskLevel.VERY_HIGH).size());
        }

        @Test
        @DisplayName("Should handle missing risk scores")
        void testMissingRiskScores() {
            PatientCohort cohort = new PatientCohort("Test Cohort", PatientCohort.CohortType.RISK_BASED);
            cohort.addPatient("PATIENT-001");
            cohort.addPatient("PATIENT-002");

            Map<String, Double> patientRiskScores = new HashMap<>();
            patientRiskScores.put("PATIENT-001", 0.55); // Only one patient has score

            Map<PatientCohort.RiskLevel, List<String>> stratification =
                service.stratifyCohortByRisk(cohort, patientRiskScores);

            assertEquals(1, stratification.get(PatientCohort.RiskLevel.MODERATE).size());
            // PATIENT-002 should not appear in any risk level
        }

        @Test
        @DisplayName("Should correctly categorize risk thresholds")
        void testRiskThresholdBoundaries() {
            PatientCohort cohort = new PatientCohort("Test Cohort", PatientCohort.CohortType.RISK_BASED);
            cohort.addPatient("P1");
            cohort.addPatient("P2");
            cohort.addPatient("P3");
            cohort.addPatient("P4");
            cohort.addPatient("P5");

            Map<String, Double> scores = new HashMap<>();
            scores.put("P1", 0.19); // VERY_LOW (< 0.2)
            scores.put("P2", 0.39); // LOW (< 0.4)
            scores.put("P3", 0.59); // MODERATE (< 0.6)
            scores.put("P4", 0.79); // HIGH (< 0.8)
            scores.put("P5", 0.81); // VERY_HIGH (>= 0.8)

            Map<PatientCohort.RiskLevel, List<String>> stratification =
                service.stratifyCohortByRisk(cohort, scores);

            assertTrue(stratification.get(PatientCohort.RiskLevel.VERY_LOW).contains("P1"));
            assertTrue(stratification.get(PatientCohort.RiskLevel.LOW).contains("P2"));
            assertTrue(stratification.get(PatientCohort.RiskLevel.MODERATE).contains("P3"));
            assertTrue(stratification.get(PatientCohort.RiskLevel.HIGH).contains("P4"));
            assertTrue(stratification.get(PatientCohort.RiskLevel.VERY_HIGH).contains("P5"));
        }
    }

    @Nested
    @DisplayName("Care Gap Detection Tests")
    class CareGapDetectionTests {

        @Test
        @DisplayName("Should identify colonoscopy screening gap")
        void testColonoscopyScreeningGap() {
            Map<String, Object> patientData = new HashMap<>();
            patientData.put("age", 60);
            patientData.put("last_colonoscopy_date", LocalDate.now().minusYears(11)); // Overdue

            List<CareGap> gaps = service.identifyCareGaps("PATIENT-001", patientData);

            // Should identify colonoscopy gap and potentially flu vaccine
            assertTrue(gaps.size() >= 1);

            CareGap colonoscopyGap = gaps.stream()
                .filter(g -> g.getGapName().equals("Colorectal Cancer Screening"))
                .findFirst()
                .orElse(null);

            assertNotNull(colonoscopyGap);
            assertEquals(CareGap.GapType.PREVENTIVE_SCREENING, colonoscopyGap.getGapType());
            assertEquals("HEDIS COL", colonoscopyGap.getQualityMeasureId());
            assertTrue(colonoscopyGap.getDaysOverdue() > 0);
        }

        @Test
        @DisplayName("Should not identify colonoscopy gap for recent screening")
        void testNoColonoscopyGapForRecent() {
            Map<String, Object> patientData = new HashMap<>();
            patientData.put("age", 60);
            patientData.put("last_colonoscopy_date", LocalDate.now().minusYears(2)); // Recent

            List<CareGap> gaps = service.identifyCareGaps("PATIENT-001", patientData);

            // Should have no colonoscopy gap
            long colonoscopyGaps = gaps.stream()
                .filter(g -> g.getGapName().contains("Colorectal"))
                .count();
            assertEquals(0, colonoscopyGaps);
        }

        @Test
        @DisplayName("Should identify mammography screening gap")
        void testMammographyScreeningGap() {
            Map<String, Object> patientData = new HashMap<>();
            patientData.put("age", 55);
            patientData.put("gender", "F"); // Must be "F" not "Female"
            patientData.put("last_mammogram_date", LocalDate.now().minusYears(3)); // Overdue

            List<CareGap> gaps = service.identifyCareGaps("PATIENT-002", patientData);

            long mammographyGaps = gaps.stream()
                .filter(g -> g.getGapName().contains("Breast Cancer"))
                .count();
            assertEquals(1, mammographyGaps);

            CareGap gap = gaps.stream()
                .filter(g -> g.getGapName().contains("Breast Cancer"))
                .findFirst()
                .orElse(null);

            assertNotNull(gap);
            assertEquals("HEDIS BCS", gap.getQualityMeasureId());
        }

        @Test
        @DisplayName("Should identify diabetes HbA1c monitoring gap")
        void testDiabetesHbA1cGap() {
            Map<String, Object> patientData = new HashMap<>();
            patientData.put("has_diabetes", true);
            patientData.put("last_hba1c_date", LocalDate.now().minusMonths(8)); // Overdue (>6 months)

            List<CareGap> gaps = service.identifyCareGaps("PATIENT-003", patientData);

            long diabetesGaps = gaps.stream()
                .filter(g -> g.getGapName().contains("HbA1c"))
                .count();
            assertEquals(1, diabetesGaps);

            CareGap gap = gaps.stream()
                .filter(g -> g.getGapName().contains("HbA1c"))
                .findFirst()
                .orElse(null);

            assertNotNull(gap);
            assertEquals(CareGap.GapType.CHRONIC_DISEASE_MONITORING, gap.getGapType());
            assertEquals(CareGap.GapSeverity.HIGH, gap.getSeverity());
            assertTrue(gap.isUrgent());
            assertEquals("HEDIS CDC-HbA1c", gap.getQualityMeasureId());
            assertEquals("E11.9", gap.getRelatedCondition());
            assertEquals("4548-4", gap.getRelatedLab());
        }

        @Test
        @DisplayName("Should identify hypertension medication adherence gap")
        void testHypertensionMedicationGap() {
            Map<String, Object> patientData = new HashMap<>();
            patientData.put("has_hypertension", true);
            patientData.put("bp_med_adherence", 0.65); // Below 80% threshold

            List<CareGap> gaps = service.identifyCareGaps("PATIENT-004", patientData);

            long medicationGaps = gaps.stream()
                .filter(g -> g.getGapType() == CareGap.GapType.MEDICATION_ADHERENCE)
                .count();
            assertEquals(1, medicationGaps);

            CareGap gap = gaps.stream()
                .filter(g -> g.getGapType() == CareGap.GapType.MEDICATION_ADHERENCE)
                .findFirst()
                .orElse(null);

            assertNotNull(gap);
            assertEquals("Hypertension Medication Non-Adherence", gap.getGapName());
            assertEquals("I10", gap.getRelatedCondition());
        }

        @Test
        @DisplayName("Should identify flu vaccine immunization gap")
        void testFluVaccineGap() {
            Map<String, Object> patientData = new HashMap<>();
            patientData.put("age", 70);
            patientData.put("last_flu_vaccine_date", LocalDate.now().minusYears(2)); // Different year

            List<CareGap> gaps = service.identifyCareGaps("PATIENT-005", patientData);

            long immunizationGaps = gaps.stream()
                .filter(g -> g.getGapType() == CareGap.GapType.IMMUNIZATION)
                .count();
            assertEquals(1, immunizationGaps);

            CareGap gap = gaps.stream()
                .filter(g -> g.getGapType() == CareGap.GapType.IMMUNIZATION)
                .findFirst()
                .orElse(null);

            assertNotNull(gap);
            assertEquals("Annual Influenza Vaccination", gap.getGapName());
            assertEquals("IMA", gap.getQualityMeasureId()); // Quality measure ID is "IMA" not "HEDIS IMA"
        }

        @Test
        @DisplayName("Should identify multiple gaps for single patient")
        void testMultipleGaps() {
            Map<String, Object> patientData = new HashMap<>();
            patientData.put("age", 65);
            patientData.put("gender", "F"); // Must be "F"
            patientData.put("has_diabetes", true);
            patientData.put("last_colonoscopy_date", LocalDate.now().minusYears(11)); // Overdue
            patientData.put("last_mammogram_date", LocalDate.now().minusYears(3)); // Overdue
            patientData.put("last_hba1c_date", LocalDate.now().minusMonths(8)); // Overdue
            patientData.put("last_flu_vaccine_date", LocalDate.now().minusYears(2)); // Different year

            List<CareGap> gaps = service.identifyCareGaps("PATIENT-006", patientData);

            // Should identify: colonoscopy, mammography, HbA1c, flu vaccine
            assertTrue(gaps.size() >= 4);
        }

        @Test
        @DisplayName("Should calculate days overdue for all gaps")
        void testDaysOverdueCalculation() {
            Map<String, Object> patientData = new HashMap<>();
            patientData.put("has_diabetes", true);
            patientData.put("last_hba1c_date", LocalDate.now().minusMonths(8));

            List<CareGap> gaps = service.identifyCareGaps("PATIENT-007", patientData);

            // All gaps should have days overdue calculated
            for (CareGap gap : gaps) {
                assertNotNull(gap.getDaysOverdue());
            }
        }
    }

    @Nested
    @DisplayName("Quality Measure Calculation Tests")
    class QualityMeasureCalculationTests {

        @Test
        @DisplayName("Should calculate quality measure with patient compliance")
        void testCalculateQualityMeasure() {
            PatientCohort cohort = new PatientCohort("Diabetes Cohort", PatientCohort.CohortType.DISEASE_BASED);
            cohort.addPatient("PATIENT-001");
            cohort.addPatient("PATIENT-002");
            cohort.addPatient("PATIENT-003");
            cohort.addPatient("PATIENT-004");

            QualityMeasure measure = new QualityMeasure(
                "Diabetes HbA1c Testing",
                QualityMeasure.MeasureType.PROCESS,
                QualityMeasure.MeasureSource.HEDIS
            );

            Map<String, Boolean> patientCompliance = new HashMap<>();
            patientCompliance.put("PATIENT-001", true);
            patientCompliance.put("PATIENT-002", true);
            patientCompliance.put("PATIENT-003", true);
            patientCompliance.put("PATIENT-004", false);

            service.calculateQualityMeasure(measure, cohort, patientCompliance);

            assertEquals(4, measure.getDenominatorCount());
            assertEquals(3, measure.getNumeratorCount());
            assertEquals(75.0, measure.getComplianceRate(), 0.001);
        }

        @Test
        @DisplayName("Should only count patients in cohort")
        void testOnlyCountCohortPatients() {
            PatientCohort cohort = new PatientCohort("Test Cohort", PatientCohort.CohortType.DISEASE_BASED);
            cohort.addPatient("PATIENT-001");
            cohort.addPatient("PATIENT-002");

            QualityMeasure measure = new QualityMeasure(
                "Test Measure",
                QualityMeasure.MeasureType.PROCESS,
                QualityMeasure.MeasureSource.HEDIS
            );

            Map<String, Boolean> patientCompliance = new HashMap<>();
            patientCompliance.put("PATIENT-001", true);
            patientCompliance.put("PATIENT-002", false);
            patientCompliance.put("PATIENT-999", true); // Not in cohort

            service.calculateQualityMeasure(measure, cohort, patientCompliance);

            assertEquals(2, measure.getDenominatorCount());
            assertEquals(1, measure.getNumeratorCount());
            assertEquals(50.0, measure.getComplianceRate(), 0.001);
        }
    }

    @Nested
    @DisplayName("Aggregate Analysis Tests")
    class AggregateAnalysisTests {

        @Test
        @DisplayName("Should get high priority care gaps")
        void testGetHighPriorityCareGaps() {
            List<CareGap> allGaps = Arrays.asList(
                createHighSeverityGap("Gap 1"),
                createLowSeverityGap("Gap 2"),
                createCriticalSeverityGap("Gap 3"),
                createModerateSeverityGap("Gap 4")
            );

            List<CareGap> highPriorityGaps = service.getHighPriorityCareGaps(allGaps);

            // Method returns gaps with HIGH/CRITICAL severity or isUrgent
            assertEquals(2, highPriorityGaps.size());
            assertTrue(highPriorityGaps.stream().allMatch(g ->
                g.getSeverity() == CareGap.GapSeverity.HIGH ||
                g.getSeverity() == CareGap.GapSeverity.CRITICAL ||
                g.isUrgent()));
        }

        @Test
        @DisplayName("Should get overdue care gaps")
        void testGetOverdueCareGaps() {
            List<CareGap> allGaps = Arrays.asList(
                createOverdueGap("Gap 1", 30),
                createOverdueGap("Gap 2", 0),
                createOverdueGap("Gap 3", 15)
            );

            List<CareGap> overdueGaps = service.getOverdueCareGaps(allGaps);

            assertEquals(2, overdueGaps.size());
            assertTrue(overdueGaps.stream().allMatch(CareGap::isOverdue));
        }

        @Test
        @DisplayName("Should group care gaps by type")
        void testGroupCareGapsByType() {
            List<CareGap> allGaps = Arrays.asList(
                new CareGap("P1", CareGap.GapType.PREVENTIVE_SCREENING, "Colonoscopy"),
                new CareGap("P2", CareGap.GapType.PREVENTIVE_SCREENING, "Mammography"),
                new CareGap("P3", CareGap.GapType.CHRONIC_DISEASE_MONITORING, "HbA1c"),
                new CareGap("P4", CareGap.GapType.MEDICATION_ADHERENCE, "Statins")
            );

            Map<CareGap.GapType, List<CareGap>> grouped = service.groupCareGapsByType(allGaps);

            assertEquals(3, grouped.size());
            assertEquals(2, grouped.get(CareGap.GapType.PREVENTIVE_SCREENING).size());
            assertEquals(1, grouped.get(CareGap.GapType.CHRONIC_DISEASE_MONITORING).size());
            assertEquals(1, grouped.get(CareGap.GapType.MEDICATION_ADHERENCE).size());
        }

        private CareGap createHighSeverityGap(String name) {
            CareGap gap = new CareGap("PATIENT-001", CareGap.GapType.PREVENTIVE_SCREENING, name);
            gap.setSeverity(CareGap.GapSeverity.HIGH);
            gap.setPriority(8);
            return gap;
        }

        private CareGap createCriticalSeverityGap(String name) {
            CareGap gap = new CareGap("PATIENT-001", CareGap.GapType.PREVENTIVE_SCREENING, name);
            gap.setSeverity(CareGap.GapSeverity.CRITICAL);
            gap.setPriority(10);
            return gap;
        }

        private CareGap createLowSeverityGap(String name) {
            CareGap gap = new CareGap("PATIENT-001", CareGap.GapType.PREVENTIVE_SCREENING, name);
            gap.setSeverity(CareGap.GapSeverity.LOW);
            gap.setPriority(3);
            return gap;
        }

        private CareGap createModerateSeverityGap(String name) {
            CareGap gap = new CareGap("PATIENT-001", CareGap.GapType.PREVENTIVE_SCREENING, name);
            gap.setSeverity(CareGap.GapSeverity.MODERATE);
            gap.setPriority(5);
            return gap;
        }

        private CareGap createOverdueGap(String name, int daysOverdue) {
            CareGap gap = new CareGap("PATIENT-001", CareGap.GapType.PREVENTIVE_SCREENING, name);
            if (daysOverdue > 0) {
                gap.setDueDate(LocalDate.now().minusDays(daysOverdue));
                gap.calculateDaysOverdue();
            } else {
                gap.setDueDate(LocalDate.now().plusDays(10)); // Future date
            }
            return gap;
        }
    }

    @Nested
    @DisplayName("Population Health Summary Tests")
    class PopulationHealthSummaryTests {

        @Test
        @DisplayName("Should generate comprehensive population health summary")
        void testGeneratePopulationHealthSummary() {
            // Setup cohort
            PatientCohort cohort = new PatientCohort("Diabetes Cohort", PatientCohort.CohortType.DISEASE_BASED);
            for (int i = 1; i <= 100; i++) {
                cohort.addPatient("PATIENT-" + String.format("%03d", i));
            }
            cohort.setAverageRiskScore(0.55);
            cohort.updateRiskDistribution(PatientCohort.RiskLevel.HIGH, 20);
            cohort.updateRiskDistribution(PatientCohort.RiskLevel.VERY_HIGH, 10);

            // Setup care gaps
            List<CareGap> careGaps = new ArrayList<>();
            for (int i = 1; i <= 45; i++) {
                CareGap gap = new CareGap("PATIENT-" + String.format("%03d", i),
                                         CareGap.GapType.CHRONIC_DISEASE_MONITORING,
                                         "Gap " + i);
                if (i <= 30) {
                    gap.setDueDate(LocalDate.now().minusDays(10)); // Overdue
                    gap.calculateDaysOverdue();
                }
                if (i > 35) {
                    gap.closeGap(CareGap.GapStatus.CLOSED_COMPLETED, "Completed");
                }
                careGaps.add(gap);
            }

            // Setup quality measures
            List<QualityMeasure> qualityMeasures = new ArrayList<>();
            QualityMeasure measure1 = new QualityMeasure("HbA1c Testing",
                                                         QualityMeasure.MeasureType.PROCESS,
                                                         QualityMeasure.MeasureSource.HEDIS);
            measure1.setDenominatorCount(100);
            measure1.setNumeratorCount(80);
            measure1.calculateComplianceRate();
            qualityMeasures.add(measure1);

            QualityMeasure measure2 = new QualityMeasure("BP Control",
                                                         QualityMeasure.MeasureType.OUTCOME,
                                                         QualityMeasure.MeasureSource.CMS);
            measure2.setDenominatorCount(100);
            measure2.setNumeratorCount(70);
            measure2.calculateComplianceRate();
            qualityMeasures.add(measure2);

            // Generate summary
            PopulationHealthService.PopulationHealthSummary summary =
                service.generatePopulationHealthSummary(cohort, careGaps, qualityMeasures);

            // Verify summary
            assertEquals("Diabetes Cohort", summary.getCohortName());
            assertEquals(100, summary.getTotalPatients());
            assertEquals(0.55, summary.getAverageRiskScore(), 0.001);
            assertEquals(30, summary.getHighRiskPatients()); // HIGH(20) + VERY_HIGH(10)
            assertEquals(45, summary.getTotalCareGaps());
            assertEquals(35, summary.getOpenCareGaps()); // 45 total - 10 closed
            assertEquals(30, summary.getOverdueCareGaps());
            assertEquals(75.0, summary.getAverageQualityCompliance(), 0.001); // (80 + 70) / 2
        }

        @Test
        @DisplayName("Should handle empty care gaps and quality measures")
        void testSummaryWithEmptyData() {
            PatientCohort cohort = new PatientCohort("Test Cohort", PatientCohort.CohortType.CUSTOM);
            cohort.addPatient("PATIENT-001");

            PopulationHealthService.PopulationHealthSummary summary =
                service.generatePopulationHealthSummary(cohort, new ArrayList<>(), new ArrayList<>());

            assertEquals(1, summary.getTotalPatients());
            assertEquals(0, summary.getTotalCareGaps());
            assertEquals(0, summary.getOpenCareGaps());
            assertEquals(0, summary.getOverdueCareGaps());
            assertEquals(0.0, summary.getAverageQualityCompliance(), 0.001);
        }

        @Test
        @DisplayName("Should calculate correct statistics for mixed data")
        void testSummaryStatistics() {
            PatientCohort cohort = new PatientCohort("Mixed Cohort", PatientCohort.CohortType.QUALITY_MEASURE);
            cohort.addPatient("P1");
            cohort.addPatient("P2");
            cohort.addPatient("P3");
            cohort.setAverageRiskScore(0.72);
            cohort.updateRiskDistribution(PatientCohort.RiskLevel.HIGH, 1);
            cohort.updateRiskDistribution(PatientCohort.RiskLevel.VERY_HIGH, 1);

            List<CareGap> gaps = Arrays.asList(
                createOpenGap(),
                createOpenGap(),
                createClosedGap()
            );

            List<QualityMeasure> measures = Arrays.asList(
                createMeasureWithCompliance(85.0),
                createMeasureWithCompliance(75.0),
                createMeasureWithCompliance(90.0)
            );

            PopulationHealthService.PopulationHealthSummary summary =
                service.generatePopulationHealthSummary(cohort, gaps, measures);

            assertEquals(3, summary.getTotalPatients());
            assertEquals(0.72, summary.getAverageRiskScore(), 0.001);
            assertEquals(2, summary.getHighRiskPatients());
            assertEquals(3, summary.getTotalCareGaps());
            assertEquals(2, summary.getOpenCareGaps());
            assertEquals(83.33, summary.getAverageQualityCompliance(), 0.01); // (85+75+90)/3
        }

        private CareGap createOpenGap() {
            return new CareGap("PATIENT-001", CareGap.GapType.PREVENTIVE_SCREENING, "Test Gap");
        }

        private CareGap createClosedGap() {
            CareGap gap = new CareGap("PATIENT-001", CareGap.GapType.PREVENTIVE_SCREENING, "Closed Gap");
            gap.closeGap(CareGap.GapStatus.CLOSED_COMPLETED, "Done");
            return gap;
        }

        private QualityMeasure createMeasureWithCompliance(double compliance) {
            QualityMeasure measure = new QualityMeasure("Test",
                                                        QualityMeasure.MeasureType.PROCESS,
                                                        QualityMeasure.MeasureSource.HEDIS);
            measure.setDenominatorCount(100);
            measure.setNumeratorCount((int) compliance);
            measure.calculateComplianceRate();
            return measure;
        }
    }
}
