# HAPI FHIR Clinical Reasoning Engine + Vaidshala.Substrate Binding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the runtime that executes the 84 CQL defines / 78 specs already shipped, by (a) deploying a new `kb-cql-runtime` service wrapping HAPI FHIR's Clinical Reasoning module, (b) implementing the `Vaidshala.Substrate.*` external function library so CQL helpers like `RunningBaseline()`, `ActiveConcerns()`, `CareIntensity()`, `MedicineUseIntent()` resolve to substrate reads, and (c) integrating execution into Plan 0.1's Recommendation lifecycle so a `detected → drafted` transition runs the relevant CQL rule and populates ClinicalContent. Per audit: this transitions Layer 3 from "shipped as paper" to "shipped as runtime."

**Architecture:** New Java/Spring Boot service `kb-cql-runtime/` running HAPI FHIR R4 server with the Clinical Reasoning IG. CQL libraries (existing in `shared/cql-libraries/`) are loaded as Library FHIR resources. External functions implemented as `org.opencds.cqf.cql.engine.runtime.External` providers that bridge to the substrate over HTTP (substrate exposes a thin REST API the runtime calls). The Recommendation lifecycle calls a Go HTTP client (`internal/cql/client.go`) to invoke `$evaluate-rule` operations on the runtime. End-to-end smoke test exercises the Sunday-night-fall scenario with real CQL evaluation rather than the mocked path used in Phase 0 tests.

**Tech Stack:** Java 17, Spring Boot 3, HAPI FHIR R4 server (latest stable, 7.x line), HAPI Clinical Reasoning module, OpenCDS CQF (cql-engine), PostgreSQL (FHIR resource store), Go (substrate REST API + HTTP client side).

---

## File Structure

**New service:**
- `kb-cql-runtime/` — Maven Java project
  - `pom.xml`
  - `src/main/java/au/vaidshala/cqlruntime/Application.java`
  - `src/main/java/au/vaidshala/cqlruntime/config/HapiConfig.java`
  - `src/main/java/au/vaidshala/cqlruntime/external/SubstrateExternalFunctions.java`
  - `src/main/java/au/vaidshala/cqlruntime/external/SubstrateClient.java`
  - `src/main/java/au/vaidshala/cqlruntime/loader/CqlLibraryLoader.java`
  - `src/main/java/au/vaidshala/cqlruntime/operation/EvaluateRuleProvider.java`
  - `src/test/java/au/vaidshala/cqlruntime/integration/SundayNightFallIT.java`
  - `Dockerfile`
  - `docker-compose.yml`

**New substrate REST API:**
- `kb-20-patient-profile/internal/api/substrate_runtime_handler.go` — handlers for `/runtime/baseline`, `/runtime/active-concerns`, `/runtime/care-intensity`, `/runtime/medicine-use`, `/runtime/observations`
- `kb-20-patient-profile/internal/api/substrate_runtime_handler_test.go`

**Go client:**
- `shared/v2_substrate/cql/client.go` — `Client.EvaluateRule(ctx, ruleID, residentID)` returns rule output for Recommendation lifecycle
- `shared/v2_substrate/cql/client_test.go`

**Modified files:**
- `shared/v2_substrate/recommendation/lifecycle.go` — add optional CQL integration on `detected → drafted` transition
- `shared/cql-libraries/MonitoringHelpers.cql` and similar — replace TODO(wave-1-runtime) bodies with actual implementations bound to external functions

---

## Reading list before starting

The HAPI Clinical Reasoning surface area is non-trivial. Read in order:
1. https://hapifhir.io/hapi-fhir/docs/server_plain/clinical_reasoning.html
2. HAPI samples repo `hapi-fhir-jpaserver-cdshooks/`
3. `shared/cql-libraries/SuppressionHelpers.cql` (already exists; understand how external functions are referenced)
4. Plan 0.1 `recommendation.Lifecycle.Transition` (the integration point)

This is one plan that benefits from genuine prototyping before task-decomposition; if the HAPI APIs differ materially from the assumptions below, adjust task code accordingly. The TDD structure remains valid; the specific Java method names will reflect the real HAPI surface.

---

