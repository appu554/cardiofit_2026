#!/bin/bash
# Script to fix FHIR Integration test API compatibility issues

TEST_DIR="src/test/java/com/cardiofit/flink/cds/fhir"

echo "Fixing FHIR Integration test API compatibility..."

# Fix Pattern 1: getCohortType() returns enum, not String
# Change assertEquals("CONDITION", cohort.getCohortType())
# To: assertEquals(PatientCohort.CohortType.DISEASE_BASED, cohort.getCohortType())
find "$TEST_DIR" -name "*Test.java" -exec sed -i '' \
  -e 's/assertEquals("CONDITION", cohort\.getCohortType())/assertEquals(PatientCohort.CohortType.DISEASE_BASED, cohort.getCohortType())/g' \
  -e 's/assertEquals("AGE", cohort\.getCohortType())/assertEquals(PatientCohort.CohortType.DEMOGRAPHIC, cohort.getCohortType())/g' \
  -e 's/assertEquals("MEDICATION", cohort\.getCohortType())/assertEquals(PatientCohort.CohortType.CUSTOM, cohort.getCohortType())/g' \
  -e 's/assertEquals("GEOGRAPHIC", cohort\.getCohortType())/assertEquals(PatientCohort.CohortType.GEOGRAPHIC, cohort.getCohortType())/g' \
  -e 's/assertEquals("COMPOSITE", cohort\.getCohortType())/assertEquals(PatientCohort.CohortType.CUSTOM, cohort.getCohortType())/g' \
  -e 's/assertEquals("RISK_STRATIFIED", cohort\.getCohortType())/assertEquals(PatientCohort.CohortType.RISK_BASED, cohort.getCohortType())/g' \
  -e 's/assertEquals("CUSTOM", cohort\.getCohortType())/assertEquals(PatientCohort.CohortType.CUSTOM, cohort.getCohortType())/g' \
  -e 's/assertEquals("HEDIS", cohort\.getCohortType())/assertEquals(PatientCohort.CohortType.QUALITY_MEASURE, cohort.getCohortType())/g' \
  {} +

# Fix Pattern 2: getInclusionCriteria() returns List<CriteriaRule>, not Map
# Remove all assertions like: cohort.getInclusionCriteria().get("condition_code")
# These will need manual review to verify the CriteriaRule objects instead

echo "Pattern 1 (CohortType enum) fixed"
echo "Pattern 2 (InclusionCriteria) requires manual review"
echo ""
echo "Remaining fixes needed:"
echo "1. Update all getInclusionCriteria().get() calls to iterate over CriteriaRule list"
echo "2. Fix QualityMeasure getMeasureType() enum assertions"
echo "3. Fix totalPatients type (int not String)"
echo ""
echo "Total test files to review: 4"
echo "- FHIRCohortBuilderTest.java"
echo "- FHIRPopulationHealthMapperTest.java"
echo "- FHIRQualityMeasureEvaluatorTest.java"
echo "- FHIRObservationMapperTest.java"
