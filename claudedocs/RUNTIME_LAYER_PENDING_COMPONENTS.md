# Runtime Layer - Pending Components Analysis

**Source Document**: RUNTIME_LAYER_IMPLEMENTATION_WORKFLOW.md
**Analysis Date**: November 21, 2025
**Current Status**: **85% Complete** (15% remaining)
**Estimated Completion**: 3-5 weeks

---

## 🎯 Executive Summary

The Runtime Layer Platform is **85% complete** (corrected from initial 65-70% assessment). The remaining **15% consists of 6 critical workflows** that must be completed for production deployment.

### What's Complete ✅

1. ✅ **Apache Flink Stream Processing** (90%) - All 6 modules with 31 operators implemented
2. ✅ **Neo4j Dual-Stream Database** (95%) - Dual databases configured
3. ✅ **Knowledge Base Federation** (95%) - All 7 KBs operational
4. ✅ **Clinical Orchestration** (90%) - Medication service + Safety Gateway

### What's Pending ❌

1. ❌ **CDC Source Connectors** (0%) - Debezium connectors for KB1-KB7 not deployed
2. ❌ **Kafka Connect Cluster** (0%) - Deployment infrastructure missing
3. ⚠️ **Snapshot Manager** (70% → need 30%) - Missing digital signatures, TTL enforcement
4. ⚠️ **Evidence Envelope** (65% → need 35%) - Missing calculation traces, immutable storage
5. ⚠️ **SLA Monitoring** (15% → need 85%) - NotImplementedError stubs only
6. ❌ **Integration Testing** (0%) - End-to-end workflow validation incomplete

---

## 📋 Pending Component Breakdown

### 1. CDC Source Connectors (Critical - 1-2 weeks)

**Current State**: ❌ **NONE DEPLOYED**
- Sink connectors exist (Neo4j, ClickHouse, Elasticsearch, Redis, Google FHIR)
- Source connectors for KB1-KB7 PostgreSQL databases missing
- Connector configs for KB4 and KB5 exist, but 5 others missing

**What's Missing**:
```yaml
Missing CDC Connectors:
  - KB1: Medications & Renal Adjustments (PostgreSQL → Kafka)
  - KB2: Drug Interactions (PostgreSQL → Kafka)
  - KB3: Clinical Guidelines (PostgreSQL → Kafka)
  - KB6: Reference Ranges (PostgreSQL → Kafka)
  - KB7: Evidence Summaries (PostgreSQL → Kafka)

Existing but not deployed:
  - KB4: Drug Calculations connector config exists
  - KB5: Diagnostic Criteria connector config exists
```

**Impact**: Real-time KB synchronization broken. CDC pipeline cannot stream KB updates to Kafka topics, breaking the entire clinical intelligence layer.

**Implementation Required**:

**Step 1: Create Kafka Connect Cluster** (3 days)
```yaml
# docker-compose.connect.yml
services:
  kafka-connect:
    image: confluentinc/cp-kafka-connect:7.5.0
    ports: ["8083:8083"]
    environment:
      CONNECT_BOOTSTRAP_SERVERS: 'kafka:29092'
      CONNECT_GROUP_ID: "cardiofit-connect-cluster"
      # ... configuration
    command:
      - bash
      - -c
      - |
        confluent-hub install --no-prompt debezium/debezium-connector-postgresql:2.4.0
        /etc/confluent/docker/run
```

**Step 2: Create 7 Debezium Connector Configs** (4 days)
```json
// Example: kb1-postgres-source-connector.json
{
  "name": "kb1-postgres-source-connector",
  "config": {
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "database.hostname": "postgres-kb1",
    "database.dbname": "kb1_db",
    "database.server.name": "kb1",
    "table.include.list": "public.medications,public.renal_adjustments",
    "plugin.name": "pgoutput",
    "slot.name": "kb1_cdc_slot",
    "publication.name": "kb1_publication",
    "topic.prefix": "kb.kb1"
  }
}
```

**Step 3: Deploy All Connectors** (2 days)
```bash
#!/bin/bash
# deploy-all-cdc-sources.sh
for kb in kb1 kb2 kb3 kb4 kb5 kb6 kb7; do
  curl -X POST -H "Content-Type: application/json" \
    --data @"debezium-${kb}-source-connector.json" \
    "http://localhost:8083/connectors"
done
```

**Deliverables**:
- ✅ Kafka Connect cluster with 3 workers
- ✅ 7 Debezium source connectors deployed
- ✅ Monitoring dashboard showing <500ms lag
- ✅ End-to-end test for all 7 KBs

