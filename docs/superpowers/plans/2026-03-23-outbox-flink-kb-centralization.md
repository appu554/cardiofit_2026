# Outbox + Flink + KB Threshold Centralization — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace ingestion service's direct Kafka writes with the Global Outbox SDK, wire ingestion events through Flink via a new Module1b canonicalizer, and centralize all hardcoded Flink clinical thresholds by connecting to KB-4/KB-20/KB-1/KB-23 via async HTTP with BroadcastState hot-swap.

**Architecture:** Ingestion service writes to PostgreSQL outbox table (atomic with FHIR Store write). Central Publisher polls outbox and relays to `ingestion.*` Kafka topics. Flink Module1b transforms outbox envelopes into CanonicalEvents for the existing Module2+ pipeline. KB services expose new GET endpoints for clinical thresholds. Flink's new ClinicalThresholdService loads thresholds at startup and refreshes every 5 minutes via Caffeine cache, distributing updates to all operators via BroadcastState.

**Tech Stack:** Go (ingestion service, KB services), Java (Flink), PostgreSQL, Kafka, Outbox SDK (`global-outbox-service-go/pkg/outbox-sdk`), Gin HTTP framework, Flink 2.1.0, Caffeine 3.1.8, Resilience4j 2.1.0

**Spec:** `docs/superpowers/specs/2026-03-23-outbox-flink-kb-centralization-design.md`

---

## File Map

### Ingestion Service (Go) — PR1, PR2

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `services/ingestion-service/internal/config/config.go` | Add `OutboxConfig` struct with SDK settings |
| Modify | `services/ingestion-service/internal/api/server.go` | Replace `kafkaProducer` with `outboxClient`, wire SDK |
| Create | `services/ingestion-service/internal/outbox/publisher.go` | Outbox publisher wrapping SDK calls, topic routing, dual-publish |
| Create | `services/ingestion-service/internal/outbox/publisher_test.go` | Unit tests for outbox publisher |
| Modify | `services/ingestion-service/internal/api/handlers.go` | Replace `publishResult()` to call outbox publisher instead of kafka producer |
| Delete | `services/ingestion-service/internal/kafka/producer.go` | Replaced by outbox publisher |
| Keep | `services/ingestion-service/internal/kafka/router.go` | Topic routing logic reused by outbox publisher |
| Keep | `services/ingestion-service/internal/kafka/envelope.go` | Envelope struct embedded in outbox `event_data` |
| Modify | `services/ingestion-service/internal/kafka/router.go` | Add wearable topic mappings (PR2) |
| Modify | `services/ingestion-service/go.mod` | Add outbox SDK dependency |

### KB-4 Patient Safety (Go) — PR4

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `kb-4-patient-safety/internal/thresholds/vitals.go` | Vital threshold data + handler |
| Create | `kb-4-patient-safety/internal/thresholds/early_warning_scores.go` | NEWS2/MEWS scoring parameters + handler |
| Create | `kb-4-patient-safety/internal/thresholds/vitals_test.go` | Tests |
| Modify | `kb-4-patient-safety/main.go` | Register new routes at lines ~370-465 |

### KB-20 Patient Profile (Go) — PR5

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `kb-20-patient-profile/internal/api/threshold_handlers.go` | Lab threshold endpoint handler |
| Create | `kb-20-patient-profile/internal/api/threshold_handlers_test.go` | Tests |
| Modify | `kb-20-patient-profile/internal/api/routes.go` | Register `/api/v1/thresholds/labs` route |

### KB-1 Drug Rules (Go) — PR6

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `kb-1-drug-rules/internal/api/high_risk_handlers.go` | High-risk categories endpoint handler |
| Create | `kb-1-drug-rules/internal/api/high_risk_handlers_test.go` | Tests |
| Modify | `kb-1-drug-rules/internal/api/server.go` | Register `/v1/high-risk/categories` route |

### KB-23 Decision Cards (Go) — PR7

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `kb-23-decision-cards/internal/api/config_handlers.go` | Risk-scoring config endpoint handler |
| Create | `kb-23-decision-cards/internal/api/config_handlers_test.go` | Tests |
| Modify | `kb-23-decision-cards/internal/api/routes.go` | Register `/api/v1/config/risk-scoring` route |

### Flink Processing (Java) — PR3, PR8

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `flink-processing/src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java` | Reads `ingestion.*` topics, transforms outbox envelope → CanonicalEvent with `sourceSystem="ingestion-service"` |
| Create | `flink-processing/src/main/java/com/cardiofit/flink/models/OutboxEnvelope.java` | Deserialization model for outbox SDK envelope format |
| Create | `flink-processing/src/main/java/com/cardiofit/flink/thresholds/ClinicalThresholdService.java` | Async HTTP to KB services, Caffeine cache, three-tier fallback |
| Create | `flink-processing/src/main/java/com/cardiofit/flink/thresholds/ClinicalThresholdSet.java` | POJO holding all threshold groups (vitals, labs, scores, risk) |
| Create | `flink-processing/src/main/java/com/cardiofit/flink/thresholds/ThresholdAwareProcessFunction.java` | Abstract base for BroadcastState-consuming operators |
| Modify | `flink-processing/src/main/java/com/cardiofit/flink/operators/TransactionalMultiSinkRouterV2_OptionC.java` | Add `sourceSystem=="ingestion-service"` check to skip FHIR write + set criticalAlerts flag for CRITICAL_VALUE |
| Modify | `flink-processing/src/main/java/com/cardiofit/flink/utils/KafkaTopics.java` | Add `ingestion.*` topic enum entries |
| Modify | `flink-processing/pom.xml` | No new deps needed (Caffeine, Resilience4j, AsyncHttpClient already present) |

### Global Outbox Service Config — PR1 prerequisite

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `global-outbox-service-go/internal/config/config.go` | Add `"ingestion-service"` to `supported_services` default list |

---

## Execution Phases

```
Phase 1 (parallel):  Task 1 (PR0) + Task 5 (PR4) + Task 6 (PR5) + Task 7 (PR6) + Task 8 (PR7)
Phase 2 (sequential): Task 2 (PR1) — depends on PR0
Phase 3 (parallel):  Task 3 (PR2) + Task 4 (PR3) — both depend on PR1
Phase 4 (sequential): Task 9 (PR8) — depends on PR3 + PR4-PR7
```

---

## Task 1: Create Kafka Topics (PR0)

**Files:**
- Reference: `docs/superpowers/specs/2026-03-23-outbox-flink-kb-centralization-design.md` (Section 3.3 Topic Architecture)

This task creates the `ingestion.*` topics in the Kafka cluster. The exact mechanism depends on your Kafka setup (Confluent Cloud CLI, admin API, or Terraform). This plan provides the topic specs; adapt the creation method to your environment.

- [ ] **Step 1: Write topic creation script**

Create a script or configuration with these topics:

```bash
# ingestion.labs — 12 partitions, 90-day retention
kafka-topics --create --topic ingestion.labs --partitions 12 --config retention.ms=7776000000

# ingestion.vitals — 8 partitions, 30-day retention
kafka-topics --create --topic ingestion.vitals --partitions 8 --config retention.ms=2592000000

# ingestion.device-data — 8 partitions, 30-day retention
kafka-topics --create --topic ingestion.device-data --partitions 8 --config retention.ms=2592000000

# ingestion.patient-reported — 8 partitions, 30-day retention
kafka-topics --create --topic ingestion.patient-reported --partitions 8 --config retention.ms=2592000000

# ingestion.wearable-aggregates — 4 partitions, 14-day retention
kafka-topics --create --topic ingestion.wearable-aggregates --partitions 4 --config retention.ms=1209600000

# ingestion.cgm-raw — 4 partitions, 7-day compacted
kafka-topics --create --topic ingestion.cgm-raw --partitions 4 --config retention.ms=604800000 --config cleanup.policy=compact

# ingestion.abdm-records — 4 partitions, 180-day retention
kafka-topics --create --topic ingestion.abdm-records --partitions 4 --config retention.ms=15552000000

# ingestion.medications — 8 partitions, 90-day retention
kafka-topics --create --topic ingestion.medications --partitions 8 --config retention.ms=7776000000

# ingestion.observations — 8 partitions, 30-day retention
kafka-topics --create --topic ingestion.observations --partitions 8 --config retention.ms=2592000000

# ingestion.safety-critical — 4 partitions, 90-day retention
kafka-topics --create --topic ingestion.safety-critical --partitions 4 --config retention.ms=7776000000

# kb.clinical-thresholds.changes — 1 partition, 7-day compacted (future use)
kafka-topics --create --topic kb.clinical-thresholds.changes --partitions 1 --config retention.ms=604800000 --config cleanup.policy=compact

# DLQ topics
kafka-topics --create --topic dlq.ingestion.labs.v1 --partitions 4 --config retention.ms=7776000000
kafka-topics --create --topic dlq.ingestion.vitals.v1 --partitions 4 --config retention.ms=7776000000
kafka-topics --create --topic dlq.ingestion.safety-critical.v1 --partitions 4 --config retention.ms=7776000000
```

- [ ] **Step 2: Verify topics created**

Run: `kafka-topics --list | grep ingestion`
Expected: All 10 `ingestion.*` topics listed

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "infra(kafka): create ingestion.* topics with partition counts and retention policies (PR0)"
```

---

## Task 2: Outbox SDK Wiring in Ingestion Service (PR1)

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/outbox/publisher.go`
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/outbox/publisher_test.go`
- Modify: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/config/config.go`
- Modify: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/api/server.go`
- Modify: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/api/handlers.go:299-401`
- Modify: `vaidshala/clinical-runtime-platform/services/ingestion-service/go.mod`
- Modify: `backend/services/global-outbox-service-go/internal/config/config.go:170` (add to supported_services)

### Sub-task 2a: Add OutboxConfig to config.go

- [ ] **Step 1: Write the failing test**

No test for config loading — this is a struct addition. Verify by compilation.

- [ ] **Step 2: Add OutboxConfig struct and env loading**

In `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/config/config.go`:

```go
// Add to Config struct (after Kafka field, line ~48):
type Config struct {
    // ... existing fields ...
    Outbox OutboxConfig
}

// Add new struct:
type OutboxConfig struct {
    Enabled          bool
    DatabaseURL      string // reuses Database.URL if empty
    GRPCAddress      string
    DefaultPriority  int32
}

// In Load() function, add after Kafka config (line ~116):
Outbox: OutboxConfig{
    Enabled:         getEnvAsBool("OUTBOX_ENABLED", false),
    DatabaseURL:     getEnv("OUTBOX_DATABASE_URL", ""),
    GRPCAddress:     getEnv("OUTBOX_GRPC_ADDRESS", "localhost:50052"),
    DefaultPriority: int32(getEnvAsInt("OUTBOX_DEFAULT_PRIORITY", 5)),
},
```

- [ ] **Step 3: Verify compilation**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go build ./...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(ingestion): add OutboxConfig to service configuration"
```

### Sub-task 2b: Create outbox publisher

- [ ] **Step 5: Write the failing test**

Create `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/outbox/publisher_test.go`:

```go
package outbox

import (
    "context"
    "testing"

    "github.com/cardiofit/ingestion-service/internal/canonical"
    "github.com/google/uuid"
    "go.uber.org/zap"
)

func TestEventDataFromObservation(t *testing.T) {
    obs := &canonical.CanonicalObservation{
        PatientID:       uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
        TenantID:        uuid.MustParse("11111111-2222-3333-4444-555555555555"),
        ObservationType: canonical.ObsLabs,
        LOINCCode:       "2160-0",
        Value:           1.4,
        Unit:            "mg/dL",
        QualityScore:    0.92,
        Flags:           []canonical.Flag{},
    }

    data := eventDataFromObservation(obs, "Observation/abc123")
    if data.PatientID != obs.PatientID.String() {
        t.Errorf("expected patient_id %s, got %s", obs.PatientID, data.PatientID)
    }
    if data.FHIRResourceID != "Observation/abc123" {
        t.Errorf("expected fhir_resource_id Observation/abc123, got %s", data.FHIRResourceID)
    }
    if data.LOINCCode != "2160-0" {
        t.Errorf("expected loinc 2160-0, got %s", data.LOINCCode)
    }
}

func TestTopicForObservationType(t *testing.T) {
    tests := []struct {
        obsType canonical.ObservationType
        want    string
    }{
        {canonical.ObsLabs, "ingestion.labs"},
        {canonical.ObsVitals, "ingestion.vitals"},
        {canonical.ObsDeviceData, "ingestion.device-data"},
        {canonical.ObsPatientReported, "ingestion.patient-reported"},
        {canonical.ObsMedications, "ingestion.medications"},
        {canonical.ObsABDMRecords, "ingestion.abdm-records"},
        {canonical.ObsGeneral, "ingestion.observations"},
    }
    for _, tt := range tests {
        got := topicForObservationType(tt.obsType)
        if got != tt.want {
            t.Errorf("topicForObservationType(%s) = %s, want %s", tt.obsType, got, tt.want)
        }
    }
}

func TestMedicalContextForObservation_Critical(t *testing.T) {
    obs := &canonical.CanonicalObservation{
        Flags: []canonical.Flag{canonical.FlagCriticalValue},
    }
    ctx, prio := medicalContextForObservation(obs)
    if ctx != "critical" {
        t.Errorf("expected critical, got %s", ctx)
    }
    if prio != int32(1) {
        t.Errorf("expected priority 1, got %d", prio)
    }
}

func TestMedicalContextForObservation_Routine(t *testing.T) {
    obs := &canonical.CanonicalObservation{
        Flags: []canonical.Flag{},
    }
    ctx, prio := medicalContextForObservation(obs)
    if ctx != "routine" {
        t.Errorf("expected routine, got %s", ctx)
    }
    if prio != int32(5) {
        t.Errorf("expected priority 5, got %d", prio)
    }
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/outbox/... -v`
Expected: FAIL — package doesn't exist yet

- [ ] **Step 7: Write the outbox publisher**

Create `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/outbox/publisher.go`:

```go
package outbox

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/sirupsen/logrus"
    "go.uber.org/zap"

    "github.com/cardiofit/ingestion-service/internal/canonical"
    outboxsdk "global-outbox-service-go/pkg/outbox-sdk"
)

// EventData is the clinical payload stored in outbox event_data JSONB.
// This is what downstream consumers (KB-20, KB-22, Flink Module1b) will read.
// Field names match CanonicalObservation struct fields in canonical/observation.go.
type EventData struct {
    EventID         string                 `json:"event_id"`
    PatientID       string                 `json:"patient_id"`
    TenantID        string                 `json:"tenant_id"`
    ObservationType string                 `json:"observation_type"`
    LOINCCode       string                 `json:"loinc_code"`
    Value           float64                `json:"value"`
    Unit            string                 `json:"unit"`
    Timestamp       time.Time              `json:"timestamp"`
    SourceType      string                 `json:"source_type"`
    SourceID        string                 `json:"source_id"`
    QualityScore    float64                `json:"quality_score"`
    Flags           []string               `json:"flags"`
    FHIRResourceID  string                 `json:"fhir_resource_id"`
}

// Publisher wraps the Outbox SDK for ingestion-specific event publishing.
type Publisher struct {
    client *outboxsdk.OutboxClient
    logger *zap.Logger
}

// NewPublisher creates an outbox publisher. The SDK auto-creates the
// outbox_events_ingestion_service table and connects to the Central Publisher via gRPC.
func NewPublisher(client *outboxsdk.OutboxClient, logger *zap.Logger) *Publisher {
    return &Publisher{client: client, logger: logger}
}

