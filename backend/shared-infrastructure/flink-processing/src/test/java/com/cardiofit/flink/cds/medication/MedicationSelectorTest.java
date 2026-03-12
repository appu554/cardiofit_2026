package com.cardiofit.flink.cds.medication;

import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.PatientState;
import com.cardiofit.flink.models.MedicationDetails;
import com.cardiofit.flink.models.PatientContextState;
import com.cardiofit.flink.models.PatientDemographics;
import com.cardiofit.flink.models.LabResult;
import com.cardiofit.flink.cds.medication.MedicationSelector.ProtocolAction;
import com.cardiofit.flink.cds.medication.MedicationSelector.SelectionCriteria;
import com.cardiofit.flink.cds.medication.MedicationSelector.MedicationSelection;
import com.cardiofit.flink.cds.medication.MedicationSelector.ClinicalMedication;
import com.cardiofit.flink.cds.medication.MedicationSelector.DrugInteraction;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Comprehensive unit tests for MedicationSelector.
 *
 * Test Coverage:
 * - Selection tests (no allergy → primary, allergy → alternative): 5 tests
 * - Criteria evaluation tests (all standard criteria): 8 tests
 * - Allergy detection tests (direct match, cross-reactivity): 6 tests
 * - CrCl calculation tests (male, female, edge cases): 5 tests
 * - Dose adjustment tests (renal, hepatic): 6 tests
 *
 * Total: 30 unit tests
 *
 * @author Module 3 CDS Team
 * @version 1.0
 */
@DisplayName("MedicationSelector Tests")
class MedicationSelectorTest {

    private MedicationSelector selector;
    private EnrichedPatientContext context;

    @BeforeEach
    void setUp() {
        selector = new MedicationSelector();
        context = createDefaultContext();
    }

    // ============================================================
    // SELECTION TESTS (5 tests)
    // ============================================================

    @Test
    @DisplayName("Selection: No allergy → use primary medication")
    void testSelectMedication_NoAllergy_UsesPrimary() {
        // Setup: Patient with no penicillin allergy
        context.getPatientState().setAllergies(Arrays.asList("iodine"));

        ProtocolAction action = createActionWithSelection(
            "NO_PENICILLIN_ALLERGY",
            "Ceftriaxone", "2 g", "IV", "q24h",
            "Levofloxacin", "750 mg", "IV", "q24h"
        );

        // Execute
        ProtocolAction selected = selector.selectMedication(action, context);

        // Verify: Primary medication selected
        assertNotNull(selected);
        assertNotNull(selected.getMedication());
        assertEquals("Ceftriaxone", selected.getMedication().getName());
        assertEquals("2 g", selected.getMedication().getDosage());
    }

    @Test
    @DisplayName("Selection: Penicillin allergy → use alternative medication")
    void testSelectMedication_PenicillinAllergy_UsesAlternative() {
        // Setup: Patient with penicillin allergy
        context.getPatientState().setAllergies(Arrays.asList("penicillin"));

        // Use NO_CONTRAINDICATION criteria so selection proceeds, then allergy detection switches to alternative
        ProtocolAction action = createActionWithSelection(
            "NO_CONTRAINDICATION",
            "Ceftriaxone", "2 g", "IV", "q24h",
            "Levofloxacin", "750 mg", "IV", "q24h"
        );

        // Execute
        ProtocolAction selected = selector.selectMedication(action, context);

        // Verify: Alternative medication selected
        assertNotNull(selected);
        assertNotNull(selected.getMedication());
        assertEquals("Levofloxacin", selected.getMedication().getName());
        assertEquals("750 mg", selected.getMedication().getDosage());
    }

    @Test
    @DisplayName("Selection: Allergy to both primary and alternative → return null (FAIL SAFE)")
    void testSelectMedication_AllergyToBoth_ReturnsNull() {
        // Setup: Patient allergic to both options
        context.getPatientState().setAllergies(Arrays.asList("ceftriaxone", "levofloxacin"));

        ProtocolAction action = createActionWithSelection(
            "NO_CONTRAINDICATION",
            "Ceftriaxone", "2 g", "IV", "q24h",
            "Levofloxacin", "750 mg", "IV", "q24h"
        );

        // Execute
        ProtocolAction selected = selector.selectMedication(action, context);

        // Verify: FAIL SAFE - returns null
        assertNull(selected);
    }