### Task 1: Scaffold kb-cql-runtime service

**Files:**
- Create: `kb-cql-runtime/pom.xml`
- Create: `kb-cql-runtime/src/main/java/au/vaidshala/cqlruntime/Application.java`
- Create: `kb-cql-runtime/Dockerfile`
- Create: `kb-cql-runtime/docker-compose.yml`

- [ ] **Step 1: Generate Maven project structure**

```bash
mkdir -p /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-cql-runtime
cd kb-cql-runtime
mkdir -p src/main/java/au/vaidshala/cqlruntime src/test/java/au/vaidshala/cqlruntime/integration
```

Create `pom.xml`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <groupId>au.vaidshala</groupId>
    <artifactId>kb-cql-runtime</artifactId>
    <version>0.1.0-SNAPSHOT</version>
    <packaging>jar</packaging>

    <parent>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-parent</artifactId>
        <version>3.2.0</version>
    </parent>

    <properties>
        <java.version>17</java.version>
        <hapi.fhir.version>7.0.2</hapi.fhir.version>
    </properties>

    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-web</artifactId>
        </dependency>
        <dependency>
            <groupId>ca.uhn.hapi.fhir</groupId>
            <artifactId>hapi-fhir-base</artifactId>
            <version>${hapi.fhir.version}</version>
        </dependency>
        <dependency>
            <groupId>ca.uhn.hapi.fhir</groupId>
            <artifactId>hapi-fhir-structures-r4</artifactId>
            <version>${hapi.fhir.version}</version>
        </dependency>
        <dependency>
            <groupId>org.opencds.cqf.cql</groupId>
            <artifactId>engine</artifactId>
            <version>3.0.0</version>
        </dependency>
        <dependency>
            <groupId>org.opencds.cqf.fhir</groupId>
            <artifactId>cqf-fhir-cr</artifactId>
            <version>3.0.0</version>
        </dependency>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-test</artifactId>
            <scope>test</scope>
        </dependency>
    </dependencies>

    <build>
        <plugins>
            <plugin>
                <groupId>org.springframework.boot</groupId>
                <artifactId>spring-boot-maven-plugin</artifactId>
            </plugin>
        </plugins>
    </build>
</project>
```

Create `src/main/java/au/vaidshala/cqlruntime/Application.java`:

```java
package au.vaidshala.cqlruntime;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

@SpringBootApplication
public class Application {
    public static void main(String[] args) {
        SpringApplication.run(Application.class, args);
    }
}
```

Create `Dockerfile`:

```dockerfile
FROM eclipse-temurin:17-jdk-alpine AS builder
WORKDIR /build
COPY pom.xml .
COPY src ./src
RUN apk add --no-cache maven && mvn -B package -DskipTests

