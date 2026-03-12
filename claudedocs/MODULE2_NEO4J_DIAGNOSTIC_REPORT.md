# Module 2 Neo4j Integration Diagnostic Report

**Date**: October 10, 2025
**Status**: 🔴 **ISSUES IDENTIFIED - PARTIAL FIX APPLIED**

---

## Executive Summary

Investigation revealed that Module 2 was **NOT** connected to Neo4j and enrichment was failing for multiple reasons. While we successfully fixed the Neo4j connectivity and encryption issues, additional problems with FHIR API connections and async timeout configuration prevent full dual-system enrichment.

---

## Initial Problem: Neo4j Not Connected

### Evidence
```log
ERROR Neo4jGraphClient - Failed to verify Neo4j connectivity
```

### Root Cause
**Network Isolation**: Neo4j container was on `bridge` network while Flink was on `kafka_cardiofit-network`. Containers couldn't communicate.

```bash
# Neo4j network
docker inspect neo4j | grep -A 10 "Networks"
→ "bridge" network (172.17.x.x)

# Flink network
docker inspect flink-jobmanager-2.1 | grep -A 10 "Networks"
→ "kafka_cardiofit-network" (172.25.x.x)
```

### Fix Applied ✅
```bash
docker network connect kafka_cardiofit-network neo4j
```

**Result**: Neo4j hostname now resolves in Flink containers:
```bash
docker exec flink-taskmanager-2-2.1 getent hosts neo4j
→ 172.25.0.6      neo4j
```

---

## Second Problem: Neo4j Encryption Mismatch

### Evidence
```log
WARN Neo4jGraphClient - Error querying Neo4j for patient PAT-ROHAN-001:
org.neo4j.driver.exceptions.ServiceUnavailableException:
Connection to the database terminated. Please ensure that you have
compatible encryption settings both on Neo4j server and driver.
```

### Root Cause
Neo4j 5.x defaults to **no encryption** in development mode, but the driver was configured with `.withEncryption()`.

**File**: `Neo4jGraphClient.java:76`
```java
Config config = Config.builder()
    .withEncryption() // ❌ Incompatible with Neo4j 5.x dev mode
    .build();
```

### Fix Applied ✅
```java
Config config = Config.builder()
    .withoutEncryption() // ✅ Compatible with Neo4j 5.x dev mode
    .build();
```

**Result**: Connection successful:
```log
INFO Neo4jGraphClient - Neo4j driver initialized and verified successfully
```

---

## Third Problem: Async Enrichment Timeout

### Evidence
```log
WARN AsyncPatientEnricher - Async enrichment timeout (2000ms) for patient PAT-ROHAN-001 - returning empty snapshot
```

Timeline of events:
```
16:38:21 - FHIR queries started (patient, conditions, medications)
16:38:23 - Timeout triggered at 2000ms, empty snapshot returned
16:38:28 - FHIR data successfully fetched and cached (5 seconds too late!)
```

### Root Cause
Dual-system enrichment (FHIR + Neo4j) requires:
1. FHIR Patient query (~1s)
2. FHIR Conditions query (~1s)
3. FHIR Medications query (~1s)
4. Neo4j graph query (~0.5s)

**Total**: ~3.5 seconds, exceeding 2-second timeout

### Fix Applied ✅
**File**: `Module2_ContextAssembly.java:85`
```java
AsyncDataStream.unorderedWait(
    canonicalEvents,
    new AsyncPatientEnricher(),
    2000,  // ❌ Too short for dual lookup
    TimeUnit.MILLISECONDS,
    300
)
```

Changed to:
```java
AsyncDataStream.unorderedWait(
    canonicalEvents,
    new AsyncPatientEnricher(),
    5000,  // ✅ Allows time for FHIR + Neo4j
    TimeUnit.MILLISECONDS,
    300
)
```

### Deployment Issue 🔴
**CRITICAL**: The updated JAR was copied but the old timeout value still appears in logs:
```log
Async enrichment timeout (2000ms)  # Should be 5000ms
```

**Possible causes**:
1. JAR not properly reloaded by Flink
2. Cached classes in TaskManager
3. Wrong JAR deployed

---

## Fourth Problem: FHIR API TLS Failures

### Evidence
```log
WARN GoogleFHIRClient - Request failed for https://healthcare.googleapis.com/.../Patient/PAT-ROHAN-001:
failure when writing TLS control frames

ERROR GoogleFHIRClient - Error fetching patient PAT-ROHAN-001:
java.net.ConnectException: failure when writing TLS control frames
```

### Root Cause
**Unknown** - Requires further investigation:
- Google Cloud credentials expiration?
- Network connectivity to Google Cloud asia-south1?
- TLS certificate validation issues in Flink Docker environment?
- Rate limiting or quota exhaustion?

### Status
🔴 **UNRESOLVED** - FHIR enrichment currently non-functional

---

## Current State Summary

### What's Working ✅
1. **Neo4j Network Connectivity**: Containers can communicate
2. **Neo4j Encryption**: Driver compatible with Neo4j 5.x
3. **Neo4j Client Initialization**: "verified successfully" in logs
4. **Module 1**: Still processing events correctly
5. **Module 2 Pipeline**: Running without crashes