// Publish sends a single observation to the outbox. For critical values,
// use PublishCritical instead (dual-publish to source + safety-critical topics).
func (p *Publisher) Publish(ctx context.Context, obs *canonical.CanonicalObservation, fhirResourceID string) error {
    topic := topicForObservationType(obs.ObservationType)
    medCtx, priority := medicalContextForObservation(obs)
    eventType := eventTypeFromObservationType(obs.ObservationType)
    data := eventDataFromObservation(obs, fhirResourceID)

    return p.client.SaveAndPublish(ctx, eventType, data, &outboxsdk.EventOptions{
        Topic:          topic,
        Priority:       priority,
        MedicalContext: medCtx,
        CorrelationID:  obs.ID.String(),
        Metadata: map[string]string{
            "patient_id": obs.PatientID.String(),
            "source":     string(obs.SourceType),
            "loinc":      obs.LOINCCode,
        },
    }, nil) // nil businessLogic — FHIR write already completed before this call
}

// PublishCritical dual-publishes a critical-value observation:
// Row 1 → source topic (e.g. ingestion.labs), Row 2 → ingestion.safety-critical.
// Both rows are written in ONE PostgreSQL transaction via SaveAndPublishBatch.
func (p *Publisher) PublishCritical(ctx context.Context, obs *canonical.CanonicalObservation, fhirResourceID string) error {
    topic := topicForObservationType(obs.ObservationType)
    eventType := eventTypeFromObservationType(obs.ObservationType)
    data := eventDataFromObservation(obs, fhirResourceID)

    events := []outboxsdk.EventRequest{
        {
            EventType: eventType,
            EventData: data,
            Options: &outboxsdk.EventOptions{
                Topic:          topic,
                Priority:       1,
                MedicalContext: "critical",
                CorrelationID:  obs.ID.String(),
                Metadata: map[string]string{
                    "patient_id": obs.PatientID.String(),
                    "source":     string(obs.SourceType),
                    "loinc":      obs.LOINCCode,
                },
            },
        },
        {
            EventType: eventType,
            EventData: data,
            Options: &outboxsdk.EventOptions{
                Topic:          "ingestion.safety-critical",
                Priority:       1,
                MedicalContext: "critical",
                CorrelationID:  obs.ID.String(),
                Metadata: map[string]string{
                    "patient_id": obs.PatientID.String(),
                    "source":     string(obs.SourceType),
                    "loinc":      obs.LOINCCode,
                },
            },
        },
    }

    return p.client.SaveAndPublishBatch(ctx, events, nil)
}

// Close releases SDK resources.
func (p *Publisher) Close() error {
    return p.client.Close()
}

// topicForObservationType maps observation types to ingestion Kafka topics.
// Mirrors the existing router.go topicMap.
func topicForObservationType(obsType canonical.ObservationType) string {
    switch obsType {
    case canonical.ObsLabs:
        return "ingestion.labs"
    case canonical.ObsVitals:
        return "ingestion.vitals"
    case canonical.ObsDeviceData:
        return "ingestion.device-data"
    case canonical.ObsPatientReported:
        return "ingestion.patient-reported"
    case canonical.ObsMedications:
        return "ingestion.medications"
    case canonical.ObsABDMRecords:
        return "ingestion.abdm-records"
    default:
        return "ingestion.observations"
    }
}

// medicalContextForObservation returns the medical context and priority for the outbox event.
func medicalContextForObservation(obs *canonical.CanonicalObservation) (string, int32) {
    for _, f := range obs.Flags {
        if f == canonical.FlagCriticalValue {
            return "critical", 1
        }
    }
    return "routine", 5
}

// eventTypeFromObservationType maps observation types to event type strings.
func eventTypeFromObservationType(obsType canonical.ObservationType) string {
    switch obsType {
    case canonical.ObsLabs:
        return "observation.lab.created"
    case canonical.ObsVitals:
        return "observation.vital.created"
    case canonical.ObsDeviceData:
        return "observation.device.created"
    case canonical.ObsPatientReported:
        return "observation.patient-reported.created"
    case canonical.ObsMedications:
        return "observation.medication.created"
    case canonical.ObsABDMRecords:
        return "observation.abdm.created"
    default:
        return "observation.created"
    }
}

// eventDataFromObservation builds the canonical event_data payload.
// Maps from CanonicalObservation fields (canonical/observation.go).
func eventDataFromObservation(obs *canonical.CanonicalObservation, fhirResourceID string) EventData {
    flags := make([]string, len(obs.Flags))
    for i, f := range obs.Flags {
        flags[i] = string(f)
    }
    return EventData{
        EventID:         uuid.New().String(),
        PatientID:       obs.PatientID.String(),
        TenantID:        obs.TenantID.String(),
        ObservationType: string(obs.ObservationType),
        LOINCCode:       obs.LOINCCode,
        Value:           obs.Value,
        Unit:            obs.Unit,
        Timestamp:       obs.Timestamp,     // CanonicalObservation.Timestamp (not EffectiveAt)
        SourceType:      string(obs.SourceType),
        SourceID:        obs.SourceID,       // CanonicalObservation.SourceID (not SourceName)
        QualityScore:    obs.QualityScore,
        Flags:           flags,
        FHIRResourceID:  fhirResourceID,
    }
}
```

- [ ] **Step 8: Run tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/outbox/... -v`
Expected: PASS (4 tests)

- [ ] **Step 9: Commit**

```bash
git add internal/outbox/
git commit -m "feat(ingestion): add outbox publisher wrapping SDK for topic routing and dual-publish"
```

### Sub-task 2c: Wire outbox into Server and replace publishResult

- [ ] **Step 10: Update go.mod to add outbox SDK dependency**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go get global-outbox-service-go/pkg/outbox-sdk`

If the module is local (not published), add a `replace` directive:
```
replace global-outbox-service-go => ../../../../backend/services/global-outbox-service-go
```

- [ ] **Step 11: Modify server.go — replace kafkaProducer with outboxPublisher**

In `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/api/server.go`:

1. Add import: `outboxpkg "github.com/cardiofit/ingestion-service/internal/outbox"` and `outboxsdk "global-outbox-service-go/pkg/outbox-sdk"` and `"github.com/sirupsen/logrus"`
2. In Server struct (line 37-38): replace `kafkaProducer *kafkapkg.Producer` with `outboxPublisher *outboxpkg.Publisher`. Keep `topicRouter` (still used for fallback logging).
3. In `NewServer()` (lines 84-88): replace kafka producer creation with outbox SDK init:

```go
// Outbox publisher (replaces direct Kafka producer)
var outboxPublisher *outboxpkg.Publisher
if cfg.Outbox.Enabled {
    dbURL := cfg.Outbox.DatabaseURL
    if dbURL == "" {
        dbURL = cfg.Database.URL
    }
    logrusLogger := logrus.New()
    sdkClient, err := outboxsdk.NewOutboxClient(&outboxsdk.ClientConfig{
        ServiceName:          "ingestion-service",
        DatabaseURL:          dbURL,
        OutboxServiceGRPCURL: cfg.Outbox.GRPCAddress,
        DefaultTopic:         "ingestion.observations",
        DefaultPriority:      cfg.Outbox.DefaultPriority,
        DefaultMedicalContext: "routine",
    }, logrusLogger)
    if err != nil {
        logger.Error("outbox SDK init failed — falling back to direct Kafka", zap.Error(err))
    } else {
        outboxPublisher = outboxpkg.NewPublisher(sdkClient, logger)
    }
}