    @Test
    @DisplayName("Selection: No alternative medication + allergy → return null (FAIL SAFE)")
    void testSelectMedication_NoAlternative_AllergyToprimary_ReturnsNull() {
        // Setup: Patient allergic to primary, no alternative defined
        context.getPatientState().setAllergies(Arrays.asList("ceftriaxone"));

        ProtocolAction action = createActionWithSelection(
            "NO_CONTRAINDICATION",
            "Ceftriaxone", "2 g", "IV", "q24h",
            null, null, null, null
        );

        // Execute
        ProtocolAction selected = selector.selectMedication(action, context);

        // Verify: FAIL SAFE - returns null
        assertNull(selected);
    }

    @Test
    @DisplayName("Selection: No medication selection algorithm → return action as-is")
    void testSelectMedication_NoSelection_ReturnsActionAsIs() {
        // Setup: Action with no medication_selection
        ProtocolAction action = new ProtocolAction();
        action.setActionId("TEST-001");
        action.setMedicationSelection(null);

        // Execute
        ProtocolAction selected = selector.selectMedication(action, context);

        // Verify: Returns original action unchanged
        assertNotNull(selected);
        assertEquals("TEST-001", selected.getActionId());
    }

    // ============================================================
    // CRITERIA EVALUATION TESTS (8 tests)
    // ============================================================

    @Test
    @DisplayName("Criteria: NO_PENICILLIN_ALLERGY - patient not allergic")
    void testEvaluateCriteria_NoPenicillinAllergy_True() {
        context.getPatientState().setAllergies(Arrays.asList("iodine"));

        boolean result = selector.evaluateCriteria("NO_PENICILLIN_ALLERGY", context);

        assertTrue(result, "Patient without penicillin allergy should satisfy criteria");
    }

    @Test
    @DisplayName("Criteria: NO_PENICILLIN_ALLERGY - patient allergic")
    void testEvaluateCriteria_HasPenicillinAllergy_False() {
        context.getPatientState().setAllergies(Arrays.asList("penicillin"));

        boolean result = selector.evaluateCriteria("NO_PENICILLIN_ALLERGY", context);

        assertFalse(result, "Patient with penicillin allergy should fail criteria");
    }

    @Test
    @DisplayName("Criteria: NO_BETA_LACTAM_ALLERGY - patient not allergic")
    void testEvaluateCriteria_NoBetaLactamAllergy_True() {
        context.getPatientState().setAllergies(Arrays.asList("iodine"));

        boolean result = selector.evaluateCriteria("NO_BETA_LACTAM_ALLERGY", context);

        assertTrue(result);
    }

    @Test
    @DisplayName("Criteria: CREATININE_CLEARANCE_GT_40 - CrCl > 40")
    void testEvaluateCriteria_CrClGT40_True() {
        // Setup: 65yo male, 70kg, Cr 1.2 → CrCl ~60.76
        setDemographics(65, 70.0, "M");
        setCreatinine(1.2);

        boolean result = selector.evaluateCriteria("CREATININE_CLEARANCE_GT_40", context);

        assertTrue(result, "CrCl ~61 should be > 40");
    }

    @Test
    @DisplayName("Criteria: CREATININE_CLEARANCE_GT_40 - CrCl < 40")
    void testEvaluateCriteria_CrClLT40_False() {
        // Setup: 72yo female, 60kg, Cr 1.5 → CrCl ~32.22
        setDemographics(72, 60.0, "F");
        setCreatinine(1.5);

        boolean result = selector.evaluateCriteria("CREATININE_CLEARANCE_GT_40", context);

        assertFalse(result, "CrCl ~32 should be < 40");
    }

    @Test
    @DisplayName("Criteria: SEVERE_SEPSIS - lactate >= 4.0")
    void testEvaluateCriteria_SevereSepsis_True() {
        // Setup: Lactate 4.5
        setVital("lactate", 4.5);

        boolean result = selector.evaluateCriteria("SEVERE_SEPSIS", context);

        assertTrue(result, "Lactate 4.5 should indicate severe sepsis");
    }

    @Test
    @DisplayName("Criteria: HIGH_BLEEDING_RISK - low platelets")
    void testEvaluateCriteria_HighBleedingRisk_LowPlatelets_True() {
        // Setup: Platelets 40,000
        setLab("platelets", 40000.0);

        boolean result = selector.evaluateCriteria("HIGH_BLEEDING_RISK", context);

        assertTrue(result, "Platelets < 50K should indicate high bleeding risk");
    }

