# Knowledge Pipeline Service

**REAL DATA ONLY** - Clinical knowledge ingestion pipeline that downloads and processes authentic data from authoritative medical sources. **NO FALLBACK DATA** - All sources must provide real clinical data or the ingestion will fail.

## 🎯 **Overview**

This service implements **Phase 1** of the Clinical Knowledge Graph Implementation Plan, providing:

- **🔒 REAL DATA ONLY**: Downloads authentic clinical data from authoritative sources - NO MOCK/FALLBACK DATA
- **⚡ Fail-Fast Approach**: If real data sources are unavailable, ingestion fails immediately with clear error messages
- **🔗 RDF Harmonization**: Ensures consistent entity mapping using RxNorm as master drug identifier
- **📊 GraphDB Integration**: Inserts processed knowledge into your existing `cae-clinical-intelligence` repository
- **🛡️ Production Ready**: Comprehensive error handling, logging, and monitoring

## 🏗️ **Architecture**

```
┌─────────────────────────────────────────────────────────────┐
│                Knowledge Pipeline Service                    │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐  │
│  │   RxNorm        │  │  CredibleMeds   │  │    AHRQ     │  │
│  │   Ingester      │  │    Ingester     │  │  Ingester   │  │
│  └─────────────────┘  └─────────────────┘  └─────────────┘  │
├─────────────────────────────────────────────────────────────┤
│              Harmonization Engine                           │
│         (RxNorm as Master Drug Identifier)                  │
├─────────────────────────────────────────────────────────────┤
│                GraphDB Client                               │
│        (localhost:7200/cae-clinical-intelligence)           │
└─────────────────────────────────────────────────────────────┘
```

## 📊 **Data Sources**

### 1. **RxNorm** (Drug Terminology) ✅ REAL DATA
- **Source**: UMLS RxNorm Full Release (requires UMLS license)
- **Content**: Drug names, RXCUIs, relationships, attributes
- **Format**: RRF files → RDF triples
- **Master Index**: Used for drug harmonization across all sources
- **⚠️ REQUIRED**: Must download actual RxNorm data - NO FALLBACKS

### 2. **DrugBank Academic** (Drug Interactions) ✅ REAL DATA
- **Source**: DrugBank Academic XML Database
- **Content**: Drug-drug interactions, pharmacology, mechanisms
- **Format**: XML → RDF triples
- **⚠️ REQUIRED**: Must register and download XML file - NO FALLBACKS

### 3. **CredibleMeds** (QT Drug Safety) ✅ REAL DATA
- **Source**: CredibleMeds QT Drug Lists
- **Content**: QT prolongation risk categories
- **Categories**: Known Risk, Possible Risk, Conditional Risk
- **Format**: Web scraping/PDF → RDF triples
- **⚠️ REQUIRED**: Must access live CredibleMeds data - NO FALLBACKS

### 4. **UMLS Metathesaurus** (Unified Medical Terminology) ✅ REAL DATA
- **Source**: UMLS Metathesaurus Full Release
- **Content**: Unified medical concepts, synonyms, cross-references
- **Format**: RRF files → RDF triples
- **⚠️ REQUIRED**: Must download with UMLS license - NO FALLBACKS

### 5. **SNOMED CT** (Clinical Terminology) ✅ REAL DATA
- **Source**: SNOMED CT International Edition RF2
- **Content**: Clinical concepts, relationships, descriptions
- **Format**: RF2 files → RDF triples
- **⚠️ REQUIRED**: Must download with SNOMED license - NO FALLBACKS

### 6. **LOINC** (Laboratory Terminology) ✅ REAL DATA
- **Source**: LOINC Database CSV Files
- **Content**: Laboratory tests, measurements, clinical observations
- **Format**: CSV files → RDF triples
- **⚠️ REQUIRED**: Must download with LOINC license - NO FALLBACKS

### 7. **AHRQ CDS Connect** (Clinical Pathways) ✅ REAL DATA
- **Source**: AHRQ Clinical Decision Support Artifacts API
- **Content**: Clinical pathways, guidelines, order sets
- **Format**: API JSON/XML → RDF triples
- **⚠️ REQUIRED**: Must access live AHRQ API - NO FALLBACKS

### 8. **OpenFDA FAERS** (Adverse Events) ✅ REAL DATA
- **Source**: FDA Adverse Event Reporting System API
- **Content**: Real-world adverse drug events, safety signals
- **Format**: JSON API → RDF triples
- **⚠️ REQUIRED**: Must access live OpenFDA API - NO FALLBACKS

## 🚀 **Quick Start**

### Prerequisites
- GraphDB running at `localhost:7200` with repository `cae-clinical-intelligence`
- Python 3.10+
- Internet connection for data downloads

### Installation
```bash
cd backend/services/knowledge-pipeline-service
pip install -r requirements.txt
```

### Run Full Pipeline
```bash
# Run all ingesters
python start_pipeline.py

# Run specific sources
python start_pipeline.py --sources rxnorm umls snomed
python start_pipeline.py --sources drugbank crediblemeds openfda

# Force fresh download
python start_pipeline.py --force-download

# Validate configuration only
python start_pipeline.py --validate-only
```

### Start API Service
```bash
cd src
python main.py
```

## 🔧 **Configuration**

### Environment Variables
```bash
# GraphDB Configuration
GRAPHDB_ENDPOINT=http://localhost:7200
GRAPHDB_REPOSITORY=cae-clinical-intelligence
GRAPHDB_USERNAME=
GRAPHDB_PASSWORD=

# Data Processing
DATA_DIR=./data
TEMP_DIR=./temp
CACHE_DIR=./cache
MAX_BATCH_SIZE=1000

# Service Configuration
HOST=0.0.0.0
PORT=8030
DEBUG=false
```