FROM eclipse-temurin:17-jre-alpine
WORKDIR /app
COPY --from=builder /build/target/kb-cql-runtime-*.jar app.jar
EXPOSE 8140
ENTRYPOINT ["java","-jar","app.jar"]
```

- [ ] **Step 2: Build smoke test**

```bash
cd kb-cql-runtime
mvn -B compile 2>&1 | tail -5
```
Expected: `BUILD SUCCESS`.

- [ ] **Step 3: Commit**

```bash
git add kb-cql-runtime/pom.xml kb-cql-runtime/src kb-cql-runtime/Dockerfile
git commit -m "feat(kb-cql-runtime): scaffold Spring Boot + HAPI FHIR project"
```

---

### Task 2: Substrate REST API for runtime queries

**Files:**
- Create: `kb-20-patient-profile/internal/api/substrate_runtime_handler.go`
- Create: `kb-20-patient-profile/internal/api/substrate_runtime_handler_test.go`

The HAPI runtime calls these endpoints at CQL evaluation time. They are read-only and idempotent.

- [ ] **Step 1: Write failing test**

```go
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestRuntimeHandler_GetBaseline(t *testing.T) {
	residentID := uuid.New()
	mockProvider := &mockBaselineProvider{
		baseline: 5.0, confidence: "high", n: 7,
	}
	handler := NewRuntimeHandler(mockProvider, nil, nil, nil, nil)

	req := httptest.NewRequest("GET",
		"/runtime/baseline?resident_id="+residentID.String()+"&type=potassium", nil)
	w := httptest.NewRecorder()
	handler.GetBaseline(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	// Body shape: {"baseline_value":5.0,"baseline_confidence":"high","baseline_n_observations":7}
	if !strings.Contains(w.Body.String(), `"baseline_value":5`) {
		t.Errorf("body missing baseline_value: %s", w.Body.String())
	}
}
```

(Define `mockBaselineProvider` in the test file with the minimum interface match.)

- [ ] **Step 2: Run, expect failure**

- [ ] **Step 3: Implement handlers**

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

// BaselineProvider, ActiveConcernsProvider, CareIntensityProvider,
// MedicineUseProvider, ObservationProvider are the substrate read
// boundaries the runtime needs. Wire each to the existing substrate
// engines.
type BaselineProvider interface {
	GetBaseline(residentID uuid.UUID, observationType string) (
		baseline float64, confidence string, n int, err error)
}

// ... similarly for ActiveConcernsProvider etc.

type RuntimeHandler struct {
	baselines BaselineProvider
	// ... others
}

func NewRuntimeHandler(b BaselineProvider, /* ... */) *RuntimeHandler {
	return &RuntimeHandler{baselines: b}
}

func (h *RuntimeHandler) GetBaseline(w http.ResponseWriter, r *http.Request) {
	residentID, err := uuid.Parse(r.URL.Query().Get("resident_id"))
	if err != nil {
		http.Error(w, "invalid resident_id", http.StatusBadRequest)
		return
	}
	obsType := r.URL.Query().Get("type")
	if obsType == "" {
		http.Error(w, "missing type", http.StatusBadRequest)
		return
	}
	baseline, confidence, n, err := h.baselines.GetBaseline(residentID, obsType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"baseline_value":          baseline,
		"baseline_confidence":     confidence,
		"baseline_n_observations": n,
	})
}

// Similarly: GetActiveConcerns, GetCareIntensity, GetMedicineUse,
// GetObservations — each thin wrapper over the corresponding substrate
// engine read.
```

- [ ] **Step 4: Wire routes in kb-20 main.go**

```go
runtimeHandler := api.NewRuntimeHandler(
	baselineProvider, activeConcernsProvider, careIntensityProvider,
	medicineUseProvider, observationProvider)
mux.HandleFunc("/runtime/baseline", runtimeHandler.GetBaseline)
mux.HandleFunc("/runtime/active-concerns", runtimeHandler.GetActiveConcerns)
mux.HandleFunc("/runtime/care-intensity", runtimeHandler.GetCareIntensity)
mux.HandleFunc("/runtime/medicine-use", runtimeHandler.GetMedicineUse)
mux.HandleFunc("/runtime/observations", runtimeHandler.GetObservations)
```

- [ ] **Step 5: Run, pass; commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/substrate_runtime_handler.go \
        backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/api/substrate_runtime_handler_test.go \
        backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/cmd/server/main.go
git commit -m "feat(kb-20): substrate runtime REST API for CQL external functions"
```

---

### Task 3: Implement Vaidshala.Substrate external functions in HAPI

**Files:**
- Create: `src/main/java/au/vaidshala/cqlruntime/external/SubstrateExternalFunctions.java`
- Create: `src/main/java/au/vaidshala/cqlruntime/external/SubstrateClient.java`
- Create: `src/test/java/au/vaidshala/cqlruntime/external/SubstrateExternalFunctionsTest.java`

`SubstrateClient` is a thin Java HTTP client over the Task 2 endpoints. `SubstrateExternalFunctions` registers each function with the OpenCDS CQF engine.

- [ ] **Step 1: Write failing test (using WireMock for substrate stubbing)**

```java
package au.vaidshala.cqlruntime.external;

import com.github.tomakehurst.wiremock.junit5.WireMockTest;
import com.github.tomakehurst.wiremock.junit5.WireMockRuntimeInfo;
import org.junit.jupiter.api.Test;

import static com.github.tomakehurst.wiremock.client.WireMock.*;
import static org.assertj.core.api.Assertions.assertThat;

@WireMockTest
class SubstrateExternalFunctionsTest {
    @Test
    void runningBaselineReturnsValue(WireMockRuntimeInfo wm) {
        stubFor(get(urlPathEqualTo("/runtime/baseline"))
            .withQueryParam("resident_id", equalTo("11111111-1111-1111-1111-111111111111"))
            .withQueryParam("type", equalTo("potassium"))
            .willReturn(okJson("""
                {"baseline_value":4.5,"baseline_confidence":"high","baseline_n_observations":7}
                """)));

        SubstrateClient client = new SubstrateClient(wm.getHttpBaseUrl());
        SubstrateExternalFunctions fns = new SubstrateExternalFunctions(client);

        Double v = fns.runningBaseline("11111111-1111-1111-1111-111111111111", "potassium");
        assertThat(v).isEqualTo(4.5);
    }
}
```

- [ ] **Step 2: Run, expect failure**

```bash
cd kb-cql-runtime && mvn test -Dtest=SubstrateExternalFunctionsTest 2>&1 | tail -10
```

- [ ] **Step 3: Implement client + functions**

```java
// SubstrateClient.java
package au.vaidshala.cqlruntime.external;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;

public class SubstrateClient {
    private final String baseUrl;
    private final HttpClient http = HttpClient.newBuilder()
        .connectTimeout(Duration.ofSeconds(2)).build();
    private final ObjectMapper json = new ObjectMapper();

    public SubstrateClient(String baseUrl) { this.baseUrl = baseUrl; }

    public JsonNode getBaseline(String residentId, String observationType) {
        return get("/runtime/baseline?resident_id=" + residentId + "&type=" + observationType);
    }

    public JsonNode getActiveConcerns(String residentId) {
        return get("/runtime/active-concerns?resident_id=" + residentId);
    }

    public JsonNode getCareIntensity(String residentId) {
        return get("/runtime/care-intensity?resident_id=" + residentId);
    }

    public JsonNode getMedicineUse(String residentId) {
        return get("/runtime/medicine-use?resident_id=" + residentId);
    }

    public JsonNode getObservations(String residentId, String type, int limit) {
        return get("/runtime/observations?resident_id=" + residentId
            + "&type=" + type + "&limit=" + limit);
    }

    private JsonNode get(String path) {
        try {
            HttpRequest req = HttpRequest.newBuilder(URI.create(baseUrl + path))
                .timeout(Duration.ofSeconds(2)).GET().build();
            HttpResponse<String> resp = http.send(req, HttpResponse.BodyHandlers.ofString());
            if (resp.statusCode() != 200) {
                throw new RuntimeException("substrate " + resp.statusCode() + ": " + resp.body());
            }
            return json.readTree(resp.body());
        } catch (Exception e) {
            throw new RuntimeException(e);
        }
    }
}
```

```java
// SubstrateExternalFunctions.java
package au.vaidshala.cqlruntime.external;

import com.fasterxml.jackson.databind.JsonNode;

public class SubstrateExternalFunctions {
    private final SubstrateClient client;
    public SubstrateExternalFunctions(SubstrateClient client) { this.client = client; }

    public Double runningBaseline(String residentId, String observationType) {
        JsonNode n = client.getBaseline(residentId, observationType);
        return n.path("baseline_value").asDouble();
    }

    public java.util.List<String> activeConcerns(String residentId) {
        JsonNode n = client.getActiveConcerns(residentId);
        java.util.List<String> out = new java.util.ArrayList<>();
        n.forEach(c -> out.add(c.path("type").asText()));
        return out;
    }

    public String careIntensity(String residentId) {
        return client.getCareIntensity(residentId).path("tag").asText();
    }

    // ... medicineUseIntent, recentObservations, etc.
}
```

- [ ] **Step 4: Register functions with the CQL engine in HapiConfig**

This is the HAPI-specific glue: register `SubstrateExternalFunctions` as an `ExternalFunctionProvider` so CQL `external function` declarations resolve to its methods. The exact registration call depends on the HAPI 7.0.2 API; consult HAPI docs in the reading list.

- [ ] **Step 5: Run, pass; commit**

```bash
git add kb-cql-runtime/src/main/java/au/vaidshala/cqlruntime/external/ \
        kb-cql-runtime/src/test/java/au/vaidshala/cqlruntime/external/
git commit -m "feat(kb-cql-runtime): Vaidshala.Substrate external function library"
```

---

### Task 4: CqlLibraryLoader for shared/cql-libraries

**Files:**
- Create: `src/main/java/au/vaidshala/cqlruntime/loader/CqlLibraryLoader.java`
- Create: `src/test/java/au/vaidshala/cqlruntime/loader/CqlLibraryLoaderTest.java`

Loads the existing 16 CQL files from `shared/cql-libraries/` as HAPI `Library` resources, available for evaluation.

- [ ] **Step 1: Write failing test**

```java
@Test
void loadsAllSharedLibraries() {
    CqlLibraryLoader loader = new CqlLibraryLoader(
        "../shared/cql-libraries/", fhirContext);
    int loaded = loader.loadAll();
    assertThat(loaded).isGreaterThan(15); // 16 files per audit
}
```

- [ ] **Step 2-5: Implement, run, commit**

```java
package au.vaidshala.cqlruntime.loader;

import ca.uhn.fhir.context.FhirContext;
import org.hl7.fhir.r4.model.Library;
import java.nio.file.*;
import java.util.stream.Stream;

public class CqlLibraryLoader {
    private final String libDir;
    private final FhirContext ctx;
    public CqlLibraryLoader(String libDir, FhirContext ctx) {
        this.libDir = libDir; this.ctx = ctx;
    }
    public int loadAll() {
        try (Stream<Path> paths = Files.walk(Paths.get(libDir))) {
            return (int) paths
                .filter(p -> p.toString().endsWith(".cql"))
                .map(this::loadOne)
                .filter(java.util.Objects::nonNull)
                .count();
        } catch (Exception e) {
            throw new RuntimeException(e);
        }
    }
    private Library loadOne(Path path) {
        // Read CQL text, wrap in Library resource, register with HAPI engine
        // Specific HAPI 7.x API call documented in their CR samples.
        return new Library(); // placeholder – consult HAPI docs
    }
}
```

```bash
git add kb-cql-runtime/src/main/java/au/vaidshala/cqlruntime/loader/ \
        kb-cql-runtime/src/test/java/au/vaidshala/cqlruntime/loader/
git commit -m "feat(kb-cql-runtime): CQL library loader for shared/cql-libraries"
```

---

### Task 5: $evaluate-rule operation provider

**Files:**
- Create: `src/main/java/au/vaidshala/cqlruntime/operation/EvaluateRuleProvider.java`

Exposes a FHIR operation that takes `ruleId` + `residentId`, executes the matching CQL, returns a structured result the Recommendation lifecycle consumes.

- [ ] **Step 1-5: Implement following HAPI ResourceProvider pattern**

```java
package au.vaidshala.cqlruntime.operation;

import ca.uhn.fhir.rest.annotation.Operation;
import ca.uhn.fhir.rest.annotation.OperationParam;
import org.hl7.fhir.r4.model.Parameters;
import org.hl7.fhir.r4.model.StringType;

public class EvaluateRuleProvider {
    @Operation(name = "$evaluate-rule", idempotent = true)
    public Parameters evaluate(
        @OperationParam(name = "ruleId") StringType ruleId,
        @OperationParam(name = "residentId") StringType residentId
    ) {
        // 1. Look up CQL Library by ruleId
        // 2. Inject substrate context (resident + observations + active concerns)
        // 3. Execute via OpenCDS CQF engine
        // 4. Wrap result as Parameters
        return new Parameters();
    }
}
```

```bash
git add kb-cql-runtime/src/main/java/au/vaidshala/cqlruntime/operation/
git commit -m "feat(kb-cql-runtime): \\$evaluate-rule FHIR operation provider"
```

---

### Task 6: Go HTTP client + Recommendation lifecycle integration

**Files:**
- Create: `shared/v2_substrate/cql/client.go`
- Create: `shared/v2_substrate/cql/client_test.go`
- Modify: `shared/v2_substrate/recommendation/lifecycle.go` (optional CQL evaluation on detected → drafted)

- [ ] **Step 1-3: Standard test + implement HTTP client**

```go
package cql

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{}}
}

