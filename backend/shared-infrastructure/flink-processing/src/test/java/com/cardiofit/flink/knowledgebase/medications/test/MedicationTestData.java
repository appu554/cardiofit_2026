package com.cardiofit.flink.knowledgebase.medications.test;

import com.cardiofit.flink.knowledgebase.medications.model.*;
import com.cardiofit.flink.models.DrugInteraction;
import java.util.*;

/**
 * Test data factory for creating medication objects and YAML test files.
 *
 * Provides standardized medication test data for common clinical scenarios:
 * - Antibiotics (Piperacillin-Tazobactam, Ceftriaxone, Vancomycin)
 * - Cardiovascular medications (Aspirin, Heparin, Warfarin, Metoprolol)
 * - High-alert medications (Insulin, Heparin)
 * - Medications requiring special dosing (renal, hepatic adjustments)
 */
public class MedicationTestData {

    // Common medication IDs
    public static final String ASPIRIN_ID = "MED-ASA-001";
    public static final String HEPARIN_ID = "MED-HEP-001";
    public static final String WARFARIN_ID = "MED-WAR-001";
    public static final String PIPERACILLIN_TAZOBACTAM_ID = "MED-PIP-001";
    public static final String CEFTRIAXONE_ID = "MED-CEF-001";
    public static final String VANCOMYCIN_ID = "MED-VAN-001";
    public static final String METFORMIN_ID = "MED-MET-001";
    public static final String CIPROFLOXACIN_ID = "MED-CIP-001";
    public static final String LEVOFLOXACIN_ID = "MED-LEV-001";

    /**
     * Creates a basic test medication with minimal configuration.
     */
    public static Medication createBasicMedication(String name) {
        return Medication.builder()
            .medicationId("MED-TEST-" + name.hashCode())
            .genericName(name)
            .brandNames(Arrays.asList(name))
            .classification(Medication.Classification.builder()
                .therapeuticClass("Test Class")
                .category("Test Category")
                .highAlert(false)
                .blackBoxWarning(false)
                .build())
            .adultDosing(Medication.AdultDosing.builder()
                .standard(Medication.AdultDosing.StandardDose.builder()
                    .dose("100mg")
                    .route("IV")
                    .frequency("q24h")
                    .build())
                .build())
            .costFormulary(Medication.CostFormulary.builder()
                .formularyStatus("PREFERRED")
                .build())
            .build();
    }

    /**
     * Creates Piperacillin-Tazobactam with complete dosing information.
     */
    public static Medication createPiperacillinTazobactam() {
        // Create renal dosing adjustments
        Map<String, Medication.RenalDosing.DoseAdjustment> renalAdjustments = new HashMap<>();
        renalAdjustments.put("40-60", Medication.RenalDosing.DoseAdjustment.builder()
            .crClRange("40-60 mL/min")
            .adjustedDose("3.375g")
            .adjustedFrequency("q6h")
            .rationale("Moderate renal impairment")
            .contraindicated(false)
            .build());
        renalAdjustments.put("20-40", Medication.RenalDosing.DoseAdjustment.builder()
            .crClRange("20-40 mL/min")
            .adjustedDose("2.25g")
            .adjustedFrequency("q6h")
            .rationale("Severe renal impairment")
            .contraindicated(false)
            .build());
        renalAdjustments.put("10-20", Medication.RenalDosing.DoseAdjustment.builder()
            .crClRange("10-20 mL/min")
            .adjustedDose("2.25g")
            .adjustedFrequency("q8h")
            .rationale("Extended interval for severe impairment")
            .contraindicated(false)
            .build());

        return Medication.builder()
            .medicationId(PIPERACILLIN_TAZOBACTAM_ID)
            .genericName("piperacillin-tazobactam")
            .brandNames(Arrays.asList("Piperacillin-Tazobactam", "Zosyn"))
            .classification(Medication.Classification.builder()
                .therapeuticClass("Anti-infective")
                .pharmacologicClass("Beta-lactam")
                .category("Antibiotic")
                .highAlert(false)
                .blackBoxWarning(false)
                .build())
            .adultDosing(Medication.AdultDosing.builder()
                .standard(Medication.AdultDosing.StandardDose.builder()
                    .dose("4.5g")
                    .route("IV")
                    .frequency("q6h")
                    .build())
                .renalAdjustment(Medication.RenalDosing.builder()
                    .creatinineClearanceMethod("Cockcroft-Gault")
                    .adjustments(renalAdjustments)
                    .requiresDialysisAdjustment(false)
                    .build())
                .build())
            .costFormulary(Medication.CostFormulary.builder()
                .formularyStatus("PREFERRED")
                .build())
            .build();
    }

