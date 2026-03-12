# Guideline Integration Quick Start Guide

**Module**: Clinical Decision Support - Module 3 Phase 5 Day 4
**Last Updated**: 2025-10-24

---

## Overview

This guide provides quick reference for using the guideline integration system to link protocol actions to evidence-based guidelines with complete citation traceability.

---

## Quick Start

### 1. Basic Evidence Chain Retrieval

```java
import com.cardiofit.flink.knowledgebase.GuidelineIntegrationService;
import com.cardiofit.flink.models.EvidenceChain;

// Get evidence chain for an action
EvidenceChain chain = integrationService.getEvidenceChain("STEMI-ACT-002");

// Display formatted evidence trail
System.out.println(chain.getFormattedEvidenceTrail());

// Check quality
String badge = chain.getQualityBadge();  // 🟢 STRONG, 🟡 MODERATE, etc.
```

### 2. Enrich Protocol Action with Evidence

```java
import com.cardiofit.flink.models.ProtocolAction;

// Load action from protocol
ProtocolAction action = protocolLoader.loadAction("STEMI-ACT-002");

// Enrich with evidence
action = integrationService.enrichActionWithEvidence(action);

// Access evidence fields
System.out.println("Guideline: " + action.getGuidelineReference());
System.out.println("Quality: " + action.getEvidenceQuality());
System.out.println("Badge: " + action.getQualityBadge());
```

### 3. Check Guideline Currency

```java
// Check if guideline is current
boolean isCurrent = integrationService.isGuidelineCurrent("GUIDE-ACCAHA-STEMI-2023");

if (!isCurrent) {
    System.out.println("⚠️ Guideline outdated - review needed");
}
```

### 4. Identify Evidence Gaps

```java
// Get actions without complete evidence
List<String> gaps = integrationService.getActionsWithoutEvidence();

// Generate detailed report
Map<String, String> gapReport = integrationService.generateEvidenceGapReport();
```

---

## Protocol YAML Format

### Action with Guideline Integration

```yaml
- action_id: "STEMI-ACT-002"
  type: "MEDICATION"
  priority: "CRITICAL"

  medication:
    name: "Aspirin"
    dose: "324"
    dose_unit: "mg"
    route: "PO (chewable)"

  # GUIDELINE INTEGRATION
  guideline_reference: "GUIDE-ACCAHA-STEMI-2023"
  recommendation_id: "ACC-STEMI-2023-REC-003"
  evidence_quality: "HIGH"
  recommendation_strength: "STRONG"
  class_of_recommendation: "CLASS_I"
  level_of_evidence: "A"
  clinical_rationale: "Aspirin reduces mortality by 23% (ISIS-2 trial)"
  citation_pmids:
    - "37079885"
    - "3081859"
    - "18160631"
```

---

## Quality Badges

| Badge | Meaning | Criteria |
|-------|---------|----------|
| 🟢 STRONG | High-quality evidence, strong recommendation | HIGH + STRONG |
| 🟡 MODERATE | Moderate evidence quality | MODERATE quality |
| 🟠 WEAK | Low evidence quality | LOW/VERY_LOW quality |
| ⚠️ OUTDATED | Guideline past review date | Past nextReviewDate |
| ⚪ UNGRADED | No evidence assessment | Missing quality data |

---

## Evidence Quality Levels (GRADE)

- **HIGH**: ≥2 RCTs + high-quality studies
- **MODERATE**: ≥1 RCT or ≥3 citations
- **LOW**: ≥2 citations
- **VERY_LOW**: <2 citations or no RCTs

---

## Recommendation Strength

### ACC/AHA Classification

- **Class I**: Strong recommendation - benefit >>> risk
- **Class IIa**: Moderate recommendation - benefit >> risk
- **Class IIb**: Weak recommendation - benefit ≥ risk
- **Class III**: No benefit or harmful - risk ≥ benefit

### Level of Evidence

- **A**: High-quality evidence from multiple RCTs
- **B-R**: Moderate evidence from randomized trials
- **B-NR**: Moderate evidence from non-randomized studies
- **C-LD**: Limited data from observational studies
- **C-EO**: Expert opinion consensus

---

## Evidence Chain Structure

