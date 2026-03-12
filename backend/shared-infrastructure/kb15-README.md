# KB-14: Documentation Intelligence Service

**Smart Clinical Documentation with Auto-Population, Validation, and E/M Calculation**

## CTO/CMO Deliberation: Why a Separate Service?

### Architectural Rationale

| Factor | KB-10 (Clinical NLP) | KB-14 (Doc Intelligence) |
|--------|---------------------|--------------------------|
| **Primary Function** | READS documents → extracts structured data | WRITES documents → generates templates |
| **Optimization** | CPU-intensive (NLP processing) | I/O-intensive (EHR queries) |
| **Update Cycle** | NLP model improvements | Regulatory/billing changes (E/M 2021) |
| **Scaling** | Scale for processing load | Scale for concurrent users |

### Clinical Rationale

1. **Documentation Burden Crisis**: Physicians spend 2+ hours daily on documentation
2. **Compliance Requirements**: E/M guidelines, Meaningful Use, quality measures
3. **Revenue Integrity**: Proper documentation supports accurate coding
4. **Quality of Care**: Complete documentation enables care continuity

### Engine Utilization Matrix

| Engine | KB-10 (NLP) | KB-14 (Doc Intel) |
|--------|-------------|-------------------|
| **AI Scribe** | Parse dictation | Generate structured note |
| **CDI Engine** | Analyze existing docs | ❌ Not needed |
| **CDSS** | ❌ Not needed | Generate care summaries |
| **Conditions Advisor** | ❌ Not needed | Patient education materials |

## Features

### Smart Templates
- 13 built-in document templates
- 11 document types (Progress Note, H&P, Discharge Summary, etc.)
- 10+ encounter types (Office Visit, Inpatient, ED, etc.)
- Conditional sections based on context
- SOAP format with comprehensive fields

### Auto-Population
- FHIR R4 data retrieval
- Auto-populate from: Conditions, Medications, Allergies, Vitals, Procedures
- Configurable data sources
- Narrative and structured formats

### Completeness Validation
- Required section checking
- Required field validation
- Completeness score calculation
- Missing element identification

### Billing Validation (2021 E/M Guidelines)
- MDM-based E/M calculation
- Time-based E/M calculation
- Diagnosis specificity checking
- E/M gap identification
- Suggested CPT codes

### Macros (Smartphrases)
- 44+ built-in clinical macros
- 7 macro categories
- Variable substitution support
- Custom macro creation

### Quality Measure Support
- Documentation gap identification
- Measure element tracking
- Quality score calculation

## API Endpoints

### Templates
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/templates` | List all templates |
| GET | `/api/v1/templates/{id}` | Get template by ID |
| GET | `/api/v1/templates/search?q=` | Search templates |
| GET | `/api/v1/templates/types` | List document types |
| GET | `/api/v1/templates/encounters` | List encounter types |

### Documents
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/documents` | List documents |
| GET | `/api/v1/documents/{id}` | Get document by ID |
| POST | `/api/v1/documents/create` | Create document from template |
| POST | `/api/v1/documents/save` | Save document progress |
| POST | `/api/v1/documents/sign` | Sign document |

### Auto-Population
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/autopopulate` | Auto-populate document |
| POST | `/api/v1/autopopulate/preview` | Preview available data |

### Validation
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/validate` | Full validation |
| POST | `/api/v1/validate/completeness` | Completeness only |
| POST | `/api/v1/validate/billing` | Billing validation |
| POST | `/api/v1/validate/quality` | Quality measures |
| POST | `/api/v1/validate/em` | E/M calculation |

### Macros
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/macros` | List macros |
| GET | `/api/v1/macros/{id}` | Get macro by ID |
| POST | `/api/v1/macros` | Create custom macro |
| POST | `/api/v1/macros/expand` | Expand macro |
| GET | `/api/v1/macros/search?q=` | Search macros |
| GET | `/api/v1/macros/categories` | List categories |

### FHIR
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/fhir/composition` | Generate FHIR Composition |
| GET | `/api/v1/fhir/documentreference` | Generate DocumentReference |

## Document Templates

| Template | Document Type | Use Case |
|----------|---------------|----------|
| Progress Note (SOAP) | PROGRESS_NOTE | Office/outpatient visits |
| History & Physical | HISTORY_AND_PHYSICAL | Admissions, consultations |
| Discharge Summary | DISCHARGE_SUMMARY | Inpatient discharges |
| Consultation Note | CONSULTATION_NOTE | Specialty consults |
| Emergency Note | EMERGENCY_NOTE | ED visits |
| Prenatal Note | PRENATAL_NOTE | OB visits |
| Psychiatric Eval | PSYCHIATRIC_EVALUATION | Psych assessments |
| Procedure Note | PROCEDURE_NOTE | General procedures |
| Operative Note | OPERATIVE_NOTE | Surgical procedures |
| Annual Wellness | PROGRESS_NOTE | Medicare AWV |
| Well Child Visit | PROGRESS_NOTE | Pediatric EPSDT |
| Telehealth Note | PROGRESS_NOTE | Virtual visits |
| Nursing Assessment | NURSING_NOTE | Nursing documentation |

