# KB-7 Multi-Region Architecture Plan

## Option A: Separate Regional Kernels

**Decision Date**: 2025-12-06
**Status**: Implementation Ready

---

## 1. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      GCS Source Buckets                                  │
│  ┌──────────────────┬──────────────────┬──────────────────┐            │
│  │ gs://kb-sources/ │ gs://kb-sources/ │ gs://kb-sources/ │            │
│  │      au/         │      in/         │      us/         │            │
│  │  ├── amt/        │  ├── cdci/       │  ├── rxnorm/     │            │
│  │  └── snomed-au/  │  └── snomed-in/  │  ├── snomed-us/  │            │
│  │                  │                  │  └── loinc/      │            │
│  └──────────────────┴──────────────────┴──────────────────┘            │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    GitHub Actions Pipeline                               │
│         kb-factory.yml (parameterized with region input)                │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  workflow_dispatch:                                              │   │
│  │    inputs:                                                       │   │
│  │      region: [au, in, us]                                       │   │
│  │      source_type: [snomed, amt, cdci, rxnorm, loinc]           │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  Stage 1: Download → Stage 2: Transform → Stage 3: Merge               │
│  Stage 4: Reason → Stage 5: Validate → Stage 6: Package                │
│  Stage 7: Upload → Stage 8: Deploy (CDC notification)                  │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      GCS Artifact Buckets                                │
│  ┌──────────────────┬──────────────────┬──────────────────┐            │
│  │ gs://kb-artifacts│ gs://kb-artifacts│ gs://kb-artifacts│            │
│  │      /au/        │      /in/        │      /us/        │            │
│  │  kb7-kernel-     │  kb7-kernel-     │  kb7-kernel-     │            │
│  │  au.ttl          │  in.ttl          │  us.ttl          │            │
│  └──────────────────┴──────────────────┴──────────────────┘            │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      Neo4j Databases                                     │
│  ┌──────────────────┬──────────────────┬──────────────────┐            │
│  │   kb7-au         │   kb7-in         │   kb7-us         │            │
│  │ (AMT + SNOMED-AU)│ (CDCI + SNOMED-IN)│ (RxNorm+SNOMED-US│            │
│  │                  │                  │  +LOINC)         │            │
│  └──────────────────┴──────────────────┴──────────────────┘            │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      KB-7 Terminology API                                │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Request Header: X-Region: au | in | us                         │   │
│  │  → Routes to appropriate Neo4j database                         │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Regional Source Configurations

### 2.1 Australia (AU)
| Source | Format | Module ID | Download URL |
|--------|--------|-----------|--------------|
| SNOMED CT-AU | RF2 | 32506021000036107 | TGA MIMS Portal |
| AMT | RF2 | 900062011000036103 | TGA MIMS Portal |

**Transformer**: `snomed-toolkit` (same as SNOMED International)
**Relationship**: `rdfs:subClassOf` (universal SNOMED path)

### 2.2 India (IN)
| Source | Format | Module ID | Download URL |
|--------|--------|-----------|--------------|
| SNOMED CT-IN | RF2 | TBD | CDSCO Portal |
| CDCI | RF2 | TBD | CDSCO Portal |

**Transformer**: `snomed-toolkit`
**Relationship**: `rdfs:subClassOf`

### 2.3 USA (US)
| Source | Format | Download URL |
|--------|--------|--------------|
| SNOMED CT-US | RF2 | NLM UMLS |
| RxNorm | RRF | NLM UMLS |
| LOINC | CSV | Regenstrief |

**Transformers**:
- `snomed-toolkit` for SNOMED
- `transform-rxnorm.py` for RxNorm
- `transform-loinc.py` for LOINC

---

## 3. GCS Bucket Structure

### 3.1 Source Buckets
```
gs://{PROJECT_ID}-kb-sources-{ENV}/
├── au/
│   ├── amt/
│   │   ├── SnomedCT_Release_AU1000036_YYYYMMDD.zip
│   │   └── amt_version.txt
│   └── snomed-au/
│       ├── SnomedCT_InternationalRF2_PRODUCTION_YYYYMMDD.zip
│       └── snomed-au_version.txt
├── in/
│   ├── cdci/
│   │   ├── CDCI_Release_YYYYMMDD.zip
│   │   └── cdci_version.txt
│   └── snomed-in/
│       ├── SnomedCT_IndiaRF2_YYYYMMDD.zip
│       └── snomed-in_version.txt
└── us/
    ├── rxnorm/
    │   ├── RxNorm_full_MMDDYYYY.zip
    │   └── rxnorm_version.txt
    ├── snomed-us/
    │   ├── SnomedCT_USEditionRF2_PRODUCTION_YYYYMMDD.zip
    │   └── snomed-us_version.txt
    └── loinc/
        ├── Loinc_X.XX.zip
        └── loinc_version.txt
```

