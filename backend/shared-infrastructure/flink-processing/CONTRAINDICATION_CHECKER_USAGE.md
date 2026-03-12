# Contraindication Checker - Usage Guide

## Quick Start

### Basic Usage

```java
import com.cardiofit.flink.safety.ContraindicationChecker;
import com.cardiofit.flink.models.*;

// Initialize checker (once per operator/task)
ContraindicationChecker checker = new ContraindicationChecker();

// Check for contraindications
List<Contraindication> contraindications = checker.checkContraindications(
    clinicalActions,
    enrichedPatientContext
);

// Apply dosing adjustments
checker.adjustDosing(clinicalActions, enrichedPatientContext);

// Or do both in one call
List<Contraindication> contraindications = checker.performSafetyCheck(
    clinicalActions,
    enrichedPatientContext
);
```

### Check for Critical Contraindications

```java
// Check if any absolute contraindications exist
if (checker.hasAbsoluteContraindications(contraindications)) {
    logger.error("CRITICAL: Absolute contraindications found - do not proceed");
    // Handle absolute contraindication (e.g., block recommendation)
}

// Get count of absolute contraindications
int absoluteCount = checker.getAbsoluteContraindicationCount(contraindications);

// Get highest risk score
double maxRisk = checker.getMaxRiskScore(contraindications);
if (maxRisk > 0.7) {
    logger.warn("High risk contraindications detected - clinical review required");
}
```

---

## Individual Checker Usage

### Allergy Checker

```java
import com.cardiofit.flink.safety.AllergyChecker;

AllergyChecker allergyChecker = new AllergyChecker();

// Check for allergies
List<Contraindication> allergyChecks = allergyChecker.checkAllergies(
    clinicalAction,
    patientAllergies
);

// Check for cross-reactivity
boolean hasCrossReactivity = allergyChecker.hasCrossReactivity(
    "ceftriaxone",
    "penicillin"
);

// Suggest alternative medication
String alternative = allergyChecker.suggestAlternative(
    "penicillin",
    "community-acquired pneumonia"
);
// Returns: "azithromycin or doxycycline"
```

### Drug Interaction Checker

```java
import com.cardiofit.flink.safety.DrugInteractionChecker;

DrugInteractionChecker interactionChecker = new DrugInteractionChecker();

// Check for interactions
List<Contraindication> interactions = interactionChecker.checkInteractions(
    newMedicationAction,
    activeMedications
);

// Find specific interaction
DrugInteractionChecker.DrugInteraction interaction =
    interactionChecker.findInteraction("warfarin", "ciprofloxacin");

if (interaction != null) {
    String significance = interactionChecker.assessClinicalSignificance(interaction);
    logger.warn("Interaction found: {}", significance);
}
```

### Renal Dosing Adjuster

```java
import com.cardiofit.flink.safety.RenalDosingAdjuster;

RenalDosingAdjuster renalAdjuster = new RenalDosingAdjuster();

// Calculate creatinine clearance
Double crCl = renalAdjuster.calculateCrCl(patientState);
logger.info("Patient CrCl: {:.1f} mL/min", crCl);

// Check if medication is contraindicated
boolean contraindicated = renalAdjuster.isContraindicatedInRenalImpairment(
    "metformin",
    crCl
);

// Check for renal contraindications
List<Contraindication> renalChecks = renalAdjuster.checkRenalContraindications(
    action,
    patientState
);

// Adjust dose (modifies MedicationDetails in-place)
boolean adjusted = renalAdjuster.adjustDose(
    medicationDetails,
    patientState
);

if (adjusted) {
    logger.info("Renal dose adjustment applied: {} {}",
        medicationDetails.getCalculatedDose(),
        medicationDetails.getDoseUnit());
}
```

### Hepatic Dosing Adjuster