// Fallback: direct Kafka producer (used when outbox is disabled or init failed)
var kafkaProducer *kafkapkg.Producer
if outboxPublisher == nil && len(cfg.Kafka.Brokers) > 0 && cfg.Kafka.Brokers[0] != "" {
    kafkaProducer = kafkapkg.NewProducer(cfg.Kafka.Brokers, logger)
}
```

4. Update Server struct initialization to include `outboxPublisher` field.

- [ ] **Step 12: Modify handlers.go — rewrite publishResult()**

In `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/api/handlers.go`, replace `publishResult()` (lines 299-401):

```go
func (s *Server) publishResult(c *gin.Context, obs *canonical.CanonicalObservation) {
    // Check for critical values
    isCritical := false
    for _, flag := range obs.Flags {
        if flag == canonical.FlagCriticalValue {
            metrics.CriticalValues.WithLabelValues(string(obs.ObservationType), obs.TenantID.String()).Inc()
            isCritical = true
            break
        }
    }

    // Map to FHIR if not already mapped
    if len(obs.RawPayload) == 0 {
        mapper := fhirmapper.NewCompositeMapper(s.logger)
        fhirJSON, err := mapper.MapToFHIR(c.Request.Context(), obs)
        if err != nil {
            s.logger.Error("FHIR mapping failed", zap.Error(err))
            return
        }
        obs.RawPayload = fhirJSON
    }

    // Write to FHIR Store (SYNCHRONOUS — before outbox)
    var fhirResourceID string
    if s.fhirClient != nil {
        resourceType := "Observation"
        if obs.ObservationType == canonical.ObsMedications {
            resourceType = "MedicationStatement"
        }
        resp, err := s.fhirClient.Create(resourceType, obs.RawPayload)
        if err != nil {
            s.logger.Error("FHIR Store write failed",
                zap.String("patient_id", obs.PatientID.String()),
                zap.Error(err),
            )
        } else {
            var created map[string]interface{}
            if json.Unmarshal(resp, &created) == nil {
                if id, ok := created["id"].(string); ok {
                    fhirResourceID = id
                }
            }
        }

        // For lab results, also create a DiagnosticReport
        if obs.ObservationType == canonical.ObsLabs && fhirResourceID != "" {
            drJSON, err := fhirmapper.MapDiagnosticReport(obs, fhirResourceID)
            if err == nil {
                _, _ = s.fhirClient.Create("DiagnosticReport", drJSON)
            }
        }
    }

    // Publish via outbox (preferred) or direct Kafka (fallback)
    if s.outboxPublisher != nil {
        var err error
        if isCritical {
            err = s.outboxPublisher.PublishCritical(c.Request.Context(), obs, fhirResourceID)
        } else {
            err = s.outboxPublisher.Publish(c.Request.Context(), obs, fhirResourceID)
        }
        if err != nil {
            metrics.DLQMessages.WithLabelValues("OUTBOX", string(obs.SourceType)).Inc()
            s.logger.Error("outbox publish failed",
                zap.String("patient_id", obs.PatientID.String()),
                zap.Error(err),
            )
        }
        return
    }

    // Fallback: direct Kafka (kept for backward compatibility during rollout)
    if s.kafkaProducer != nil && s.topicRouter != nil {
        topic, key, err := s.topicRouter.Route(c.Request.Context(), obs)
        if err == nil {
            resourceType := "Observation"
            if obs.ObservationType == canonical.ObsMedications {
                resourceType = "MedicationStatement"
            }
            if pubErr := s.kafkaProducer.Publish(
                c.Request.Context(), topic, key, obs,
                resourceType, fhirResourceID, s.config.Kafka.Brokers,
            ); pubErr != nil {
                metrics.DLQMessages.WithLabelValues("PUBLISH", string(obs.SourceType)).Inc()
                s.logger.Error("Kafka publish failed", zap.String("topic", topic), zap.Error(pubErr))
            }
            if isCritical {
                _ = s.kafkaProducer.Publish(c.Request.Context(), "ingestion.safety-critical", key, obs, resourceType, fhirResourceID, s.config.Kafka.Brokers)
            }
        }
    }
}
```

- [ ] **Step 13: Add ingestion-service to outbox supported_services**

In `backend/services/global-outbox-service-go/internal/config/config.go`, add `"ingestion-service"` to the `supported_services` default list at line ~170.

- [ ] **Step 14: Run existing tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./... -v`
Expected: All existing tests pass (outbox disabled by default, falls back to Kafka path)

- [ ] **Step 15: Commit**

```bash
git add -A
git commit -m "feat(ingestion): wire outbox SDK into server, replace publishResult with outbox-first + Kafka fallback (PR1)"
```

---

## Task 3: Wearable Gap Fix (PR2)

**Files:**
- Modify: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/router.go:14-22`
- Modify: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/router_test.go`
- Modify: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/outbox/publisher.go`
- Modify: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/canonical/observation.go`

- [ ] **Step 1: Write the failing test**

Add to `router_test.go`:
```go
func TestRouteWearableAggregates(t *testing.T) {
    obs := &canonical.CanonicalObservation{
        PatientID:       uuid.New(),
        ObservationType: canonical.ObsWearableAggregates,
    }
    r := NewTopicRouter(zap.NewNop())
    topic, key, err := r.Route(context.Background(), obs)
    if err != nil {
        t.Fatal(err)
    }
    if topic != "ingestion.wearable-aggregates" {
        t.Errorf("expected ingestion.wearable-aggregates, got %s", topic)
    }
    if key != obs.PatientID.String() {
        t.Errorf("partition key mismatch")
    }
}

func TestRouteCGMRaw(t *testing.T) {
    obs := &canonical.CanonicalObservation{
        PatientID:       uuid.New(),
        ObservationType: canonical.ObsCGMRaw,
    }
    r := NewTopicRouter(zap.NewNop())
    topic, _, err := r.Route(context.Background(), obs)
    if err != nil {
        t.Fatal(err)
    }
    if topic != "ingestion.cgm-raw" {
        t.Errorf("expected ingestion.cgm-raw, got %s", topic)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/kafka/... -run TestRouteWearable -v`
Expected: FAIL — `ObsWearableAggregates` undefined

- [ ] **Step 3: Add wearable observation types to canonical**

In `internal/canonical/observation.go`, add:
```go
ObsWearableAggregates ObservationType = "WEARABLE_AGGREGATES"
ObsCGMRaw             ObservationType = "CGM_RAW"
```

- [ ] **Step 4: Add wearable topics to router.go and outbox publisher**

In `internal/kafka/router.go` topicMap (line 14-22), add:
```go
canonical.ObsWearableAggregates: "ingestion.wearable-aggregates",
canonical.ObsCGMRaw:             "ingestion.cgm-raw",
```

In `internal/outbox/publisher.go` `topicForObservationType()`, add:
```go
case canonical.ObsWearableAggregates:
    return "ingestion.wearable-aggregates"
case canonical.ObsCGMRaw:
    return "ingestion.cgm-raw"
```

And in `medicalContextForObservation()`, add background context for wearables:
```go
func medicalContextForObservation(obs *canonical.CanonicalObservation) (string, int32) {
    for _, f := range obs.Flags {
        if f == canonical.FlagCriticalValue {
            return "critical", 1
        }
    }
    switch obs.ObservationType {
    case canonical.ObsWearableAggregates, canonical.ObsCGMRaw:
        return "background", 8
    default:
        return "routine", 5
    }
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/kafka/... ./internal/outbox/... -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat(ingestion): add wearable-aggregates and cgm-raw topic routing with background priority (PR2)"
```

---

## Task 4: Flink Module1b — IngestionCanonicalizer (PR3)

**Files:**
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/OutboxEnvelope.java`
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java`
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/KafkaTopics.java`
- Modify: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/routers/CriticalAlertRouter.java`

### Sub-task 4a: OutboxEnvelope model

- [ ] **Step 1: Create OutboxEnvelope.java**

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;
import java.util.Map;