### 3.2 Artifact Buckets
```
gs://{PROJECT_ID}-kb-artifacts-{ENV}/
├── au/
│   ├── latest/
│   │   ├── kb7-kernel-au.ttl
│   │   ├── kb7-kernel-au.ttl.sha256
│   │   ├── manifest.json
│   │   └── versions.json
│   └── archive/
│       └── YYYY-MM-DD/
│           └── kb7-kernel-au.ttl
├── in/
│   ├── latest/
│   │   ├── kb7-kernel-in.ttl
│   │   └── ...
│   └── archive/
└── us/
    ├── latest/
    │   ├── kb7-kernel-us.ttl
    │   └── ...
    └── archive/
```

---

## 4. GitHub Actions Workflow Updates

### 4.1 Parameterized Workflow
```yaml
# .github/workflows/kb-factory.yml
name: KB-7 Factory Pipeline

on:
  workflow_dispatch:
    inputs:
      region:
        description: 'Target region'
        required: true
        type: choice
        options:
          - au
          - in
          - us
      full_rebuild:
        description: 'Force full rebuild'
        type: boolean
        default: false
  schedule:
    # AU: Wednesday 2 AM AEST (Tuesday 4 PM UTC)
    - cron: '0 16 * * 2'
    # IN: Thursday 2 AM IST (Wednesday 8:30 PM UTC)
    - cron: '30 20 * * 3'
    # US: Monday 2 AM EST (Monday 7 AM UTC)
    - cron: '0 7 * * 1'

env:
  REGION: ${{ github.event.inputs.region || 'us' }}
  GCS_SOURCE_BUCKET: ${{ secrets.GCP_PROJECT_ID }}-kb-sources-production
  GCS_ARTIFACT_BUCKET: ${{ secrets.GCP_PROJECT_ID }}-kb-artifacts-production

jobs:
  detect-region:
    runs-on: ubuntu-latest
    outputs:
      region: ${{ steps.detect.outputs.region }}
    steps:
      - id: detect
        run: |
          if [ "${{ github.event_name }}" == "schedule" ]; then
            HOUR=$(date -u +%H)
            DAY=$(date -u +%u)
            if [ "$DAY" == "2" ] && [ "$HOUR" == "16" ]; then
              echo "region=au" >> $GITHUB_OUTPUT
            elif [ "$DAY" == "3" ] && [ "$HOUR" == "20" ]; then
              echo "region=in" >> $GITHUB_OUTPUT
            else
              echo "region=us" >> $GITHUB_OUTPUT
            fi
          else
            echo "region=${{ github.event.inputs.region }}" >> $GITHUB_OUTPUT
          fi

  stage-1-download:
    needs: detect-region
    runs-on: ubuntu-latest
    env:
      REGION: ${{ needs.detect-region.outputs.region }}
    steps:
      - name: Download regional sources
        run: |
          gsutil -m cp -r gs://${GCS_SOURCE_BUCKET}/${REGION}/* /input/

  # ... remaining stages with REGION environment variable
```

### 4.2 Region-Specific Source Downloads
```yaml
# Stage 1: Download (region-aware)
stage-1-download:
  steps:
    - name: Download AU sources
      if: env.REGION == 'au'
      run: |
        gsutil cp gs://${GCS_SOURCE_BUCKET}/au/amt/*.zip /input/amt/
        gsutil cp gs://${GCS_SOURCE_BUCKET}/au/snomed-au/*.zip /input/snomed/

    - name: Download IN sources
      if: env.REGION == 'in'
      run: |
        gsutil cp gs://${GCS_SOURCE_BUCKET}/in/cdci/*.zip /input/cdci/
        gsutil cp gs://${GCS_SOURCE_BUCKET}/in/snomed-in/*.zip /input/snomed/

    - name: Download US sources
      if: env.REGION == 'us'
      run: |
        gsutil cp gs://${GCS_SOURCE_BUCKET}/us/rxnorm/*.zip /input/rxnorm/
        gsutil cp gs://${GCS_SOURCE_BUCKET}/us/snomed-us/*.zip /input/snomed/
        gsutil cp gs://${GCS_SOURCE_BUCKET}/us/loinc/*.zip /input/loinc/
```