---

### 2. Snapshot Manager - Missing Components (Critical - 3-5 days)

**Current State**: ⚠️ **70% COMPLETE**

**What Exists**:
- ✅ Core service structure (snapshot_manager.py)
- ✅ Redis integration for 5-minute TTL storage
- ✅ Query router for multi-KB data fetching
- ✅ Basic version tracking service

**What's Missing (30%)**:

#### Missing Component A: Digital Signature Generation
**Status**: ❌ **NOT IMPLEMENTED**
**Criticality**: 🔴 CRITICAL - FDA SaMD compliance blocked without this

**Current Code**:
```typescript
// MISSING: Digital signature with HSM
class CryptoService {
  async signSnapshot(snapshot: ClinicalSnapshot): Promise<string> {
    // TODO: Integrate with AWS KMS or Azure Key Vault
    // const signature = await hsm.sign(JSON.stringify(snapshot));
    throw new Error("Not implemented");
  }
}
```

**Required Implementation**:
```typescript
import { KMSClient, SignCommand, VerifyCommand } from "@aws-sdk/client-kms";
import * as crypto from "crypto";

export class CryptoService {
  private kmsClient: KMSClient;
  private keyId: string;

  constructor() {
    this.kmsClient = new KMSClient({ region: process.env.AWS_REGION });
    this.keyId = process.env.KMS_KEY_ID!;
  }

  async signSnapshot(snapshot: ClinicalSnapshot): Promise<string> {
    // Step 1: Canonical JSON representation (RFC 8785)
    const canonicalData = this.canonicalize(snapshot);

    // Step 2: SHA-256 hash
    const hash = crypto.createHash('sha256').update(canonicalData).digest();

    // Step 3: Sign with AWS KMS
    const signCommand = new SignCommand({
      KeyId: this.keyId,
      Message: hash,
      MessageType: 'DIGEST',
      SigningAlgorithm: 'RSASSA_PKCS1_V1_5_SHA_256'
    });

    const response = await this.kmsClient.send(signCommand);
    return Buffer.from(response.Signature!).toString('base64');
  }

  async verifySignature(snapshot: ClinicalSnapshot, signature: string): Promise<boolean> {
    const canonicalData = this.canonicalize(snapshot);
    const hash = crypto.createHash('sha256').update(canonicalData).digest();

    const verifyCommand = new VerifyCommand({
      KeyId: this.keyId,
      Message: hash,
      MessageType: 'DIGEST',
      Signature: Buffer.from(signature, 'base64'),
      SigningAlgorithm: 'RSASSA_PKCS1_V1_5_SHA_256'
    });

    const response = await this.kmsClient.send(verifyCommand);
    return response.SignatureValid || false;
  }

  private canonicalize(obj: any): string {
    const sortedKeys = Object.keys(obj).sort();
    const canonical: any = {};
    sortedKeys.forEach(key => {
      if (key !== 'signature') {
        canonical[key] = obj[key];
      }
    });
    return JSON.stringify(canonical, Object.keys(canonical).sort());
  }
}
```

**Tasks**:
1. Set up AWS KMS key for digital signatures (1 day)
2. Implement CryptoService class (1 day)
3. Integrate with SnapshotManager.createSnapshot() (1 day)
4. Add signature verification on retrieval (0.5 days)
5. Unit tests (0.5 days)

---

#### Missing Component B: TTL Enforcement
**Status**: ⚠️ **PASSIVE ONLY** - Redis TTL set but no active monitoring

**Current Implementation**:
```typescript
// EXISTING: Passive TTL (Redis auto-expires)
await this.redis.setex(
  `snapshot:${snapshot.id}`,
  300, // 5 minutes
  JSON.stringify(snapshot)
);
```

