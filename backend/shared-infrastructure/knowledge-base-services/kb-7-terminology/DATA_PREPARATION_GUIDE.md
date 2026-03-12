# 📊 KB-7 Terminology Data Preparation Guide

This guide provides instructions for downloading and preparing terminology data sources for the KB-7 Terminology service ETL pipeline.

## 🎯 Required Data Sources

### 1. **RxNorm** - Drug Names and Codes
**Publisher:** National Library of Medicine (NLM)  
**License:** Free for use  
**Download:** https://www.nlm.nih.gov/research/umls/rxnorm/docs/rxnormfiles.html

**Required Files:**
- `RXNCONSO.RRF` - Concept names and sources
- `RXNREL.RRF` - Relationships between concepts  
- `RXNSAT.RRF` - Simple concept and source attributes

**Preparation:**
```bash
# Create RxNorm data directory
mkdir -p ./data/rxnorm

# Download RxNorm files (requires UMLS license agreement)
# Extract to ./data/rxnorm/
# Files should be directly in the rxnorm directory
```

### 2. **LOINC** - Laboratory Codes
**Publisher:** Regenstrief Institute  
**License:** Free with registration  
**Download:** https://loinc.org/downloads/

**Required Files:**
- `LOINC.CSV` - Core LOINC data
- `LOINC_HIERARCHY.CSV` - Hierarchical relationships
- `MULTI-AXIAL_HIERARCHY.CSV` - Multi-axial hierarchy (optional)

**Preparation:**
```bash
# Create LOINC data directory
mkdir -p ./data/loinc

# Download LOINC Complete database
# Extract to ./data/loinc/
# Ensure CSV files are directly accessible
```

### 3. **SNOMED CT** - Clinical Terms
**Publisher:** SNOMED International  
**License:** Requires licensing agreement  
**Download:** https://www.nlm.nih.gov/healthit/snomedct/

**Required Files:**
- `sct2_Concept_Snapshot_INT.txt` - Core concepts
- `sct2_Description_Snapshot-en_INT.txt` - English descriptions
- `sct2_Relationship_Snapshot_INT.txt` - Concept relationships

**Preparation:**
```bash
# Create SNOMED data directory
mkdir -p ./data/snomed

# Download SNOMED CT International Edition
# Extract to ./data/snomed/
# Files are typically in Snapshot/Terminology/ subdirectory
# Or place files directly in ./data/snomed/
```

### 4. **ICD-10-CM** - Diagnosis Codes
**Publisher:** Centers for Medicare & Medicaid Services (CMS)  
**License:** Public Domain  
**Download:** https://www.cms.gov/medicare/icd-10/icd-10-cm-official-guidelines-coding-reporting

**Required Files:**
- `icd10cm_codes.txt` - ICD-10-CM diagnosis codes
- `icd10cm_order.txt` - ICD-10-CM tabular order

**Preparation:**
```bash
# Create ICD-10 data directory
mkdir -p ./data/icd10

# Download ICD-10-CM files
# Extract to ./data/icd10/
# Note: ICD-10 loader includes sample data for demonstration
```

---

## 📁 Final Directory Structure

After downloading and extracting all data sources, your directory structure should look like:

```
kb-7-terminology/data/
├── rxnorm/
│   ├── RXNCONSO.RRF
│   ├── RXNREL.RRF
│   ├── RXNSAT.RRF
│   └── [other RxNorm files]
├── loinc/
│   ├── LOINC.CSV
│   ├── LOINC_HIERARCHY.CSV
│   ├── MULTI-AXIAL_HIERARCHY.CSV
│   └── [other LOINC files]
├── snomed/
│   ├── Snapshot/
│   │   └── Terminology/
│   │       ├── sct2_Concept_Snapshot_INT.txt
│   │       ├── sct2_Description_Snapshot-en_INT.txt
│   │       └── sct2_Relationship_Snapshot_INT.txt
│   └── [other SNOMED directories]
└── icd10/
    ├── icd10cm_codes.txt
    ├── icd10cm_order.txt
    └── [other ICD-10 files]
```

---

## 🚀 Running the ETL Process

### 1. **Build the ETL Tool**
```bash
cd kb-7-terminology
go build -o etl-tool ./cmd/etl
```

### 2. **Validate Data Sources**
```bash
# Validate all systems
./etl-tool -data=./data -validate

# Validate specific system
./etl-tool -data=./data -system=rxnorm -validate
```

### 3. **Load Terminology Data**

**Load All Systems:**
```bash
./etl-tool -data=./data -batch=2000 -workers=4
```

**Load Specific System:**
```bash
# Load RxNorm only
./etl-tool -data=./data -system=rxnorm -batch=1000

# Load LOINC only  
./etl-tool -data=./data -system=loinc -batch=1000

# Load SNOMED CT only
./etl-tool -data=./data -system=snomed -batch=500

# Load ICD-10 only
./etl-tool -data=./data -system=icd10 -batch=1000
```

**Advanced Options:**
```bash
# Enable debug logging
./etl-tool -data=./data -debug

# Skip loading if data already exists
./etl-tool -data=./data -skip-existing

# Custom batch sizes for performance tuning
./etl-tool -data=./data -batch=5000 -workers=8
```

---

## 📊 Expected Data Volumes