```
ProtocolAction
    ↓
EvidenceChain
    ├─ Guideline Info
    │   ├─ guidelineId: "GUIDE-ACCAHA-STEMI-2023"
    │   ├─ guidelineName: "ACC/AHA STEMI 2023"
    │   ├─ organization: "ACC/AHA/SCAI"
    │   ├─ publicationDate: "2023-04-20"
    │   ├─ nextReviewDate: "2028-04-20"
    │   └─ status: "CURRENT"
    │
    ├─ Recommendation Details
    │   ├─ recommendationId: "ACC-STEMI-2023-REC-003"
    │   ├─ statement: "Aspirin 162-325 mg immediately"
    │   ├─ strength: "STRONG"
    │   ├─ classOfRecommendation: "CLASS_I"
    │   ├─ levelOfEvidence: "A"
    │   └─ evidenceQuality: "HIGH"
    │
    └─ Citations
        ├─ Citation 1 (PMID 3081859)
        │   ├─ title: "ISIS-2 trial"
        │   ├─ authors: "ISIS-2 Collaborative Group"
        │   ├─ year: 1988
        │   └─ summary: "23% mortality reduction"
        │
        └─ Citation 2 (PMID 18160631)
            └─ title: "De Luca meta-analysis"
```

---

## Common Use Cases

### Use Case 1: Display Evidence in UI

```java
// Get evidence chain
EvidenceChain chain = integrationService.getEvidenceChain(actionId);

// Display quality badge
ui.showBadge(chain.getQualityBadge());

// Show evidence summary
ui.showSummary(resolver.getEvidenceSummary(actionId));

// Show full trail (on click)
ui.showDetail(chain.getFormattedEvidenceTrail());
```

### Use Case 2: Validate Protocol Quality

```java
// Load all protocol actions
List<ProtocolAction> actions = protocolLoader.loadAllActions("STEMI-PROTOCOL-001");

// Check evidence quality
for (ProtocolAction action : actions) {
    if (!action.hasHighQualityEvidence()) {
        System.out.println("⚠️ " + action.getActionId() + " needs evidence review");
    }
}
```

### Use Case 3: Guideline Update Workflow

```java
// Check all guidelines for currency
List<String> guidelines = Arrays.asList(
    "GUIDE-ACCAHA-STEMI-2023",
    "GUIDE-SSC-2021",
    "GUIDE-ACCAHA-STEMI-2013"
);

for (String guidelineId : guidelines) {
    if (!integrationService.isGuidelineCurrent(guidelineId)) {
        System.out.println("⚠️ " + guidelineId + " requires review");
        // Trigger update workflow
    }
}
```

### Use Case 4: Evidence Gap Report for Quality Improvement

```java
// Generate gap report
Map<String, String> gaps = integrationService.generateEvidenceGapReport();

// Prioritize by action criticality
List<String> criticalGaps = gaps.keySet().stream()
    .filter(actionId -> actionId.contains("ACT-001") || actionId.contains("ACT-002"))
    .collect(Collectors.toList());

// Send to QI team
qualityImprovementService.reportGaps(criticalGaps);
```

---

## Pre-configured Mappings

### STEMI Protocol (GUIDE-ACCAHA-STEMI-2023)

| Action ID | Recommendation | Evidence | Citations |
|-----------|---------------|----------|-----------|
| STEMI-ACT-001 | ACC-STEMI-2023-REC-001 | HIGH/STRONG | 3 |
| STEMI-ACT-002 | ACC-STEMI-2023-REC-003 | HIGH/STRONG | 3 |
| STEMI-ACT-003 | ACC-STEMI-2023-REC-004 | HIGH/STRONG | 4 |
| STEMI-ACT-004 | ACC-STEMI-2023-REC-005 | MODERATE/STRONG | 4 |
| STEMI-ACT-005 | ACC-STEMI-2023-REC-002 | HIGH/STRONG | 4 |

### Sepsis Protocol (GUIDE-SSC-2021)

| Action ID | Recommendation | Evidence | Citations |
|-----------|---------------|----------|-----------|
| SEPSIS-ACT-001 | SSC-2021-REC-001 | MODERATE/STRONG | 2 |
| SEPSIS-ACT-002 | SSC-2021-REC-002 | MODERATE/STRONG | 2 |
| SEPSIS-ACT-004 | SSC-2021-REC-004 | HIGH/STRONG | 2 |
| SEPSIS-ACT-005 | SSC-2021-REC-005 | MODERATE/STRONG | 1 |

---

## API Reference

### GuidelineIntegrationService

```java
// Core methods
EvidenceChain getEvidenceChain(String actionId)
List<Guideline> getGuidelinesForAction(String actionId)
boolean isGuidelineCurrent(String guidelineId)
List<String> getActionsWithoutEvidence()
String getQualityBadge(String actionId)
ProtocolAction enrichActionWithEvidence(ProtocolAction action)
Map<String, String> generateEvidenceGapReport()
```