## E/M Calculation (2021 Guidelines)

### MDM-Based Levels (Office/Outpatient)

| Level | Code (Est) | Code (New) | Problems | Data | Risk |
|-------|------------|------------|----------|------|------|
| Straightforward | 99212 | 99202 | 1 self-limited | Minimal | Minimal |
| Low | 99213 | 99203 | 2 self-limited | Limited | Low |
| Moderate | 99214 | 99204 | 1+ chronic | Moderate | Moderate |
| High | 99215 | 99205 | Acute threat | Extensive | High |

### Time-Based Levels

| Code | Est. Patient | New Patient |
|------|--------------|-------------|
| 99212/99202 | 10-19 min | 15-29 min |
| 99213/99203 | 20-29 min | 30-44 min |
| 99214/99204 | 30-39 min | 45-59 min |
| 99215/99205 | 40+ min | 60-74 min |

## Macro Categories

| Category | Count | Examples |
|----------|-------|----------|
| Physical Exam | 20+ | .ngen, .ncv, .nresp, .nabd |
| Review of Systems | 5+ | .rosneg, .rosconst |
| HPI Templates | 5+ | .hpipain, .hpisob, .hpicp |
| Plan Templates | 5+ | .plandm, .planhtn, .planchf |
| Attestations | 3+ | .attesttime, .attestres |
| Procedures | 3+ | .procconsent, .timeout |
| Discharge | 3+ | .dcreturn, .dcwound |

## Quick Start

### Running with Docker

```bash
# Build
docker build -t kb14-documentation-intelligence .

# Run
docker run -p 8087:8087 kb14-documentation-intelligence
```

### Running Locally

```bash
# Install dependencies
go mod download

# Run
go run cmd/server/main.go

# Test
go test ./test/...
```

### Example: Create Document with Auto-Population

```bash
# Create document
curl -X POST http://localhost:8087/api/v1/documents/create \
  -H "Content-Type: application/json" \
  -d '{
    "templateId": "tpl-progress-note-001",
    "patientId": "patient-123",
    "encounterId": "encounter-456",
    "encounterType": "OFFICE_VISIT",
    "authorId": "dr-smith",
    "authorName": "Dr. Smith",
    "autoPopulate": true
  }'

# Auto-populate data
curl -X POST http://localhost:8087/api/v1/autopopulate \
  -H "Content-Type: application/json" \
  -d '{
    "templateId": "tpl-progress-note-001",
    "patientId": "patient-123",
    "encounterType": "OFFICE_VISIT"
  }'
```

### Example: Validate and Calculate E/M

```bash
curl -X POST http://localhost:8087/api/v1/validate/em \
  -H "Content-Type: application/json" \
  -d '{
    "documentId": "doc_abc123",
    "encounterType": "OFFICE_VISIT",
    "patientType": "established",
    "method": "MDM"
  }'
```

### Example: Expand Macro

```bash
curl -X POST http://localhost:8087/api/v1/macros/expand \
  -H "Content-Type: application/json" \
  -d '{
    "abbreviation": ".ncv",
    "variables": {}
  }'
```

## File Structure

```
kb14-documentation-intelligence/
├── cmd/server/main.go              (850 lines)   HTTP server with 25+ endpoints
├── pkg/docint/
│   ├── service.go                  (750 lines)   Core service, types, E/M rules
│   ├── templates.go                (950 lines)   Built-in template definitions
│   ├── auto_population.go          (550 lines)   FHIR auto-population
│   ├── validation.go               (650 lines)   Completeness, billing, quality
│   └── macros.go                   (450 lines)   Macro management
├── test/service_test.go            (400 lines)   Comprehensive tests
├── Dockerfile                      (65 lines)    Multi-stage build
├── go.mod                          (5 lines)
└── README.md                                     This file
```

**Total: ~4,700 lines**

## Integration Points

### Upstream (Data Sources)
- **FHIR Server**: Patient data for auto-population
- **EHR**: Encounter context, user authentication

### Downstream (Consumers)
- **AI Scribe**: Template generation after transcription
- **CDSS**: Care summary generation
- **Patient Portal**: After Visit Summary

### Sibling Services
- **KB-10 (Clinical NLP)**: Content validation
- **KB-7 (Terminology)**: Code validation
- **KB-11 (CDI)**: Documentation improvement suggestions

## Compliance

- **2021 AMA E/M Guidelines**: Full MDM and time-based support
- **FHIR R4**: Standard data exchange
- **LOINC**: Document type coding
- **SNOMED CT**: Clinical terminology

## License

Proprietary - Healthcare Platform