/**
 * Deserialization model for the Global Outbox SDK envelope format.
 * Ingestion service writes this to Kafka via the Central Publisher.
 * Module1b transforms this into CanonicalEvent for the Flink pipeline.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class OutboxEnvelope {

    @JsonProperty("id")
    private String id;

    @JsonProperty("service_name")
    private String serviceName;

    @JsonProperty("event_type")
    private String eventType;

    @JsonProperty("topic")
    private String topic;

    @JsonProperty("correlation_id")
    private String correlationId;

    @JsonProperty("priority")
    private int priority;

    @JsonProperty("medical_context")
    private String medicalContext;

    @JsonProperty("metadata")
    private Map<String, String> metadata;

    @JsonProperty("event_data")
    private IngestionEventData eventData;

    @JsonProperty("created_at")
    private String createdAt;

    // Getters and setters
    public String getId() { return id; }
    public void setId(String id) { this.id = id; }
    public String getServiceName() { return serviceName; }
    public void setServiceName(String serviceName) { this.serviceName = serviceName; }
    public String getEventType() { return eventType; }
    public void setEventType(String eventType) { this.eventType = eventType; }
    public String getTopic() { return topic; }
    public void setTopic(String topic) { this.topic = topic; }
    public String getCorrelationId() { return correlationId; }
    public void setCorrelationId(String correlationId) { this.correlationId = correlationId; }
    public int getPriority() { return priority; }
    public void setPriority(int priority) { this.priority = priority; }
    public String getMedicalContext() { return medicalContext; }
    public void setMedicalContext(String medicalContext) { this.medicalContext = medicalContext; }
    public Map<String, String> getMetadata() { return metadata; }
    public void setMetadata(Map<String, String> metadata) { this.metadata = metadata; }
    public IngestionEventData getEventData() { return eventData; }
    public void setEventData(IngestionEventData eventData) { this.eventData = eventData; }
    public String getCreatedAt() { return createdAt; }
    public void setCreatedAt(String createdAt) { this.createdAt = createdAt; }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class IngestionEventData {
        @JsonProperty("event_id") private String eventId;
        @JsonProperty("patient_id") private String patientId;
        @JsonProperty("tenant_id") private String tenantId;
        @JsonProperty("observation_type") private String observationType;
        @JsonProperty("loinc_code") private String loincCode;
        @JsonProperty("value") private double value;
        @JsonProperty("unit") private String unit;
        @JsonProperty("timestamp") private String timestamp;
        @JsonProperty("source_type") private String sourceType;
        @JsonProperty("source_id") private String sourceId;
        @JsonProperty("quality_score") private double qualityScore;
        @JsonProperty("flags") private List<String> flags;
        @JsonProperty("fhir_resource_id") private String fhirResourceId;

        // Getters
        public String getEventId() { return eventId; }
        public String getPatientId() { return patientId; }
        public String getTenantId() { return tenantId; }
        public String getObservationType() { return observationType; }
        public String getLoincCode() { return loincCode; }
        public double getValue() { return value; }
        public String getUnit() { return unit; }
        public String getTimestamp() { return timestamp; }
        public String getSourceType() { return sourceType; }
        public String getSourceId() { return sourceId; }
        public double getQualityScore() { return qualityScore; }
        public List<String> getFlags() { return flags; }
        public String getFhirResourceId() { return fhirResourceId; }

        // Setters
        public void setEventId(String eventId) { this.eventId = eventId; }
        public void setPatientId(String patientId) { this.patientId = patientId; }
        public void setTenantId(String tenantId) { this.tenantId = tenantId; }
        public void setObservationType(String observationType) { this.observationType = observationType; }
        public void setLoincCode(String loincCode) { this.loincCode = loincCode; }
        public void setValue(double value) { this.value = value; }
        public void setUnit(String unit) { this.unit = unit; }
        public void setTimestamp(String timestamp) { this.timestamp = timestamp; }
        public void setSourceType(String sourceType) { this.sourceType = sourceType; }
        public void setSourceId(String sourceId) { this.sourceId = sourceId; }
        public void setQualityScore(double qualityScore) { this.qualityScore = qualityScore; }
        public void setFlags(List<String> flags) { this.flags = flags; }
        public void setFhirResourceId(String fhirResourceId) { this.fhirResourceId = fhirResourceId; }
    }
}
```

- [ ] **Step 2: Compile**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -q`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add src/main/java/com/cardiofit/flink/models/OutboxEnvelope.java
git commit -m "feat(flink): add OutboxEnvelope deserialization model for ingestion events"
```

### Sub-task 4b: Add ingestion topic constants

- [ ] **Step 4: Add topic constants to KafkaTopics.java**

Add these entries to the existing `KafkaTopics` **enum** in `src/main/java/com/cardiofit/flink/utils/KafkaTopics.java`. The enum uses a 3-arg constructor: `(topicName, partitions, retentionDays)`:

```java
// ============= Ingestion Service Topics (consumed by Module1b) =============
INGESTION_LABS("ingestion.labs", 12, 90),
INGESTION_VITALS("ingestion.vitals", 8, 30),
INGESTION_DEVICE_DATA("ingestion.device-data", 8, 30),
INGESTION_PATIENT_REPORTED("ingestion.patient-reported", 8, 30),
INGESTION_WEARABLE_AGGREGATES("ingestion.wearable-aggregates", 4, 14),
INGESTION_CGM_RAW("ingestion.cgm-raw", 4, 7),
INGESTION_ABDM_RECORDS("ingestion.abdm-records", 4, 180),
INGESTION_MEDICATIONS("ingestion.medications", 8, 90),
INGESTION_OBSERVATIONS("ingestion.observations", 8, 30),
INGESTION_SAFETY_CRITICAL("ingestion.safety-critical", 4, 90),

// KB threshold hot-swap topic
KB_CLINICAL_THRESHOLDS_CHANGES("kb.clinical-thresholds.changes", 1, 7),
```

**Note:** When referencing topics in code, use `KafkaTopics.INGESTION_LABS.getTopicName()` (not the enum directly, since KafkaSource expects String).

- [ ] **Step 5: Commit**

```bash
git add src/main/java/com/cardiofit/flink/models/KafkaTopics.java
git commit -m "feat(flink): add ingestion.* and kb threshold topic constants"
```

### Sub-task 4c: Module1b IngestionCanonicalizer

- [ ] **Step 6: Create Module1b_IngestionCanonicalizer.java**

This is the core ~200-line Flink job. It reads all `ingestion.*` topics, transforms `OutboxEnvelope` → `CanonicalEvent`, and outputs to the Module2 input topic (`enriched-patient-events-v1`).

**Key design decisions (based on actual codebase model hierarchy):**
- `CanonicalEvent` has NO `routing` field — routing decisions are made later by `TransactionalMultiSinkRouterV2_OptionC` which wraps `EnrichedClinicalEvent` into `RoutedEnrichedEvent` with a `RoutingDecision`
- Module1b sets `sourceSystem = "ingestion-service"` on the CanonicalEvent — this is how downstream operators identify ingestion events
- The FHIR skip and critical alert routing are handled by modifying `TransactionalMultiSinkRouterV2_OptionC.shouldPersistToFHIR()` and `isCriticalEvent()` (see Sub-task 4d)
- `KafkaTopics` is an **enum** in `com.cardiofit.flink.utils` — use `.getTopicName()` to get the String
- `CanonicalObservation.Timestamp` maps to `EventData.timestamp` (not `EffectiveAt`)
- `CanonicalObservation.SourceID` maps to `EventData.source_id` (not `SourceName`)
- `ObservationType` values use UPPER_CASE: `"LABS"`, `"VITALS"`, etc.

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.utils.KafkaTopics;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.serialization.SimpleStringSchema;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Instant;
import java.util.*;

/**
 * Module1b: IngestionCanonicalizer
 *
 * DEPLOYMENT: Separate Flink job (not co-deployed with Module1)
 * CONSUMER GROUP: flink-module1b-ingestion (independent from Module1)
 * PARALLELISM: 4
 * CHECKPOINTING: 60 seconds
 *
 * Transforms: OutboxEnvelope (from ingestion service) → CanonicalEvent (Flink pipeline format)
 * Sets sourceSystem="ingestion-service" so downstream routing operator can:
 *   1. Skip FHIR write (raw Observation already in FHIR from ingestion Stage 3)
 *   2. Set sendToCriticalAlerts=true for events with CRITICAL_VALUE flag
 *
 * FAILURE ISOLATION: Module1b crash does NOT affect Module1 (EHR pipeline).
 */