---

## 5. Docker Image Updates

### 5.1 snomed-toolkit (No Changes Needed)
AMT and CDCI use RF2 format - same as SNOMED International.
The existing snomed-toolkit container works for all regions.

### 5.2 converters (Region-Aware)
```dockerfile
# Dockerfile.converters
FROM python:3.11-slim

# Install dependencies
COPY requirements.txt .
RUN pip install -r requirements.txt

# Copy all converters
COPY scripts/transform-rxnorm.py /app/
COPY scripts/transform-loinc.py /app/
COPY scripts/transform-amt.py /app/      # Optional: AMT-specific if needed
COPY scripts/transform-cdci.py /app/     # Optional: CDCI-specific if needed

ENV REGION=us
WORKDIR /app
```

### 5.3 robot (No Changes Needed)
ELK reasoner works on any OWL ontology regardless of region.

---

## 6. Neo4j Client Updates

### 6.1 Region-Aware Database Selection
```go
// internal/repository/neo4j_client.go

type Neo4jClient struct {
    drivers   map[string]neo4j.Driver  // region -> driver
    defaultDB string
}

func NewNeo4jClient(cfg *config.Config) (*Neo4jClient, error) {
    client := &Neo4jClient{
        drivers:   make(map[string]neo4j.Driver),
        defaultDB: cfg.DefaultRegion,
    }

    // Initialize drivers for each region
    for region, dbConfig := range cfg.Neo4jRegions {
        driver, err := neo4j.NewDriver(
            dbConfig.URI,
            neo4j.BasicAuth(dbConfig.User, dbConfig.Password, ""),
        )
        if err != nil {
            return nil, fmt.Errorf("failed to create driver for %s: %w", region, err)
        }
        client.drivers[region] = driver
    }

    return client, nil
}

func (c *Neo4jClient) getDriver(region string) neo4j.Driver {
    if driver, ok := c.drivers[region]; ok {
        return driver
    }
    return c.drivers[c.defaultDB]  // fallback to default
}

func (c *Neo4jClient) getDatabase(region string) string {
    switch region {
    case "au":
        return "kb7-au"
    case "in":
        return "kb7-in"
    case "us":
        return "kb7-us"
    default:
        return "kb7-us"
    }
}
```

### 6.2 Remove RxNorm-Specific Code Paths
The current neo4j_client.go has RxNorm-specific relationship queries using `ns1__rxnorm_RB` and `ns1__rxnorm_RN`. These must be removed since:

1. **AMT/CDCI use `rdfs:subClassOf`** (same as SNOMED)
2. **RxNorm branch creates inconsistency** between regions
3. **Universal SNOMED path works for all RF2-based terminologies**

**Files to Modify**:
- `internal/repository/neo4j_client.go`:
  - Lines 334-347: Remove RxNorm IsSubsumedBy branch
  - Lines 409-423: Remove RxNorm GetAncestors branch
  - Lines 458-472: Remove RxNorm GetDescendants branch

**New Universal Query** (works for SNOMED, AMT, CDCI):
```cypher
MATCH path = (child:Resource)-[:rdfs__subClassOf*1..]->(parent:Resource)
WHERE child.uri = $childUri AND parent.uri = $parentUri
RETURN path IS NOT NULL AS isSubsumed
```

---

## 7. API Updates

### 7.1 Region Header Handling
```go
// internal/handlers/terminology_handler.go

func (h *Handler) getRegion(c *gin.Context) string {
    region := c.GetHeader("X-Region")
    if region == "" {
        region = c.Query("region")
    }
    if region == "" {
        region = "us"  // default
    }
    return strings.ToLower(region)
}

func (h *Handler) ValidateCode(c *gin.Context) {
    region := h.getRegion(c)

    // Route to region-specific database
    result, err := h.service.ValidateCode(c.Request.Context(), region, req)
    // ...
}
```

### 7.2 Updated Service Interface
```go
// internal/services/terminology_service.go

type TerminologyService interface {
    ValidateCode(ctx context.Context, region string, code, system string) (*ValidationResult, error)
    LookupCode(ctx context.Context, region string, code, system string) (*Concept, error)
    Translate(ctx context.Context, region string, code, sourceSystem, targetSystem string) (*Concept, error)
    IsSubsumedBy(ctx context.Context, region string, childCode, parentCode, system string) (bool, error)
    GetAncestors(ctx context.Context, region string, code, system string) ([]*Concept, error)
    GetDescendants(ctx context.Context, region string, code, system string) ([]*Concept, error)
}
```