    /**
     * Creates Ceftriaxone with renal dosing information.
     */
    public static Medication createCeftriaxone() {
        Map<String, Medication.RenalDosing.DoseAdjustment> renalAdjustments = new HashMap<>();
        renalAdjustments.put("40-60", Medication.RenalDosing.DoseAdjustment.builder()
            .crClRange("40-60 mL/min")
            .adjustedDose("1g")
            .adjustedFrequency("q24h")
            .rationale("CrCl 40-60")
            .contraindicated(false)
            .build());
        renalAdjustments.put("10-40", Medication.RenalDosing.DoseAdjustment.builder()
            .crClRange("10-40 mL/min")
            .adjustedDose("500mg")
            .adjustedFrequency("q24h")
            .rationale("severe renal impairment")
            .contraindicated(false)
            .build());

        return Medication.builder()
            .medicationId(CEFTRIAXONE_ID)
            .genericName("ceftriaxone")
            .brandNames(Arrays.asList("Ceftriaxone", "Rocephin"))
            .classification(Medication.Classification.builder()
                .therapeuticClass("Anti-infective")
                .pharmacologicClass("Cephalosporin")
                .category("Antibiotic")
                .highAlert(false)
                .blackBoxWarning(false)
                .build())
            .adultDosing(Medication.AdultDosing.builder()
                .standard(Medication.AdultDosing.StandardDose.builder()
                    .dose("2g")
                    .route("IV")
                    .frequency("q24h")
                    .build())
                .renalAdjustment(Medication.RenalDosing.builder()
                    .creatinineClearanceMethod("Cockcroft-Gault")
                    .adjustments(renalAdjustments)
                    .requiresDialysisAdjustment(false)
                    .build())
                .build())
            .costFormulary(Medication.CostFormulary.builder()
                .formularyStatus("PREFERRED")
                .build())
            .build();
    }

    /**
     * Creates Vancomycin with TDM requirements and renal dosing.
     */
    public static Medication createVancomycin() {
        Map<String, Medication.RenalDosing.DoseAdjustment> renalAdjustments = new HashMap<>();
        renalAdjustments.put("40-60", Medication.RenalDosing.DoseAdjustment.builder()
            .crClRange("40-60 mL/min")
            .adjustedDose("750mg")
            .adjustedFrequency("q12h")
            .rationale("Moderate impairment")
            .contraindicated(false)
            .build());
        renalAdjustments.put("20-40", Medication.RenalDosing.DoseAdjustment.builder()
            .crClRange("20-40 mL/min")
            .adjustedDose("500mg")
            .adjustedFrequency("q24h")
            .rationale("Severe impairment")
            .contraindicated(false)
            .build());
        renalAdjustments.put("10-20", Medication.RenalDosing.DoseAdjustment.builder()
            .crClRange("10-20 mL/min")
            .adjustedDose("500mg")
            .adjustedFrequency("q48h")
            .rationale("ESRD")
            .contraindicated(false)
            .build());

        return Medication.builder()
            .medicationId(VANCOMYCIN_ID)
            .genericName("vancomycin")
            .brandNames(Arrays.asList("Vancomycin"))
            .classification(Medication.Classification.builder()
                .therapeuticClass("Anti-infective")
                .pharmacologicClass("Glycopeptide")
                .category("Antibiotic")
                .highAlert(true)
                .blackBoxWarning(false)
                .build())
            .adultDosing(Medication.AdultDosing.builder()
                .standard(Medication.AdultDosing.StandardDose.builder()
                    .dose("1g")
                    .route("IV")
                    .frequency("q12h")
                    .build())
                .renalAdjustment(Medication.RenalDosing.builder()
                    .creatinineClearanceMethod("Cockcroft-Gault")
                    .adjustments(renalAdjustments)
                    .requiresDialysisAdjustment(true)
                    .hemodialysis(Medication.RenalDosing.DoseAdjustment.builder()
                        .adjustedDose("500mg")
                        .adjustedFrequency("post-dialysis")
                        .rationale("Administer after dialysis, monitor trough levels")
                        .contraindicated(false)
                        .build())
                    .build())
                .build())
            .monitoring(Medication.Monitoring.builder()
                .therapeuticRange("10-20 mcg/mL")
                .labTests(Arrays.asList("Trough level"))
                .build())
            .costFormulary(Medication.CostFormulary.builder()
                .formularyStatus("PREFERRED")
                .build())
            .build();
    }