public class Module1b_IngestionCanonicalizer {

    private static final Logger LOG = LoggerFactory.getLogger(Module1b_IngestionCanonicalizer.class);
    private static final ObjectMapper MAPPER = new ObjectMapper();

    public static void main(String[] args) throws Exception {
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
        env.enableCheckpointing(60_000);
        env.setParallelism(4);

        String kafkaBrokers = System.getenv().getOrDefault("KAFKA_BROKERS", "localhost:9092");

        // Multi-topic subscription: all ingestion.* topics
        List<String> ingestionTopics = Arrays.asList(
            KafkaTopics.INGESTION_LABS.getTopicName(),
            KafkaTopics.INGESTION_VITALS.getTopicName(),
            KafkaTopics.INGESTION_DEVICE_DATA.getTopicName(),
            KafkaTopics.INGESTION_PATIENT_REPORTED.getTopicName(),
            KafkaTopics.INGESTION_WEARABLE_AGGREGATES.getTopicName(),
            KafkaTopics.INGESTION_CGM_RAW.getTopicName(),
            KafkaTopics.INGESTION_ABDM_RECORDS.getTopicName(),
            KafkaTopics.INGESTION_MEDICATIONS.getTopicName(),
            KafkaTopics.INGESTION_OBSERVATIONS.getTopicName(),
            KafkaTopics.INGESTION_SAFETY_CRITICAL.getTopicName()
        );

        KafkaSource<String> source = KafkaSource.<String>builder()
            .setBootstrapServers(kafkaBrokers)
            .setTopics(ingestionTopics)
            .setGroupId("flink-module1b-ingestion")
            .setStartingOffsets(OffsetsInitializer.latest())
            .setValueOnlyDeserializer(new SimpleStringSchema())
            .build();

        DataStream<String> ingestionStream = env.fromSource(
            source, WatermarkStrategy.noWatermarks(), "IngestionSource");

        // Transform OutboxEnvelope → CanonicalEvent JSON
        DataStream<String> canonicalStream = ingestionStream
            .map(raw -> {
                try {
                    OutboxEnvelope envelope = MAPPER.readValue(raw, OutboxEnvelope.class);
                    CanonicalEvent event = transformToCanonical(envelope);
                    return MAPPER.writeValueAsString(event);
                } catch (Exception e) {
                    LOG.error("Failed to canonicalize ingestion event: {}", e.getMessage());
                    return null;
                }
            })
            .filter(Objects::nonNull)
            .name("IngestionCanonicalizer");

        // Output to Module2 input topic (same junction as Module1)
        KafkaSink<String> sink = KafkaSink.<String>builder()
            .setBootstrapServers(kafkaBrokers)
            .setRecordSerializer(
                KafkaRecordSerializationSchema.builder()
                    .setTopic(KafkaTopics.ENRICHED_PATIENT_EVENTS.getTopicName())
                    .setValueSerializationSchema(new SimpleStringSchema())
                    .build()
            )
            .build();

        canonicalStream.sinkTo(sink);

        env.execute("Module1b-IngestionCanonicalizer");
    }

    /**
     * Transform an outbox envelope from the ingestion service into the
     * Flink pipeline's CanonicalEvent format.
     *
     * NOTE: CanonicalEvent has no routing field. The FHIR skip and critical
     * alert routing are handled by TransactionalMultiSinkRouterV2_OptionC
     * which checks sourceSystem == "ingestion-service".
     */
    static CanonicalEvent transformToCanonical(OutboxEnvelope envelope) {
        OutboxEnvelope.IngestionEventData data = envelope.getEventData();

        CanonicalEvent event = new CanonicalEvent();
        event.setId(data.getEventId());
        event.setPatientId(data.getPatientId());
        event.setEventType(mapEventType(data.getObservationType()));

        // Parse timestamp or fall back to envelope created_at
        try {
            event.setEventTime(Instant.parse(data.getTimestamp()).toEpochMilli());
        } catch (Exception e) {
            event.setEventTime(System.currentTimeMillis());
        }
        event.setProcessingTime(System.currentTimeMillis());
        event.setSourceSystem("ingestion-service"); // Key marker for downstream routing
        event.setCorrelationId(envelope.getCorrelationId());

        // Build payload map (consumed by Module2+ operators)
        Map<String, Object> payload = new HashMap<>();
        payload.put("loinc_code", data.getLoincCode());
        payload.put("value", data.getValue());
        payload.put("unit", data.getUnit());
        payload.put("observation_type", data.getObservationType());
        payload.put("quality_score", data.getQualityScore());
        payload.put("source_type", data.getSourceType());
        payload.put("source_id", data.getSourceId());
        payload.put("fhir_resource_id", data.getFhirResourceId());
        if (data.getFlags() != null) {
            payload.put("flags", data.getFlags());
        }
        event.setPayload(payload);

        // Metadata
        CanonicalEvent.EventMetadata metadata = new CanonicalEvent.EventMetadata();
        metadata.setSource("ingestion-service");
        event.setMetadata(metadata);

        return event;
    }

    private static EventType mapEventType(String observationType) {
        if (observationType == null) return EventType.OBSERVATION;
        switch (observationType) {
            case "LABS":             return EventType.LAB_RESULT;
            case "VITALS":           return EventType.VITAL_SIGN;
            case "DEVICE_DATA":      return EventType.DEVICE_READING;
            case "PATIENT_REPORTED": return EventType.PATIENT_REPORT;
            case "MEDICATIONS":      return EventType.MEDICATION;
            default:                 return EventType.OBSERVATION;
        }
    }
}
```

- [ ] **Step 7: Compile**

Run: `cd backend/shared-infrastructure/flink-processing && mvn compile -q`
Expected: BUILD SUCCESS

- [ ] **Step 8: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/Module1b_IngestionCanonicalizer.java
git commit -m "feat(flink): add Module1b IngestionCanonicalizer — transforms outbox envelopes to CanonicalEvents (PR3)"
```

### Sub-task 4d: Modify routing operator for ingestion FHIR skip + critical alerts

The FHIR skip and critical alert routing for ingestion events are handled in `TransactionalMultiSinkRouterV2_OptionC.java` — the operator that wraps `EnrichedClinicalEvent` into `RoutedEnrichedEvent` with a `RoutingDecision`. This is where `sendToFHIR` and `sendToCriticalAlerts` flags are set.

**Why NOT modify CriticalAlertRouter directly:** CriticalAlertRouter reads `RoutedEnrichedEvent` from `prod.ehr.events.enriched.routing`. It cannot read raw outbox envelopes from `ingestion.safety-critical` (different serialization format). Instead, ingestion events flow through the normal pipeline (Module1b → Module2 → ... → routing operator) and the routing operator sets the correct flags.

- [ ] **Step 9: Modify TransactionalMultiSinkRouterV2_OptionC.shouldPersistToFHIR()**

In `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/TransactionalMultiSinkRouterV2_OptionC.java`, in the `shouldPersistToFHIR()` method (around line 144), add an early return for ingestion events:

```java
private boolean shouldPersistToFHIR(EnrichedClinicalEvent event) {
    // Ingestion service events: raw Observation already written to FHIR Store
    // in ingestion Stage 3. Skip to prevent duplicate FHIR writes.
    // Derived resources (RiskAssessment, DetectedIssue) from other operators
    // will NOT have sourceSystem="ingestion-service" and will be written normally.
    if ("ingestion-service".equals(event.getSourceSystem())) {
        return false;
    }

    // ... existing logic unchanged ...
}
```

- [ ] **Step 10: Modify TransactionalMultiSinkRouterV2_OptionC.isCriticalEvent()**

Add ingestion CRITICAL_VALUE flag detection to the existing `isCriticalEvent()` method:

```java
private boolean isCriticalEvent(EnrichedClinicalEvent event) {
    // Existing critical event detection...
    // ... (keep all existing logic) ...

    // Ingestion service critical values (K+ > 5.5, eGFR < 15, etc.)
    // These arrive with CRITICAL_VALUE in the payload flags list
    if ("ingestion-service".equals(event.getSourceSystem())) {
        Object flags = event.getPayload() != null ? event.getPayload().get("flags") : null;
        if (flags instanceof List && ((List<?>) flags).contains("CRITICAL_VALUE")) {
            return true;
        }
    }

    return false; // or existing return
}
```

- [ ] **Step 11: Compile**

Run: `mvn compile -q`
Expected: BUILD SUCCESS

- [ ] **Step 12: Commit**

```bash
git add src/main/java/com/cardiofit/flink/operators/TransactionalMultiSinkRouterV2_OptionC.java
git commit -m "feat(flink): routing operator skips FHIR for ingestion events + routes CRITICAL_VALUE to critical alerts (PR3)"
```

---

## Task 5: KB-4 Vital Threshold Endpoints (PR4)

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-4-patient-safety/internal/thresholds/vitals.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-4-patient-safety/internal/thresholds/early_warning_scores.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-4-patient-safety/internal/thresholds/vitals_test.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-4-patient-safety/main.go` (~line 370)

- [ ] **Step 1: Write the failing test**

Create `kb-4-patient-safety/internal/thresholds/vitals_test.go`:

```go
package thresholds

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)

func TestGetVitalThresholds(t *testing.T) {
    gin.SetMode(gin.TestMode)
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Request, _ = http.NewRequest("GET", "/v1/thresholds/vitals", nil)

    HandleGetVitalThresholds(c)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", w.Code)
    }

    var resp VitalThresholds
    if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
        t.Fatal(err)
    }
    if resp.HeartRate.BradycardiaSevere != 40 {
        t.Errorf("expected HR bradycardia_severe=40, got %v", resp.HeartRate.BradycardiaSevere)
    }
    if resp.SpO2.Critical != 90 {
        t.Errorf("expected SpO2 critical=90, got %v", resp.SpO2.Critical)
    }
    if resp.Version == "" {
        t.Error("expected non-empty version")
    }
}