| System | Concepts | Load Time | Disk Space |
|--------|----------|-----------|------------|
| RxNorm | ~400K | 10-15 min | ~50 MB |
| LOINC | ~90K | 5-8 min | ~30 MB |
| SNOMED CT | ~350K | 15-25 min | ~200 MB |
| ICD-10 | ~10K* | 1-2 min | ~5 MB |

*Sample implementation - full ICD-10-CM has ~70K codes

---

## ⚡ Performance Optimization

### Database Tuning
```sql
-- Increase work memory for bulk operations
SET work_mem = '256MB';

-- Disable autovacuum during loading
SET autovacuum = off;

-- Increase checkpoint segments
SET checkpoint_segments = 32;
```

### ETL Tuning
- **Batch Size:** Start with 1000, increase to 5000 for larger systems
- **Workers:** Use 2-4 workers for optimal performance
- **Memory:** Ensure adequate RAM (4GB+ recommended)
- **Storage:** Use SSD storage for better I/O performance

---

## 🔍 Troubleshooting

### Common Issues

**1. File Not Found Errors**
```bash
# Check directory structure
ls -la ./data/rxnorm/
ls -la ./data/loinc/
ls -la ./data/snomed/Snapshot/Terminology/

# Verify file permissions
chmod -R 755 ./data/
```

**2. Database Connection Issues**
```bash
# Test database connectivity
psql -h localhost -p 5432 -U postgres -d kb_terminology -c "SELECT version();"

# Check service status
./check-databases.bat
```

**3. Memory Issues During Loading**
```bash
# Reduce batch size
./etl-tool -data=./data -batch=500 -workers=2

# Monitor memory usage
docker stats kb-postgres
```

**4. Licensing and Access Issues**
- **RxNorm:** Requires free UMLS license agreement
- **LOINC:** Requires free registration
- **SNOMED CT:** Requires institutional license
- **ICD-10:** Public domain, no restrictions

---

## 📈 Verification and Testing

### 1. **Database Verification**
```sql
-- Check loaded systems
SELECT system_name, version, status, 
       (SELECT COUNT(*) FROM terminology_concepts WHERE system_id = terminology_systems.id) as concept_count
FROM terminology_systems;

-- Verify search functionality
SELECT code, display, system_id 
FROM terminology_concepts 
WHERE search_vector @@ to_tsquery('hypertension')
LIMIT 10;
```

### 2. **API Testing**
```bash
# Test terminology lookup
curl "http://localhost:8087/api/v1/terminology/lookup?code=10509002&system=snomed"

# Test terminology search
curl "http://localhost:8087/api/v1/terminology/search?query=diabetes&system=rxnorm"

# Test concept validation
curl "http://localhost:8087/api/v1/terminology/validate" \
  -H "Content-Type: application/json" \
  -d '{"system": "http://loinc.org", "code": "33747-0"}'
```

### 3. **Performance Testing**
```bash
# Load test terminology API
npm install -g artillery
artillery quick --count 100 --num 10 http://localhost:8087/api/v1/terminology/search?query=medication
```

---

## 📚 Data Source Documentation

### RxNorm Documentation
- **File Format Guide:** https://www.nlm.nih.gov/research/umls/rxnorm/docs/2013/rxnorm_doco_full_2013-2.html
- **API Documentation:** https://rxnav.nlm.nih.gov/RxNavVisualization.html
- **Technical Guide:** https://www.nlm.nih.gov/research/umls/rxnorm/docs/techguide.html

### LOINC Documentation  
- **User Guide:** https://loinc.org/downloads/files/LOINCManual.pdf
- **Database Structure:** https://loinc.org/downloads/files/loinc-database-structure/
- **Implementation Guide:** https://loinc.org/kb/users-guide/

### SNOMED CT Documentation
- **Technical Implementation Guide:** https://confluence.ihtsdotools.org/display/DOCTIG
- **Browser Guide:** https://browser.ihtsdotools.org/
- **API Guide:** https://github.com/IHTSDO/snowstorm

### ICD-10-CM Documentation
- **Official Guidelines:** https://www.cms.gov/medicare/icd-10/icd-10-cm-official-guidelines-coding-reporting
- **Code Files:** https://www.cms.gov/Medicare/Coding/ICD10/2023-ICD-10-CM
- **Implementation Guide:** https://www.cdc.gov/nchs/icd/icd10cm.htm

---

## 🎯 Next Steps

After successfully loading terminology data:

1. **Test Knowledge Base Integration:**
   - Verify KB-1 Drug Rules can access RxNorm codes
   - Test KB-2 Clinical Context phenotype detection
   - Confirm KB-6 Formulary drug lookups

2. **Performance Monitoring:**
   - Set up Prometheus metrics collection
   - Configure Grafana dashboards
   - Monitor query performance and cache hit rates

3. **Data Maintenance:**
   - Schedule regular terminology updates
   - Implement delta loading for incremental updates
   - Set up automated backup and recovery

4. **Integration Testing:**
   - Test Apollo Federation endpoint integration
   - Verify Evidence Envelope audit trail capture
   - Confirm FHIR compliance for terminology operations

---

**📞 Support:** For technical issues, check the troubleshooting section or review the ETL tool logs for detailed error information.