### What's Broken 🔴
1. **FHIR API Connectivity**: TLS connection failures to Google Healthcare API
2. **Async Timeout**: Code change to 5000ms not reflected in deployed JAR
3. **Neo4j Queries**: Timing out after 30 seconds (connection pool issues?)
4. **Enrichment Output**: All null values for patient context

### Enriched Output Analysis
```json
{
  "id": "evt-rohan-final-test",
  "patient_context": {
    "activeConditions": null,          // ❌ Should have Prediabetes, Hypertension
    "demographics": null,               // ❌ Should have age:42, gender:male
    "currentMedications": null,         // ❌ Should have Telmisartan
    "care_team": null,                  // ❌ Should have Dr. Priya Rao from Neo4j
    "chronic_conditions": null          // ❌ Should have conditions from FHIR
  },
  "risk_indicators": {
    "hasDiabetes": false                // ❌ Should be true
  }
}
```

---

## Next Steps (Recommended Priority)

### Immediate (P0)
1. **Verify JAR Deployment**:
   ```bash
   # Check JAR timestamp in container
   docker exec flink-jobmanager-2.1 ls -lh /opt/flink/flink-ehr-intelligence-1.0.0.jar

   # Verify timeout value in deployed JAR
   docker exec flink-jobmanager-2.1 jar tf /opt/flink/flink-ehr-intelligence-1.0.0.jar | grep Module2_ContextAssembly
   ```

2. **Fix FHIR TLS Issues**:
   - Check Google Cloud credentials: `/Users/apoorvabk/Downloads/cardiofit/backend/services/patient-service/credentials/google-credentials.json`
   - Verify credentials are mounted in Flink containers
   - Test FHIR API from Flink container:
     ```bash
     docker exec flink-taskmanager-2-2.1 curl -v https://healthcare.googleapis.com/v1/projects/cardiofit-905a8/locations/asia-south1/datasets/clinical-synthesis-hub/fhirStores/fhir-store/fhir/Patient/PAT-ROHAN-001
     ```

3. **Restart Flink Cluster**:
   - Full restart to clear all cached classes
   - Redeploy both modules with fresh JARs

### Short-term (P1)
1. **Neo4j Connection Pool Tuning**:
   - Current timeout: 30 seconds (too long)
   - Recommended: 2-5 seconds for async context
   - Update `Neo4jGraphClient.java` connection acquisition timeout

2. **Add Circuit Breaker Logging**:
   - Log when FHIR circuit breaker opens
   - Track failure rates and recovery

3. **Monitoring Dashboard**:
   - Track async enrichment success/timeout rates
   - Monitor FHHIR API latency percentiles
   - Alert on Neo4j connection failures

### Medium-term (P2)
1. **Graceful Degradation**:
   - Allow partial enrichment (FHIR-only or Neo4j-only)
   - Return best-effort data instead of all nulls
   - Flag enrichment completeness in output

2. **Retry Logic**:
   - Implement exponential backoff for transient failures
   - Separate retry strategies for FHIR vs Neo4j

3. **Caching Strategy**:
   - FHIR client has 5-minute TTL cache (good)
   - Add Neo4j query result caching
   - Consider distributed cache (Redis) for cross-TaskManager sharing

---

## Files Modified

1. **Neo4jGraphClient.java** ([/backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/clients/Neo4jGraphClient.java](/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/clients/Neo4jGraphClient.java:76))
   - Line 76: `.withEncryption()` → `.withoutEncryption()`

2. **Module2_ContextAssembly.java** ([/backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module2_ContextAssembly.java](/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module2_ContextAssembly.java:85))
   - Line 85: `2000,` → `5000,` (async timeout)

---

## Test Commands for Verification

### Check Neo4j Connectivity
```bash
# From Flink container
docker exec flink-taskmanager-2-2.1 nc -zv neo4j 7687

# Manual Cypher query
docker exec neo4j cypher-shell -u neo4j -p CardioFit2024! \
  "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'}) RETURN p"
```

### Check FHIR API Access
```bash
# Test with gcloud credentials
docker exec flink-taskmanager-2-2.1 cat /etc/resolv.conf
docker exec flink-taskmanager-2-2.1 curl -I https://healthcare.googleapis.com
```

### Monitor Real-time Enrichment
```bash
# Tail Flink logs for enrichment activity
docker logs flink-taskmanager-2-2.1 --follow | grep -E "(enriching|timeout|Neo4j|FHIR)"
```

---

## Conclusion

**Answer to "Is it connected to Neo4j and fetch data?"**

**Before fixes**: ❌ NO
- Network isolation prevented any connection
- Encryption mismatch caused auth failures

**After fixes**: ⚠️ PARTIALLY
- ✅ Network connectivity established
- ✅ Neo4j driver initialized successfully
- ❌ Actual queries timing out
- ❌ FHIR API also failing with TLS errors
- ❌ Enrichment output still contains all null values

**Root cause of current state**:
The async enrichment timeout change to 5000ms wasn't properly deployed, AND the FHIR API has developed TLS connection issues that prevent any enrichment from succeeding.

**Recommendation**:
1. Fix FHIR credentials/TLS issues first (blocking both FHIR and Neo4j enrichment testing)
2. Fully restart Flink cluster with clean JAR deployment
3. Re-test with proper timeout configuration

---

**Report Generated**: 2025-10-10T16:43:00+05:30
**Investigator**: Claude Code
**Status**: Requires additional troubleshooting for production readiness