```java
import com.cardiofit.flink.safety.HepaticDosingAdjuster;

HepaticDosingAdjuster hepaticAdjuster = new HepaticDosingAdjuster();

// Calculate Child-Pugh score
Integer childPughScore = hepaticAdjuster.calculateChildPughScore(patientState);
if (childPughScore != null) {
    String childPughClass = childPughScore <= 6 ? "A" :
                           (childPughScore <= 9 ? "B" : "C");
    logger.info("Child-Pugh Class {}: score {}", childPughClass, childPughScore);
}

// Check if medication is hepatotoxic
boolean hepatotoxic = hepaticAdjuster.isHepatotoxic("acetaminophen");

// Check for hepatic contraindications
List<Contraindication> hepaticChecks = hepaticAdjuster.checkHepaticContraindications(
    action,
    patientState
);

// Adjust dose (modifies MedicationDetails in-place)
boolean adjusted = hepaticAdjuster.adjustDose(
    medicationDetails,
    patientState
);
```

---

## Integration Examples

### Flink Operator Integration

```java
public class ClinicalRecommendationOperator
    extends ProcessFunction<EnrichedPatientContext, ClinicalRecommendation> {

    private transient ContraindicationChecker contraindicationChecker;

    @Override
    public void open(Configuration parameters) throws Exception {
        super.open(parameters);
        this.contraindicationChecker = new ContraindicationChecker();
    }

    @Override
    public void processElement(
            EnrichedPatientContext context,
            Context ctx,
            Collector<ClinicalRecommendation> out) throws Exception {

        // Generate clinical actions
        List<ClinicalAction> actions = generateActions(context);

        // Perform safety check
        List<Contraindication> contraindications =
            contraindicationChecker.performSafetyCheck(actions, context);

        // Check for absolute contraindications
        if (contraindicationChecker.hasAbsoluteContraindications(contraindications)) {
            logger.error("Absolute contraindications found - filtering actions");
            // Filter out contraindicated actions
            actions = filterContraindicatedActions(actions, contraindications);
        }

        // Create recommendation with safety warnings
        ClinicalRecommendation recommendation = new ClinicalRecommendation();
        recommendation.setActions(actions);
        recommendation.setContraindications(contraindications);

        out.collect(recommendation);
    }
}
```

### Action Builder Integration

```java
public class ActionBuilder {

    private final ContraindicationChecker contraindicationChecker;

    public ActionBuilder() {
        this.contraindicationChecker = new ContraindicationChecker();
    }

    public ClinicalAction buildMedicationAction(
            String medicationName,
            double dose,
            String doseUnit,
            EnrichedPatientContext context) {

        // Create action
        ClinicalAction action = new ClinicalAction();
        action.setActionType(ClinicalAction.ActionType.THERAPEUTIC);

        MedicationDetails medication = new MedicationDetails();
        medication.setName(medicationName);
        medication.setCalculatedDose(dose);
        medication.setDoseUnit(doseUnit);
        medication.setRoute("PO");
        medication.setFrequency("q12h");

        action.setMedicationDetails(medication);

        // Apply dosing adjustments
        List<ClinicalAction> actions = Arrays.asList(action);
        contraindicationChecker.adjustDosing(actions, context);

        // Check for contraindications
        List<Contraindication> contraindications =
            contraindicationChecker.checkContraindications(actions, context);

        // Log warnings if contraindications found
        for (Contraindication c : contraindications) {
            if (c.isAbsolute()) {
                logger.error("ABSOLUTE contraindication: {}", c.getContraindicationDescription());
            } else if (c.requiresCaution()) {
                logger.warn("CAUTION: {}", c.getContraindicationDescription());
            }
        }

        return action;
    }
}
```

---

## Data Requirements

### Required Patient Data

The checkers require specific data from `EnrichedPatientContext` and `PatientContextState`:

**For Allergy Checking**:
```java
List<String> allergies = context.getPatientState().getAllergies();
// Example: ["penicillin", "sulfa antibiotics"]
```

**For Drug Interaction Checking**:
```java
Map<String, Medication> activeMedications = context.getPatientState().getActiveMedications();
// Key: RxNorm code, Value: Medication object
```

