# KB-7 Terminology Database Schema & ETL Documentation

This document provides comprehensive documentation for the database schema and ETL (Extract, Transform, Load) processes used by the KB-7 Terminology Service.

## 🗄️ Database Overview

The KB-7 Terminology Service uses PostgreSQL with specialized extensions for clinical terminology management. The schema is designed to support:

- **Multiple Terminology Systems**: SNOMED CT, ICD-10, RxNorm, LOINC
- **Hierarchical Relationships**: Parent-child concept navigation
- **Cross-System Mappings**: Translation between terminologies  
- **Full-Text Search**: Optimized text search with ranking
- **Value Set Management**: Collections of concepts for specific use cases
- **Audit Trail**: Complete change tracking and versioning

## 📊 Database Schema

### Core Tables

#### 1. `terminology_systems`
Stores metadata for terminology systems (SNOMED CT, ICD-10, etc.)

```sql
CREATE TABLE terminology_systems (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    system_uri VARCHAR(500) UNIQUE NOT NULL,
    system_name VARCHAR(255) NOT NULL,
    version VARCHAR(100) NOT NULL,
    description TEXT,
    publisher VARCHAR(255),
    status VARCHAR(20) DEFAULT 'active',
    metadata JSONB DEFAULT '{}',
    supported_regions TEXT[] DEFAULT ARRAY['US'],
    concept_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(system_uri, version)
);
```

**Key Features:**
- UUID primary keys for global uniqueness
- JSONB metadata for flexible system-specific properties
- Array support for multi-regional terminology versions
- Automatic concept counting via triggers

#### 2. `terminology_concepts`
Core concept storage with hierarchical relationships

```sql
CREATE TABLE terminology_concepts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    system_id UUID REFERENCES terminology_systems(id),
    code VARCHAR(255) NOT NULL,
    display VARCHAR(500) NOT NULL,
    definition TEXT,
    status VARCHAR(20) DEFAULT 'active',
    parent_codes TEXT[] DEFAULT ARRAY[]::TEXT[],
    child_codes TEXT[] DEFAULT ARRAY[]::TEXT[],
    properties JSONB DEFAULT '{}',
    designations JSONB DEFAULT '[]',
    clinical_domain VARCHAR(100),
    specialty VARCHAR(100),
    search_terms TSVECTOR, -- Automatically maintained
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(system_id, code)
);
```

**Optimization Features:**
- Denormalized hierarchy (parent_codes/child_codes arrays) for fast traversal
- Full-text search vector automatically maintained via triggers
- GIN indexes on arrays and JSONB for efficient querying
- Clinical domain classification for filtering

#### 3. `concept_mappings`
Cross-terminology mappings with confidence scoring

```sql
CREATE TABLE concept_mappings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_system_id UUID REFERENCES terminology_systems(id),
    source_code VARCHAR(255) NOT NULL,
    target_system_id UUID REFERENCES terminology_systems(id),
    target_code VARCHAR(255) NOT NULL,
    equivalence VARCHAR(20) DEFAULT 'equivalent',
    mapping_type VARCHAR(50) DEFAULT 'manual',
    confidence DECIMAL(3,2) DEFAULT 1.0,
    comment TEXT,
    evidence JSONB DEFAULT '{}',
    verified BOOLEAN DEFAULT FALSE,
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

**Quality Features:**
- FHIR-compliant equivalence relationships
- Confidence scoring (0.0-1.0) for mapping quality
- Verification workflow support
- Usage tracking for optimization

#### 4. `value_sets`
FHIR-compliant value set definitions

```sql
CREATE TABLE value_sets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    url VARCHAR(500) UNIQUE NOT NULL,
    version VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    title VARCHAR(500),
    description TEXT,
    status VARCHAR(20) DEFAULT 'draft',
    compose JSONB DEFAULT '{}',
    expansion JSONB DEFAULT '{}',
    clinical_domain VARCHAR(100),
    supported_regions TEXT[] DEFAULT ARRAY['US'],
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    expired_at TIMESTAMPTZ,
    UNIQUE(url, version)
);
```

### Performance Indexes

#### Full-Text Search Optimization
```sql
-- GIN index for full-text search
CREATE INDEX idx_terminology_concepts_search 
ON terminology_concepts USING gin(search_terms);

-- Trigram index for fuzzy matching
CREATE INDEX idx_terminology_concepts_display 
ON terminology_concepts USING gin(display gin_trgm_ops);
```

#### Hierarchy Navigation
```sql
-- Array indexes for parent/child traversal
CREATE INDEX idx_terminology_concepts_parents 
ON terminology_concepts USING gin(parent_codes);