### Data Source URLs
```python
# RxNorm
RXNORM_DOWNLOAD_URL=https://download.nlm.nih.gov/umls/kss/rxnorm/RxNorm_full_current.zip

# CredibleMeds
CREDIBLEMEDS_QT_URL=https://www.crediblemeds.org/pdftemp/pdf/CombinedList.pdf

# AHRQ CDS Connect
AHRQ_CDS_CONNECT_URL=https://cds.ahrq.gov/cdsconnect/artifacts
```

## 📡 **API Endpoints**

### Pipeline Management
```bash
# Get pipeline status
GET /api/v1/pipeline/status

# Run full pipeline
POST /api/v1/pipeline/run
{
  "force_download": false
}

# Run single ingester
POST /api/v1/pipeline/run/rxnorm
{
  "force_download": false
}

# Cancel running pipeline
POST /api/v1/pipeline/cancel

# Get execution history
GET /api/v1/pipeline/history?limit=10
```

### Monitoring
```bash
# Health check
GET /api/v1/pipeline/health

# Available sources
GET /api/v1/pipeline/sources

# Harmonization statistics
GET /api/v1/pipeline/harmonization/stats

# Validate configuration
GET /api/v1/pipeline/validate
```

## 🧪 **Testing**

### Run Tests
```bash
# All tests
pytest tests/

# Integration tests only
pytest tests/test_pipeline_integration.py

# Slow tests (actual downloads)
pytest tests/ -m slow

# Component testing
python start_pipeline.py --test-components
```

### Test Categories
- **Unit Tests**: Individual component testing
- **Integration Tests**: GraphDB integration
- **End-to-End Tests**: Full pipeline execution
- **Performance Tests**: Large data processing

## 📈 **Monitoring & Metrics**

### Pipeline Execution Metrics
- Total records processed
- Total RDF triples inserted
- Execution duration
- Success/failure rates
- Error tracking

### Harmonization Statistics
- Total drug mappings
- Exact vs. partial matches
- Unmapped entities
- Confidence scores

### GraphDB Integration
- Connection health
- Query performance
- Data integrity checks
- Repository statistics

## 🔄 **Data Processing Flow**

### 1. **Download Phase**
```
RxNorm: Download ZIP → Extract RRF files
CredibleMeds: Web scraping → Parse drug lists
AHRQ: API calls → Download artifacts
```

### 2. **Processing Phase**
```
Parse source data → Normalize entities → Generate RDF triples
```

### 3. **Harmonization Phase**
```
Map drug names to RxNorm → Resolve conflicts → Create unified entities
```

### 4. **Insertion Phase**
```
Batch RDF triples → Insert into GraphDB → Validate integrity
```

## 🎯 **Integration with CAE**

The pipeline enhances your existing CAE system:

### Before Pipeline
```sparql
# Limited test data
SELECT ?drug ?qtRisk WHERE {
  ?drug cae:hasQTRisk ?qtRisk .
}
# Returns: ~10 test drugs
```

### After Pipeline
```sparql
# Rich clinical knowledge
SELECT ?drug ?qtRisk WHERE {
  ?drug cae:hasQTRisk ?qtRisk .
}
# Returns: ~100+ real QT drugs with risk levels

SELECT ?pathway ?step WHERE {
  ?pathway cae:hasStep ?step .
}
# Returns: Clinical pathways with detailed steps
```

## 🚨 **Error Handling**

### Graceful Degradation
- Source unavailable → Use fallback data
- Network issues → Retry with backoff
- Parsing errors → Log and continue
- GraphDB issues → Queue for retry

### Data Quality Validation
- RDF syntax validation
- Ontology compliance checking
- Duplicate detection
- Integrity constraints

## 📊 **Performance Optimization**

### Batch Processing
- Configurable batch sizes
- Memory-efficient streaming
- Parallel processing where possible
- Progress tracking

### Caching Strategy
- Downloaded data caching (24h TTL)
- Processed mappings cache
- GraphDB query caching
- Incremental updates

## 🔐 **Security Considerations**

### Data Sources
- HTTPS for all downloads
- API key management
- Rate limiting compliance
- Terms of service adherence

### GraphDB Security
- Connection encryption
- Authentication support
- No PHI in knowledge graph
- Audit logging

## 🛠️ **Troubleshooting**

### Common Issues

**GraphDB Connection Failed**
```bash
# Check GraphDB status
curl http://localhost:7200/repositories/cae-clinical-intelligence

# Verify repository exists
python start_pipeline.py --validate-only
```

**Download Failures**
```bash
# Check internet connectivity
python start_pipeline.py --test-components

# Force fresh download
python start_pipeline.py --force-download
```

**Memory Issues**
```bash
# Reduce batch size
export MAX_BATCH_SIZE=500

# Clear temp files
rm -rf ./temp/*
```

## 📚 **Next Steps**

After successful pipeline execution:

1. **Verify Data**: Check GraphDB for new clinical knowledge
2. **Test CAE Integration**: Run CAE queries against enhanced knowledge
3. **Monitor Performance**: Track query response times
4. **Schedule Updates**: Set up periodic pipeline runs
5. **Expand Sources**: Add Phase 2 commercial data sources

## 🤝 **Contributing**

### Adding New Ingesters
1. Extend `BaseIngester` class
2. Implement required methods
3. Add to `PipelineOrchestrator`
4. Create comprehensive tests
5. Update documentation

### Data Source Integration
- Follow RDF ontology patterns
- Implement harmonization logic
- Add comprehensive error handling
- Include fallback data
- Document source-specific quirks