    @Test
    @DisplayName("Criteria: Unknown criteria → return false")
    void testEvaluateCriteria_UnknownCriteria_ReturnsFalse() {
        boolean result = selector.evaluateCriteria("UNKNOWN_CRITERIA_XYZ", context);

        assertFalse(result, "Unknown criteria should return false");
    }

    // ============================================================
    // ALLERGY DETECTION TESTS (6 tests)
    // ============================================================

    @Test
    @DisplayName("Allergy: Direct match - medication name contains allergy")
    void testHasAllergy_DirectMatch_ReturnsTrue() {
        context.getPatientState().setAllergies(Arrays.asList("ceftriaxone"));

        ClinicalMedication med = createMedication("Ceftriaxone", "2 g", "IV", "q24h");
        boolean result = selector.hasAllergy(med, context);

        assertTrue(result, "Direct match should detect allergy");
    }

    @Test
    @DisplayName("Allergy: Direct match - allergy contains medication name")
    void testHasAllergy_AllergyContainsMed_ReturnsTrue() {
        context.getPatientState().setAllergies(Arrays.asList("all cephalosporins"));

        ClinicalMedication med = createMedication("Cephalexin", "500 mg", "PO", "q6h");
        boolean result = selector.hasAllergy(med, context);

        assertTrue(result, "Allergy containing medication name should be detected");
    }

    @Test
    @DisplayName("Allergy: Cross-reactivity - penicillin → ceftriaxone")
    void testHasAllergy_CrossReactivity_PenicillinToCeftriaxone_ReturnsTrue() {
        context.getPatientState().setAllergies(Arrays.asList("penicillin"));

        ClinicalMedication med = createMedication("Ceftriaxone", "2 g", "IV", "q24h");
        boolean result = selector.hasAllergy(med, context);

        assertTrue(result, "Penicillin allergy should cross-react with ceftriaxone");
    }

    @Test
    @DisplayName("Allergy: Cross-reactivity - penicillin → cefepime")
    void testHasAllergy_CrossReactivity_PenicillinToCefepime_ReturnsTrue() {
        context.getPatientState().setAllergies(Arrays.asList("penicillin"));

        ClinicalMedication med = createMedication("Cefepime", "2 g", "IV", "q8h");
        boolean result = selector.hasAllergy(med, context);

        assertTrue(result, "Penicillin allergy should cross-react with cefepime");
    }

    @Test
    @DisplayName("Allergy: Cross-reactivity - sulfa → sulfamethoxazole")
    void testHasAllergy_CrossReactivity_SulfaToSMX_ReturnsTrue() {
        context.getPatientState().setAllergies(Arrays.asList("sulfa"));

        ClinicalMedication med = createMedication("Sulfamethoxazole/Trimethoprim", "800/160 mg", "PO", "q12h");
        boolean result = selector.hasAllergy(med, context);

        assertTrue(result, "Sulfa allergy should cross-react with sulfamethoxazole");
    }

    @Test
    @DisplayName("Allergy: No allergy → return false")
    void testHasAllergy_NoAllergy_ReturnsFalse() {
        context.getPatientState().setAllergies(Arrays.asList("iodine"));

        ClinicalMedication med = createMedication("Levofloxacin", "750 mg", "IV", "q24h");
        boolean result = selector.hasAllergy(med, context);

        assertFalse(result, "No allergy should return false");
    }

    // ============================================================
    // CRCL CALCULATION TESTS (5 tests)
    // ============================================================

    @Test
    @DisplayName("CrCl: Male 65yo, 70kg, Cr 1.2 → ~60.76 mL/min")
    void testCalculateCrCl_Male65yo_ReturnsCorrectValue() {
        // Setup
        setDemographics(65, 70.0, "M");
        setCreatinine(1.2);

        // Execute
        double crCl = selector.calculateCrCl(context);

        // Verify: (140-65) * 70 / (72 * 1.2) = 60.76
        assertEquals(60.76, crCl, 1.0, "CrCl should be approximately 60.76");
    }