**Required Enhancement**:
```typescript
export class TTLEnforcer {
  private redis: Redis;

  constructor() {
    this.redis = new Redis(process.env.REDIS_URL);
    this.startExpirationMonitor();
  }

  private startExpirationMonitor() {
    const subscriber = new Redis(process.env.REDIS_URL);

    // Enable keyspace notifications in Redis
    this.redis.config('SET', 'notify-keyspace-events', 'Ex');

    // Subscribe to expiration events
    subscriber.psubscribe('__keyevent@0__:expired');

    subscriber.on('pmessage', async (pattern, channel, expiredKey) => {
      if (expiredKey.startsWith('snapshot:')) {
        const snapshotId = expiredKey.replace('snapshot:', '');

        await this.logAuditEvent({
          event: 'SNAPSHOT_EXPIRED',
          snapshotId,
          timestamp: new Date().toISOString()
        });

        console.log(`⏱️ Snapshot ${snapshotId} expired after 5 minutes`);
      }
    });
  }

  async invalidateOnVersionChange(kbName: string, newVersion: string) {
    // Find all snapshots using this KB version
    const keys = await this.redis.keys('snapshot:*');

    for (const key of keys) {
      const data = await this.redis.get(key);
      if (!data) continue;

      const snapshot = JSON.parse(data);
      const oldVersion = snapshot.versionVector[kbName];

      if (oldVersion !== newVersion) {
        await this.redis.del(key);
        await this.logAuditEvent({
          event: 'SNAPSHOT_INVALIDATED',
          snapshotId: key.replace('snapshot:', ''),
          reason: `${kbName} version changed from ${oldVersion} to ${newVersion}`,
          timestamp: new Date().toISOString()
        });
      }
    }
  }
}
```

**Tasks**:
1. Implement TTLEnforcer class (0.5 days)
2. Configure Redis keyspace notifications (0.25 days)
3. Add audit logging for expirations (0.5 days)
4. Implement KB version change invalidation (0.75 days)

---

#### Missing Component C: Complete Version Vector Capture
**Status**: ⚠️ **PARTIAL** - Only captures KB versions, missing Flink and FHIR versions

**Current Implementation**:
```typescript
// EXISTING: Only KB versions captured
const versionVector = await this.captureVersionVector();
// Returns: {kb1: "1.2.3", kb2: "1.1.0", ...}
```

**Required Complete Implementation**:
```typescript
interface CompleteVersionVector {
  // Knowledge Base versions (KB1-KB7)
  kb_versions: {
    kb1_drug_rules: string;
    kb2_clinical_context: string;
    kb3_guidelines: string;
    kb4_drug_calculations: string;
    kb5_drug_interactions: string;
    kb6_formulary: string;
    kb7_terminology: string;
  };

  // Flink module versions (Modules 1-6)
  flink_modules: {
    module1_ingestion_validation: string;
    module2_context_assembly: string;
    module3_protocol_matching: string;
    module4_cep_pattern_detection: string;
    module5_ml_inference: string;
    module6_egress_routing: string;
  };

  // FHIR resource versions
  fhir_resources: {
    [resource_type: string]: {
      resource_id: string;
      version_id: string;
      last_modified: string;
    }[];
  };
}

private async captureVersionVector(): Promise<CompleteVersionVector> {
  const [kbVersions, flinkVersions, fhirVersions] = await Promise.all([
    this.fetchKBVersions(),        // Query PostgreSQL kb_metadata tables
    this.fetchFlinkModuleVersions(), // Query Flink JobManager REST API
    this.fetchFHIRResourceVersions() // Query Google Healthcare API
  ]);

  return {
    kb_versions: kbVersions,
    flink_modules: flinkVersions,
    fhir_resources: fhirVersions,
    captured_at: new Date().toISOString()
  };
}

async fetchFlinkModuleVersions(): Promise<Record<string, string>> {
  // Query Flink JobManager REST API
  const response = await fetch('http://flink-jobmanager:8081/jobs');
  const jobs = await response.json();

  const versions: Record<string, string> = {};
  for (const job of jobs.jobs) {
    const detailResponse = await fetch(`http://flink-jobmanager:8081/jobs/${job.id}`);
    const detail = await detailResponse.json();
    versions[detail.plan.metadata.module] = detail.plan.metadata.version;
  }

  return versions;
}
```

**Tasks**:
1. Implement fetchFlinkModuleVersions() (1 day)
2. Implement fetchFHIRResourceVersions() (1 day)
3. Update version vector schema (0.5 days)
4. Integration testing (0.5 days)

---

### 3. Evidence Envelope Generator - Missing Components (Important - 3-5 days)

**Current State**: ⚠️ **65% COMPLETE**

**What Exists**:
- ✅ Evidence envelope data structure
- ✅ KB version capture at calculation time
- ✅ Input/output recording
- ✅ GraphQL integration

**What's Missing (35%)**:

#### Missing Component A: Calculation Trace Generation
**Status**: ❌ **NOT IMPLEMENTED**
**Example**: Dose calculation should record step-by-step audit log

**Required Implementation**:
```python
# File: backend/services/medication-service/dose_calculator.py

