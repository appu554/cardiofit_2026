# Phase 5 Day 4: Evidence Chain Tests - Quick Start Guide

## Files Created

### Implementation Files (7 files)
```
src/main/java/com/cardiofit/flink/knowledgebase/
├── Guideline.java              ✅ Complete
├── Recommendation.java         ✅ Complete
├── Citation.java               ✅ Complete (modified by linter)
├── EvidenceChain.java          ✅ Complete
├── GuidelineLoader.java        ✅ Complete
├── CitationLoader.java         ⚠️ Needs update for linter changes
└── GuidelineLinker.java        ✅ Complete
```

### Test Files (5 files)
```
src/test/java/com/cardiofit/flink/knowledgebase/
├── GuidelineLoaderTest.java              ✅ 12 tests
├── CitationLoaderTest.java               ⚠️ Needs update for linter changes
├── GuidelineLinkerTest.java              ✅ 9 tests
├── EvidenceChainIntegrationTest.java     ⚠️ Needs update for linter changes
└── GuidelineValidationTest.java          ✅ 10 tests
```

## Required Updates

### Citation.java Linter Changes

The linter changed:
- `Citation.StudyType` enum → `String studyType`
- Removed: `setYear()` → Use `setPublicationYear()`
- Removed: `setIntervention()`, `setPrimaryOutcome()`, `setMainFinding()`
- Added: `isHighQuality()` method (was `isHighQualityEvidence()`)

### Update CitationLoader.java

Replace lines 59-151 (studyType assignments):
```java
// OLD (enum):
isis2.setStudyType(Citation.StudyType.RCT);

// NEW (string):
isis2.setStudyType("RCT");
```

### Update CitationLoaderTest.java

Replace line 27:
```java
// OLD:
assertEquals(Citation.StudyType.RCT, citation.getStudyType());

// NEW:
assertEquals("RCT", citation.getStudyType());
```

### Update GuidelineLinker.java

Replace lines 118-130:
```java
// OLD:
long rctCount = citations.stream()
    .filter(c -> c.getStudyType() == Citation.StudyType.RCT)
    .count();

// NEW:
long rctCount = citations.stream()
    .filter(c -> "RCT".equals(c.getStudyType()))
    .count();
```

### Update EvidenceChainIntegrationTest.java

Replace line 283:
```java
// OLD:
citation.setStudyType(Citation.StudyType.RCT);

// NEW:
citation.setStudyType("RCT");
```

## Fix Script

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing

# Update CitationLoader.java
sed -i '' 's/Citation\.StudyType\.RCT/"RCT"/g' src/main/java/com/cardiofit/flink/knowledgebase/CitationLoader.java
sed -i '' 's/Citation\.StudyType\.META_ANALYSIS/"META_ANALYSIS"/g' src/main/java/com/cardiofit/flink/knowledgebase/CitationLoader.java
sed -i '' 's/Citation\.StudyType\.COHORT/"COHORT"/g' src/main/java/com/cardiofit/flink/knowledgebase/CitationLoader.java
sed -i '' 's/setYear(/setPublicationYear(/g' src/main/java/com/cardiofit/flink/knowledgebase/CitationLoader.java
sed -i '' 's/isHighQualityEvidence/isHighQuality/g' src/main/java/com/cardiofit/flink/knowledgebase/CitationLoader.java

# Update GuidelineLinker.java
sed -i '' 's/c\.getStudyType() == Citation\.StudyType\.RCT/"RCT".equals(c.getStudyType())/g' src/main/java/com/cardiofit/flink/knowledgebase/GuidelineLinker.java
sed -i '' 's/c\.getStudyType() == Citation\.StudyType\.META_ANALYSIS/"META_ANALYSIS".equals(c.getStudyType())/g' src/main/java/com/cardiofit/flink/knowledgebase/GuidelineLinker.java
sed -i '' 's/c\.getStudyType() == Citation\.StudyType\.COHORT/"COHORT".equals(c.getStudyType())/g' src/main/java/com/cardiofit/flink/knowledgebase/GuidelineLinker.java
sed -i '' 's/c\.getStudyType() == Citation\.StudyType\.OBSERVATIONAL/"OBSERVATIONAL".equals(c.getStudyType())/g' src/main/java/com/cardiofit/flink/knowledgebase/GuidelineLinker.java

