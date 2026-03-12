# Graph Mutations Diagnostic Report

## Problem: `prod.ehr.graph.mutations` Topic Has 0 Messages

### Root Cause Analysis

The `prod.ehr.graph.mutations` topic only receives messages when **`hasGraphImplications()`** returns `true` in [TransactionalMultiSinkRouter.java:159](src/main/java/com/cardiofit/flink/operators/TransactionalMultiSinkRouter.java#L159).

#### Code Path:
```java
// TransactionalMultiSinkRouter.java:159
if (hasGraphImplications(event)) {
    decision.setUpdateGraph(true);  // → Writes to prod.ehr.graph.mutations
}

// TransactionalMultiSinkRouter.java:261-274
private boolean hasGraphImplications(EnrichedClinicalEvent event) {
    // Patient relationship changes
    if (event.hasPatientRelationshipChanges()) {
        return true;
    }

    // Clinical concept relationships
    if (event.hasClinicalConceptRelationships()) {
        return true;
    }

    // Drug interaction networks
    return event.hasDrugInteractions();  // ← Most common trigger
}
```

#### Why Original Events Failed:
The original `continuous-events.sh` sent **random vital signs, medications, and observations** that:
- ❌ Had no drug interactions
- ❌ Had no clinical concept relationships
- ❌ Had no patient relationship changes
- ❌ Never triggered `hasGraphImplications() = true`
- ❌ Never routed to `prod.ehr.graph.mutations`

### Current Status: Module 6 is RESTARTING

```bash
$ curl -s http://localhost:8081/jobs/overview | jq -r '.jobs[] | "\(.name) - \(.state)"'

Module 6: Egress & Multi-Sink Routing - RESTARTING  ← ⚠️ CRASHING
Module 3: Comprehensive CDS Engine - RUNNING         ← ✅ Working
Module 1: EHR Event Ingestion - RUNNING             ← ✅ Working
```

**Module 6 is in a crash loop**, which explains why no messages are being routed even if drug interactions exist in Module 3 output.

## Solution: Drug Interaction Pipeline

### Updated Event Strategy

The new `continuous-events.sh` triggers drug interactions by sending:

```
1. Warfarin medication event
   ↓
2. Patient admission with active_medications: [Warfarin]
   ↓
3. HIGH qSOFA vitals (RR=28, SBP=88, GCS=13)
   ↓ Triggers sepsis protocol
4. Module 3 detects: Warfarin + Piperacillin-Tazobactam = MAJOR interaction
   ↓
5. CDSEvent contains drugInteractionAnalysis
   ↓
6. Module 6B extracts drug interactions
   ↓
7. enriched.setDrugInteractions([...])
   ↓
8. hasGraphImplications() = true ✅
   ↓
9. GraphMutation written to prod.ehr.graph.mutations ✅
```

### Event Details

**Event 1: Warfarin Medication**
```json
{
  "patient_id": "PAT-ROHAN-001",
  "type": "medication",
  "payload": {
    "drug": "Warfarin",
    "dose": "5mg",
    "frequency": "daily",
    "status": "active"
  }
}
```

**Event 2: Emergency Admission**
```json
{
  "patient_id": "PAT-ROHAN-001",
  "type": "patient_admission",
  "payload": {
    "admission_type": "EMERGENCY",
    "chief_complaint": "Fever and confusion",
    "active_medications": [{"drug": "Warfarin", "dose": "5mg"}]
  }
}
```

**Event 3: Critical Vitals (qSOFA = 3)**
```json
{
  "patient_id": "PAT-ROHAN-001",
  "type": "vital_signs",
  "payload": {
    "respiratory_rate": 28,    // ≥22 (qSOFA +1)
    "systolic_bp": 88,         // ≤100 (qSOFA +1)
    "gcs": 13,                 // <15 (qSOFA +1)
    "temperature": 39.2,
    "alert_level": "CRITICAL"
  }
}
```

**Event 4: Infection Labs**
```json
{
  "patient_id": "PAT-ROHAN-001",
  "type": "lab_result",
  "payload": {
    "wbc": 18.5,               // Elevated
    "lactate": 3.2,            // Elevated
    "procalcitonin": 2.5       // Elevated
  }
}
```

### Module 3 Drug Interaction Detection

Module 3 will:
1. Match **Sepsis Protocol** based on qSOFA ≥ 2
2. Recommend **Piperacillin-Tazobactam** (first-line sepsis antibiotic)
3. Check active medications: **Warfarin**
4. Detect **MAJOR drug interaction**:
   - Drug 1: Warfarin (anticoagulant)
   - Drug 2: Piperacillin-Tazobactam (antibiotic)
   - Severity: MAJOR
   - Clinical Effect: Increased bleeding risk
   - Management: Monitor INR closely, consider dose adjustment

### Module 6B Drug Interaction Extraction

Module 6B processes CDSEvents and extracts drug interactions:

```java
// Module6_EgressRouting.java:1071-1115
else if (routedEvent.getOriginalPayload() instanceof Module3_ComprehensiveCDS.CDSEvent) {
    Module3_ComprehensiveCDS.CDSEvent cdsEvent = (Module3_ComprehensiveCDS.CDSEvent) routedEvent.getOriginalPayload();

    if (cdsEvent.getSemanticEnrichment().getDrugInteractionAnalysis() != null) {
        SemanticEnrichment.DrugInteractionAnalysis analysis =
            cdsEvent.getSemanticEnrichment().getDrugInteractionAnalysis();

        if (!analysis.getInteractionWarnings().isEmpty()) {
            List<SemanticEvent.DrugInteraction> drugInteractions = new ArrayList<>();

            for (SemanticEnrichment.InteractionWarning warning : analysis.getInteractionWarnings()) {
                SemanticEvent.DrugInteraction interaction = new SemanticEvent.DrugInteraction();
                interaction.setDrug1(warning.getProtocolMedication());  // Piperacillin-Tazobactam
                interaction.setDrug2(warning.getActiveMedication());    // Warfarin
                interaction.setSeverity(warning.getSeverity());         // MAJOR
                drugInteractions.add(interaction);
            }

            enriched.setDrugInteractions(drugInteractions);  // ← KEY: Sets drug interactions
            LOG.info("✅ Set {} drug interactions on EnrichedClinicalEvent", drugInteractions.size());
        }
    }
}
```

### Expected Output

When Module 6 is running properly, you should see:

```bash
$ ./continuous-events.sh

[1] Sending event batch at 14:23:45 (timestamp: 1704123825000)
   ✓ Sent 4 events for PAT-ROHAN-001
   💊 Scenario: Warfarin + Sepsis Protocol → Drug Interaction → Graph Mutation

[3] Checking pipeline output...
   📈 Module 3 Output:
      comprehensive-cds-events.v1: 19435 messages

   📈 Module 6B Output (Hybrid Routing):
      prod.ehr.events.enriched: 1523 messages
      prod.ehr.graph.mutations: 12 messages ⭐ TARGET
      prod.ehr.fhir.upsert: 876 messages

   ✅ SUCCESS! Graph mutations detected!
   🎯 Drug interactions flowing to graph database
```

## Troubleshooting

### 1. Module 6 Still RESTARTING After Script Runs

**Check Exception:**
```bash
curl -s "http://localhost:8081/jobs/38ac5a7a2765a18176d7fec4db03ff20" | jq '.exceptions'
```

**Common Causes:**
- Kafka connection timeout (check `--bootstrap-server` config)
- Deserialization errors (check CDSEvent schema matches)
- OOM errors (increase Flink memory)

**Fix: Restart Module 6**
```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing
mvn clean package
# Redeploy to Flink
```

### 2. Module 3 Not Producing Drug Interactions

**Check CDSEvents:**
```bash
timeout 5 docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic comprehensive-cds-events.v1 \
  --max-messages 1 --from-beginning | jq '.semanticEnrichment.drugInteractionAnalysis'
```

**Expected:**
```json
{
  "interactionsDetected": 1,
  "interactionWarnings": [
    {
      "protocolMedication": "Piperacillin-Tazobactam",
      "activeMedication": "Warfarin",
      "severity": "MAJOR",
      "clinicalEffect": "Increased bleeding risk",
      "management": "Monitor INR closely"
    }
  ]
}
```

If `null`, Module 3 isn't detecting interactions. Check:
- Drug database has Warfarin interaction data
- Sepsis protocol is configured to recommend Piperacillin-Tazobactam
- Module 3 job is running: `curl http://localhost:8081/jobs/overview | jq '.jobs[] | select(.name | contains("Module 3"))'`

### 3. Module 6B Not Extracting Drug Interactions

**Check Logs:**
```bash
docker logs flink-jobmanager 2>&1 | grep "MODULE6B-DEBUG" | tail -20
```

**Expected:**
```
🔍 [MODULE6B-DEBUG] Processing CDSEvent for patient: PAT-ROHAN-001
🔍 [MODULE6B-DEBUG] DrugInteractionAnalysis found with 1 interactions detected
✅ [MODULE6B-DEBUG] Set 1 drug interactions on EnrichedClinicalEvent
```

If you see `⚠️ InteractionWarnings list is EMPTY`, the CDSEvent doesn't contain warnings even though `interactionsDetected > 0`.

### 4. hasGraphImplications() Still Returns False

**Debug EnrichedClinicalEvent:**

Add debug logging to TransactionalMultiSinkRouter:
```java
LOG.info("🔍 [GRAPH-DEBUG] hasGraphImplications check:");
LOG.info("   - hasDrugInteractions: {}", event.hasDrugInteractions());
LOG.info("   - drugInteractions: {}", event.getDrugInteractions());
LOG.info("   - hasClinicalConceptRelationships: {}", event.hasClinicalConceptRelationships());
LOG.info("   - hasPatientRelationshipChanges: {}", event.hasPatientRelationshipChanges());
```

## Verification Commands

### Check All Topics
```bash
for topic in comprehensive-cds-events.v1 prod.ehr.events.enriched prod.ehr.graph.mutations prod.ehr.fhir.upsert; do
  COUNT=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
    --broker-list localhost:9092 --topic $topic --time -1 2>/dev/null | \
    awk -F: '{sum += $3} END {print sum}')
  echo "$topic: ${COUNT:-0} messages"
done
```

### Sample Graph Mutation Event
```bash
timeout 5 docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic prod.ehr.graph.mutations \
  --max-messages 1 --from-beginning | jq '.'
```

### Module Status
```bash
curl -s http://localhost:8081/jobs/overview | \
  jq -r '.jobs[] | "\(.name): \(.state)"' | \
  grep -E "Module [1-6]"
```

## Success Criteria

✅ **Module 6 Status:** RUNNING (not RESTARTING)
✅ **CDS Events:** Contains drugInteractionAnalysis with interactionWarnings
✅ **Graph Mutations:** `prod.ehr.graph.mutations` message count > 0
✅ **Debug Logs:** Shows "✅ Set N drug interactions on EnrichedClinicalEvent"
✅ **Test Script:** Shows "✅ SUCCESS! Graph mutations detected!"