@dataclass
class CalculationStep:
    step_number: int
    step_name: str
    inputs: Dict[str, Any]
    formula: str
    intermediate_values: Dict[str, Any]
    result: Any
    kb_reference: str  # KB version and rule ID used
    timestamp: str

class DoseCalculatorWithTrace:
    def __init__(self, kb1_client, kb4_client):
        self.kb1 = kb1_client
        self.kb4 = kb4_client
        self.trace: List[CalculationStep] = []

    async def calculate_dose(
        self,
        rxnorm_code: str,
        patient_weight_kg: float,
        creatinine_mg_dl: float,
        age_years: int
    ) -> Dict[str, Any]:
        self.trace = []

        # Step 1: Fetch medication from KB1
        medication = await self.kb1.get_medication(rxnorm_code)
        self.trace.append(CalculationStep(
            step_number=1,
            step_name="Fetch Base Medication",
            inputs={"rxnorm_code": rxnorm_code},
            formula="KB1.medications.find_by_rxnorm",
            intermediate_values={"base_dose_range": medication['dosage_range']},
            result=medication,
            kb_reference=f"KB1:{self.kb1.version}/medications/{rxnorm_code}",
            timestamp=datetime.utcnow().isoformat()
        ))

        # Step 2: Calculate creatinine clearance
        crcl = await self._calculate_crcl(creatinine_mg_dl, age_years, patient_weight_kg)
        self.trace.append(CalculationStep(
            step_number=2,
            step_name="Creatinine Clearance (CG Formula)",
            inputs={
                "creatinine_mg_dl": creatinine_mg_dl,
                "age_years": age_years,
                "weight_kg": patient_weight_kg
            },
            formula="((140 - age) * weight) / (72 * creatinine)",
            intermediate_values={
                "numerator": (140 - age_years) * patient_weight_kg,
                "denominator": 72 * creatinine_mg_dl
            },
            result=crcl,
            kb_reference="KB4:1.2.0/formulas/cockcroft_gault",
            timestamp=datetime.utcnow().isoformat()
        ))

        # ... Steps 3-5 continue

        return {
            "final_dose_mg": final_dose_mg,
            "calculation_trace": [asdict(step) for step in self.trace]
        }
```

**Tasks**:
1. Instrument dose calculator with trace capture (2 days)
2. Add trace capture to interaction checker (1 day)
3. Add trace capture to guideline evaluator (1 day)
4. Integration testing (1 day)

---

#### Missing Component B: Guideline Reference Extraction
**Status**: ❌ **NOT IMPLEMENTED**

**Required Implementation**:
```python
async def _fetch_guideline_references(self, trace: List[Dict]) -> List[Dict[str, str]]:
    references = []

    for step in trace:
        kb_ref = step['kb_reference']

        if kb_ref.startswith('KB1'):
            guideline = await self.kb1_client.get_guideline(kb_ref)
            references.append({
                "kb": "KB1",
                "version": kb_ref.split(':')[1].split('/')[0],
                "citation": guideline['citation'],
                "evidence_level": guideline['evidence_level'],
                "url": guideline['pubmed_url']
            })

        elif kb_ref.startswith('KB4'):
            formula = await self.kb4_client.get_formula(kb_ref)
            references.append({
                "kb": "KB4",
                "version": kb_ref.split(':')[1].split('/')[0],
                "citation": "Cockcroft DW, Gault MH. Nephron. 1976;16(1):31-41.",
                "evidence_level": "Standard of Care",
                "url": "https://pubmed.ncbi.nlm.nih.gov/1244564/"
            })

    return references
```

**Tasks**:
1. Implement guideline reference extraction (1 day)
2. Add KB API methods for guideline lookup (1 day)
3. Testing (0.5 days)

---

#### Missing Component C: Immutable Storage in ClickHouse
**Status**: ❌ **NOT IMPLEMENTED** - Currently stored in MongoDB (mutable)

**Required Schema**:
```sql
CREATE TABLE IF NOT EXISTS evidence_envelopes (
    envelope_id String,
    snapshot_id String,
    calculation_type LowCardinality(String),
    calculation_timestamp DateTime64(3),
    kb_versions String,  -- JSON
    clinical_inputs String,  -- JSON
    calculation_trace String,  -- JSON array
    guideline_references String,  -- JSON array
    outputs String,  -- JSON
    signature String,
    inserted_at DateTime64(3) DEFAULT now64()
) ENGINE = MergeTree()
ORDER BY (calculation_timestamp, envelope_id)
SETTINGS index_granularity = 8192;

