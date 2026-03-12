# ADR-001: Medication Database Architecture

**Date**: October 21, 2025
**Status**: ACCEPTED
**Decision Makers**: Module 3 CDS Team
**Relates to**: Drug-Drug Interaction Implementation (Gap Analysis Priority 1)

---

## Context and Problem Statement

During Module 3 implementation, we needed to decide how to store and manage medication information for clinical decision support. The original Phase 3 design specified a separate medication database (YAML files in `medications/` directory), but the implementation embedded medication details directly within protocol YAML files.

**Key Question**: Should medications be:
1. **Separate database** (as designed): Independent YAML files with centralized medication library
2. **Embedded in protocols** (as implemented): Medication details within each protocol's action definitions

---

## Decision Drivers

### Implementation Phase Constraints
- **Time to Market**: Phase 1-3 delivery timeline (12 weeks)
- **Complexity**: Minimize cross-references and data lookups during initial development
- **Testing**: Simpler data model for unit testing individual protocols

###Patient Safety Requirements
- **Drug-Drug Interactions**: Need to check medications against patient's current medication list
- **Allergy Checking**: Must validate medications against patient allergies
- **Renal/Hepatic Dosing**: Need medication-specific dose adjustment rules

### Future Scalability Needs
- **Protocol Count**: Initial 25 protocols → Future 100+ protocols
- **Medication Count**: Initial ~50 medications → Future 500+ medications
- **Maintenance**: Guideline updates, new evidence, medication recalls

---

## Decision Outcome

### Chosen Option: **Embedded Medications in Protocols (Phase 1-5)**

**Rationale**:
1. **Faster Initial Implementation**
   - No need to build separate medication loading infrastructure
   - Simpler data model for MVP delivery
   - Protocols are self-contained and easier to understand

2. **Sufficient for Initial Scope**
   - 25 protocols with 3-5 medication options each
   - ~75 unique medications total (manageable duplication)
   - Medication details rarely change during Phase 1-5

3. **Maintains Patient Safety**
   - Drug-drug interaction checking implemented separately in MedicationSelector.java (in-memory database)
   - Allergy checking works with embedded medication names
   - Renal/hepatic dosing logic in MedicationSelector, not in medication database

4. **Faster Testing and Iteration**
   - Each protocol test file has all data needed
   - No cross-file dependencies during testing
   - Protocol updates don't require medication database synchronization

---

## Implementation Details

### Current Architecture (Phase 1-5)

**Protocol YAML Structure**:
```yaml
# sepsis-management.yaml
protocol_id: "SEPSIS-BUNDLE-001"
actions:
  - action_id: "SEPSIS-001-A2"
    type: "ORDER_MEDICATION"
    medication_selection:
      selection_criteria:
        - criteria_id: "NO_PENICILLIN_ALLERGY"
          primary_medication:
            name: "Piperacillin-Tazobactam"
            dose: "4.5 g"
            route: "IV"
            frequency: "q6h"
          alternative_medication:
            name: "Cefepime"
            dose: "2 g"
            route: "IV"
            frequency: "q8h"
```

**Drug Interaction Database**:
```java
// MedicationSelector.java
private static final Map<String, List<DrugInteraction>> DRUG_INTERACTIONS = initializeInteractions();

// In-memory interaction database (top 20 critical interactions)
interactions.put("warfarin", Arrays.asList(
    new DrugInteraction("piperacillin-tazobactam", "MAJOR", "Increased INR and bleeding risk", "Monitor INR daily"),
    // ... more interactions
));
```

### Consequences

#### ✅ Positive
- **Phase 1-5 Delivery**: Met all deadlines without medication database complexity
- **Self-Contained Protocols**: Each protocol file is complete and independently testable
- **Simpler Codebase**: No medication loader, no cross-references, fewer abstraction layers
- **Patient Safety Maintained**: Drug interaction checking works effectively with in-memory database