    /**
     * Creates Warfarin with black box warning.
     */
    public static Medication createWarfarin() {
        return Medication.builder()
            .medicationId(WARFARIN_ID)
            .genericName("warfarin")
            .brandNames(Arrays.asList("Warfarin", "Coumadin"))
            .classification(Medication.Classification.builder()
                .therapeuticClass("Anticoagulant")
                .pharmacologicClass("Vitamin K antagonist")
                .category("Cardiovascular")
                .highAlert(true)
                .blackBoxWarning(true)
                .build())
            .adultDosing(Medication.AdultDosing.builder()
                .standard(Medication.AdultDosing.StandardDose.builder()
                    .dose("5mg")
                    .route("PO")
                    .frequency("daily")
                    .build())
                .build())
            .adverseEffects(Medication.AdverseEffects.builder()
                .blackBoxWarnings(Arrays.asList("Bleeding risk - INR monitoring required"))
                .build())
            .costFormulary(Medication.CostFormulary.builder()
                .formularyStatus("PREFERRED")
                .build())
            .build();
    }

    /**
     * Creates Aspirin for cardiovascular testing.
     */
    public static Medication createAspirin() {
        return Medication.builder()
            .medicationId(ASPIRIN_ID)
            .genericName("aspirin")
            .brandNames(Arrays.asList("Aspirin", "Ecotrin"))
            .classification(Medication.Classification.builder()
                .therapeuticClass("Antiplatelet")
                .pharmacologicClass("NSAID")
                .category("Cardiovascular")
                .highAlert(false)
                .blackBoxWarning(false)
                .build())
            .adultDosing(Medication.AdultDosing.builder()
                .standard(Medication.AdultDosing.StandardDose.builder()
                    .dose("81mg")
                    .route("PO")
                    .frequency("daily")
                    .build())
                .build())
            .costFormulary(Medication.CostFormulary.builder()
                .formularyStatus("PREFERRED")
                .build())
            .build();
    }

    /**
     * Creates Heparin for anticoagulation testing.
     */
    public static Medication createHeparin() {
        return Medication.builder()
            .medicationId(HEPARIN_ID)
            .genericName("heparin")
            .brandNames(Arrays.asList("Heparin"))
            .classification(Medication.Classification.builder()
                .therapeuticClass("Anticoagulant")
                .pharmacologicClass("Unfractionated heparin")
                .category("Cardiovascular")
                .highAlert(true)
                .blackBoxWarning(false)
                .build())
            .adultDosing(Medication.AdultDosing.builder()
                .standard(Medication.AdultDosing.StandardDose.builder()
                    .dose("60 units/kg bolus")
                    .route("IV")
                    .frequency("continuous")
                    .build())
                .build())
            .costFormulary(Medication.CostFormulary.builder()
                .formularyStatus("PREFERRED")
                .build())
            .build();
    }

    /**
     * Creates Metformin with renal contraindications.
     */
    public static Medication createMetformin() {
        Map<String, Medication.RenalDosing.DoseAdjustment> renalAdjustments = new HashMap<>();
        renalAdjustments.put("<30", Medication.RenalDosing.DoseAdjustment.builder()
            .crClRange("<30 mL/min")
            .adjustedDose("0mg")
            .adjustedFrequency("N/A")
            .rationale("Lactic acidosis risk")
            .contraindicated(true)
            .build());

        return Medication.builder()
            .medicationId(METFORMIN_ID)
            .genericName("metformin")
            .brandNames(Arrays.asList("Metformin", "Glucophage"))
            .classification(Medication.Classification.builder()
                .therapeuticClass("Antidiabetic")
                .pharmacologicClass("Biguanide")
                .category("Endocrine")
                .highAlert(false)
                .blackBoxWarning(false)
                .build())
            .adultDosing(Medication.AdultDosing.builder()
                .standard(Medication.AdultDosing.StandardDose.builder()
                    .dose("500mg")
                    .route("PO")
                    .frequency("BID")
                    .build())
                .renalAdjustment(Medication.RenalDosing.builder()
                    .creatinineClearanceMethod("Cockcroft-Gault")
                    .adjustments(renalAdjustments)
                    .requiresDialysisAdjustment(false)
                    .build())
                .build())
            .contraindications(Medication.Contraindications.builder()
                .absolute(Arrays.asList("CrCl <30 mL/min"))
                .build())
            .costFormulary(Medication.CostFormulary.builder()
                .formularyStatus("PREFERRED")
                .build())
            .build();
    }