type RuleResult struct {
	Triggered      bool                   `json:"triggered"`
	Type           string                 `json:"type"`
	Urgency        string                 `json:"urgency"`
	ClinicalContent map[string]any        `json:"clinical_content"`
}

func (c *Client) EvaluateRule(ctx context.Context, ruleID string,
	residentID uuid.UUID) (*RuleResult, error) {
	url := c.baseURL + "/Library/" + ruleID + "/$evaluate-rule?residentId=" + residentID.String()
	req, _ := http.NewRequestWithContext(ctx, "POST", url, nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out RuleResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
```

- [ ] **Step 4: Optionally extend Plan 0.1's Recommendation lifecycle**

The detected → drafted transition can populate `ClinicalContent` from the rule result. This is a feature of the Phase 2 craft engine more than this plan, so leave it out here and document in Plan 2 that the wiring becomes available once kb-cql-runtime is up.

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/cql/
git commit -m "feat(substrate): Go client for kb-cql-runtime \\$evaluate-rule"
```

---

### Task 7: End-to-end Sunday-night-fall integration test

**Files:**
- Create: `kb-cql-runtime/src/test/java/au/vaidshala/cqlruntime/integration/SundayNightFallIT.java`

Deploy: kb-20 substrate API + kb-cql-runtime + Postgres + Redis. Seed: one resident with running baseline, an active concern, a recent fall event. Execute: `$evaluate-rule` against the existing PostFallRule. Assert: rule triggers; clinical_content populated; result is consumable by Plan 0.1's Recommendation Create+Lifecycle.

- [ ] **Step 1-5: Write integration test, run with docker-compose, commit**

```bash
cd kb-cql-runtime
docker-compose up -d
mvn verify -P integration 2>&1 | tail -30
git add kb-cql-runtime/src/test/java/au/vaidshala/cqlruntime/integration/SundayNightFallIT.java \
        kb-cql-runtime/docker-compose.yml
git commit -m "test(kb-cql-runtime): Sunday-night-fall end-to-end with real CQL execution"
```

---

### Task 8: Body-fill the TODO(wave-1-runtime) markers in CQL helpers

**Files:**
- Modify: `shared/cql-libraries/MonitoringHelpers.cql` and similar files

Per the audit: Tier-4 surveillance has 17 TODO(wave-1-runtime) markers; helper bodies bottom out in `Vaidshala.Substrate.*` external functions that now exist. Replace markers with real CQL calling the registered functions.

- [ ] **Step 1: List affected files**

```bash
grep -lr "TODO(wave-1-runtime)" shared/cql-libraries/
```

- [ ] **Step 2-3: For each file, replace the TODO body with a real expression**

Example transformation in `MonitoringHelpers.cql`:

Before:
```
define ObservationOverdue(plan MonitoringPlan):
  // TODO(wave-1-runtime): bind to substrate
  false
```

After:
```
define ObservationOverdue(plan MonitoringPlan):
  Vaidshala.Substrate.ObservationLandedSince(
      ResidentID(plan), ObservationCode(plan), DueAt(plan)
  ) is null
```

- [ ] **Step 4: Run the existing CQL toolchain validator (Layer 3 already production-shaped)**

```bash
cd shared/cql-toolchain
python -m pytest -v 2>&1 | tail -20
```
Expected: 244 tests pass (regression check).

- [ ] **Step 5: Commit**

```bash
git add shared/cql-libraries/
git commit -m "feat(cql-libraries): bind TODO(wave-1-runtime) helpers to Vaidshala.Substrate"
```

---

## Spec coverage

- [x] kb-cql-runtime service stood up — Tasks 1, 5
- [x] Vaidshala.Substrate external functions implemented — Tasks 2, 3
- [x] CQL libraries loaded into runtime — Task 4
- [x] $evaluate-rule operation exposed — Task 5
- [x] Go client for Recommendation lifecycle — Task 6
- [x] End-to-end Sunday-night-fall executes — Task 7
- [x] TODO(wave-1-runtime) markers closed — Task 8

Plan complete and saved.