---

## 8. Configuration Updates

### 8.1 config.yaml
```yaml
# config/config.yaml
server:
  port: 8092
  default_region: us

neo4j:
  regions:
    au:
      uri: bolt://neo4j-au:7687
      user: neo4j
      password: ${NEO4J_AU_PASSWORD}
      database: kb7-au
    in:
      uri: bolt://neo4j-in:7687
      user: neo4j
      password: ${NEO4J_IN_PASSWORD}
      database: kb7-in
    us:
      uri: bolt://neo4j-us:7687
      user: neo4j
      password: ${NEO4J_US_PASSWORD}
      database: kb7-us

terminology:
  systems:
    au:
      - name: SNOMED CT-AU
        uri: http://snomed.info/sct/32506021000036107
      - name: AMT
        uri: http://snomed.info/sct/900062011000036103
    in:
      - name: SNOMED CT-IN
        uri: http://snomed.info/sct/TBD
      - name: CDCI
        uri: http://snomed.info/sct/TBD
    us:
      - name: SNOMED CT-US
        uri: http://snomed.info/sct
      - name: RxNorm
        uri: http://www.nlm.nih.gov/research/umls/rxnorm
      - name: LOINC
        uri: http://loinc.org

cache:
  redis:
    addr: redis:6379
    db: 0
    ttl: 24h
```

---

## 9. Implementation Tasks

### Phase 1: Infrastructure (Day 1-2)
- [ ] Create regional GCS bucket structure
- [ ] Set up Neo4j databases (kb7-au, kb7-in, kb7-us)
- [ ] Configure Terraform for multi-region infrastructure

### Phase 2: Pipeline Updates (Day 3-4)
- [ ] Update kb-factory.yml with region parameter
- [ ] Create region-specific download stages
- [ ] Test pipeline for each region

### Phase 3: API Updates (Day 5-6)
- [ ] Update neo4j_client.go with region-aware routing
- [ ] Remove RxNorm-specific code paths
- [ ] Add X-Region header handling
- [ ] Update service interface signatures

### Phase 4: Testing (Day 7-8)
- [ ] Unit tests for region routing
- [ ] Integration tests for each region
- [ ] E2E tests for cross-region queries

### Phase 5: Documentation (Day 9)
- [ ] API documentation for region header
- [ ] Deployment guide for regional instances
- [ ] Runbook for regional updates

---

## 10. Rollout Strategy

### 10.1 Phased Deployment
1. **Week 1**: Deploy US region (existing, validate no regression)
2. **Week 2**: Deploy AU region (AMT + SNOMED-AU)
3. **Week 3**: Deploy IN region (CDCI + SNOMED-IN)
4. **Week 4**: Enable cross-region features

### 10.2 Feature Flags
```yaml
feature_flags:
  multi_region_enabled: true
  regions:
    au: true   # Enable after Week 2
    in: false  # Enable after Week 3
    us: true   # Already enabled
```

---

## 11. Monitoring & Observability

### 11.1 Metrics per Region
```
kb7_query_duration_seconds{region="au|in|us"}
kb7_cache_hit_ratio{region="au|in|us"}
kb7_concepts_total{region="au|in|us"}
kb7_pipeline_duration_seconds{region="au|in|us"}
```

### 11.2 Alerts
- Pipeline failure per region
- Query latency threshold per region
- Database connectivity per region
- Cache hit ratio degradation per region

---

## Appendix A: Terminology System URIs

| Region | System | URI |
|--------|--------|-----|
| AU | SNOMED CT-AU | http://snomed.info/sct/32506021000036107 |
| AU | AMT | http://snomed.info/sct/900062011000036103 |
| IN | SNOMED CT-IN | http://snomed.info/sct/TBD |
| IN | CDCI | http://snomed.info/sct/TBD |
| US | SNOMED CT-US | http://snomed.info/sct |
| US | RxNorm | http://www.nlm.nih.gov/research/umls/rxnorm |
| US | LOINC | http://loinc.org |

## Appendix B: RF2 Module IDs

| Terminology | Module ID | Description |
|-------------|-----------|-------------|
| SNOMED International | 900000000000207008 | Core module |
| SNOMED CT-AU | 32506021000036107 | Australian extension |
| AMT | 900062011000036103 | Australian Medicines Terminology |
| SNOMED CT-US | 731000124108 | US National extension |