CREATE INDEX idx_terminology_concepts_children 
ON terminology_concepts USING gin(child_codes);
```

#### JSONB Property Queries
```sql
-- GIN indexes for JSONB properties
CREATE INDEX idx_terminology_concepts_properties_gin 
ON terminology_concepts USING gin(properties);

CREATE INDEX idx_value_sets_compose_gin 
ON value_sets USING gin(compose);
```

### Database Triggers

#### 1. Automatic Search Vector Maintenance
```sql
CREATE OR REPLACE FUNCTION update_concept_search_terms()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_terms := 
        setweight(to_tsvector('english', COALESCE(NEW.display, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.definition, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.code, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_concept_search_terms
    BEFORE INSERT OR UPDATE ON terminology_concepts
    FOR EACH ROW EXECUTE FUNCTION update_concept_search_terms();
```

#### 2. Concept Count Maintenance
```sql
CREATE OR REPLACE FUNCTION update_system_concept_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE terminology_systems 
        SET concept_count = concept_count + 1,
            updated_at = NOW()
        WHERE id = NEW.system_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE terminology_systems 
        SET concept_count = concept_count - 1,
            updated_at = NOW()
        WHERE id = OLD.system_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
```

## 🔄 ETL Pipeline Architecture

### ETL Overview

The ETL pipeline processes terminology source files and loads them into the PostgreSQL database with appropriate transformations and validations.

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Source Files   │───▶│  ETL Processor   │───▶│   PostgreSQL    │
│ (RF2, RRF, XML) │    │                  │    │    Database     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                               │
                               ▼
                       ┌──────────────────┐
                       │  Validation &    │
                       │    Logging       │
                       └──────────────────┘
```

### ETL Command-Line Tool

The ETL tool is located at `cmd/etl/main.go` and supports multiple terminology systems:

```bash
# Basic usage
go run ./cmd/etl/main.go --data=./data/snomed --system=snomed

# With debugging
go run ./cmd/etl/main.go --data=./data/rxnorm --system=rxnorm --debug

# All available options
go run ./cmd/etl/main.go \
  --data=/path/to/data \
  --system=snomed \
  --batch-size=10000 \
  --workers=4 \
  --validate-only \
  --debug
```

### System-Specific ETL Processors

#### 1. SNOMED CT RF2 Loader (`internal/etl/snomed_loader.go`)

**Input Files:**
- `sct2_Concept_*.txt` - Core concepts
- `sct2_Description_*.txt` - Terms and descriptions
- `sct2_Relationship_*.txt` - Concept relationships
- `sct2_TextDefinition_*.txt` - Formal definitions

**Processing Flow:**
```go
func (s *SNOMEDLoader) ProcessRF2Files(dataPath string) error {
    // 1. Load concepts first
    concepts, err := s.loadConcepts(filepath.Join(dataPath, "Terminology/sct2_Concept_*.txt"))
    
    // 2. Load descriptions and link to concepts
    descriptions, err := s.loadDescriptions(filepath.Join(dataPath, "Terminology/sct2_Description_*.txt"))
    
    // 3. Load relationships and build hierarchy
    relationships, err := s.loadRelationships(filepath.Join(dataPath, "Terminology/sct2_Relationship_*.txt"))
    
    // 4. Transform and insert into database
    return s.insertConcepts(concepts, descriptions, relationships)
}
```

**Hierarchy Building:**
```go
func (s *SNOMEDLoader) buildHierarchy(relationships []Relationship) map[string]ConceptHierarchy {
    hierarchy := make(map[string]ConceptHierarchy)
    
    for _, rel := range relationships {
        if rel.TypeId == "116680003" { // "Is a" relationship
            concept := hierarchy[rel.SourceId]
            concept.Parents = append(concept.Parents, rel.DestinationId)
            hierarchy[rel.SourceId] = concept
            
            targetConcept := hierarchy[rel.DestinationId]
            targetConcept.Children = append(targetConcept.Children, rel.SourceId)
            hierarchy[rel.DestinationId] = targetConcept
        }
    }
    
    return hierarchy
}
```

#### 2. RxNorm RRF Loader (`internal/etl/rxnorm_loader.go`)

**Input Files:**
- `RXNCONSO.RRF` - Concept names and sources
- `RXNREL.RRF` - Concept relationships
- `RXNSAT.RRF` - Concept attributes
- `RXNSTY.RRF` - Semantic types

**Processing Example:**
```go
func (r *RxNormLoader) ProcessRRFFiles(dataPath string) error {
    // Load concepts from RXNCONSO.RRF
    conceptsFile := filepath.Join(dataPath, "RXNCONSO.RRF")
    concepts, err := r.loadRxNormConcepts(conceptsFile)
    if err != nil {
        return fmt.Errorf("failed to load RxNorm concepts: %w", err)
    }
    
    // Load relationships from RXNREL.RRF
    relationshipsFile := filepath.Join(dataPath, "RXNREL.RRF")
    relationships, err := r.loadRxNormRelationships(relationshipsFile)
    if err != nil {
        return fmt.Errorf("failed to load RxNorm relationships: %w", err)
    }
    
    // Process drug mappings
    return r.processDrugMappings(concepts, relationships)
}
```

#### 3. ICD-10-CM Loader (`internal/etl/icd10_loader.go`)

**Input Files:**
- `icd10cm_codes_*.txt` - Diagnosis codes
- `icd10cm_order_*.txt` - Code ordering
- `icd10cm_index_*.xml` - Alphabetic index

**Processing Features:**
- Hierarchical code structure (3-7 character codes)
- Chapter and section organization
- Inclusion/exclusion note processing
- Cross-references and "see also" relationships

#### 4. LOINC Loader (`internal/etl/loinc_loader.go`)

**Input Files:**
- `LOINC.csv` - Core LOINC concepts
- `LOINC_HIERARCHY.csv` - Multi-axial hierarchy
- `MAP_TO.csv` - Mappings to other terminologies
- `LINGUISTIC_VARIANTS.csv` - Language variants

### ETL Configuration

#### Batch Processing Configuration
```go
type ETLConfig struct {
    BatchSize        int    // Default: 10000 records per batch
    WorkerCount      int    // Default: 4 concurrent workers  
    ValidateOnly     bool   // Default: false
    EnableDebug      bool   // Default: false
    MaxRetries       int    // Default: 3
    RetryDelay       time.Duration // Default: 5 seconds
}
```

#### Error Handling
```go
type ETLError struct {
    FileName   string
    LineNumber int
    RecordData string
    ErrorType  string
    Message    string
    Timestamp  time.Time
}

func (e *ETLCoordinator) handleProcessingError(err ETLError) {
    // Log error with context
    e.logger.WithFields(logrus.Fields{
        "file": err.FileName,
        "line": err.LineNumber,
        "error_type": err.ErrorType,
    }).Error(err.Message)
    
    // Store in error table for later analysis
    e.storeETLError(err)
    
    // Check error threshold
    if e.errorCount > e.config.MaxErrors {
        return fmt.Errorf("error threshold exceeded")
    }
}
```

## 📈 Performance Tuning

### Database Configuration

#### PostgreSQL Settings for Terminology Workloads
```ini
# Memory settings
shared_buffers = 2GB                    # 25% of RAM
effective_cache_size = 6GB              # 75% of RAM
work_mem = 256MB                        # For large sorts/joins
maintenance_work_mem = 512MB           # For CREATE INDEX

# Checkpoint settings
checkpoint_completion_target = 0.9
checkpoint_segments = 32

# Query planner
default_statistics_target = 100
random_page_cost = 1.1                 # SSD optimized
effective_io_concurrency = 200

# Logging for optimization
log_min_duration_statement = 1000     # Log slow queries
log_statement = 'mod'                  # Log data modifications
```

#### Connection Pool Optimization
```go
// Production settings in internal/database/connection.go
func Connect(databaseURL string) (*sql.DB, error) {
    db, err := sql.Open("postgres", databaseURL)
    if err != nil {
        return nil, err
    }
    
    // Connection pool settings
    db.SetMaxOpenConns(50)              // Max concurrent connections
    db.SetMaxIdleConns(10)              // Idle connection pool
    db.SetConnMaxLifetime(5 * time.Minute) // Connection lifetime
    
    return db, nil
}
```

### Query Optimization

#### Full-Text Search Optimization
```sql
-- Optimize search queries with proper ranking
SELECT 
    c.code, c.display, c.definition,
    ts_rank(c.search_terms, query) AS rank
FROM terminology_concepts c,
     plainto_tsquery('english', 'paracetamol') query
WHERE c.search_terms @@ query
  AND c.status = 'active'
ORDER BY rank DESC, c.display
LIMIT 20;
```

#### Hierarchy Traversal Optimization
```sql
-- Efficient parent traversal using arrays
WITH RECURSIVE concept_hierarchy AS (
    -- Base case: start concept
    SELECT code, display, parent_codes, 1 as level
    FROM terminology_concepts 
    WHERE code = '387517004'
    
    UNION ALL
    
    -- Recursive case: find parents
    SELECT p.code, p.display, p.parent_codes, h.level + 1
    FROM terminology_concepts p
    JOIN concept_hierarchy h ON p.code = ANY(h.parent_codes)
    WHERE h.level < 5  -- Limit depth
)
SELECT * FROM concept_hierarchy ORDER BY level;
```

### Index Maintenance

#### Regular Maintenance Tasks
```sql
-- Weekly index maintenance
REINDEX INDEX CONCURRENTLY idx_terminology_concepts_search;
REINDEX INDEX CONCURRENTLY idx_terminology_concepts_display;

-- Update table statistics
ANALYZE terminology_concepts;
ANALYZE concept_mappings;
ANALYZE value_sets;

-- Vacuum to reclaim space
VACUUM ANALYZE terminology_concepts;
```

## 🔍 Monitoring & Diagnostics

### Database Monitoring Queries

#### Performance Analysis
```sql
-- Slow query identification
SELECT query, mean_time, calls, total_time
FROM pg_stat_statements 
WHERE query LIKE '%terminology%'
ORDER BY mean_time DESC
LIMIT 10;

-- Index usage statistics
SELECT 
    schemaname, tablename, indexname,
    idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes 
WHERE schemaname = 'public'
ORDER BY idx_scan DESC;

-- Table size analysis
SELECT 
    tablename,
    pg_size_pretty(pg_total_relation_size(tablename::regclass)) as size
FROM pg_tables 
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(tablename::regclass) DESC;
```

#### ETL Process Monitoring
```sql
-- ETL error analysis
SELECT 
    error_type, 
    COUNT(*) as error_count,
    MAX(timestamp) as last_occurrence
FROM etl_errors 
WHERE timestamp > NOW() - INTERVAL '24 hours'
GROUP BY error_type
ORDER BY error_count DESC;

-- Data freshness check
SELECT 
    system_name,
    version,
    updated_at,
    EXTRACT(EPOCH FROM (NOW() - updated_at))/3600 as hours_old
FROM terminology_systems
ORDER BY updated_at DESC;
```

### ETL Validation Queries

#### Data Quality Checks
```sql
-- Check for orphaned concepts
SELECT COUNT(*) as orphaned_concepts
FROM terminology_concepts c
LEFT JOIN terminology_systems s ON c.system_id = s.id
WHERE s.id IS NULL;

-- Validate hierarchy consistency
SELECT 
    c.code,
    array_length(c.parent_codes, 1) as parent_count,
    array_length(c.child_codes, 1) as child_count
FROM terminology_concepts c
WHERE array_length(c.parent_codes, 1) > 10  -- Unusually many parents
   OR array_length(c.child_codes, 1) > 100; -- Unusually many children

-- Check mapping quality
SELECT 
    equivalence,
    COUNT(*) as mapping_count,
    AVG(confidence) as avg_confidence,
    MIN(confidence) as min_confidence
FROM concept_mappings
GROUP BY equivalence
ORDER BY mapping_count DESC;
```

## 🔧 Troubleshooting

### Common ETL Issues

#### 1. File Format Problems
```bash
# Check file encoding
file -bi /path/to/terminology/file.txt

# Convert encoding if needed
iconv -f iso-8859-1 -t utf-8 input.txt > output.txt
```

#### 2. Memory Issues During ETL
```go
// Implement batching for large files
func (e *ETLProcessor) processBatches(reader *csv.Reader) error {
    batch := make([]Record, 0, e.config.BatchSize)
    
    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        
        batch = append(batch, parseRecord(record))
        
        if len(batch) >= e.config.BatchSize {
            if err := e.processBatch(batch); err != nil {
                return err
            }
            batch = batch[:0]  // Reset slice
            runtime.GC()       // Force garbage collection
        }
    }
    
    return e.processBatch(batch)  // Process remaining records
}
```

#### 3. Database Lock Issues
```sql
-- Check for locks during ETL
SELECT 
    pid, query, state, wait_event_type, wait_event
FROM pg_stat_activity 
WHERE state = 'active' AND query LIKE '%terminology%';

-- Kill problematic connections if needed
SELECT pg_terminate_backend(pid) FROM pg_stat_activity 
WHERE state = 'idle in transaction' 
  AND query_start < NOW() - INTERVAL '1 hour';
```

### Performance Troubleshooting

#### Query Performance Issues
```sql
-- Enable detailed query logging
SET log_statement = 'all';
SET log_min_duration_statement = 0;

-- Analyze specific query performance
EXPLAIN (ANALYZE, BUFFERS, TIMING) 
SELECT * FROM terminology_concepts 
WHERE search_terms @@ plainto_tsquery('clinical finding');
```

#### Memory Usage Optimization
```sql
-- Monitor memory usage
SELECT 
    setting as shared_buffers,
    unit
FROM pg_settings 
WHERE name = 'shared_buffers';

-- Check cache hit ratios
SELECT 
    sum(heap_blks_read) as heap_read,
    sum(heap_blks_hit) as heap_hit,
    sum(heap_blks_hit) / (sum(heap_blks_hit) + sum(heap_blks_read)) as ratio
FROM pg_statio_user_tables;
```

For additional database support, contact the Clinical Platform Team at clinical-platform@hospital.com.