### EvidenceChainResolver

```java
// Resolution methods
EvidenceChain resolveChain(String actionId)
Map<String, EvidenceChain> resolveChains(List<String> actionIds)
List<Citation> aggregateCitations(String actionId)
String assessEvidenceQuality(List<Citation> citations)
String generateFormattedEvidenceTrail(String actionId)
String getEvidenceSummary(String actionId)
```

### ProtocolAction

```java
// Utility methods
boolean hasGuidelineSupport()
boolean hasHighQualityEvidence()
boolean hasStrongRecommendation()
boolean isTimeCritical()
String getQualityBadge()
String getEvidenceSummary()
```

### EvidenceChain

```java
// Utility methods
boolean isGuidelineCurrent()
boolean isOutdated()
boolean hasHighQualityEvidence()
boolean hasStrongRecommendation()
boolean isChainComplete()
void calculateCompletenessScore()
String getQualityBadge()
String getFormattedEvidenceTrail()
```

---

## Testing

### Run Integration Example

```bash
cd /backend/shared-infrastructure/flink-processing
javac -cp ".:lib/*" src/main/java/com/cardiofit/flink/knowledgebase/GuidelineIntegrationExample.java
java -cp ".:lib/*" com.cardiofit.flink.knowledgebase.GuidelineIntegrationExample
```

### Expected Output

```
=============================================================
GUIDELINE INTEGRATION EXAMPLE
Phase 5 Day 4: Protocol-Guideline-Citation Integration
=============================================================

EXAMPLE 1: STEMI Aspirin (STEMI-ACT-002)
=========================================

📋 FORMATTED EVIDENCE TRAIL:
Action: Aspirin 324 mg PO (STEMI-ACT-002)
  ↓
Guideline: 2023 ACC/AHA/SCAI STEMI Guideline (2023-04-20)
  ↓
Recommendation: ACC-STEMI-2023-REC-003 - Aspirin 162-325 mg loading dose
  Strength: STRONG (CLASS_I)
  Evidence: HIGH (Level A)
  ↓
Citations:
  • PMID 3081859: ISIS-2 trial - 23% mortality reduction
  • PMID 18160631: De Luca meta-analysis
  ↓
Quality Badge: 🟢 STRONG (High-quality evidence, strong recommendation)

[... additional examples ...]
```

---

## Troubleshooting

### Issue: Evidence chain returns null

**Cause**: Action ID not found in mappings
**Solution**: Check EvidenceChainResolver.initializeActionMappings() for configured mappings

### Issue: Guideline shows as outdated

**Cause**: nextReviewDate has passed
**Solution**: Update guideline YAML with new version or extend review date

### Issue: Low completeness score

**Cause**: Missing evidence fields in chain
**Solution**: Add missing guidelineReference, recommendationId, or citations to protocol YAML

### Issue: Quality badge shows UNGRADED

**Cause**: Missing evidenceQuality or recommendationStrength
**Solution**: Add quality fields to protocol action YAML

---

## Best Practices

1. **Always check currency**: Verify guideline currency before displaying evidence
2. **Cache evidence chains**: Cache for performance in high-traffic scenarios
3. **Handle gaps gracefully**: Display partial evidence when complete chain unavailable
4. **Log evidence access**: Track which actions are queried for analytics
5. **Update regularly**: Review and update action mappings quarterly
6. **Validate completeness**: Aim for ≥80% completeness score on all critical actions
7. **Monitor outdated guidelines**: Set up alerts for approaching review dates

---

## Related Documentation

- **Phase 5 Day 4 Complete Report**: `/claudedocs/PHASE5_DAY4_GUIDELINE_INTEGRATION_COMPLETE.md`
- **Guideline YAML Format**: `/backend/shared-infrastructure/flink-processing/src/main/resources/knowledge-base/guidelines/`
- **Protocol YAML Format**: `/backend/shared-infrastructure/flink-processing/src/main/resources/clinical-protocols/`
- **Integration Example**: `/backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/knowledgebase/GuidelineIntegrationExample.java`

---

## Support

For questions or issues:
- Review the complete implementation report in `/claudedocs/PHASE5_DAY4_GUIDELINE_INTEGRATION_COMPLETE.md`
- Check the integration example for working code samples
- Verify protocol YAML format matches the expected structure
- Ensure all required guideline and citation YAMLs are present in knowledge base

---

**Quick Start Guide Version**: 1.0
**Last Updated**: 2025-10-24
**Status**: Production Ready