#### ⚠️ Negative
- **Duplication**: Same medication appears in multiple protocol files
- **Maintenance Burden**: Medication updates require changes to multiple protocols
- **Scalability Limit**: Not sustainable beyond ~100 protocols and ~200 unique medications
- **Drug Interaction Database**: In-memory database in code (not externalized like protocols)
- **Limited Metadata**: Cannot easily track medication formulary, costs, availability

---

## Alternatives Considered

### Alternative 1: Separate Medication Database (Original Design)

**Structure**:
```yaml
# medications/piperacillin-tazobactam.yaml
medication_id: "MED-PIPT-001"
name: "Piperacillin-Tazobactam"
generic_name: "piperacillin-tazobactam"
drug_class: "beta-lactam antibiotic"
dosing:
  standard:
    dose: "4.5 g"
    route: "IV"
    frequency: "q6h"
  renal_adjustment:
    crcl_30_60:
      dose: "3.375 g"
      frequency: "q6h"
    crcl_10_30:
      dose: "2.25 g"
      frequency: "q8h"
interactions:
  - medication: "warfarin"
    severity: "MAJOR"
    description: "Increased INR and bleeding risk"
    recommendation: "Monitor INR daily"
contraindications:
  - "Known hypersensitivity to beta-lactams"
pregnancy_category: "B"
lactation_safety: "Compatible"
```

**Protocol Reference**:
```yaml
# sepsis-management.yaml
actions:
  - action_id: "SEPSIS-001-A2"
    type: "ORDER_MEDICATION"
    medication_selection:
      selection_criteria:
        - criteria_id: "NO_PENICILLIN_ALLERGY"
          primary_medication_id: "MED-PIPT-001"  # Reference by ID
          alternative_medication_id: "MED-CEFE-002"
```