# Update CitationLoaderTest.java
sed -i '' 's/Citation\.StudyType\.RCT/"RCT"/g' src/test/java/com/cardiofit/flink/knowledgebase/CitationLoaderTest.java
sed -i '' 's/Citation\.StudyType\.META_ANALYSIS/"META_ANALYSIS"/g' src/test/java/com/cardiofit/flink/knowledgebase/CitationLoaderTest.java
sed -i '' 's/Citation\.StudyType\.COHORT/"COHORT"/g' src/test/java/com/cardiofit/flink/knowledgebase/CitationLoaderTest.java
sed -i '' 's/isHighQualityEvidence/isHighQuality/g' src/test/java/com/cardiofit/flink/knowledgebase/CitationLoaderTest.java

# Update EvidenceChainIntegrationTest.java
sed -i '' 's/Citation\.StudyType\.RCT/"RCT"/g' src/test/java/com/cardiofit/flink/knowledgebase/EvidenceChainIntegrationTest.java
sed -i '' 's/Citation\.StudyType\.META_ANALYSIS/"META_ANALYSIS"/g' src/test/java/com/cardiofit/flink/knowledgebase/EvidenceChainIntegrationTest.java
sed -i '' 's/Citation\.StudyType\.COHORT/"COHORT"/g' src/test/java/com/cardiofit/flink/knowledgebase/EvidenceChainIntegrationTest.java
```

## After Fixes - Test Execution

```bash
# Compile
mvn clean compile

# Run tests
mvn test -Dtest=GuidelineLoaderTest
mvn test -Dtest=CitationLoaderTest
mvn test -Dtest=GuidelineLinkerTest
mvn test -Dtest=EvidenceChainIntegrationTest
mvn test -Dtest=GuidelineValidationTest

# All tests
mvn test -Dtest="com.cardiofit.flink.knowledgebase.*"
```

## Expected Test Results

```
GuidelineLoaderTest           12 tests ✅
CitationLoaderTest            10 tests ✅
GuidelineLinkerTest            9 tests ✅
EvidenceChainIntegrationTest   8 tests ✅
GuidelineValidationTest       10 tests ✅
-------------------------------------------
TOTAL                         49 tests ✅
```

## Quick Validation

```bash
# Check if files exist
ls -l src/main/java/com/cardiofit/flink/knowledgebase/
ls -l src/test/java/com/cardiofit/flink/knowledgebase/

# Count test methods
grep -r "@Test" src/test/java/com/cardiofit/flink/knowledgebase/ | wc -l

# Verify guideline YAMLs
ls -l src/main/resources/knowledge-base/guidelines/cardiac/
```

## Integration Points

- **Protocol Actions**: STEMI-ACT-001 through STEMI-ACT-012
- **Guidelines**: ACC/AHA STEMI 2023, ESC STEMI 2023, etc.
- **Citations**: ISIS-2, PLATO, PROVE-IT, etc.

## Next Steps

1. Run fix script to update for linter changes
2. Execute `mvn clean compile`
3. Run test suite
4. If tests pass → Move to Phase 5 Day 5
5. If tests fail → Check compilation errors and fix

## Documentation

- **Full Report**: `PHASE5_DAY4_TEST_IMPLEMENTATION_COMPLETE.md`
- **This Guide**: `PHASE5_DAY4_QUICK_START_GUIDE.md`

## Contact

For issues, check:
- Compilation errors in other modules (may block build)
- YAML file paths (must be in `src/main/resources/knowledge-base/guidelines/`)
- Dependency versions in `pom.xml`