**For Renal Dosing**:
```java
// Required lab values
Double creatinine = getLabValue(state.getRecentLabs(), "creatinine");
// Required demographics
Integer age = state.getDemographics().getAge();
String sex = state.getDemographics().getSex();
// Required vitals
Double weight = state.getLatestVitals().get("weight");
```

**For Hepatic Dosing**:
```java
// Required lab values
Double bilirubin = getLabValue(state.getRecentLabs(), "bilirubin");
Double albumin = getLabValue(state.getRecentLabs(), "albumin");
Double inr = getLabValue(state.getRecentLabs(), "inr");
```

---

## Contraindication Model

### Contraindication Fields

```java
public class Contraindication {
    private ContraindicationType contraindicationType; // ALLERGY, DRUG_INTERACTION, ORGAN_DYSFUNCTION
    private String contraindicationDescription;        // Human-readable description
    private Severity severity;                         // ABSOLUTE, RELATIVE, CAUTION
    private boolean found;                             // Always true if in list
    private String evidence;                           // Supporting evidence
    private double riskScore;                          // 0.0-1.0
    private boolean alternativeAvailable;              // Alternative exists?
    private String alternativeMedication;              // Alternative name
    private String alternativeRationale;               // Why alternative is better
    private String clinicalGuidance;                   // What to do
    private String overrideJustification;              // For clinician override
}
```

### Severity Levels

- **ABSOLUTE**: Absolutely contraindicated, do not use
- **RELATIVE**: Relative contraindication, use with caution
- **CAUTION**: Use with monitoring and caution

### Example Contraindication

```java
Contraindication contraindication = new Contraindication(
    ContraindicationType.ALLERGY,
    "Cross-reactivity: Penicillin allergy with cephalosporin use (1-3% risk)"
);
contraindication.setSeverity(Severity.RELATIVE);
contraindication.setFound(true);
contraindication.setEvidence("Patient has penicillin allergy with 3% cross-reactivity risk");
contraindication.setRiskScore(0.3);
contraindication.setClinicalGuidance(
    "Consider alternative class (e.g., aztreonam, fluoroquinolone). " +
    "If no alternative, may proceed with caution and monitoring."
);
contraindication.setAlternativeAvailable(true);
contraindication.setAlternativeMedication("aztreonam or fluoroquinolone");
contraindication.setAlternativeRationale("No cross-reactivity with beta-lactams");
```

---

## Error Handling

### Missing Patient Data

The checkers handle missing data gracefully:

```java
// If patient allergies are null or empty
List<Contraindication> allergyChecks = allergyChecker.checkAllergies(action, null);
// Returns: empty list (no allergies to check)

// If creatinine clearance cannot be calculated
Double crCl = renalAdjuster.calculateCrCl(state);
// Returns: null
boolean adjusted = renalAdjuster.adjustDose(medication, state);
// Returns: false (cannot adjust without CrCl)

// If Child-Pugh score cannot be calculated
Integer childPughScore = hepaticAdjuster.calculateChildPughScore(state);
// Returns: null
// Still flags hepatotoxic medications with caution warning
```

### Null Safety

All checkers perform null checks:

```java
// Safe to pass null or empty lists
List<Contraindication> checks = checker.checkContraindications(null, context);
// Returns: empty list

// Safe to pass null context
List<Contraindication> checks = checker.checkContraindications(actions, null);
// Logs warning, returns: empty list
```

---

## Performance Considerations

### Initialization

- Create `ContraindicationChecker` once per Flink operator (in `open()` method)
- Static databases initialized once per JVM
- Minimal memory footprint (~100KB for all rule databases)

### Execution Time

- Allergy checking: O(n×m) where n=allergies, m=cross-reactivity rules (~5-20 iterations)
- Drug interaction checking: O(n×m) where n=active meds, m=interaction database (~50-100 iterations)
- Renal dosing: O(1) calculation + O(n) guideline lookup (~8 medications)
- Hepatic dosing: O(1) calculation + O(n) guideline lookup (~8 medications)

### Scalability

- Thread-safe (stateless checkers)
- Serializable for Flink distribution
- Can be parallelized across task slots
- No external dependencies (all data in-memory)