    @Test
    @DisplayName("CrCl: Female 72yo, 60kg, Cr 1.5 → ~32.22 mL/min")
    void testCalculateCrCl_Female72yo_ReturnsCorrectValue() {
        // Setup
        setDemographics(72, 60.0, "F");
        setCreatinine(1.5);

        // Execute
        double crCl = selector.calculateCrCl(context);

        // Verify: ((140-72) * 60 / (72 * 1.5)) * 0.85 = 32.22
        assertEquals(32.22, crCl, 1.0, "CrCl should be approximately 32.22");
    }

    @Test
    @DisplayName("CrCl: Female adjustment (0.85 multiplier)")
    void testCalculateCrCl_Female_AppliesCorrection() {
        // Setup: Same age/weight/Cr for male and female
        setDemographics(70, 70.0, "M");
        setCreatinine(1.0);
        double crClMale = selector.calculateCrCl(context);

        setDemographics(70, 70.0, "F");
        setCreatinine(1.0);
        double crClFemale = selector.calculateCrCl(context);

        // Verify: Female should be 0.85 * male
        assertEquals(crClMale * 0.85, crClFemale, 0.1,
            "Female CrCl should be 0.85 * male CrCl");
    }

    @Test
    @DisplayName("CrCl: Missing parameters → return default 60.0")
    void testCalculateCrCl_MissingParameters_ReturnsDefault() {
        // Setup: No demographics or creatinine
        context.getPatientState().setDemographics(null);

        // Execute
        double crCl = selector.calculateCrCl(context);

        // Verify: Default safe value
        assertEquals(60.0, crCl, 0.01, "Should return default CrCl 60.0");
    }

    @Test
    @DisplayName("CrCl: Edge case - very high creatinine")
    void testCalculateCrCl_HighCreatinine_ReturnsLowCrCl() {
        // Setup: 80yo, 60kg, Cr 3.0 (severely impaired)
        setDemographics(80, 60.0, "M");
        setCreatinine(3.0);

        // Execute
        double crCl = selector.calculateCrCl(context);

        // Verify: (140-80) * 60 / (72 * 3.0) = 16.67 (severe renal impairment)
        assertTrue(crCl < 20.0, "High creatinine should result in low CrCl");
        assertEquals(16.67, crCl, 1.0);
    }

    // ============================================================
    // DOSE ADJUSTMENT TESTS (6 tests)
    // ============================================================

    @Test
    @DisplayName("Renal adjustment: Ceftriaxone - CrCl < 30 → reduce to 1g")
    void testAdjustDoseForRenalFunction_Ceftriaxone_ReducesDose() {
        ClinicalMedication med = createMedication("Ceftriaxone", "2 g", "IV", "q24h");

        ClinicalMedication adjusted = selector.adjustDoseForRenalFunction(med, 25.0);

        assertEquals("1 g", adjusted.getDose(), "Should reduce dose to 1g for CrCl < 30");
        assertNotNull(adjusted.getAdministrationInstructions());
        assertTrue(adjusted.getAdministrationInstructions().contains("Renal dose adjustment"));
    }

    @Test
    @DisplayName("Renal adjustment: Vancomycin - CrCl < 60 → pharmacist consult")
    void testAdjustDoseForRenalFunction_Vancomycin_RequiresPharmacist() {
        ClinicalMedication med = createMedication("Vancomycin", "1 g", "IV", "q12h");

        ClinicalMedication adjusted = selector.adjustDoseForRenalFunction(med, 45.0);

        assertNotNull(adjusted.getAdministrationInstructions());
        assertTrue(adjusted.getAdministrationInstructions().contains("Pharmacist consult"));
    }

    @Test
    @DisplayName("Renal adjustment: Levofloxacin - CrCl < 50 → 500mg q48h")
    void testAdjustDoseForRenalFunction_Levofloxacin_AdjustsDoseAndInterval() {
        ClinicalMedication med = createMedication("Levofloxacin", "750 mg", "IV", "q24h");

        ClinicalMedication adjusted = selector.adjustDoseForRenalFunction(med, 40.0);

        assertEquals("500 mg", adjusted.getDose(), "Should reduce dose to 500mg");
        assertEquals("q48h", adjusted.getFrequency(), "Should extend interval to q48h");
    }

    @Test
    @DisplayName("Renal adjustment: Gentamicin - CrCl < 60 → extended interval")
    void testAdjustDoseForRenalFunction_Gentamicin_ExtendsInterval() {
        ClinicalMedication med = createMedication("Gentamicin", "5 mg/kg", "IV", "q8h");

        ClinicalMedication adjusted = selector.adjustDoseForRenalFunction(med, 50.0);

        assertEquals("q24h", adjusted.getFrequency(), "Should extend to q24h dosing");
        assertTrue(adjusted.getAdministrationInstructions().contains("Extended interval"));
    }