    /**
     * Creates Ciprofloxacin for drug interaction testing.
     */
    public static Medication createCiprofloxacin() {
        return Medication.builder()
            .medicationId(CIPROFLOXACIN_ID)
            .genericName("ciprofloxacin")
            .brandNames(Arrays.asList("Ciprofloxacin", "Cipro"))
            .classification(Medication.Classification.builder()
                .therapeuticClass("Anti-infective")
                .pharmacologicClass("Fluoroquinolone")
                .category("Antibiotic")
                .highAlert(false)
                .blackBoxWarning(true)
                .build())
            .adultDosing(Medication.AdultDosing.builder()
                .standard(Medication.AdultDosing.StandardDose.builder()
                    .dose("400mg")
                    .route("IV")
                    .frequency("q12h")
                    .build())
                .build())
            .adverseEffects(Medication.AdverseEffects.builder()
                .blackBoxWarnings(Arrays.asList("Tendon rupture risk, especially in elderly"))
                .build())
            .costFormulary(Medication.CostFormulary.builder()
                .formularyStatus("PREFERRED")
                .build())
            .build();
    }

    /**
     * Creates Levofloxacin with renal dosing.
     */
    public static Medication createLevofloxacin() {
        Map<String, Medication.RenalDosing.DoseAdjustment> renalAdjustments = new HashMap<>();
        renalAdjustments.put("30-50", Medication.RenalDosing.DoseAdjustment.builder()
            .crClRange("30-50 mL/min")
            .adjustedDose("500mg")
            .adjustedFrequency("q24h")
            .rationale("CrCl 30-50")
            .contraindicated(false)
            .build());
        renalAdjustments.put("20-30", Medication.RenalDosing.DoseAdjustment.builder()
            .crClRange("20-30 mL/min")
            .adjustedDose("500mg")
            .adjustedFrequency("q48h")
            .rationale("CrCl 20-30")
            .contraindicated(false)
            .build());
        renalAdjustments.put("10-20", Medication.RenalDosing.DoseAdjustment.builder()
            .crClRange("10-20 mL/min")
            .adjustedDose("500mg")
            .adjustedFrequency("q48h")
            .rationale("CrCl <20")
            .contraindicated(false)
            .build());

        return Medication.builder()
            .medicationId(LEVOFLOXACIN_ID)
            .genericName("levofloxacin")
            .brandNames(Arrays.asList("Levofloxacin", "Levaquin"))
            .classification(Medication.Classification.builder()
                .therapeuticClass("Anti-infective")
                .pharmacologicClass("Fluoroquinolone")
                .category("Antibiotic")
                .highAlert(false)
                .blackBoxWarning(false)
                .build())
            .adultDosing(Medication.AdultDosing.builder()
                .standard(Medication.AdultDosing.StandardDose.builder()
                    .dose("750mg")
                    .route("IV")
                    .frequency("q24h")
                    .build())
                .renalAdjustment(Medication.RenalDosing.builder()
                    .creatinineClearanceMethod("Cockcroft-Gault")
                    .adjustments(renalAdjustments)
                    .requiresDialysisAdjustment(false)
                    .build())
                .build())
            .costFormulary(Medication.CostFormulary.builder()
                .formularyStatus("PREFERRED")
                .build())
            .build();
    }

    /**
     * Creates Metoprolol with hepatic metabolism.
     */
    public static Medication createMetoprolol() {
        Map<String, Medication.HepaticDosing.DoseAdjustment> hepaticAdjustments = new HashMap<>();
        hepaticAdjustments.put("A", Medication.HepaticDosing.DoseAdjustment.builder()
            .severity("Mild")
            .childPughClass("A")
            .adjustedDose("75% of standard dose")
            .adjustedFrequency("BID")
            .rationale("25% dose reduction for Child-Pugh A")
            .contraindicated(false)
            .build());
        hepaticAdjustments.put("B", Medication.HepaticDosing.DoseAdjustment.builder()
            .severity("Moderate")
            .childPughClass("B")
            .adjustedDose("50% of standard dose")
            .adjustedFrequency("BID")
            .rationale("50% dose reduction for Child-Pugh B")
            .contraindicated(false)
            .build());
        hepaticAdjustments.put("C", Medication.HepaticDosing.DoseAdjustment.builder()
            .severity("Severe")
            .childPughClass("C")
            .adjustedDose("50% of standard dose")
            .adjustedFrequency("BID")
            .rationale("50% dose reduction, consider alternative")
            .contraindicated(false)
            .build());

        return Medication.builder()
            .medicationId("MED-MET-002")
            .genericName("metoprolol")
            .brandNames(Arrays.asList("Metoprolol", "Lopressor", "Toprol-XL"))
            .classification(Medication.Classification.builder()
                .therapeuticClass("Antihypertensive")
                .pharmacologicClass("Beta-blocker")
                .category("Cardiovascular")
                .highAlert(false)
                .blackBoxWarning(false)
                .build())
            .adultDosing(Medication.AdultDosing.builder()
                .standard(Medication.AdultDosing.StandardDose.builder()
                    .dose("25mg")
                    .route("PO")
                    .frequency("BID")
                    .build())
                .hepaticAdjustment(Medication.HepaticDosing.builder()
                    .assessmentMethod("Child-Pugh")
                    .adjustments(hepaticAdjustments)
                    .requiresMonitoring(true)
                    .build())
                .build())
            .costFormulary(Medication.CostFormulary.builder()
                .formularyStatus("PREFERRED")
                .build())
            .build();
    }

