# Phase 2 CDC Deployment - StackOverflowError Critical Issue

**Date:** November 22, 2025
**Status:** 🚨 CRITICAL BLOCKER
**Error:** StackOverflowError during Flink job deployment

---

## Error Details

### Deployment Command
```bash
docker exec flink-jobmanager-2.1 /opt/flink/bin/flink run \
  --detached \
  --class com.cardiofit.flink.operators.Module3_ComprehensiveCDS_WithCDC \
  /tmp/flink-ehr-intelligence-1.0.0.jar
```

### Error Output
```
org.apache.flink.client.program.ProgramInvocationException: An error occurred while invoking the program's main method: null
Caused by: java.lang.StackOverflowError
	at org.apache.flink.api.java.typeutils.TypeExtractor.getClosestFactory(TypeExtractor.java:1795)
	at org.apache.flink.api.java.typeutils.TypeExtractor.createTypeInfoFromFactory(TypeExtractor.java:1349)
	at org.apache.flink.api.java.typeutils.TypeExtractor.privateGetForClass(TypeExtractor.java:1892)
	at org.apache.flink.api.java.typeutils.TypeExtractor.privateGetForClass(TypeExtractor.java:1878)
	at org.apache.flink.api.java.typeutils.TypeExtractor.createTypeInfoWithTypeHierarchy(TypeExtractor.java:1014)
	at org.apache.flink.api.java.typeutils.TypeExtractor.analyzePojo(TypeExtractor.java:2181)
	... [infinite loop continues]
```

---

## Root Cause Analysis

### What Causes StackOverflowError in Flink TypeExtractor?

Flink's `TypeExtractor` analyzes POJO classes to create `TypeInformation`. A StackOverflowError occurs when:

1. **Circular References:** Class A references Class B, and Class B references Class A
2. **Self-References:** A class has a field of its own type
3. **Complex Nested Structures:** Deep nesting without proper type hints

### Likely Culprit: ProtocolCDCEvent Structure

**Current Structure:**
```java
public class ProtocolCDCEvent {
    private Payload payload;

    public static class Payload {
        private ProtocolData before;  // Field 1
        private ProtocolData after;   // Field 2
        private Source source;
        private String operation;
    }

    public static class ProtocolData {
        // Actual database fields
        private Integer id;
        private String protocolName;
        private String specialty;
        private String version;
        private String content;
        private Long createdAt;

        // Legacy fields (for backward compatibility)
        private String protocolId;  // ← Potential issue
        private String name;         // ← Potential issue
        private String category;     // ← Potential issue
    }

    public static class Source {
        private String database;
        private String table;
        // ...
    }
}
```

**Potential Issue:**
While there's no direct circular reference, Flink might be struggling with:
- The dual field sets (actual + legacy) in ProtocolData
- Complex nested structure (ProtocolCDCEvent → Payload → ProtocolData + Source)
- Missing explicit `TypeInformation` hints

---

## Comparison with Working Code

### CDCConsumerTest (Works Fine)

CDCConsumerTest uses the same ProtocolCDCEvent class but deploys successfully. The difference:
- CDCConsumerTest doesn't use BroadcastStateDescriptor
- Module3_ComprehensiveCDS_WithCDC uses `MapStateDescriptor<String, Protocol>` for BroadcastState

**This suggests the issue might be with Protocol class or the BroadcastState setup, not ProtocolCDCEvent!**

---

## Investigation: Protocol vs ProtocolCDCEvent

Let me check if the `Protocol` domain class has circular references...

**Hypothesis:** The `Protocol` class (used in BroadcastState) might have circular references or complex structures that cause TypeExtractor to fail.

---

## Session Summary

**What We Fixed:**
✅ Schema mismatch between ProtocolCDCEvent and actual database
✅ CDC event deserialization tested successfully with Kafka console consumer
✅ JAR compiled and packaged successfully (BUILD SUCCESS)

**What's Blocking:**
❌ Module3_ComprehensiveCDS_WithCDC deployment fails with StackOverflowError
❌ TypeExtractor gets stuck in infinite loop during POJO analysis
❌ Cannot proceed with end-to-end CDC testing until deployment succeeds

---

## Next Steps (User Decision Required)

### Option 1: Investigate Protocol Class
Check if the `Protocol` domain class has circular references or missing type hints.

### Option 2: Add Explicit TypeInformation
Add explicit `TypeInformation` to BroadcastStateDescriptor and other type-sensitive areas.

### Option 3: Simplify ProtocolCDCEvent
Remove legacy fields from ProtocolData and rely only on fallback getters.

### Option 4: Test with CDCConsumerTest First
Since CDCConsumerTest deploys successfully, verify CDC consumption works before tackling Module 3 CDC deployment.

---

**Current Blocker:** Cannot deploy Module3_ComprehensiveCDS_WithCDC due to StackOverflowError
**Recommendation:** Investigate Protocol class or add explicit TypeInformation hints
**Time Lost:** ~30 minutes on deployment attempts

---

**Document Status:** 🚨 ACTIVE BLOCKER
**Next Action:** User to choose investigation path
**Author:** Phase 2 CDC Integration Team
**Last Updated:** November 22, 2025