**Why Not Chosen for Phase 1-5**:
- ❌ Additional 2-3 weeks development time (medication loader, ID resolution, cross-file validation)
- ❌ More complex testing (need to load both protocols AND medications)
- ❌ Overkill for 25 protocols with ~75 total medications
- ❌ Premature optimization (YAGNI - You Aren't Gonna Need It... yet)

**When to Revisit**: Phase 6 (Comprehensive Medication Database) when scaling to 100+ protocols

---

### Alternative 2: Hybrid Approach

**Concept**: Essential medications in separate database, protocol-specific medications embedded

**Why Not Chosen**:
- ❌ Worst of both worlds - complexity of separate database + duplication of embedded
- ❌ Inconsistent architecture (some meds referenced by ID, some embedded)
- ❌ Confusing for future developers

---

## Future Work Plan

### Phase 6: Medication Database Refactoring (Est. Q2 2025)

**Triggers** (any one of these):
1. Protocol count exceeds 50
2. Unique medication count exceeds 150
3. Medication updates require changing >10 protocol files
4. Formulary management becomes requirement
5. Drug interaction database exceeds 100 interactions

**Refactoring Steps**:
1. **Create Medication Database Structure**
   ```
   medications/
   ├── antibiotics/
   │   ├── piperacillin-tazobactam.yaml
   │   ├── cefepime.yaml
   │   └── ...
   ├── cardiovascular/
   │   ├── metoprolol.yaml
   │   └── ...
   └── interactions/
       └── drug-interactions.yaml  (externalized from code)
   ```

2. **Implement MedicationLoader**
   ```java
   public class MedicationLoader {
       public static Map<String, MedicationDefinition> loadAllMedications();
       public static MedicationDefinition getMedication(String medicationId);
       public static List<DrugInteraction> getInteractions(String medicationId);
   }
   ```

3. **Add to KnowledgeBaseManager**
   ```java
   public class KnowledgeBaseManager {
       private Map<String, Protocol> protocols;
       private Map<String, MedicationDefinition> medications;  // NEW
       private Map<String, List<DrugInteraction>> interactions;  // NEW (externalized)
   }
   ```

4. **Update Protocol YAML** (one-time migration)
   - Replace embedded medication objects with medication_id references
   - Automated migration script: `migrate-protocols-to-medication-ids.py`

5. **Enhanced Drug Interaction Database**
   ```yaml
   # interactions/drug-interactions.yaml
   interactions:
     - primary_drug: "warfarin"
       interacting_drugs:
         - drug: "piperacillin-tazobactam"
           severity: "MAJOR"
           mechanism: "Altered vitamin K metabolism"
           evidence_level: "1A"
           reference: "PMID:12345678"
           onset: "2-5 days"
           monitoring: "Monitor INR daily for first week"
           management: "Consider dose reduction"
   ```

6. **Add Medication Metadata**
   - Formulary status (preferred, alternative, restricted)
   - Cost tier (generic, brand, specialty)
   - Availability (in stock, special order, unavailable)
   - Pregnancy/lactation safety
   - Pediatric dosing

**Estimated Effort**: 3-4 weeks for complete refactoring

---

## Compliance and Standards

### Current Compliance
- ✅ **Patient Safety**: Drug-drug interaction checking implemented (Priority 1 gap addressed)
- ✅ **Allergy Checking**: Cross-reactivity detection for beta-lactams, sulfa drugs
- ✅ **Renal Dosing**: Cockcroft-Gault formula with evidence-based adjustments
- ✅ **Audit Logging**: All medication selection decisions logged with rationale

### Standards Alignment
- ✅ **FHIR Medication Resource**: Can export to FHIR Medication/MedicationRequest
- ✅ **RxNorm**: Medication names map to RxNorm concepts
- ✅ **Evidence-Based**: Dose adjustments cite clinical guidelines

### Future Enhancements (Phase 6)
- 📋 **CPOE Integration**: Computerized Physician Order Entry compliance
- 📋 **ASHP Guidelines**: American Society of Health-System Pharmacists formulary structure
- 📋 **FDA MedWatch**: Medication recall integration
- 📋 **HL7 FHIR R5**: Enhanced medication knowledge resource

---

## Metrics and Monitoring

### Current Performance
- **Protocol Loading**: ~200ms for 25 protocols (acceptable)
- **Medication Selection**: <5ms per patient
- **Drug Interaction Check**: <2ms per medication
- **Memory Usage**: ~2MB for all protocols

### Phase 6 Target Performance
- **Protocol Loading**: <500ms for 100+ protocols
- **Medication Database Loading**: <300ms for 500+ medications
- **Medication Lookup**: <1ms by ID (indexed)
- **Interaction Database**: <1000ms for 500+ interactions
- **Memory Usage**: <50MB total

---

## Decision Review

**Review Date**: Q1 2025 (after Phase 5 completion)
**Review Criteria**:
1. Protocol count approaching 50?
2. Medication update pain points?
3. Drug interaction database exceeding 50 interactions?
4. Formulary management requirements emerging?

**Escalation Path**: If review triggers refactoring, create Phase 6 implementation plan with stakeholder approval.

---

## References

- **Original Design**: Phase 3 design specification (Week 6-7)
- **Gap Analysis**: DESIGN_SPECS_VERIFICATION.md (October 21, 2025)
- **Drug Interaction Implementation**: MedicationSelector.java (Lines 59-155)
- **Performance Benchmark**: Phase1PerformanceBenchmark.java

---

## Approval

**Decision**: ACCEPTED
**Approved by**: Module 3 CDS Team
**Date**: October 21, 2025
**Status**: IMPLEMENTED (Phase 1-5)
**Review Scheduled**: Q1 2025 (before Phase 6)

---

## Summary

✅ **Current State**: Medications embedded in protocols (Phase 1-5)
✅ **Drug-Drug Interactions**: Implemented in-memory database (20+ critical interactions)
✅ **Patient Safety**: All safety features functional
✅ **Technical Debt**: Acknowledged and planned for Phase 6 refactoring
✅ **Timeline**: 3-4 weeks refactoring when protocol count exceeds 50 or medication count exceeds 150

**Bottom Line**: Pragmatic architectural decision that delivered working system on time while preserving path to scalable future architecture.
