# KB7 Terminology Migration System

## Overview

This migration system implements Phase 3.5.1 of the KB7 Terminology Service, migrating from pure GraphDB to an optimized hybrid PostgreSQL/GraphDB architecture.

### Migration Goals
- ✅ Extract all 23,337 triples from GraphDB
- ✅ Transform data for optimized PostgreSQL schema
- ✅ Load with 100% integrity validation
- ✅ Optimize GraphDB to contain only core reasoning data (<5,000 triples)
- ✅ Maintain complete data provenance and audit trails

## Architecture

### Before Migration
```
Client → GraphDB (23,337 triples)
```

### After Migration
```
Client → Query Router → PostgreSQL (fast lookups) + GraphDB (reasoning only)
```

## Quick Start

### 1. Setup Environment

```bash
# Install dependencies
pip install -r requirements.txt

# Configure migration settings
cp migration_config.yaml.example migration_config.yaml
# Edit migration_config.yaml with your connection details
```

### 2. Run Migration

```bash
# Full migration with config file
python scripts/migrate_to_hybrid.py --config migration_config.yaml

# Or with command line parameters
python scripts/migrate_to_hybrid.py \
  --graphdb-endpoint http://localhost:7200 \
  --graphdb-repository kb7-terminology \
  --postgres-url postgresql://user:pass@localhost:5433/kb7_terminology
```

### 3. Validate Results

```bash
# Run validation only
python scripts/migrate_to_hybrid.py --config migration_config.yaml --validate-only

# Check migration reports
ls -la logs/
```

## Migration Components

### Core Scripts

1. **`migrate_to_hybrid.py`** - Main orchestrator
   - Coordinates entire migration workflow
   - Provides comprehensive CLI interface
   - Generates detailed reports and statistics

2. **`graphdb_extractor.py`** - GraphDB data extraction
   - Extracts concepts, mappings, and relationships
   - Uses optimized SPARQL queries
   - Provides progress tracking and error handling

3. **`postgres_loader.py`** - PostgreSQL data loading
   - Batch loading with transaction safety
   - Schema validation and optimization
   - Hierarchical relationship calculation

4. **`data_validator.py`** - Integrity validation
   - 100% data integrity verification
   - Cross-system validation checks
   - Detailed mismatch reporting

## Migration Phases

### Phase 1: Pre-migration Setup
- ✅ Validate all connections
- ✅ Create backup if requested
- ✅ Verify PostgreSQL schema readiness
- ✅ Initialize comprehensive logging

### Phase 2: Data Extraction
- ✅ Extract concepts with metadata
- ✅ Extract terminology mappings
- ✅ Extract concept relationships
- ✅ Generate extraction statistics

### Phase 3: Data Loading
- ✅ Load concepts into PostgreSQL
- ✅ Load mappings with validation
- ✅ Load relationships with integrity checks
- ✅ Post-load optimizations (search vectors, hierarchies)

### Phase 4: Integrity Validation
- ✅ Compare source vs target data
- ✅ Validate concept integrity
- ✅ Validate mapping accuracy
- ✅ Validate relationship consistency
- ✅ Generate integrity score (target: ≥99%)

### Phase 5: GraphDB Optimization
- ✅ Backup current GraphDB state
- ✅ Clear non-essential data
- ✅ Load only core reasoning data
- ✅ Verify triple count (<5,000 target)

### Phase 6: Post-migration Verification
- ✅ Test PostgreSQL performance
- ✅ Verify GraphDB reasoning capabilities
- ✅ Generate comprehensive final report

## Configuration

### Configuration File (YAML)

```yaml
# GraphDB Source
graphdb_endpoint: "http://localhost:7200"
graphdb_repository: "kb7-terminology"
graphdb_username: null
graphdb_password: null

# PostgreSQL Target
postgres_url: "postgresql://user:pass@localhost:5433/kb7_terminology"

# Migration Settings
data_dir: "data"
logs_dir: "logs"
batch_size: 1000
validate_integrity: true
optimize_graphdb: true
backup_before_migration: true
```

### Environment Variables

```bash
export GRAPHDB_ENDPOINT="http://localhost:7200"
export GRAPHDB_REPOSITORY="kb7-terminology"
export POSTGRES_URL="postgresql://user:pass@localhost:5433/kb7_terminology"
```

## CLI Usage

### Basic Migration

```bash
# Full migration
python scripts/migrate_to_hybrid.py --config migration_config.yaml

# Migration with custom settings
python scripts/migrate_to_hybrid.py \
  --graphdb-endpoint http://localhost:7200 \
  --graphdb-repository kb7-terminology \
  --postgres-url postgresql://user:pass@localhost:5433/kb7_terminology \
  --batch-size 2000 \
  --data-dir ./migration_data
```

### Validation and Testing

```bash
# Dry run (configuration validation only)
python scripts/migrate_to_hybrid.py --config migration_config.yaml --dry-run

# Validation only (check integrity without migration)
python scripts/migrate_to_hybrid.py --config migration_config.yaml --validate-only

# Skip certain phases
python scripts/migrate_to_hybrid.py --config migration_config.yaml \
  --no-optimize \
  --no-backup \
  --no-validate
```

