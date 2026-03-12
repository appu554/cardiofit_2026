# Testing and Validation Guide

## Table of Contents
1. [Overview](#overview)
2. [Unit Tests](#unit-tests)
3. [Integration Tests](#integration-tests)
4. [Validation Scripts](#validation-scripts)
5. [Performance Benchmarks](#performance-benchmarks)
6. [Continuous Validation](#continuous-validation)

---

## Overview

### Testing Strategy

Comprehensive testing ensures that the guideline library system is:
- **Accurate**: Evidence chains resolve correctly
- **Complete**: All protocol actions have evidence support
- **Current**: Guidelines are up-to-date and not superseded
- **Performant**: Fast enough for real-time clinical use
- **Valid**: YAML files are syntactically and semantically correct

### Test Pyramid

```
┌─────────────────────────────────────────┐
│  E2E Tests                              │
│  - Complete protocol execution          │  ← Few, comprehensive
│  - Evidence chain generation            │
└─────────────────────────────────────────┘
          ▲
          │
┌─────────────────────────────────────────┐
│  Integration Tests                      │
│  - Guideline loading                    │  ← More, focused
│  - Citation resolution                  │
│  - Evidence linking                     │
└─────────────────────────────────────────┘
          ▲
          │
┌─────────────────────────────────────────┐
│  Unit Tests                             │
│  - Individual class methods             │  ← Most, isolated
│  - YAML parsing                         │
│  - Evidence quality assessment          │
└─────────────────────────────────────────┘
```

### Test Categories

| Category | Purpose | Coverage Target |
|----------|---------|----------------|
| **Unit Tests** | Test individual components in isolation | >90% code coverage |
| **Integration Tests** | Test component interactions | All major workflows |
| **Validation Scripts** | Validate YAML data quality | 100% of guidelines |
| **Performance Tests** | Ensure acceptable response times | Critical paths |
| **E2E Tests** | Validate complete user scenarios | Key clinical workflows |

---

## Unit Tests

### GuidelineLoader Tests

```java
package com.cds.knowledgebase.evidence.loader;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.BeforeEach;
import static org.junit.jupiter.api.Assertions.*;

import java.util.List;

public class GuidelineLoaderTest {

    private GuidelineLoader loader;

    @BeforeEach
    public void setUp() {
        loader = new GuidelineLoader();
    }

    @Test
    public void testLoadGuidelineById() {
        // Load guideline
        Guideline guideline = loader.loadGuideline("GUIDE-ACCAHA-STEMI-2023");

        // Assertions
        assertNotNull(guideline, "Guideline should not be null");
        assertEquals("GUIDE-ACCAHA-STEMI-2023", guideline.getGuidelineId());
        assertEquals("ACC/AHA STEMI 2023", guideline.getShortName());
        assertEquals(GuidelineStatus.CURRENT, guideline.getStatus());
        assertNotNull(guideline.getRecommendations());
        assertTrue(guideline.getRecommendations().size() > 0,
                  "Should have recommendations");
    }

    @Test
    public void testLoadNonexistentGuideline() {
        // Attempt to load non-existent guideline
        assertThrows(GuidelineNotFoundException.class, () -> {
            loader.loadGuideline("GUIDE-NONEXISTENT-2099");
        });
    }

    @Test
    public void testLoadGuidelinesByTopic() {
        // Load all STEMI guidelines
        List<Guideline> guidelines = loader.loadGuidelinesByTopic("STEMI");

        assertNotNull(guidelines);
        assertTrue(guidelines.size() >= 2,
                  "Should have at least ACC/AHA and ESC guidelines");

        // Verify all are STEMI-related
        for (Guideline g : guidelines) {
            assertTrue(g.getTopic().contains("STEMI") ||
                      g.getTopic().contains("Myocardial Infarction"));
        }
    }

    @Test
    public void testGetCurrentGuideline() {
        // Get current (non-superseded) STEMI guideline
        Guideline current = loader.getCurrentGuideline("STEMI");

        assertNotNull(current);
        assertEquals(GuidelineStatus.CURRENT, current.getStatus());
        assertNull(current.getSupersededBy(),
                  "Current guideline should not be superseded");
    }

    @Test
    public void testLoadRecommendation() {
        // Load specific recommendation
        Recommendation rec = loader.loadRecommendation("ACC-STEMI-2023-REC-003");

        assertNotNull(rec);
        assertEquals("ACC-STEMI-2023-REC-003", rec.getRecommendationId());
        assertEquals("STRONG", rec.getStrength());
        assertEquals("HIGH", rec.getEvidenceQuality());
        assertTrue(rec.getKeyEvidence().size() > 0,
                  "Should have supporting evidence");
    }

    @Test
    public void testRecommendationHasLinkedActions() {
        // Verify recommendation links to protocol actions
        Recommendation rec = loader.loadRecommendation("ACC-STEMI-2023-REC-003");

        assertNotNull(rec.getLinkedProtocolActions());
        assertTrue(rec.getLinkedProtocolActions().size() > 0,
                  "Recommendation should link to protocol actions");
        assertTrue(rec.getLinkedProtocolActions().contains("STEMI-ACT-002"),
                  "Should link to aspirin action");
    }
}
```

### CitationLoader Tests

```java
package com.cds.knowledgebase.evidence.loader;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.BeforeEach;
import static org.junit.jupiter.api.Assertions.*;

public class CitationLoaderTest {

    private CitationLoader loader;

    @BeforeEach
    public void setUp() {
        loader = new CitationLoader();
    }

    @Test
    public void testLoadCitationByPmid() {
        // Load ISIS-2 trial citation
        Citation citation = loader.loadCitation("3081859");

        assertNotNull(citation);
        assertEquals("3081859", citation.getPmid());
        assertEquals("Lancet", citation.getJournal());
        assertEquals(1988, citation.getPublicationYear());
        assertEquals(StudyType.RCT, citation.getStudyType());
        assertEquals("HIGH", citation.getEvidenceQuality());
        assertEquals(17187, citation.getSampleSize());
    }

    @Test
    public void testLoadCitationByDoi() {
        // Load by DOI
        Citation citation = loader.loadCitationByDoi(
            "10.1016/S0140-6736(88)92833-4"
        );

        assertNotNull(citation);
        assertEquals("3081859", citation.getPmid());
    }

    @Test
    public void testFetchFromPubMed() throws Exception {
        // Fetch from PubMed API
        Citation citation = loader.fetchFromPubMed("3081859");

        assertNotNull(citation);
        assertEquals("3081859", citation.getPmid());
        assertNotNull(citation.getTitle());
        assertNotNull(citation.getAuthors());
        assertTrue(citation.getAuthors().size() > 0);
        assertEquals("PubMed API", citation.getSource());
    }

    @Test
    public void testLoadMultipleCitations() {
        // Load multiple citations
        List<String> pmids = List.of("3081859", "12517460", "18160631");
        List<Citation> citations = loader.loadCitations(pmids);

        assertEquals(3, citations.size());
        assertEquals("3081859", citations.get(0).getPmid());
        assertEquals("12517460", citations.get(1).getPmid());
        assertEquals("18160631", citations.get(2).getPmid());
    }

    @Test
    public void testCitationHasKeyFindings() {
        Citation citation = loader.loadCitation("3081859");

        assertNotNull(citation.getKeyFindings());
        assertTrue(citation.getKeyFindings().size() > 0,
                  "Citation should have key findings");

        // Check for aspirin mortality reduction finding
        boolean hasAspirinFinding = citation.getKeyFindings().stream()
            .anyMatch(f -> f.contains("23%") && f.contains("mortality"));
        assertTrue(hasAspirinFinding,
                  "Should document aspirin mortality benefit");
    }
}
```

### GuidelineLinker Tests

```java
package com.cds.knowledgebase.evidence.linker;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.BeforeEach;
import static org.junit.jupiter.api.Assertions.*;

public class GuidelineLinkerTest {

    private GuidelineLinker linker;

    @BeforeEach
    public void setUp() {
        linker = new GuidelineLinker();
    }

    @Test
    public void testGetEvidenceChain() {
        // Get complete evidence chain for STEMI aspirin action
        EvidenceChain chain = linker.getEvidenceChain("STEMI-ACT-002");

        assertNotNull(chain);
        assertEquals("STEMI-ACT-002", chain.getActionId());
        assertNotNull(chain.getActionDescription());
        assertTrue(chain.getActionDescription().contains("Aspirin"));

        // Guideline layer
        assertEquals("GUIDE-ACCAHA-STEMI-2023", chain.getGuidelineId());
        assertEquals("ACC/AHA STEMI 2023", chain.getGuidelineName());
        assertEquals("ACC-STEMI-2023-REC-003", chain.getRecommendationId());

        // Evidence layer
        assertNotNull(chain.getKeyCitations());
        assertTrue(chain.getKeyCitations().size() > 0,
                  "Should have supporting citations");

        // Quality assessment
        assertEquals("STRONG", chain.getStrengthOfRecommendation());
        assertEquals("HIGH", chain.getEvidenceQuality());
        assertEquals("🟢 STRONG", chain.getQualityBadge());
        assertTrue(chain.isCurrent());
    }

    @Test
    public void testGetGuidelinesForAction() {
        // Action may be supported by multiple guidelines
        List<Guideline> guidelines = linker.getGuidelinesForAction("STEMI-ACT-002");

        assertNotNull(guidelines);
        assertTrue(guidelines.size() > 0);

        // Should include at least ACC/AHA guideline
        boolean hasAccAha = guidelines.stream()
            .anyMatch(g -> g.getGuidelineId().equals("GUIDE-ACCAHA-STEMI-2023"));
        assertTrue(hasAccAha);
    }

    @Test
    public void testGetQualityBadge() {
        String badge = linker.getQualityBadge("STEMI-ACT-002");

        assertNotNull(badge);
        assertTrue(badge.contains("🟢") || badge.contains("🟡"),
                  "Should have quality badge emoji");
    }

    @Test
    public void testIsEvidenceCurrent() {
        // Check if evidence is current (not superseded)
        boolean isCurrent = linker.isEvidenceCurrent("STEMI-ACT-002");

        assertTrue(isCurrent,
                  "STEMI aspirin action should have current evidence");
    }

    @Test
    public void testEvidenceTrailFormat() {
        EvidenceChain chain = linker.getEvidenceChain("STEMI-ACT-002");
        String trail = chain.getEvidenceTrail();

        assertNotNull(trail);
        assertTrue(trail.contains("Action:"), "Should describe action");
        assertTrue(trail.contains("Guideline:"), "Should reference guideline");
        assertTrue(trail.contains("Citation"), "Should include citations");
        assertTrue(trail.contains("Quality"), "Should show quality assessment");
    }
}
```

### EvidenceQualityAssessor Tests

```java
package com.cds.knowledgebase.evidence.linker;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class EvidenceQualityAssessorTest {

    private EvidenceQualityAssessor assessor = new EvidenceQualityAssessor();

    @Test
    public void testAssessQualityHighRct() {
        Recommendation rec = Recommendation.builder()
            .strength("STRONG")
            .evidenceQuality("HIGH")
            .build();

        List<Citation> citations = List.of(
            Citation.builder()
                .pmid("3081859")
                .studyType(StudyType.RCT)
                .sampleSize(17187)
                .build()
        );

        String quality = assessor.assessQuality(rec, citations);
        assertEquals("HIGH", quality);
    }

    @Test
    public void testAssessQualitySmallRct() {
        List<Citation> citations = List.of(
            Citation.builder()
                .studyType(StudyType.RCT)
                .sampleSize(100)  // Small sample
                .build()
        );

        String quality = assessor.mapStudyTypeToQuality(
            citations.get(0).getStudyType(),
            citations.get(0).getSampleSize(),
            false
        );

        assertEquals("MODERATE", quality,
                    "Small RCT should be MODERATE quality");
    }

    @Test
    public void testGenerateBadge() {
        assertEquals("🟢 STRONG",
                    assessor.generateBadge("STRONG", "HIGH"));
        assertEquals("🟡 MODERATE",
                    assessor.generateBadge("STRONG", "MODERATE"));
        assertEquals("🟠 WEAK",
                    assessor.generateBadge("WEAK", "LOW"));
        assertEquals("🔴 VERY_WEAK",
                    assessor.generateBadge("WEAK", "VERY_LOW"));
    }

    @Test
    public void testMapStudyTypeToQuality() {
        assertEquals("HIGH",
                    assessor.mapStudyTypeToQuality(StudyType.META_ANALYSIS));
        assertEquals("HIGH",
                    assessor.mapStudyTypeToQuality(StudyType.RCT));
        assertEquals("MODERATE",
                    assessor.mapStudyTypeToQuality(StudyType.SYSTEMATIC_REVIEW));
        assertEquals("LOW",
                    assessor.mapStudyTypeToQuality(StudyType.COHORT));
        assertEquals("VERY_LOW",
                    assessor.mapStudyTypeToQuality(StudyType.EXPERT_OPINION));
    }
}
```

---

## Integration Tests

### Complete Evidence Chain Resolution

```java
package com.cds.knowledgebase.evidence;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class EvidenceChainIntegrationTest {

    @Test
    public void testCompleteEvidenceChainResolution() {
        // Start with protocol action
        String actionId = "STEMI-ACT-002";

        // Load protocol
        ProtocolLoader protocolLoader = new ProtocolLoader();
        Protocol protocol = protocolLoader.loadProtocol("STEMI");
        Action action = protocol.getAction(actionId);

        assertNotNull(action, "Action should exist in protocol");

        // Get guideline references from action
        List<String> guidelineRefs = action.getGuidelineReferences();
        assertTrue(guidelineRefs.size() > 0,
                  "Action should reference guidelines");

        // Load guideline
        GuidelineLoader guidelineLoader = new GuidelineLoader();
        Recommendation rec = guidelineLoader.loadRecommendation(
            guidelineRefs.get(0)
        );

        assertNotNull(rec, "Recommendation should load");
        assertEquals("STRONG", rec.getStrength());
        assertEquals("HIGH", rec.getEvidenceQuality());

        // Load citations
        CitationLoader citationLoader = new CitationLoader();
        List<Citation> citations = citationLoader.loadCitations(
            rec.getKeyEvidence()
        );

        assertTrue(citations.size() > 0, "Should have citations");

        // Verify at least one high-quality citation
        boolean hasHighQuality = citations.stream()
            .anyMatch(c -> "HIGH".equals(c.getEvidenceQuality()));
        assertTrue(hasHighQuality,
                  "Should have at least one HIGH quality citation");

        // Build complete evidence chain
        GuidelineLinker linker = new GuidelineLinker();
        EvidenceChain chain = linker.getEvidenceChain(actionId);

        assertNotNull(chain);
        assertEquals(action.getActionId(), chain.getActionId());
        assertEquals(rec.getRecommendationId(), chain.getRecommendationId());
        assertTrue(chain.getKeyCitations().size() > 0);
        assertEquals("🟢 STRONG", chain.getQualityBadge());
    }

    @Test
    public void testBidirectionalLinkage() {
        // Test that guideline → action and action → guideline links match

        GuidelineLoader guidelineLoader = new GuidelineLoader();
        ProtocolLoader protocolLoader = new ProtocolLoader();

        // Load guideline recommendation
        Recommendation rec = guidelineLoader.loadRecommendation(
            "ACC-STEMI-2023-REC-003"
        );

        // Get linked actions
        List<String> linkedActions = rec.getLinkedProtocolActions();
        assertTrue(linkedActions.size() > 0);

        // Load each action and verify reverse link
        for (String actionId : linkedActions) {
            Protocol protocol = protocolLoader.loadProtocolForAction(actionId);
            Action action = protocol.getAction(actionId);

            assertTrue(action.getGuidelineReferences()
                            .contains("ACC-STEMI-2023-REC-003"),
                      "Action should reference guideline");
        }
    }

    @Test
    public void testSupersededGuidelineDetection() {
        // Load old SSC 2016 guideline
        GuidelineLoader loader = new GuidelineLoader();
        Guideline oldGuideline = loader.loadGuideline("GUIDE-SSC-2016");

        assertEquals(GuidelineStatus.SUPERSEDED, oldGuideline.getStatus());
        assertEquals("GUIDE-SSC-2021", oldGuideline.getSupersededBy());

        // Load new guideline
        Guideline newGuideline = loader.loadGuideline("GUIDE-SSC-2021");
        assertEquals(GuidelineStatus.CURRENT, newGuideline.getStatus());

        // Verify actions use current guideline
        GuidelineLinker linker = new GuidelineLinker();
        boolean current = linker.isEvidenceCurrent("SEPSIS-ACT-003");
        assertTrue(current, "Sepsis actions should use current guideline");
    }
}
```

### Protocol Coverage Tests

```java
public class ProtocolCoverageTest {

    @Test
    public void testAllActionsHaveEvidence() {
        ProtocolLoader protocolLoader = new ProtocolLoader();
        GuidelineLinker linker = new GuidelineLinker();

        // Test all protocols
        List<String> protocolIds = List.of("STEMI", "SEPSIS", "RESPIRATORY");

        for (String protocolId : protocolIds) {
            Protocol protocol = protocolLoader.loadProtocol(protocolId);

            for (Action action : protocol.getActions()) {
                EvidenceChain chain = linker.getEvidenceChain(
                    action.getActionId()
                );

                assertNotNull(chain,
                    "Action " + action.getActionId() + " missing evidence");

                assertTrue(chain.getKeyCitations().size() > 0,
                    "Action " + action.getActionId() + " has no citations");

                assertTrue(chain.isCurrent(),
                    "Action " + action.getActionId() +
                    " uses superseded guideline");
            }
        }
    }

    @Test
    public void testEvidenceQualityDistribution() {
        // Verify we have good mix of evidence qualities
        GuidelineLinker linker = new GuidelineLinker();
        ProtocolLoader protocolLoader = new ProtocolLoader();

        List<String> allActions = getAllProtocolActions(protocolLoader);

        Map<String, Long> qualityCounts = allActions.stream()
            .map(actionId -> linker.getEvidenceChain(actionId))
            .collect(Collectors.groupingBy(
                EvidenceChain::getEvidenceQuality,
                Collectors.counting()
            ));

        // Expect majority to be HIGH or MODERATE
        long highAndModerate = qualityCounts.getOrDefault("HIGH", 0L) +
                               qualityCounts.getOrDefault("MODERATE", 0L);
        long total = qualityCounts.values().stream()
                                  .mapToLong(Long::longValue)
                                  .sum();

        double percentage = (double) highAndModerate / total;
        assertTrue(percentage > 0.70,
                  "At least 70% of actions should have HIGH or MODERATE evidence");
    }
}
```

---

## Validation Scripts

### YAML Syntax Validation

```bash
#!/bin/bash
# validate-yaml-syntax.sh
# Validates YAML syntax for all guideline files

echo "Validating YAML syntax..."

GUIDELINES_DIR="src/main/resources/knowledge-base/guidelines"
ERROR_COUNT=0

for file in $(find $GUIDELINES_DIR -name "*.yaml"); do
    echo -n "Checking $(basename $file)... "

    # Use yamllint or Python to validate
    python3 -c "
import yaml
import sys
try:
    with open('$file', 'r') as f:
        yaml.safe_load(f)
    print('✓')
except Exception as e:
    print('✗ ERROR:', str(e))
    sys.exit(1)
    " || ((ERROR_COUNT++))
done

if [ $ERROR_COUNT -eq 0 ]; then
    echo ""
    echo "✅ All YAML files are syntactically valid"
    exit 0
else
    echo ""
    echo "❌ Found $ERROR_COUNT files with YAML syntax errors"
    exit 1
fi
```

### Citation Coverage Validation

```bash
#!/bin/bash
# validate-citations.sh
# Ensures all PMIDs referenced in guidelines have citation files

echo "Validating citation coverage..."

GUIDELINES_DIR="src/main/resources/knowledge-base/guidelines"
CITATIONS_DIR="src/main/resources/knowledge-base/citations"
MISSING_COUNT=0

# Extract all PMIDs from guidelines
ALL_PMIDS=$(grep -rh "keyEvidence:" $GUIDELINES_DIR | \
            grep -oP '- "\K\d+' | \
            sort -u)

for pmid in $ALL_PMIDS; do
    CITATION_FILE="$CITATIONS_DIR/pmid-$pmid.yaml"

    if [ ! -f "$CITATION_FILE" ]; then
        echo "⚠️  Missing citation file for PMID $pmid"
        ((MISSING_COUNT++))
    fi
done

if [ $MISSING_COUNT -eq 0 ]; then
    echo "✅ All citations present"
    exit 0
else
    echo "❌ Missing $MISSING_COUNT citation files"
    exit 1
fi
```

### Protocol Linkage Validation

```bash
#!/bin/bash
# validate-protocol-links.sh
# Validates bidirectional links between guidelines and protocols

echo "Validating protocol linkage..."

GUIDELINES_DIR="src/main/resources/knowledge-base/guidelines"
PROTOCOLS_DIR="src/main/resources/protocols"
ERROR_COUNT=0

# Extract all linkedProtocolActions from guidelines
grep -rh "linkedProtocolActions:" $GUIDELINES_DIR -A 10 | \
    grep -oP '- "\K[A-Z\-0-9]+' | \
    sort -u | \
while read action_id; do

    # Search for action in protocol files
    if ! grep -rq "actionId: \"$action_id\"" $PROTOCOLS_DIR; then
        echo "⚠️  Action $action_id referenced in guideline but not found in protocols"
        ((ERROR_COUNT++))
    fi
done

if [ $ERROR_COUNT -eq 0 ]; then
    echo "✅ All protocol links valid"
else
    echo "❌ Found $ERROR_COUNT broken protocol links"
fi
```

### Comprehensive Validation Script

```python
#!/usr/bin/env python3
"""
Comprehensive validation of guideline knowledge base
"""

import yaml
import os
import sys
from pathlib import Path

class GuidelineValidator:

    def __init__(self, base_dir):
        self.base_dir = Path(base_dir)
        self.guidelines_dir = self.base_dir / "guidelines"
        self.citations_dir = self.base_dir / "citations"
        self.protocols_dir = self.base_dir / "protocols"
        self.errors = []
        self.warnings = []

    def validate_all(self):
        """Run all validations"""
        print("Running comprehensive validation...")
        print("=" * 60)

        self.validate_yaml_syntax()
        self.validate_required_fields()
        self.validate_citation_coverage()
        self.validate_protocol_linkage()
        self.validate_superseded_guidelines()
        self.validate_evidence_quality()

        self.print_report()

    def validate_yaml_syntax(self):
        """Check YAML syntax"""
        print("\n1. Validating YAML syntax...")

        for yaml_file in self.guidelines_dir.rglob("*.yaml"):
            try:
                with open(yaml_file) as f:
                    yaml.safe_load(f)
                print(f"  ✓ {yaml_file.name}")
            except Exception as e:
                error = f"YAML syntax error in {yaml_file.name}: {e}"
                self.errors.append(error)
                print(f"  ✗ {yaml_file.name}: {e}")

    def validate_required_fields(self):
        """Check required fields present"""
        print("\n2. Validating required fields...")

        required_fields = [
            'guidelineId', 'name', 'shortName', 'organization',
            'topic', 'version', 'publicationDate', 'status',
            'publication', 'scope', 'methodology', 'recommendations'
        ]

        for yaml_file in self.guidelines_dir.rglob("*.yaml"):
            try:
                with open(yaml_file) as f:
                    data = yaml.safe_load(f)

                for field in required_fields:
                    if field not in data:
                        error = f"{yaml_file.name}: Missing required field '{field}'"
                        self.errors.append(error)

                # Check recommendations
                if 'recommendations' in data:
                    for rec in data['recommendations']:
                        if 'recommendationId' not in rec:
                            self.errors.append(
                                f"{yaml_file.name}: Recommendation missing ID"
                            )
                        if 'strength' not in rec or rec['strength'] not in \
                           ['STRONG', 'WEAK', 'CONDITIONAL']:
                            self.errors.append(
                                f"{yaml_file.name}: Invalid recommendation strength"
                            )

                print(f"  ✓ {yaml_file.name}")

            except Exception as e:
                self.errors.append(f"Error validating {yaml_file.name}: {e}")

    def validate_citation_coverage(self):
        """Ensure all PMIDs have citation files"""
        print("\n3. Validating citation coverage...")

        all_pmids = set()

        # Collect all PMIDs from guidelines
        for yaml_file in self.guidelines_dir.rglob("*.yaml"):
            try:
                with open(yaml_file) as f:
                    data = yaml.safe_load(f)

                for rec in data.get('recommendations', []):
                    pmids = rec.get('keyEvidence', [])
                    all_pmids.update(pmids)

            except Exception as e:
                pass

        # Check if citation files exist
        missing = []
        for pmid in all_pmids:
            citation_file = self.citations_dir / f"pmid-{pmid}.yaml"
            if not citation_file.exists():
                missing.append(pmid)
                self.warnings.append(f"Missing citation file for PMID {pmid}")

        if missing:
            print(f"  ⚠️  Missing {len(missing)} citation files")
        else:
            print(f"  ✓ All {len(all_pmids)} citations present")

    def validate_protocol_linkage(self):
        """Validate bidirectional protocol links"""
        print("\n4. Validating protocol linkage...")

        # Collect all linkedProtocolActions from guidelines
        linked_actions = set()

        for yaml_file in self.guidelines_dir.rglob("*.yaml"):
            try:
                with open(yaml_file) as f:
                    data = yaml.safe_load(f)

                for rec in data.get('recommendations', []):
                    actions = rec.get('linkedProtocolActions', [])
                    linked_actions.update(actions)

            except Exception as e:
                pass

        # Check if actions exist in protocols
        protocol_actions = set()
        for yaml_file in self.protocols_dir.rglob("*.yaml"):
            try:
                with open(yaml_file) as f:
                    data = yaml.safe_load(f)

                for action in data.get('actions', []):
                    action_id = action.get('actionId')
                    if action_id:
                        protocol_actions.add(action_id)

            except Exception as e:
                pass

        # Find broken links
        broken = linked_actions - protocol_actions
        if broken:
            for action_id in broken:
                self.warnings.append(
                    f"Action {action_id} referenced but not found in protocols"
                )
            print(f"  ⚠️  {len(broken)} broken protocol links")
        else:
            print(f"  ✓ All {len(linked_actions)} protocol links valid")

    def validate_superseded_guidelines(self):
        """Check superseded guideline references"""
        print("\n5. Validating superseded guidelines...")

        for yaml_file in self.guidelines_dir.rglob("*.yaml"):
            try:
                with open(yaml_file) as f:
                    data = yaml.safe_load(f)

                if data.get('status') == 'SUPERSEDED':
                    if 'supersededBy' not in data:
                        self.errors.append(
                            f"{yaml_file.name}: SUPERSEDED but missing 'supersededBy'"
                        )
                    else:
                        # Check if new guideline exists
                        new_id = data['supersededBy']
                        # Search for new guideline
                        found = False
                        for search_file in self.guidelines_dir.rglob("*.yaml"):
                            with open(search_file) as sf:
                                search_data = yaml.safe_load(sf)
                                if search_data.get('guidelineId') == new_id:
                                    found = True
                                    break

                        if not found:
                            self.warnings.append(
                                f"{yaml_file.name}: Superseded by {new_id} which doesn't exist"
                            )

            except Exception as e:
                pass

        print("  ✓ Superseded guideline references validated")

    def validate_evidence_quality(self):
        """Validate evidence quality assignments"""
        print("\n6. Validating evidence quality...")

        valid_strengths = ['STRONG', 'WEAK', 'CONDITIONAL']
        valid_qualities = ['HIGH', 'MODERATE', 'LOW', 'VERY_LOW']

        for yaml_file in self.guidelines_dir.rglob("*.yaml"):
            try:
                with open(yaml_file) as f:
                    data = yaml.safe_load(f)

                for rec in data.get('recommendations', []):
                    strength = rec.get('strength')
                    quality = rec.get('evidenceQuality')

                    if strength not in valid_strengths:
                        self.errors.append(
                            f"{yaml_file.name}: Invalid strength '{strength}'"
                        )

                    if quality not in valid_qualities:
                        self.errors.append(
                            f"{yaml_file.name}: Invalid evidenceQuality '{quality}'"
                        )

            except Exception as e:
                pass

        print("  ✓ Evidence quality values validated")

    def print_report(self):
        """Print validation report"""
        print("\n" + "=" * 60)
        print("VALIDATION REPORT")
        print("=" * 60)

        if self.errors:
            print(f"\n❌ ERRORS ({len(self.errors)}):")
            for error in self.errors:
                print(f"  - {error}")

        if self.warnings:
            print(f"\n⚠️  WARNINGS ({len(self.warnings)}):")
            for warning in self.warnings:
                print(f"  - {warning}")

        if not self.errors and not self.warnings:
            print("\n✅ All validations passed!")
            return 0
        elif self.errors:
            print(f"\n❌ Validation failed with {len(self.errors)} errors")
            return 1
        else:
            print(f"\n✅ Validation passed with {len(self.warnings)} warnings")
            return 0


if __name__ == '__main__':
    base_dir = sys.argv[1] if len(sys.argv) > 1 else "src/main/resources/knowledge-base"

    validator = GuidelineValidator(base_dir)
    exit_code = validator.validate_all()
    sys.exit(exit_code)
```

---

## Performance Benchmarks

### Expected Response Times

| Operation | Target Time | Maximum Time |
|-----------|-------------|--------------|
| Load guideline by ID | <10 ms | 50 ms |
| Load recommendation | <5 ms | 20 ms |
| Load citation by PMID | <5 ms | 20 ms |
| Resolve evidence chain | <50 ms | 200 ms |
| Generate evidence report (10 actions) | <500 ms | 2000 ms |

### Performance Test Suite

```java
package com.cds.knowledgebase.evidence.performance;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class PerformanceTest {

    @Test
    public void testGuidelineLoadPerformance() {
        GuidelineLoader loader = new GuidelineLoader();

        long start = System.currentTimeMillis();
        Guideline guideline = loader.loadGuideline("GUIDE-ACCAHA-STEMI-2023");
        long duration = System.currentTimeMillis() - start;

        assertNotNull(guideline);
        assertTrue(duration < 50,
                  "Guideline load took " + duration + "ms (target <50ms)");
    }

    @Test
    public void testEvidenceChainPerformance() {
        GuidelineLinker linker = new GuidelineLinker();

        long start = System.currentTimeMillis();
        EvidenceChain chain = linker.getEvidenceChain("STEMI-ACT-002");
        long duration = System.currentTimeMillis() - start;

        assertNotNull(chain);
        assertTrue(duration < 200,
                  "Evidence chain resolution took " + duration + "ms (target <200ms)");
    }

    @Test
    public void testBatchEvidenceChainPerformance() {
        GuidelineLinker linker = new GuidelineLinker();
        List<String> actionIds = List.of(
            "STEMI-ACT-001", "STEMI-ACT-002", "STEMI-ACT-003",
            "STEMI-ACT-004", "STEMI-ACT-005", "STEMI-ACT-006",
            "STEMI-ACT-007", "STEMI-ACT-008", "STEMI-ACT-009",
            "STEMI-ACT-010"
        );

        long start = System.currentTimeMillis();
        List<EvidenceChain> chains = actionIds.stream()
            .map(id -> linker.getEvidenceChain(id))
            .collect(Collectors.toList());
        long duration = System.currentTimeMillis() - start;

        assertEquals(10, chains.size());
        assertTrue(duration < 2000,
                  "Batch resolution (10 actions) took " + duration + "ms (target <2000ms)");
    }
}
```

---

## Continuous Validation

### GitHub Actions Workflow

```yaml
# .github/workflows/validate-guidelines.yml
name: Validate Guideline Knowledge Base

on:
  push:
    paths:
      - 'src/main/resources/knowledge-base/**'
  pull_request:
    paths:
      - 'src/main/resources/knowledge-base/**'

jobs:
  validate:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v3

      - name: Set up Python
        uses: actions/setup-python@v4
        with:
          python-version: '3.11'

      - name: Install dependencies
        run: |
          pip install pyyaml

      - name: Validate YAML syntax
        run: |
          ./scripts/validate-yaml-syntax.sh

      - name: Validate citations
        run: |
          ./scripts/validate-citations.sh

      - name: Validate protocol links
        run: |
          ./scripts/validate-protocol-links.sh

      - name: Comprehensive validation
        run: |
          python scripts/comprehensive-validator.py

      - name: Run Java tests
        run: |
          mvn test -Dtest=GuidelineLoaderTest,CitationLoaderTest,GuidelineLinkerTest
```

---

## Conclusion

Comprehensive testing and validation ensures:
- **Correctness**: Evidence chains resolve accurately
- **Completeness**: All protocol actions have evidence support
- **Currency**: Guidelines are up-to-date
- **Performance**: Fast enough for real-time clinical use
- **Quality**: YAML data is valid and well-formed

### Quick Test Checklist

Before committing guideline changes:

- [ ] Run YAML syntax validation
- [ ] Run citation coverage check
- [ ] Run protocol linkage validation
- [ ] Run unit tests
- [ ] Run integration tests
- [ ] Verify performance benchmarks
- [ ] Check for superseded guidelines

For additional guidance, see:
- [Evidence Chain Implementation Guide](./Evidence_Chain_Implementation_Guide.md)
- [Guideline YAML Authoring Guide](./Guideline_YAML_Authoring_Guide.md)
- [Citation Management Guide](./Citation_Management_Guide.md)