---

## Logging

### Log Levels

```java
// DEBUG: Calculation details, decision points
logger.debug("Calculated CrCl: {:.1f} mL/min", crCl);

// INFO: Successful operations, adjustments applied
logger.info("Renal dose adjustment applied: {:.1f} → {:.1f} mg", originalDose, adjustedDose);

// WARN: Contraindications found, safety warnings
logger.warn("Drug interaction detected: {} + {} (severity: MAJOR)", drug1, drug2);

// ERROR: Absolute contraindications, critical safety issues
logger.error("ABSOLUTE CONTRAINDICATION: {} in severe renal impairment", medication);
```

### Example Log Output

```
[INFO] ContraindicationChecker - Checking 3 clinical actions for contraindications (patient: P12345)
[DEBUG] AllergyChecker - Checking allergies for medication: ceftriaxone against 2 patient allergies
[WARN] AllergyChecker - Cross-reactivity contraindication: Penicillin allergy with cephalosporin use (risk: 3.0%)
[DEBUG] RenalDosingAdjuster - Calculated CrCl: 42.5 mL/min for renal dosing check of gabapentin
[INFO] RenalDosingAdjuster - Renal dose adjustment applied for gabapentin: 900.0 → 450.0 mg (CrCl 42.5)
[INFO] ContraindicationChecker - Contraindication checking complete - found 2 contraindications
```

---

## Best Practices

### 1. Always Check for Absolute Contraindications

```java
List<Contraindication> contraindications = checker.performSafetyCheck(actions, context);

if (checker.hasAbsoluteContraindications(contraindications)) {
    // CRITICAL: Do not proceed with contraindicated actions
    // Filter actions or alert clinician
}
```

### 2. Log All Contraindications

```java
for (Contraindication c : contraindications) {
    if (c.isAbsolute()) {
        logger.error("ABSOLUTE: {} - Evidence: {}",
            c.getContraindicationDescription(), c.getEvidence());
    } else if (c.requiresCaution()) {
        logger.warn("CAUTION: {} - Guidance: {}",
            c.getContraindicationDescription(), c.getClinicalGuidance());
    }
}
```

### 3. Provide Alternatives

```java
for (Contraindication c : contraindications) {
    if (c.isAlternativeAvailable()) {
        logger.info("Alternative available: {} - Rationale: {}",
            c.getAlternativeMedication(), c.getAlternativeRationale());
    }
}
```

### 4. Document Dosing Adjustments

```java
if (medication.getRenalAdjustmentApplied() != null) {
    logger.info("Renal adjustment: {}", medication.getRenalAdjustmentApplied());
}

if ("hepatic_adjusted".equals(medication.getDoseCalculationMethod())) {
    logger.info("Hepatic adjustment applied");
}
```

---

## Future Enhancements

### External Knowledge Base Integration (Phase 5)

Replace static databases with external KB services:

```java
// Future: Replace static interaction database
DrugInteraction interaction = kb5Service.getInteraction(drug1, drug2);

// Future: Replace static renal dosing guidelines
RenalDosingGuideline guideline = kb5Service.getRenalDosing(medication);
```

### Additional Contraindication Types

```java
// Pregnancy contraindications
List<Contraindication> pregnancyChecks = checker.checkPregnancyContraindications(
    action,
    context
);

// Age-based contraindications (Beers Criteria)
List<Contraindication> ageChecks = checker.checkAgeContraindications(
    action,
    context
);

// Pharmacogenomic contraindications
List<Contraindication> pgxChecks = checker.checkPharmacogenomicContraindications(
    action,
    context
);
```

---

## Support

For questions or issues:
- Review JavaDoc in source files for detailed method documentation
- Check `/Users/apoorvabk/Downloads/cardiofit/claudedocs/MODULE3_PHASE4_CONTRAINDICATION_DOSING_IMPLEMENTATION.md`
- Refer to clinical references cited in class headers
- Contact: CardioFit Platform - Module 3 development team
