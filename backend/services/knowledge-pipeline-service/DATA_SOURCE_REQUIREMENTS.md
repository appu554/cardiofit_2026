# 🔒 Data Source Requirements - REAL DATA ONLY

**CRITICAL**: This knowledge pipeline service operates with **REAL CLINICAL DATA ONLY**. No fallback, mock, or synthetic data is used. All data sources must provide authentic clinical information or the ingestion will fail.

## 🚨 No Fallback Policy

- ❌ **NO MOCK DATA**: No synthetic or simulated clinical data
- ❌ **NO FALLBACK DATA**: No hardcoded backup datasets
- ❌ **NO PLACEHOLDER DATA**: No example or demo data
- ✅ **REAL DATA ONLY**: Authentic clinical data from authoritative sources
- ✅ **FAIL-FAST**: Clear error messages when real data is unavailable

## 📋 Required Data Sources

### 1. 🏥 **RxNorm** (Drug Terminology Master Index)

**Status**: ✅ REAL DATA REQUIRED  
**Source**: UMLS RxNorm Full Release  
**License**: UMLS License Agreement Required  

#### Requirements:
- **Download URL**: https://download.nlm.nih.gov/umls/kss/rxnorm/RxNorm_full_current.zip
- **License**: Must accept UMLS Metathesaurus License Agreement
- **Account**: UMLS Terminology Services (UTS) account required
- **File Size**: ~500MB compressed, ~2GB extracted
- **Update Frequency**: Monthly releases

#### Required Files:
```
backend/services/knowledge-pipeline-service/data/rxnorm/rrf/
├── RXNCONSO.RRF    # Concept names and sources
├── RXNREL.RRF      # Relationships between concepts
├── RXNSAT.RRF      # Simple attributes
└── RXNCUI.RRF      # Concept unique identifiers
```

#### Download Instructions:
1. Create account at: https://uts.nlm.nih.gov/uts/
2. Accept UMLS Metathesaurus License Agreement
3. Download RxNorm Full Release
4. Extract RRF files to required directory
5. Verify files exist before running pipeline

---

### 2. 💊 **DrugBank Academic** (Drug Interactions & Pharmacology)

**Status**: ✅ REAL DATA REQUIRED  
**Source**: DrugBank Academic XML Database  
**License**: Academic License (Free for Academic Use)  

#### Requirements:
- **Download URL**: https://go.drugbank.com/releases/latest#open-data
- **License**: Academic use only (commercial license available)
- **Account**: DrugBank account required
- **File Size**: ~1GB compressed XML
- **Update Frequency**: Quarterly releases

#### Required Files:
```
backend/services/knowledge-pipeline-service/data/drugbank/
└── drugbank_full_database.xml  # OR drugbank_all_full_database.xml.zip
```

#### Download Instructions:
1. Create account at: https://go.drugbank.com/
2. Navigate to: Releases → Latest → Open Data
3. Download "All drugs (XML)" file
4. Save to required directory (can be ZIP or extracted XML)
5. Verify file exists before running pipeline

---

### 3. ⚡ **CredibleMeds** (QT Drug Safety Data)

**Status**: ✅ REAL DATA REQUIRED  
**Source**: CredibleMeds QT Drug Lists  
**License**: Public access (with terms of use)  

#### Requirements:
- **Website**: https://www.crediblemeds.org/
- **Data Access**: Live web scraping from public lists
- **Internet**: Active internet connection required
- **Update Frequency**: Continuous updates

#### Data Categories:
- **Known Risk of TdP**: Drugs with established QT prolongation risk
- **Possible Risk of TdP**: Drugs with potential QT prolongation risk  
- **Conditional Risk of TdP**: Drugs with conditional QT prolongation risk

#### Validation:
- Pipeline validates CredibleMeds website accessibility
- Scrapes current drug lists in real-time
- No cached or offline data used

---

### 4. 🏥 **AHRQ CDS Connect** (Clinical Decision Support)

**Status**: ✅ REAL DATA REQUIRED  
**Source**: AHRQ CDS Connect Artifacts API  
**License**: Public domain (US Government)  

#### Requirements:
- **API URL**: https://cds.ahrq.gov/cdsconnect/api/artifacts
- **Access**: Public API (no authentication required)
- **Internet**: Active internet connection required
- **Update Frequency**: Continuous updates

#### Data Types:
- **Clinical Pathways**: Evidence-based care pathways
- **Clinical Guidelines**: Practice guidelines and recommendations
- **Order Sets**: Standardized clinical order sets
- **Decision Support**: Clinical decision support artifacts