    /**
     * Creates medication with specific formulary status.
     */
    public static Medication createNonFormularyMedication(String name) {
        Medication baseMed = createBasicMedication(name);
        return Medication.builder()
            .medicationId(baseMed.getMedicationId())
            .genericName(baseMed.getGenericName())
            .brandNames(baseMed.getBrandNames())
            .classification(baseMed.getClassification())
            .adultDosing(baseMed.getAdultDosing())
            .costFormulary(Medication.CostFormulary.builder()
                .formularyStatus("NON_FORMULARY")
                .build())
            .build();
    }

    /**
     * Creates brand medication with cost information.
     */
    public static Medication createBrandMedication(String brandName, double cost) {
        return Medication.builder()
            .medicationId("MED-TEST-" + brandName.hashCode())
            .genericName(brandName.toLowerCase())
            .brandNames(Arrays.asList(brandName))
            .classification(Medication.Classification.builder()
                .therapeuticClass("Test Class")
                .category("Test Category")
                .highAlert(false)
                .blackBoxWarning(false)
                .build())
            .adultDosing(Medication.AdultDosing.builder()
                .standard(Medication.AdultDosing.StandardDose.builder()
                    .dose("100mg")
                    .route("IV")
                    .frequency("q24h")
                    .build())
                .build())
            .costFormulary(Medication.CostFormulary.builder()
                .formularyStatus("PREFERRED")
                .institutionalCost(cost)
                .genericAvailable(true)
                .build())
            .build();
    }

    /**
     * Creates YAML string for testing medication loader.
     */
    public static String createMedicationYAML(String id, String name, String category) {
        return String.format("""
            medication_id: %s
            name: %s
            generic_name: %s
            category: %s
            drug_class: test-class
            formulary_status: PREFERRED
            high_alert: false
            standard_dose: 100mg
            route: IV
            frequency: q24h
            """, id, name, name.toLowerCase(), category);
    }

    /**
     * Creates a list of common drug interactions for testing.
     */
    public static List<DrugInteraction> createCommonInteractions() {
        return Arrays.asList(
            DrugInteraction.builder()
                .interactionId("INT-WAR-CIP-001")
                .medicationIds(Arrays.asList("Warfarin", "Ciprofloxacin"))
                .interactionType("PHARMACODYNAMIC")
                .severity("MAJOR")
                .description("INR increase, bleeding risk. Monitor INR closely")
                .build(),
            DrugInteraction.builder()
                .interactionId("INT-VAN-PIP-001")
                .medicationIds(Arrays.asList("Vancomycin", "Piperacillin-Tazobactam"))
                .interactionType("PHARMACODYNAMIC")
                .severity("MODERATE")
                .description("Increased nephrotoxicity risk. Monitor renal function")
                .build(),
            DrugInteraction.builder()
                .interactionId("INT-ASA-HEP-001")
                .medicationIds(Arrays.asList("Aspirin", "Heparin"))
                .interactionType("PHARMACODYNAMIC")
                .severity("MODERATE")
                .description("Additive bleeding risk. Monitor for bleeding")
                .build(),
            DrugInteraction.builder()
                .interactionId("INT-DIG-FUR-001")
                .medicationIds(Arrays.asList("Digoxin", "Furosemide"))
                .interactionType("PHARMACODYNAMIC")
                .severity("MODERATE")
                .description("Hypokalemia increases digoxin toxicity. Monitor potassium and digoxin levels")
                .build()
        );
    }

    /**
     * Creates list of common contraindications.
     */
    public static List<String> createCommonContraindications() {
        return Arrays.asList(
            "Pregnancy Category X",
            "Severe renal impairment (CrCl <10)",
            "Active bleeding",
            "Known hypersensitivity"
        );
    }
}