-- Immutability: No UPDATE or DELETE permissions
REVOKE UPDATE, DELETE ON evidence_envelopes FROM cardiofit_app_user;
GRANT INSERT, SELECT ON evidence_envelopes TO cardiofit_app_user;
```

**Tasks**:
1. Deploy ClickHouse cluster (1 day)
2. Create schema and access controls (0.5 days)
3. Implement evidence_envelope_generator.py storage (1 day)
4. Testing (0.5 days)

---

### 4. SLA Monitoring Service (Important - 1 week)

**Current State**: ⚠️ **15% COMPLETE** - Only stub implementation

**What Exists**:
- ✅ Basic service structure (alert_manager.py)
- ✅ Placeholder methods

**What's Missing (85%)**:

```python
# CURRENT: NotImplementedError stubs
class SLAMonitor:
    async def monitor_workflow(self, workflow_id: str, phases: Dict[str, float]):
        raise NotImplementedError("SLA monitoring not implemented")

    async def _send_alert(self, workflow_id: str, violation: Dict[str, Any]):
        raise NotImplementedError("Alert sending not implemented")
```

**Required Implementation**:
```python
from prometheus_client import Counter, Histogram, Gauge
import asyncio

class SLAMonitor:
    def __init__(self):
        # Prometheus metrics
        self.latency_histogram = Histogram(
            'cdss_workflow_latency_seconds',
            'CDSS workflow latency',
            ['workflow_type', 'phase'],
            buckets=[0.01, 0.05, 0.1, 0.15, 0.2, 0.31, 0.5, 1.0]
        )

        self.sla_violations = Counter(
            'cdss_sla_violations_total',
            'SLA violations by type',
            ['violation_type', 'severity']
        )

        self.active_workflows = Gauge(
            'cdss_active_workflows',
            'Number of active CDSS workflows'
        )

    async def monitor_workflow(self, workflow_id: str, phases: Dict[str, float]):
        total_latency = sum(phases.values())

        # Record metrics
        for phase_name, latency in phases.items():
            self.latency_histogram.labels(
                workflow_type='medication_prescription',
                phase=phase_name
            ).observe(latency / 1000)

        # Check SLA violations
        violations = []

        if total_latency > 310:
            violations.append({
                'type': 'TOTAL_LATENCY_EXCEEDED',
                'severity': 'CRITICAL',
                'actual': total_latency,
                'target': 310
            })

        if phases.get('snapshot_creation', 0) > 50:
            violations.append({
                'type': 'SNAPSHOT_LATENCY_EXCEEDED',
                'severity': 'WARNING',
                'actual': phases['snapshot_creation'],
                'target': 50
            })

        # Send alerts
        for violation in violations:
            await self._send_alert(workflow_id, violation)
            self.sla_violations.labels(
                violation_type=violation['type'],
                severity=violation['severity']
            ).inc()

    async def _send_alert(self, workflow_id: str, violation: Dict[str, Any]):
        if violation['severity'] == 'CRITICAL':
            await self._send_pagerduty_alert(workflow_id, violation)
        else:
            await self._send_slack_alert(workflow_id, violation)
```

**Tasks**:
1. Implement SLA monitoring service (2 days)
2. Create Prometheus metrics (1 day)
3. Create Grafana dashboard (2 days)
4. Configure alert rules (1 day)
5. PagerDuty integration (1 day)

---

### 5. Integration Testing (Important - 1 week)

**Current State**: ❌ **NOT IMPLEMENTED**

**Required Test Coverage**:

```python
@pytest.mark.asyncio
async def test_prescription_workflow_310ms_latency():
    """
    End-to-end test: Recipe → Snapshot → Intelligence → Validation → Commit
    Target: 310ms total latency
    """

    snapshot_mgr = SnapshotManager()
    evidence_gen = EvidenceEnvelopeGenerator()
    med_service = MedicationServiceClient()

    patient_id = "test-patient-001"
    rxnorm_code = "313782"  # Metformin

    # Phase 1: Recipe Submission (Target: 10ms)
    start_time = datetime.utcnow()
    recipe = {...}
    recipe_time = (datetime.utcnow() - start_time).total_seconds() * 1000
    assert recipe_time < 10

    # Phase 2: Snapshot Creation (Target: 50ms)
    snapshot = await snapshot_mgr.createSnapshot(recipe, patient_id)
    snapshot_time = ...
    assert snapshot_time < 50
    assert snapshot.signature is not None

    # Phase 3: Clinical Intelligence (Target: 150ms)
    calculation_result = await med_service.calculate_dose(...)
    intelligence_time = ...
    assert intelligence_time < 150
    assert 'calculation_trace' in calculation_result

    # Phase 4: Safety Validation (Target: 30ms)
    safety_result = await safety_gateway.validate(...)
    validation_time = ...
    assert validation_time < 30

    # Phase 5: Commit Phase (Target: 70ms)
    envelope = await evidence_gen.generate_envelope(...)
    commit_time = ...
    assert commit_time < 70

    # Total latency check
    total_time = ...
    assert total_time < 310