#### Validation:
- Pipeline validates AHRQ API accessibility
- Downloads current artifacts in real-time
- Parses XML/JSON clinical content

---

### 5. 🗄️ **GraphDB** (Knowledge Storage)

**Status**: ✅ REQUIRED INFRASTRUCTURE  
**Source**: Local GraphDB Instance  
**Repository**: cae-clinical-intelligence  

#### Requirements:
- **Endpoint**: http://localhost:7200
- **Repository**: cae-clinical-intelligence (must exist)
- **Status**: Running and accessible
- **Permissions**: Read/write access to repository

#### Validation:
- Tests GraphDB connection
- Verifies repository exists
- Validates read/write permissions

## 🔧 Validation Process

### Pre-Pipeline Validation
```bash
# Run comprehensive validation
python validate_data_sources.py

# Expected output for success:
✅ RXNORM: AVAILABLE
✅ DRUGBANK: AVAILABLE  
✅ CREDIBLEMEDS: AVAILABLE
✅ AHRQ: AVAILABLE
✅ GRAPHDB: AVAILABLE

🎉 ALL DATA SOURCES VALIDATED - PIPELINE READY TO RUN
```

### Failure Scenarios
```bash
# Example failure output:
❌ RXNORM: UNAVAILABLE
   ⚠️  Missing RxNorm RRF files: ['RXNCONSO.RRF', 'RXNREL.RRF']
   ⚠️  Download from: https://download.nlm.nih.gov/umls/kss/rxnorm/

❌ DRUGBANK: UNAVAILABLE
   ⚠️  DrugBank XML file not found
   ⚠️  Manual download required from: https://go.drugbank.com/

🚨 PIPELINE CANNOT RUN - MISSING REQUIRED DATA SOURCES
```

## 📊 Data Quality Assurance

### Real Data Verification
- **Source Authentication**: Validates data comes from official sources
- **Content Validation**: Checks data format and structure
- **Freshness Checks**: Ensures data is current and not stale
- **Integrity Validation**: Verifies data completeness

### Error Handling
- **Immediate Failure**: Pipeline stops if real data unavailable
- **Clear Error Messages**: Specific instructions for fixing issues
- **No Silent Fallbacks**: No hidden use of backup data
- **Audit Trail**: Complete logging of data source validation

## 🚀 Getting Started

### Step 1: Validate Prerequisites
```bash
cd backend/services/knowledge-pipeline-service
python validate_data_sources.py
```

### Step 2: Download Required Data
Follow instructions for each failed validation:
- RxNorm: Download from UMLS
- DrugBank: Download from DrugBank Academic
- CredibleMeds: Ensure internet connectivity
- AHRQ: Ensure internet connectivity
- GraphDB: Start local instance

### Step 3: Re-validate
```bash
python validate_data_sources.py
```

### Step 4: Run Pipeline
```bash
python start_pipeline.py
```

## ⚠️ Important Notes

### Legal Compliance
- **UMLS License**: Required for RxNorm data
- **DrugBank License**: Academic use only
- **Terms of Service**: Respect all data source terms
- **Attribution**: Proper citation of data sources

### Data Retention
- **Local Storage**: Downloaded data stored locally
- **Cache Management**: Automatic cache expiration
- **Update Cycles**: Regular data refresh required
- **Backup Strategy**: Implement appropriate backups

### Performance Considerations
- **Download Time**: Initial downloads may take significant time
- **Processing Time**: Large datasets require substantial processing
- **Storage Space**: Ensure adequate disk space
- **Memory Usage**: Monitor memory consumption during processing

## 🆘 Troubleshooting

### Common Issues

**"RxNorm files not found"**
- Solution: Download RxNorm from UMLS with proper license
- Check: File paths and permissions

**"DrugBank XML missing"**  
- Solution: Register and download from DrugBank Academic
- Check: File size and format (XML or ZIP)

**"CredibleMeds website inaccessible"**
- Solution: Check internet connection and website status
- Check: Firewall and proxy settings

**"AHRQ API unavailable"**
- Solution: Verify internet connectivity and API status
- Check: Network restrictions and timeouts

**"GraphDB connection failed"**
- Solution: Start GraphDB service and verify repository
- Check: Port 7200 accessibility and repository existence

### Support Resources
- **UMLS Support**: https://www.nlm.nih.gov/research/umls/
- **DrugBank Support**: https://go.drugbank.com/support
- **AHRQ Resources**: https://cds.ahrq.gov/
- **GraphDB Documentation**: https://graphdb.ontotext.com/

---

**Remember**: This pipeline is designed for production use with real clinical data. No shortcuts, no fallbacks, no compromises on data authenticity.
