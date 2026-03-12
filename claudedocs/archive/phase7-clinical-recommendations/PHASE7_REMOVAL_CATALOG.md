# Phase 7 Clinical Recommendation Engine - Removal Catalog

**Date**: 2025-10-26
**Purpose**: Catalog all Phase 7 files before removal
**Reason**: Starting fresh with Design Specification (Evidence Repository)

---

## Files to Remove

### Java Classes - Clinical Package (5 files)
1. `src/main/java/com/cardiofit/flink/clinical/MedicationActionBuilder.java` (492 lines)
2. `src/main/java/com/cardiofit/flink/clinical/SafetyValidator.java` (340 lines)
3. `src/main/java/com/cardiofit/flink/clinical/AlternativeActionGenerator.java` (370 lines)
4. `src/main/java/com/cardiofit/flink/clinical/RecommendationEnricher.java` (480 lines)
5. `src/main/java/com/cardiofit/flink/clinical/SafetyValidationResult.java` (180 lines)

### Java Classes - Protocols Package (5 files)
6. `src/main/java/com/cardiofit/flink/protocols/ClinicalProtocolDefinition.java` (310 lines)
7. `src/main/java/com/cardiofit/flink/protocols/ProtocolLibraryLoader.java` (320 lines)
8. `src/main/java/com/cardiofit/flink/protocols/EnhancedProtocolMatcher.java` (268 lines)
9. `src/main/java/com/cardiofit/flink/protocols/ProtocolActionBuilder.java` (285 lines)
10. NOTE: `Protocol.java` and `ProtocolMatcher.java` - Check if pre-existing

### Java Classes - Operators Package (2 files)
11. `src/main/java/com/cardiofit/flink/operators/Module3_ClinicalRecommendationEngine.java` (187 lines)
12. `src/main/java/com/cardiofit/flink/operators/ClinicalRecommendationProcessor.java` (490 lines)

### Java Classes - Serialization Package (2 files)
13. `src/main/java/com/cardiofit/flink/serialization/EnrichedPatientContextDeserializer.java` (103 lines)
14. `src/main/java/com/cardiofit/flink/serialization/ClinicalRecommendationSerializer.java` (78 lines)

### Java Classes - Models Package (Phase 7 specific)
15. `src/main/java/com/cardiofit/flink/models/StructuredAction.java` (283 lines)
16. `src/main/java/com/cardiofit/flink/models/ContraindicationCheck.java` (173 lines)
17. `src/main/java/com/cardiofit/flink/models/AlternativeAction.java` (145 lines)
18. `src/main/java/com/cardiofit/flink/models/ProtocolState.java` (178 lines)
19. `src/main/java/com/cardiofit/flink/models/ClinicalAction.java` (check if Phase 7 specific)
20. `src/main/java/com/cardiofit/flink/models/ProtocolAction.java` (check if Phase 7 specific)
21. `src/main/java/com/cardiofit/flink/models/ClinicalRecommendation.java` (check if Phase 7 specific)

### YAML Protocol Files (10 files - 2,128 lines total)
22. `src/main/resources/protocols/SEPSIS-BUNDLE-001.yaml` (247 lines)
23. `src/main/resources/protocols/STEMI-001.yaml` (208 lines)
24. `src/main/resources/protocols/HF-ACUTE-001.yaml` (195 lines)
25. `src/main/resources/protocols/DKA-001.yaml` (212 lines)
26. `src/main/resources/protocols/ARDS-001.yaml` (223 lines)
27. `src/main/resources/protocols/STROKE-001.yaml` (198 lines)
28. `src/main/resources/protocols/ANAPHYLAXIS-001.yaml` (187 lines)
29. `src/main/resources/protocols/HYPERKALEMIA-001.yaml` (201 lines)
30. `src/main/resources/protocols/ACS-NSTEMI-001.yaml` (235 lines)
31. `src/main/resources/protocols/HYPERTENSIVE-CRISIS-001.yaml` (222 lines)

### Test Files
32. `src/test/java/com/cardiofit/flink/phase7/Phase7CompilationTest.java` (165 lines)
33. NOTE: Phase7IntegrationTest.java and ClinicalScenarioTest.java already removed

### Directories to Remove
- `src/main/java/com/cardiofit/flink/clinical/` (entire directory)
- `src/main/java/com/cardiofit/flink/protocols/` (check for pre-existing files first)
- `src/main/resources/protocols/` (entire directory)
- `src/test/java/com/cardiofit/flink/phase7/` (entire directory)

---

## Files to Archive (Documentation)

Keep in `claudedocs/archive/phase7-clinical-recommendations/`:

1. MODULE3_PHASE7_COMPLETION_REPORT.md
2. PHASE7_COMPILATION_FIX_COMPLETE.md
3. PHASE7_PRODUCTION_DEPLOYMENT_STATUS.md
4. PHASE7_QUICK_START.md
5. PHASE7_SPECIFICATION_VS_IMPLEMENTATION_ANALYSIS.md
6. PHASE7_DETAILED_CROSSCHECK_ANALYSIS.md
7. PHASE7_CROSSCHECK_EXECUTIVE_SUMMARY.md
8. PHASE7_FINAL_STATUS.md
9. PHASE7_REMOVAL_CATALOG.md (this file)

---

## Pre-Removal Verification

### Check for Shared/Pre-existing Files

Files that MAY have existed before Phase 7:
- `Protocol.java` - Check git history
- `ProtocolMatcher.java` - Check git history
- `ClinicalAction.java` - Check git history
- `ProtocolAction.java` - Check git history
- `ClinicalRecommendation.java` - Check git history

### Compilation Test After Removal

After removal, verify:
```bash
mvn clean compile
# Should succeed (minus Phase 7 components)
```

---

## Total Removal Summary

- **Java Files**: ~21-28 files (need to verify shared files)
- **YAML Files**: 10 protocol files
- **Test Files**: 1-3 files
- **Total Lines**: ~5,860 production + 2,128 YAML = 7,988 lines
- **Directories**: 3-4 directories

---

*Catalog Created: 2025-10-26*
*Purpose: Safe removal with ability to restore if needed*