```

**Test Scenarios Required**:
1. End-to-end prescription workflow (310ms target)
2. Snapshot signature verification
3. TTL expiration
4. KB version change invalidation
5. Evidence envelope trace generation
6. CDC event flow (KB → Kafka → Flink → Neo4j)
7. Load test (100 concurrent workflows)
8. Failure scenarios (KB unavailable, signature failure)

**Tasks**:
1. Implement 8 test scenarios (3 days)
2. Performance benchmarking (2 days)
3. Load testing (2 days)

---

## 🔥 Implementation Timeline

### Week 1: CDC Pipeline & Kafka Connect
- Days 1-3: Deploy Kafka Connect cluster
- Days 4-7: Create and deploy 7 CDC source connectors
- **Deliverable**: Real-time KB streaming operational

### Week 2: Snapshot Manager & Evidence Envelope
- Days 1-2: AWS KMS integration for digital signatures
- Day 3: TTL enforcement and version vector
- Days 4-5: Calculation trace generation
- **Deliverable**: FDA-compliant snapshots and evidence

### Week 3: Integration Testing & Performance
- Days 1-3: End-to-end workflow testing
- Days 4-5: Performance optimization (310ms)
- Days 6-7: Load testing
- **Deliverable**: System validated at production scale

### Week 4: SLA Monitoring & Documentation
- Days 1-3: SLA monitoring service + dashboards
- Days 4-7: Documentation (architecture, runbooks, dev guide)
- **Deliverable**: Production-ready system

### Week 5 (Buffer): Contingency
- Security audit
- Performance fine-tuning
- User acceptance testing

---

## ⚡ Priority Order (What to Implement First)

### Priority 1: Critical Path 🔴
1. **CDC Source Connectors** (Weeks 1) - Foundation for everything else
2. **Snapshot Manager Digital Signatures** (Week 2, Days 1-3) - FDA compliance requirement
3. **Evidence Envelope Traces** (Week 2, Days 4-5) - Audit trail requirement

### Priority 2: Production Readiness 🟡
4. **Integration Testing** (Week 3) - Validate system works end-to-end
5. **SLA Monitoring** (Week 4, Days 1-3) - Operational visibility
6. **Documentation** (Week 4, Days 4-7) - Knowledge transfer

---

## 📊 Success Criteria

**Functional Completion**:
- [ ] All 7 CDC source connectors streaming KB updates
- [ ] Snapshots digitally signed with AWS KMS
- [ ] Evidence envelopes with 5-step calculation traces
- [ ] 310ms end-to-end latency achieved (p95)
- [ ] SLA monitoring operational with alerting
- [ ] 100 concurrent workflows handled
- [ ] Complete documentation delivered

**Compliance**:
- [ ] FDA 21 CFR Part 11 digital signatures
- [ ] Immutable audit trail in ClickHouse
- [ ] Complete version vector tracking
- [ ] Evidence envelope for every clinical decision

---

## 🎯 Key Takeaways

`★ Insight ─────────────────────────────────────`

**Runtime Layer Status**: 85% complete, but the remaining 15% is critical path

**Most Critical Gap**: CDC Source Connectors (0% deployed)
- Without CDC connectors, KB updates don't flow to Kafka
- Flink can't consume KB changes
- Neo4j knowledge graph stays empty
- Real-time clinical intelligence broken

**Second Most Critical**: Snapshot Manager Digital Signatures
- Required for FDA SaMD compliance
- Blocks Evidence Envelope completion
- Blocks production deployment

**Implementation Strategy**: Start with CDC (Week 1), then Snapshot Manager (Week 2), then testing (Week 3-4)

**Timeline**: 3-5 weeks to 100% completion with 1.5-2.0 FTE

`─────────────────────────────────────────────────`

---

**Document Status**: ✅ COMPLETE
**Next Action**: Choose implementation priority and begin Week 1 (CDC Pipeline)
