# Evidence Chain Implementation Guide

## Table of Contents
1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Evidence Chain Model](#evidence-chain-model)
4. [GRADE Methodology](#grade-methodology)
5. [Integration Points](#integration-points)
6. [Code Examples](#code-examples)
7. [API Reference](#api-reference)

---

## Overview

### What is Evidence Chain Traceability?

Evidence chain traceability is a systematic approach to linking clinical decision support (CDS) recommendations back to their source evidence through a verifiable chain of citations. This ensures that every clinical action recommended by the CardioFit platform can be traced to peer-reviewed research and authoritative clinical guidelines.

**Why It Matters:**

1. **Legal Protection**: Provides defensible documentation for clinical decisions
2. **Clinical Trust**: Physicians trust recommendations backed by recognized evidence
3. **Audit Trail**: Complete traceability for regulatory compliance (Joint Commission, CMS)
4. **Quality Assurance**: Ensures recommendations stay current with evolving evidence
5. **Transparency**: Makes the reasoning behind recommendations explicit and reviewable

### Evidence Chain Flow

```
Protocol Action (What to do)
    ↓
Guideline Recommendation (Why to do it)
    ↓
Citation/Research (Evidence supporting it)
    ↓
Evidence Quality Badge (How strong the evidence is)
```

**Example:**
```
Action: "Administer aspirin 324 mg PO" (STEMI-ACT-002)
    ↓
Guideline: ACC/AHA STEMI 2023, Recommendation 3.1
    ↓
Citation: ISIS-2 Trial (PMID 3081859)
    ↓
Quality: HIGH evidence, STRONG recommendation 🟢
```

---

## Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                   Clinical Decision Support                  │
│                                                               │
│  ┌───────────────┐     ┌──────────────┐    ┌──────────────┐ │
│  │   Protocol    │────▶│  Guideline   │───▶│   Citation   │ │
│  │    Action     │     │Recommendation│    │   (PMID)     │ │
│  └───────────────┘     └──────────────┘    └──────────────┘ │
│         │                      │                    │         │
│         │                      │                    │         │
│         ▼                      ▼                    ▼         │
│  ┌───────────────────────────────────────────────────────┐  │
│  │            Evidence Chain Resolver                     │  │
│  │  - Loads guidelines from YAML                          │  │
│  │  - Resolves citations from PubMed                      │  │
│  │  - Assesses evidence quality (GRADE)                   │  │
│  │  - Generates quality badges                            │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘

Storage Layer:
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│ guidelines/     │  │  citations/     │  │  protocols/     │
│  - sepsis/      │  │  - pmid-*.yaml  │  │  - sepsis.yaml  │
│  - cardiac/     │  │  - doi-*.yaml   │  │  - stemi.yaml   │
│  - respiratory/ │  │  - indexed by   │  │  - respiratory  │
│                 │  │    PMID         │  │    .yaml        │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

### Data Flow

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Protocol Execution                                        │
│    Action triggered: "STEMI-ACT-002: Aspirin 324 mg"        │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. Guideline Lookup                                          │
│    GuidelineLinker.getEvidenceForAction("STEMI-ACT-002")    │
│    → Returns: ACC/AHA STEMI 2023, Recommendation 3.1        │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. Citation Resolution                                       │
│    CitationLoader.loadCitation("3081859")                   │
│    → Returns: ISIS-2 Trial metadata                         │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 4. Evidence Quality Assessment                               │
│    EvidenceQuality: HIGH (RCT with large sample)            │
│    Recommendation Strength: STRONG                           │
│    Quality Badge: 🟢 STRONG                                  │
└─────────────────────────────────────────────────────────────┘
```

---

## Evidence Chain Model

### Core Classes

#### 1. EvidenceChain.java

The `EvidenceChain` class represents the complete linkage from a protocol action to its supporting evidence.

```java
package com.cds.knowledgebase.evidence.linker;

import lombok.Data;
import lombok.Builder;
import java.util.List;

/**
 * Evidence Chain - Complete traceability from action to evidence
 */
@Data
@Builder
public class EvidenceChain {

    // Origin
    private String actionId;              // "STEMI-ACT-002"
    private String actionDescription;     // "Aspirin 324 mg PO"

    // Guideline Layer
    private String guidelineId;           // "GUIDE-ACCAHA-STEMI-2023"
    private String guidelineName;         // "ACC/AHA STEMI 2023"
    private String recommendationId;      // "ACC-STEMI-2023-REC-003"
    private String recommendationText;    // "Aspirin 162-325 mg should be given..."

    // Evidence Layer
    private List<Citation> keyCitations; // Primary supporting evidence
    private List<Citation> supportingCitations; // Additional evidence

    // Quality Assessment
    private String strengthOfRecommendation; // STRONG, WEAK, CONDITIONAL
    private String evidenceQuality;          // HIGH, MODERATE, LOW, VERY_LOW
    private String qualityBadge;             // 🟢 STRONG, 🟡 MODERATE, 🟠 WEAK

    // Metadata
    private String evidenceTrail;         // Formatted text representation
    private boolean isCurrent;            // Is guideline still current?
    private String supersededBy;          // If outdated, what replaces it?
}
```

#### 2. Guideline.java

Represents a clinical practice guideline with full metadata.

```java
@Data
@Builder
public class Guideline {

    // Identification
    private String guidelineId;
    private String name;
    private String shortName;
    private String organization;
    private String topic;

    // Versioning
    private String version;
    private LocalDate publicationDate;
    private LocalDate nextReviewDate;
    private GuidelineStatus status; // CURRENT, SUPERSEDED, WITHDRAWN

    // Recommendations
    private List<Recommendation> recommendations;

    // Publication
    private PublicationInfo publication;
}
```

#### 3. Recommendation.java

Individual guideline recommendations with GRADE assessment.

```java
@Data
@Builder
public class Recommendation {

    private String recommendationId;
    private String number;                // "3.1"
    private String section;               // "Antiplatelet Therapy"
    private String title;
    private String statement;             // The recommendation text

    // GRADE Assessment
    private String strength;              // STRONG, WEAK, CONDITIONAL
    private String evidenceQuality;       // HIGH, MODERATE, LOW, VERY_LOW
    private String classOfRecommendation; // Class I, IIa, IIb, III
    private String levelOfEvidence;       // A, B-R, B-NR, C-LD, C-EO

    // Evidence
    private List<String> keyEvidence;     // PMIDs
    private String rationale;

    // Linkage
    private List<String> linkedProtocolActions;
}
```

#### 4. Citation.java

Research citation with study metadata.

```java
@Data
@Builder
public class Citation {

    private String citationId;
    private String pmid;
    private String doi;

    // Publication
    private String title;
    private List<String> authors;
    private String journal;
    private Integer publicationYear;

    // Study Characteristics
    private StudyType studyType; // RCT, META_ANALYSIS, COHORT, etc.
    private Integer sampleSize;

    // Evidence Quality
    private EvidenceQuality evidenceQuality;
}
```

---

## GRADE Methodology

### What is GRADE?

GRADE (Grading of Recommendations Assessment, Development and Evaluation) is a systematic approach to rating the quality of evidence and strength of recommendations in clinical practice guidelines.

### Evidence Quality Levels

| Level | Description | Study Types | Certainty |
|-------|-------------|-------------|-----------|
| **HIGH** | Very confident that true effect is close to estimated effect | Multiple RCTs with consistent results, well-designed meta-analyses | High certainty |
| **MODERATE** | Moderately confident; true effect likely close to estimate but could be different | RCTs with limitations, very strong observational evidence | Moderate certainty |
| **LOW** | Limited confidence; true effect may be substantially different | Observational studies, RCTs with serious limitations | Low certainty |
| **VERY_LOW** | Very little confidence; true effect likely substantially different | Case series, expert opinion, poor quality studies | Very low certainty |

### Recommendation Strength

| Strength | Description | Implications |
|----------|-------------|--------------|
| **STRONG** | Benefits clearly outweigh risks/burdens | "We recommend..." - Most patients should receive intervention |
| **WEAK** | Benefits and risks closely balanced | "We suggest..." - Different choices appropriate for different patients |
| **CONDITIONAL** | Depends on patient values/preferences | Shared decision-making emphasized |

### Evidence Quality Mapping

```java
/**
 * Map study type to evidence quality
 */
public class EvidenceQualityMapper {

    public static String mapStudyTypeToQuality(StudyType studyType,
                                               int sampleSize,
                                               boolean hasLimitations) {

        // High Quality
        if (studyType == StudyType.META_ANALYSIS && !hasLimitations) {
            return "HIGH";
        }
        if (studyType == StudyType.RCT && sampleSize > 1000 && !hasLimitations) {
            return "HIGH";
        }

        // Moderate Quality
        if (studyType == StudyType.RCT && (sampleSize < 1000 || hasLimitations)) {
            return "MODERATE";
        }
        if (studyType == StudyType.SYSTEMATIC_REVIEW) {
            return "MODERATE";
        }

        // Low Quality
        if (studyType == StudyType.COHORT || studyType == StudyType.CASE_CONTROL) {
            return "LOW";
        }

        // Very Low Quality
        if (studyType == StudyType.CASE_SERIES ||
            studyType == StudyType.EXPERT_OPINION) {
            return "VERY_LOW";
        }

        return "LOW"; // Default
    }
}
```

### Quality Badges

Visual indicators of evidence quality:

```
🟢 STRONG       HIGH evidence + STRONG recommendation
🟢 HIGH         HIGH evidence + WEAK recommendation
🟡 MODERATE     MODERATE evidence + STRONG recommendation
🟡 CONDITIONAL  MODERATE evidence + WEAK recommendation
🟠 WEAK         LOW evidence + any recommendation
🔴 VERY_WEAK    VERY_LOW evidence + any recommendation
```

---

## Integration Points

### How Guidelines Link to Protocol Actions

Protocols define clinical workflows, and each action within a protocol can be linked to guideline recommendations.

#### Protocol Action Example (STEMI Protocol)

```yaml
# stemi-protocol.yaml
actions:
  - actionId: "STEMI-ACT-002"
    description: "Aspirin 324 mg PO chewable"
    medication: "Aspirin"
    dose: "324 mg"
    route: "PO"
    timing: "Immediately"
    guidelineReferences:
      - "ACC-STEMI-2023-REC-003"  # Links to guideline recommendation
```

#### Guideline Recommendation Example

```yaml
# accaha-stemi-2023.yaml
recommendations:
  - recommendationId: "ACC-STEMI-2023-REC-003"
    statement: "Aspirin 162 to 325 mg should be given as soon as possible..."
    strength: "STRONG"
    evidenceQuality: "HIGH"
    keyEvidence:
      - "3081859"  # ISIS-2 trial PMID
      - "18160631" # De Luca G study
    linkedProtocolActions:
      - "STEMI-ACT-002"  # Bidirectional link
```

#### Citation Example

```yaml
# pmid-3081859.yaml
pmid: "3081859"
title: "Randomised trial of intravenous streptokinase, oral aspirin, both, or neither among 17,187 cases of suspected acute myocardial infarction: ISIS-2"
studyType: "RCT"
sampleSize: 17187
evidenceQuality: "HIGH"
keyFindings:
  - "Aspirin reduced 5-week mortality by 23%"
  - "Number needed to treat: 40 patients"
```

### Resolution Flow

```java
// Start with action ID
String actionId = "STEMI-ACT-002";

// 1. Load protocol to find guideline references
Protocol protocol = protocolLoader.loadProtocol("STEMI");
Action action = protocol.getAction(actionId);
List<String> guidelineRefs = action.getGuidelineReferences();

// 2. Load guidelines
for (String recId : guidelineRefs) {
    Recommendation rec = guidelineLoader.loadRecommendation(recId);

    // 3. Load citations for this recommendation
    for (String pmid : rec.getKeyEvidence()) {
        Citation citation = citationLoader.loadCitation(pmid);

        // 4. Assess evidence quality
        String quality = assessEvidenceQuality(rec, citation);

        // 5. Build evidence chain
        EvidenceChain chain = buildChain(action, rec, citation, quality);
    }
}
```

---

## Code Examples

### Example 1: Resolve Complete Evidence Chain

```java
package com.cds.example;

import com.cds.knowledgebase.evidence.linker.GuidelineLinker;
import com.cds.knowledgebase.evidence.linker.EvidenceChain;

public class EvidenceChainExample {

    public static void main(String[] args) {

        // Initialize linker
        GuidelineLinker linker = new GuidelineLinker();

        // Get evidence chain for STEMI aspirin action
        EvidenceChain chain = linker.getEvidenceChain("STEMI-ACT-002");

        // Print evidence trail
        System.out.println(chain.getEvidenceTrail());

        /* Output:
         * Evidence Chain for Action: STEMI-ACT-002
         * ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
         *
         * Action: Aspirin 324 mg PO chewable
         *   ↓
         * Guideline: ACC/AHA STEMI 2023
         *   Organization: American College of Cardiology / AHA
         *   Status: CURRENT (published 2023-04-20)
         *   ↓
         * Recommendation: REC-003 - Aspirin 162-325 mg Loading Dose
         *   Statement: "Aspirin 162 to 325 mg should be given as soon
         *              as possible to all patients with STEMI who do
         *              not have a true aspirin allergy"
         *   Strength: STRONG
         *   Evidence: HIGH
         *   ↓
         * Key Citations:
         *   [1] ISIS-2 Trial (PMID 3081859)
         *       "Randomised trial of intravenous streptokinase,
         *        oral aspirin, both, or neither..."
         *       Type: RCT | Sample: 17,187 patients
         *       Finding: Aspirin reduced 5-week mortality by 23%
         *       Quality: HIGH
         *
         *   [2] De Luca G et al (PMID 18160631)
         *       "Aspirin in primary PCI"
         *       Type: Meta-Analysis | Sample: 3,119 patients
         *       Quality: HIGH
         *   ↓
         * Quality Assessment: 🟢 STRONG
         *   HIGH evidence + STRONG recommendation
         *   Benefits clearly outweigh risks
         */
    }
}
```

### Example 2: Check Evidence Currency

```java
public class EvidenceCurrencyCheck {

    public static void checkCurrency(String actionId) {

        GuidelineLinker linker = new GuidelineLinker();
        EvidenceChain chain = linker.getEvidenceChain(actionId);

        // Check if guideline is current
        if (!chain.isCurrent()) {
            System.out.println("⚠️  WARNING: Guideline is SUPERSEDED");
            System.out.println("    Current guideline: " + chain.getGuidelineId());
            System.out.println("    Superseded by: " + chain.getSupersededBy());

            // Load new guideline
            EvidenceChain updatedChain = linker.getEvidenceChain(
                chain.getSupersededBy()
            );

            System.out.println("\n✅ Updated recommendation:");
            System.out.println(updatedChain.getRecommendationText());
        } else {
            System.out.println("✅ Guideline is CURRENT");
        }
    }
}
```

### Example 3: Compare Historical Guidelines

```java
public class GuidelineEvolutionTracker {

    public static void compareGuidelines(String oldGuidelineId,
                                         String newGuidelineId) {

        GuidelineLoader loader = new GuidelineLoader();

        Guideline oldGuideline = loader.loadGuideline(oldGuidelineId);
        Guideline newGuideline = loader.loadGuideline(newGuidelineId);

        System.out.println("Guideline Evolution Analysis");
        System.out.println("════════════════════════════");
        System.out.println("Old: " + oldGuideline.getName());
        System.out.println("     Published: " + oldGuideline.getPublicationDate());
        System.out.println("New: " + newGuideline.getName());
        System.out.println("     Published: " + newGuideline.getPublicationDate());
        System.out.println();

        // Compare recommendations
        for (Recommendation oldRec : oldGuideline.getRecommendations()) {
            Recommendation newRec = findMatchingRecommendation(
                newGuideline, oldRec.getNumber()
            );

            if (newRec != null && hasChanged(oldRec, newRec)) {
                System.out.println("Changed: Recommendation " + oldRec.getNumber());
                System.out.println("  Old strength: " + oldRec.getStrength());
                System.out.println("  New strength: " + newRec.getStrength());
                System.out.println("  Old evidence: " + oldRec.getEvidenceQuality());
                System.out.println("  New evidence: " + newRec.getEvidenceQuality());
                System.out.println();
            }
        }
    }
}
```

### Example 4: Generate Evidence Report

```java
public class EvidenceReportGenerator {

    public static void generateReport(String protocolId) {

        ProtocolLoader protocolLoader = new ProtocolLoader();
        GuidelineLinker linker = new GuidelineLinker();

        Protocol protocol = protocolLoader.loadProtocol(protocolId);

        System.out.println("Evidence Report: " + protocol.getName());
        System.out.println("═".repeat(60));
        System.out.println();

        for (Action action : protocol.getActions()) {
            EvidenceChain chain = linker.getEvidenceChain(action.getActionId());

            System.out.println("Action: " + action.getDescription());
            System.out.println("  Guideline: " + chain.getGuidelineName());
            System.out.println("  Recommendation: " + chain.getRecommendationId());
            System.out.println("  Evidence Quality: " + chain.getEvidenceQuality());
            System.out.println("  Strength: " + chain.getStrengthOfRecommendation());
            System.out.println("  Badge: " + chain.getQualityBadge());
            System.out.println("  Citations: " + chain.getKeyCitations().size());
            System.out.println();
        }

        // Summary statistics
        long highQuality = protocol.getActions().stream()
            .map(a -> linker.getEvidenceChain(a.getActionId()))
            .filter(c -> "HIGH".equals(c.getEvidenceQuality()))
            .count();

        System.out.println("Summary:");
        System.out.println("  Total actions: " + protocol.getActions().size());
        System.out.println("  High-quality evidence: " + highQuality);
        System.out.println("  Evidence coverage: 100%");
    }
}
```

---

## API Reference

### GuidelineLoader

Loads guidelines from YAML files.

```java
public class GuidelineLoader {

    /**
     * Load a guideline by ID
     * @param guidelineId Guideline identifier (e.g., "GUIDE-ACCAHA-STEMI-2023")
     * @return Guideline object with all recommendations
     */
    public Guideline loadGuideline(String guidelineId);

    /**
     * Load all guidelines for a topic
     * @param topic Clinical topic (e.g., "STEMI", "Sepsis")
     * @return List of guidelines
     */
    public List<Guideline> loadGuidelinesByTopic(String topic);

    /**
     * Load a specific recommendation
     * @param recommendationId Recommendation identifier
     * @return Recommendation object
     */
    public Recommendation loadRecommendation(String recommendationId);

    /**
     * Get current guideline for a topic (non-superseded)
     * @param topic Clinical topic
     * @return Most recent active guideline
     */
    public Guideline getCurrentGuideline(String topic);
}
```

### CitationLoader

Loads citations from YAML files or PubMed API.

```java
public class CitationLoader {

    /**
     * Load citation by PMID
     * @param pmid PubMed ID
     * @return Citation object
     */
    public Citation loadCitation(String pmid);

    /**
     * Load citation by DOI
     * @param doi Digital Object Identifier
     * @return Citation object
     */
    public Citation loadCitationByDoi(String doi);

    /**
     * Fetch citation from PubMed API
     * @param pmid PubMed ID
     * @return Citation with metadata from PubMed
     */
    public Citation fetchFromPubMed(String pmid);

    /**
     * Load multiple citations
     * @param pmids List of PubMed IDs
     * @return List of citations
     */
    public List<Citation> loadCitations(List<String> pmids);
}
```

### GuidelineLinker

Links protocol actions to evidence chains.

```java
public class GuidelineLinker {

    /**
     * Get complete evidence chain for an action
     * @param actionId Protocol action ID
     * @return EvidenceChain with guidelines, recommendations, citations
     */
    public EvidenceChain getEvidenceChain(String actionId);

    /**
     * Get all guidelines supporting an action
     * @param actionId Protocol action ID
     * @return List of guidelines
     */
    public List<Guideline> getGuidelinesForAction(String actionId);

    /**
     * Get quality badge for an action
     * @param actionId Protocol action ID
     * @return Quality badge (🟢, 🟡, 🟠, 🔴)
     */
    public String getQualityBadge(String actionId);

    /**
     * Check if evidence is current
     * @param actionId Protocol action ID
     * @return true if all supporting guidelines are current
     */
    public boolean isEvidenceCurrent(String actionId);
}
```

### EvidenceQualityAssessor

Assesses evidence quality using GRADE methodology.

```java
public class EvidenceQualityAssessor {

    /**
     * Assess overall evidence quality
     * @param recommendation Guideline recommendation
     * @param citations Supporting citations
     * @return Evidence quality (HIGH, MODERATE, LOW, VERY_LOW)
     */
    public String assessQuality(Recommendation recommendation,
                                List<Citation> citations);

    /**
     * Generate quality badge
     * @param strength Recommendation strength
     * @param quality Evidence quality
     * @return Badge emoji and text
     */
    public String generateBadge(String strength, String quality);

    /**
     * Map study type to evidence level
     * @param studyType Type of study
     * @return Evidence quality level
     */
    public String mapStudyTypeToQuality(StudyType studyType);
}
```

---

## Testing Evidence Chains

### Unit Test Example

```java
@Test
public void testEvidenceChainResolution() {
    GuidelineLinker linker = new GuidelineLinker();

    EvidenceChain chain = linker.getEvidenceChain("STEMI-ACT-002");

    assertNotNull(chain);
    assertEquals("STEMI-ACT-002", chain.getActionId());
    assertEquals("GUIDE-ACCAHA-STEMI-2023", chain.getGuidelineId());
    assertEquals("HIGH", chain.getEvidenceQuality());
    assertEquals("STRONG", chain.getStrengthOfRecommendation());
    assertTrue(chain.getKeyCitations().size() > 0);
    assertEquals("🟢 STRONG", chain.getQualityBadge());
}
```

### Integration Test Example

```java
@Test
public void testCompleteEvidenceTrail() {
    // Load full protocol
    Protocol protocol = protocolLoader.loadProtocol("STEMI");

    // Verify all actions have evidence chains
    for (Action action : protocol.getActions()) {
        EvidenceChain chain = linker.getEvidenceChain(action.getActionId());

        assertNotNull("Action " + action.getActionId() + " missing evidence",
                     chain);
        assertTrue("Action " + action.getActionId() + " has no citations",
                  chain.getKeyCitations().size() > 0);
        assertTrue("Action " + action.getActionId() + " using superseded guideline",
                  chain.isCurrent());
    }
}
```

---

## Best Practices

1. **Always Link Actions to Guidelines**: Every protocol action should reference at least one guideline recommendation

2. **Use Current Guidelines**: Regularly check for updated guidelines and mark superseded versions

3. **Maintain Citation Quality**: Prefer RCTs and meta-analyses over observational studies

4. **Document Evidence Gaps**: If high-quality evidence is lacking, document this explicitly

5. **Update Evidence Chains**: When guidelines change, update all linked protocol actions

6. **Audit Evidence Trails**: Regularly validate that evidence chains are complete and current

7. **Display Quality Badges**: Show evidence quality to clinicians to build trust

---

## Conclusion

The evidence chain implementation provides complete traceability from clinical actions to peer-reviewed evidence. This documentation should enable developers to:

- Understand the evidence chain architecture
- Implement guideline and citation loading
- Link protocol actions to evidence
- Assess evidence quality using GRADE
- Generate evidence reports and quality badges

For additional support, see:
- [Guideline YAML Authoring Guide](./Guideline_YAML_Authoring_Guide.md)
- [Citation Management Guide](./Citation_Management_Guide.md)
- [Testing and Validation Guide](./Testing_Validation_Guide.md)
