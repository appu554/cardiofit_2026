# KB-7 Knowledge Factory - Australia Edition

Australian terminology kernel builder for the CardioFit Clinical Synthesis Hub.

## Overview

This repository builds the Australian knowledge kernel (`kb7-kernel-au.ttl`) containing:

| Terminology | Module ID | Description |
|-------------|-----------|-------------|
| **SNOMED CT-AU** | 32506021000036107 | Australian SNOMED CT Extension |
| **AMT** | 900062011000036103 | Australian Medicines Terminology |
| **LOINC** | N/A | International Laboratory Codes |

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  GCP Cloud Workflow (australia-southeast1)                      │
│  ├── kb7-snomed-au-job-production (download from NTS)           │
│  └── kb7-github-dispatcher-au-job (trigger this workflow)       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  GitHub Actions Pipeline (8 Stages)                              │
│  ┌─────────┐ ┌───────────┐ ┌───────┐ ┌───────────┐              │
│  │Download │→│ Transform │→│ Merge │→│ Reasoning │              │
│  └─────────┘ └───────────┘ └───────┘ └───────────┘              │
│       │                                     │                    │
│       ▼                                     ▼                    │
│  ┌────────────┐ ┌─────────┐ ┌────────┐ ┌────────┐               │
│  │ Validation │→│ Package │→│ Upload │→│ Deploy │               │
│  └────────────┘ └─────────┘ └────────┘ └────────┘               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  Output: kb7-kernel-au.ttl                                       │
│  ├── gs://kb-artifacts/au/latest/                               │
│  └── Neo4j (kb7-au database)                                    │
└─────────────────────────────────────────────────────────────────┘
```

## Pipeline Stages

| Stage | Description | Output |
|-------|-------------|--------|
| 1. Download | Fetch AU sources from GCS | `sources/*.zip` |
| 2. Transform | Convert RF2 to OWL (SNOMED-OWL-Toolkit) | `*.owl`, `*.ttl` |
| 3. Merge | Combine ontologies (ROBOT) | `kb7-merged.ttl` |
| 4. Reasoning | Download pre-computed ELK reasoning | `kb7-inferred.ttl` |
| 5. Validation | Quality gates and namespace checks | Validation report |
| 6. Package | Create versioned kernel + manifest | `kb7-kernel-au.ttl` |
| 7. Upload | Push to GCS (versioned + latest) | GCS paths |
| 8. Deploy | Import to Neo4j (optional) | Database updated |

## Source Data

### SNOMED CT-AU
- **Source**: Australian National Terminology Service (NTS)
- **URL**: https://api.healthterminologies.gov.au
- **Authentication**: OAuth2 (client credentials)
- **Package**: SNOMEDCT-AU RF2 SNAPSHOT
- **Includes**: SNOMED International + AU Extension + AMT

### LOINC
- **Source**: Regenstrief Institute
- **Format**: CSV files
- **Shared**: Same as US kernel (international standard)

## Docker Images

| Image | Purpose |
|-------|---------|
| `snomed-toolkit` | RF2 to OWL conversion (SNOMED-OWL-Toolkit v5.3.0) |
| `robot` | Ontology operations (ROBOT v1.9.5) |
| `converters` | LOINC CSV to RDF conversion |

## Usage

### Manual Trigger
```bash
gh workflow run kb-factory-au.yml
```

### Scheduled
- **Wednesday 2 AM AEST** (Tuesday 4 PM UTC)

### GCP Trigger
```bash
gcloud workflows execute kb7-factory-multiregion-workflow-production \
  --location=us-central1 \
  --data='{"region":"au"}'
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GCS_BUCKET_SOURCES` | Source files bucket |
| `GCS_BUCKET_ARTIFACTS` | Output artifacts bucket |
| `GCP_PROJECT_ID` | GCP project ID |
| `REGION` | Fixed to `au` |

## Secrets Required

| Secret | Description |
|--------|-------------|
| `GCS_SERVICE_ACCOUNT_KEY` | GCP service account JSON |
| `NEO4J_URL_AU` | AU Neo4j connection URL |
| `NEO4J_PASSWORD_AU` | AU Neo4j password |

## Output Paths

```
gs://kb-artifacts-production/
├── au/
│   ├── 20251206/                    # Versioned
│   │   ├── kb7-kernel-au.ttl
│   │   └── kb7-manifest.json
│   └── latest/                      # Latest pointer
│       ├── kb7-kernel-au.ttl
│       └── kb7-manifest.json
└── local-reasoning/
    └── kb7-inferred.ttl             # Pre-computed reasoning
```

## Manifest Example

```json
{
  "version": "20251206",
  "region": "au",
  "region_display": "Australia",
  "terminologies": {
    "snomed_au": {
      "version": "20251130",
      "module_id": "32506021000036107"
    },
    "amt": {
      "version": "20251130",
      "module_id": "900062011000036103"
    },
    "loinc": {
      "version": "2.77"
    }
  }
}
```

## Related Repositories

- **knowledge-factory** (US): SNOMED-US + RxNorm + LOINC
- **knowledge-factory-au** (AU): SNOMED-AU + AMT + LOINC (this repo)
- **knowledge-factory-in** (IN): SNOMED-IN + CDCI + LOINC (future)

## License

Proprietary - CardioFit Clinical Synthesis Hub