    @Test
    @DisplayName("Renal adjustment: Enoxaparin - CrCl < 30 → reduce to 30mg")
    void testAdjustDoseForRenalFunction_Enoxaparin_ReducesDose() {
        ClinicalMedication med = createMedication("Enoxaparin", "40 mg", "SC", "q12h");

        ClinicalMedication adjusted = selector.adjustDoseForRenalFunction(med, 25.0);

        assertEquals("30 mg", adjusted.getDose(), "Should reduce dose to 30mg");
    }

    @Test
    @DisplayName("Renal adjustment: Normal CrCl → no adjustment")
    void testAdjustDoseForRenalFunction_NormalCrCl_NoAdjustment() {
        ClinicalMedication med = createMedication("Ceftriaxone", "2 g", "IV", "q24h");

        ClinicalMedication adjusted = selector.adjustDoseForRenalFunction(med, 80.0);

        assertEquals("2 g", adjusted.getDose(), "Should not adjust dose for normal CrCl");
        assertNull(adjusted.getAdministrationInstructions());
    }

    // ============================================================
    // HELPER METHODS
    // ============================================================

    private EnrichedPatientContext createDefaultContext() {
        EnrichedPatientContext ctx = new EnrichedPatientContext();
        PatientContextState state = new PatientContextState();

        state.setPatientId("TEST-PATIENT-001");
        state.setAllergies(new ArrayList<>());
        state.setLatestVitals(new HashMap<>());
        state.setRecentLabs(new HashMap<>());

        // Default demographics: 65yo male, 70kg
        PatientDemographics demographics = new PatientDemographics();
        demographics.setAge(65);
        demographics.setWeight(70.0);
        demographics.setSex("M");
        state.setDemographics(demographics);

        ctx.setPatientId("TEST-PATIENT-001");
        ctx.setPatientState(state);

        return ctx;
    }

    private ProtocolAction createActionWithSelection(
        String criteriaId,
        String primaryName, String primaryDose, String primaryRoute, String primaryFreq,
        String altName, String altDose, String altRoute, String altFreq) {

        ProtocolAction action = new ProtocolAction();
        action.setActionId("TEST-ACTION-001");
        action.setType("MEDICATION");

        MedicationSelection selection = new MedicationSelection();
        List<SelectionCriteria> criteriaList = new ArrayList<>();

        SelectionCriteria criteria = new SelectionCriteria();
        criteria.setCriteriaId(criteriaId);

        ClinicalMedication primary = createMedication(primaryName, primaryDose, primaryRoute, primaryFreq);
        criteria.setPrimaryMedication(primary);

        if (altName != null) {
            ClinicalMedication alternative = createMedication(altName, altDose, altRoute, altFreq);
            criteria.setAlternativeMedication(alternative);
        }

        criteriaList.add(criteria);
        selection.setSelectionCriteria(criteriaList);
        action.setMedicationSelection(selection);

        return action;
    }

    private ClinicalMedication createMedication(String name, String dose, String route, String frequency) {
        ClinicalMedication med = new ClinicalMedication();
        med.setName(name);
        med.setDose(dose);
        med.setRoute(route);
        med.setFrequency(frequency);
        return med;
    }

    private void setDemographics(int age, double weight, String sex) {
        PatientDemographics demographics = new PatientDemographics();
        demographics.setAge(age);
        demographics.setWeight(weight);
        demographics.setSex(sex);
        context.getPatientState().setDemographics(demographics);
    }

    private void setCreatinine(double value) {
        LabResult labResult = new LabResult();
        labResult.setValue(value);
        labResult.setTimestamp(System.currentTimeMillis());
        context.getPatientState().getRecentLabs().put("creatinine", labResult);
    }

    private void setVital(String vitalName, double value) {
        context.getPatientState().getLatestVitals().put(vitalName.toLowerCase(), value);
    }

    private void setLab(String labName, double value) {
        LabResult labResult = new LabResult();
        labResult.setValue(value);
        labResult.setTimestamp(System.currentTimeMillis());
        context.getPatientState().getRecentLabs().put(labName.toLowerCase(), labResult);
    }
}