func TestGetEarlyWarningScores(t *testing.T) {
    gin.SetMode(gin.TestMode)
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Request, _ = http.NewRequest("GET", "/v1/thresholds/early-warning-scores", nil)

    HandleGetEarlyWarningScores(c)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", w.Code)
    }

    var resp map[string]interface{}
    if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
        t.Fatal(err)
    }
    if _, ok := resp["news2"]; !ok {
        t.Error("expected news2 key in response")
    }
    if _, ok := resp["mews"]; !ok {
        t.Error("expected mews key in response")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-4-patient-safety && go test ./internal/thresholds/... -v`
Expected: FAIL — package doesn't exist

- [ ] **Step 3: Create vitals.go**

Create `kb-4-patient-safety/internal/thresholds/vitals.go` with the `VitalThresholds` struct and `HandleGetVitalThresholds` handler. The exact threshold values are defined in the spec (Section 5.3, KB-4). Use the closure handler pattern matching KB-4's existing style.

- [ ] **Step 4: Create early_warning_scores.go**

Create `kb-4-patient-safety/internal/thresholds/early_warning_scores.go` with NEWS2 + MEWS scoring parameter structs and `HandleGetEarlyWarningScores` handler. Include both SpO2 Scale 1 (on air) and Scale 2 (on oxygen). Values from spec Section 5.3.

- [ ] **Step 5: Register routes in main.go**

In `kb-4-patient-safety/main.go`, in the `setupRoutes()` function (around line 370), add:
```go
// Threshold endpoints for Flink ClinicalThresholdService
thresholds := v1.Group("/thresholds")
{
    thresholds.GET("/vitals", thresholdpkg.HandleGetVitalThresholds)
    thresholds.GET("/early-warning-scores", thresholdpkg.HandleGetEarlyWarningScores)
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/thresholds/... -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "feat(kb-4): add GET /v1/thresholds/vitals and /v1/thresholds/early-warning-scores endpoints (PR4)"
```

---

## Task 6: KB-20 Lab Threshold Endpoint (PR5)

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/threshold_handlers.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/threshold_handlers_test.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/routes.go`

- [ ] **Step 1: Write the failing test**

Create `kb-20-patient-profile/internal/api/threshold_handlers_test.go`:

```go
package api

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)

func TestGetLabThresholds(t *testing.T) {
    gin.SetMode(gin.TestMode)

    // Create a minimal server with just the threshold handler
    s := &Server{} // Threshold endpoint doesn't need DB — returns static config
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Request, _ = http.NewRequest("GET", "/api/v1/thresholds/labs", nil)

    s.getLabThresholds(c)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", w.Code)
    }

    var resp map[string]interface{}
    if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
        t.Fatal(err)
    }

    // Verify key thresholds exist
    potassium, ok := resp["potassium"].(map[string]interface{})
    if !ok {
        t.Fatal("expected potassium key")
    }
    if potassium["alert_high"] != 5.5 {
        t.Errorf("expected K+ alert_high=5.5, got %v", potassium["alert_high"])
    }
    if potassium["halt_high"] != 6.0 {
        t.Errorf("expected K+ halt_high=6.0, got %v", potassium["halt_high"])
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile && go test ./internal/api/... -run TestGetLabThresholds -v`
Expected: FAIL — `getLabThresholds` undefined

- [ ] **Step 3: Create threshold_handlers.go**

Create `kb-20-patient-profile/internal/api/threshold_handlers.go`. This handler exposes the existing `lab_validator.go` plausibility ranges plus KDIGO/clinical thresholds from the spec (Section 5.3, KB-20). Use the Server method pattern matching KB-20's existing `routes.go` style.

Key: include both `alert_high`/`alert_low` (Flink alerting) AND `halt_high`/`halt_low` (V-MCU Channel B) fields for potassium and eGFR. Same KB-20 endpoint, different consumers use different fields.

- [ ] **Step 4: Register route in routes.go**

In `kb-20-patient-profile/internal/api/routes.go`, add inside the `v1` group (after line 89):
```go
// Threshold endpoints for Flink ClinicalThresholdService
thresholds := v1.Group("/thresholds")
{
    thresholds.GET("/labs", s.getLabThresholds)
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/api/... -run TestGetLabThresholds -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat(kb-20): add GET /api/v1/thresholds/labs endpoint with alert+halt context fields (PR5)"
```

---

## Task 7: KB-1 High-Risk Categories Endpoint (PR6)

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-1-drug-rules/internal/api/high_risk_handlers.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-1-drug-rules/internal/api/high_risk_handlers_test.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-1-drug-rules/internal/api/server.go` (~line 188)

- [ ] **Step 1: Write the failing test**

Test that `GET /v1/high-risk/categories` returns the expected JSON structure with categories and drug list.

- [ ] **Step 2: Run test to verify it fails**

Expected: FAIL — handler undefined

- [ ] **Step 3: Create high_risk_handlers.go**

Handler returns high-risk medication categories and drug list per spec Section 5.3, KB-1. Use the Server method pattern matching KB-1's `server.go`.

- [ ] **Step 4: Register route in server.go**

In `kb-1-drug-rules/internal/api/server.go`, in the route setup (around line 248), add:
```go
v1.GET("/high-risk/categories", s.handleGetHighRiskCategories)
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/api/... -run TestHighRisk -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat(kb-1): add GET /v1/high-risk/categories endpoint (PR6)"
```

---

## Task 8: KB-23 Risk-Scoring Config Endpoint (PR7)

**Files:**
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/config_handlers.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/config_handlers_test.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/routes.go`

- [ ] **Step 1: Write the failing test**

Test that `GET /api/v1/config/risk-scoring` returns daily_risk_weights, risk_levels, alert_severity_scores, time_sensitivity_scores, patient_vulnerability, and version.

- [ ] **Step 2: Run test to verify it fails**

Expected: FAIL — handler undefined

- [ ] **Step 3: Create config_handlers.go**

Handler returns risk-scoring configuration per spec Section 5.3, KB-23. Use Server method pattern matching KB-23's `routes.go`.

- [ ] **Step 4: Register route in routes.go**

In `kb-23-decision-cards/internal/api/routes.go`, add inside the `v1` group:
```go
config := v1.Group("/config")
{
    config.GET("/risk-scoring", s.getRiskScoringConfig)
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/api/... -run TestRiskScoring -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat(kb-23): add GET /api/v1/config/risk-scoring endpoint (PR7)"
```

---

## Task 9: Flink ClinicalThresholdService + BroadcastState (PR8)

**Files:**
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/thresholds/ClinicalThresholdSet.java`
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/thresholds/ClinicalThresholdService.java`
- Create: `backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/thresholds/ThresholdBroadcastFunction.java`
- Create: `backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/thresholds/ClinicalThresholdServiceTest.java`

This is the most complex task. It reuses existing patterns from the codebase:
- **AsyncHttpClient** pattern from `GoogleFHIRClient.java`
- **Caffeine L1+L2 cache** pattern from `GoogleFHIRClient.java`
- **Resilience4j circuit breaker** pattern from `AsyncPatientEnricher.java`
- **BroadcastState** pattern from `Module3_ComprehensiveCDS_WithCDC.java`

### Sub-task 9a: ClinicalThresholdSet POJO

- [ ] **Step 1: Create ClinicalThresholdSet.java**

A POJO holding all threshold groups deserialized from the 4 KB endpoints. Include `hardcodedDefaults()` static method that returns the same values currently hardcoded in `NEWS2Calculator`, `ClinicalScoreCalculator`, `SmartAlertGenerator`, `RiskScoreCalculator`, `AlertPrioritizer`, etc.

Key sections: `VitalThresholds`, `LabThresholds`, `NEWS2Params`, `MEWSParams`, `RiskScoringConfig`, `HighRiskDrugCategories`.

- [ ] **Step 2: Compile**

Run: `mvn compile -q`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add src/main/java/com/cardiofit/flink/thresholds/ClinicalThresholdSet.java
git commit -m "feat(flink): add ClinicalThresholdSet POJO with hardcoded defaults fallback"
```

### Sub-task 9b: ClinicalThresholdService

- [ ] **Step 4: Write the test**

Create `ClinicalThresholdServiceTest.java`:
```java
@Test
public void testGetThresholdsReturnsDefaultsWhenCacheEmpty() {
    ClinicalThresholdService service = new ClinicalThresholdService(/* mock URLs */);
    ClinicalThresholdSet thresholds = service.getThresholds();
    assertNotNull(thresholds);
    assertEquals(40, thresholds.getVitals().getHeartRate().getBradycardiaSevere());
    assertEquals(5.5, thresholds.getLabs().getPotassium().getAlertHigh(), 0.01);
}
```

- [ ] **Step 5: Create ClinicalThresholdService.java**

Implement the three-tier fallback service:
- **Tier 1:** Caffeine L1 cache (5-min TTL) — populated from live KB service HTTP responses
- **Tier 2:** Caffeine L2 stale cache (24-hour TTL)
- **Tier 3:** `ClinicalThresholdSet.hardcodedDefaults()` (same values as today, zero regression)

Key methods:
- `loadInitialThresholds()` — blocking at job startup, 10s timeout per KB service
- `refreshThresholds()` — async, called every 5 minutes by a scheduled executor
- `getThresholds()` — three-tier resolution (L1 → L2 → defaults)

Uses per-KB circuit breakers (Resilience4j) to prevent cascading failures.

- [ ] **Step 6: Run test**

Run: `mvn test -pl . -Dtest=ClinicalThresholdServiceTest -q`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add src/main/java/com/cardiofit/flink/thresholds/ClinicalThresholdService.java
git add src/test/java/com/cardiofit/flink/thresholds/ClinicalThresholdServiceTest.java
git commit -m "feat(flink): add ClinicalThresholdService with 3-tier fallback (live → stale → defaults)"
```

### Sub-task 9c: BroadcastState wiring

- [ ] **Step 8: Create ThresholdBroadcastFunction.java**

Abstract base class that analytics operators extend. Reads `ClinicalThresholdSet` from BroadcastState. Reuses the `MapStateDescriptor` pattern from `Module3_ComprehensiveCDS_WithCDC.java`.

```java
public static final MapStateDescriptor<String, ClinicalThresholdSet> THRESHOLD_STATE =
    new MapStateDescriptor<>("clinical-thresholds",
        Types.STRING, Types.POJO(ClinicalThresholdSet.class));
```

- [ ] **Step 9: Compile full project**

Run: `mvn compile -q`
Expected: BUILD SUCCESS

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "feat(flink): add BroadcastState wiring for clinical threshold hot-swap (PR8)"
```

### Sub-task 9d: Refactor hardcoded thresholds (incremental)

This is the largest sub-task — refactoring each analytics operator to read from BroadcastState instead of hardcoded constants. Do this incrementally, one operator at a time:

- [ ] **Step 11: Refactor NEWS2Calculator to use BroadcastState thresholds**
- [ ] **Step 12: Refactor ClinicalScoreCalculator (MEWS) to use BroadcastState thresholds**
- [ ] **Step 13: Refactor SmartAlertGenerator to use BroadcastState thresholds**
- [ ] **Step 14: Refactor RiskScoreCalculator to use BroadcastState thresholds**
- [ ] **Step 15: Refactor AlertPrioritizer to use BroadcastState thresholds**
- [ ] **Step 16: Compile and run full test suite**

Run: `mvn clean test -q`
Expected: All tests PASS (hardcoded defaults match current values — zero behavioral change)

- [ ] **Step 17: Commit**

```bash
git add -A
git commit -m "refactor(flink): replace hardcoded clinical thresholds with BroadcastState reads from KB services (PR8)"
```

---

## Verification Checklist

After all tasks complete:

- [ ] `OUTBOX_ENABLED=true` → ingestion POST returns 200, outbox table has pending rows
- [ ] Central Publisher picks up rows and publishes to `ingestion.*` topics
- [ ] Critical K+ 6.8 → two outbox rows (labs + safety-critical) in one transaction
- [ ] `OUTBOX_ENABLED=false` → falls back to direct Kafka (backward compatible)
- [ ] Module1b transforms outbox envelopes to CanonicalEvents
- [ ] FHIRRouter does NOT write raw Observations from ingestion (sendToFHIR=false)
- [ ] CriticalAlertRouter receives ingestion.safety-critical events
- [ ] KB-4 `/v1/thresholds/vitals` returns expected JSON
- [ ] KB-20 `/api/v1/thresholds/labs` returns K+ alert_high=5.5, halt_high=6.0
- [ ] KB-1 `/v1/high-risk/categories` returns expected categories
- [ ] KB-23 `/api/v1/config/risk-scoring` returns expected weights
- [ ] Flink starts with all KBs up → thresholds loaded from KB (not defaults)
- [ ] Flink starts with KB-4 down → uses stale cache, then defaults
- [ ] All existing tests still pass (zero regression)