### Individual Components

```bash
# Extract data only
python scripts/graphdb_extractor.py \
  --endpoint http://localhost:7200 \
  --repository kb7-terminology \
  --output-dir ./data

# Load data only
python scripts/postgres_loader.py \
  --database-url postgresql://user:pass@localhost:5433/kb7_terminology \
  --input-dir ./data

# Validate integrity only
python scripts/data_validator.py \
  --graphdb-endpoint http://localhost:7200 \
  --graphdb-repository kb7-terminology \
  --postgres-url postgresql://user:pass@localhost:5433/kb7_terminology
```

## Docker Support

### Build Migration Container

```bash
docker build -t kb7-migration:3.5.1 .
```

### Run Migration in Container

```bash
# With volume-mounted config
docker run --rm \
  -v $(pwd)/migration_config.yaml:/app/migration/config/migration.yaml \
  -v $(pwd)/output:/app/migration/output \
  kb7-migration:3.5.1 \
  python scripts/migrate_to_hybrid.py --config config/migration.yaml

# With environment variables
docker run --rm \
  -e GRAPHDB_ENDPOINT=http://host.docker.internal:7200 \
  -e POSTGRES_URL=postgresql://user:pass@host.docker.internal:5433/kb7_terminology \
  -v $(pwd)/output:/app/migration/output \
  kb7-migration:3.5.1 \
  python scripts/migrate_to_hybrid.py \
    --graphdb-endpoint $GRAPHDB_ENDPOINT \
    --graphdb-repository kb7-terminology \
    --postgres-url $POSTGRES_URL
```

## Migration Statistics

### Expected Performance

| Metric | Target | Actual |
|--------|--------|--------|
| Total Records | 23,337 | TBD |
| Extraction Time | <5 minutes | TBD |
| Loading Time | <10 minutes | TBD |
| Validation Time | <5 minutes | TBD |
| Integrity Score | ≥99% | TBD |
| GraphDB Triples (after) | <5,000 | TBD |

### Output Files

```
migration/
├── data/                           # Extracted data
│   ├── concepts.json              # Concept data
│   ├── mappings.json              # Terminology mappings
│   ├── relationships.json         # Concept relationships
│   └── extraction_report.json     # Extraction statistics
├── logs/                          # Migration logs
│   ├── migration_YYYYMMDD_HHMMSS.log
│   ├── migration_stats.json       # Overall statistics
│   ├── loading_report.json        # Loading statistics
│   ├── validation_report.json     # Validation results
│   └── migration_final_report.md  # Human-readable summary
└── backups/                       # Pre-migration backups
    └── pre_migration_backup_*.json
```

## Troubleshooting

### Common Issues

1. **Connection Failures**
   ```bash
   # Test GraphDB connection
   curl http://localhost:7200/repositories/kb7-terminology

   # Test PostgreSQL connection
   psql postgresql://user:pass@localhost:5433/kb7_terminology -c "SELECT 1"
   ```

2. **Memory Issues**
   ```bash
   # Reduce batch size
   python scripts/migrate_to_hybrid.py --config migration_config.yaml --batch-size 500

   # Monitor memory usage
   docker stats  # If using Docker
   ```

3. **Performance Issues**
   ```bash
   # Check PostgreSQL performance
   psql -c "EXPLAIN ANALYZE SELECT COUNT(*) FROM concepts"

   # Optimize PostgreSQL
   psql -c "VACUUM ANALYZE concepts"
   ```

### Error Recovery

```bash
# Resume from specific phase (manual editing of phase completion tracking)
# Check migration_stats.json for completed phases

# Restart with validation only
python scripts/migrate_to_hybrid.py --config migration_config.yaml --validate-only

# Clean restart (clears all target data)
python scripts/migrate_to_hybrid.py --config migration_config.yaml --force-clean
```

## Security Considerations

- Store credentials in environment variables, not config files
- Use encrypted connections for production environments
- Backup original data before migration
- Validate data integrity before cutover
- Test rollback procedures

## Integration with KB7 System

### Post-Migration Steps

1. **Update Service Configuration**
   ```bash
   # Update query router to use hybrid mode
   cd ../query-router
   make configure-hybrid
   ```

2. **Test API Endpoints**
   ```bash
   # Test terminology lookup
   curl "http://localhost:8081/api/v1/terminology/lookup/rxnorm/123456"

   # Test reasoning queries
   curl "http://localhost:8081/api/v1/reasoning/interactions?drug1=123&drug2=456"
   ```

3. **Monitor Performance**
   ```bash
   # Check query performance
   cd ../monitoring
   ./check-query-performance.sh
   ```

## Support

- 📧 Team: KB7 Development Team
- 📋 Issues: Create GitHub issue with migration logs
- 📖 Documentation: See KB7_IMPLEMENTATION_PLAN.md
- 🔧 Status: Phase 3.5.1 Implementation