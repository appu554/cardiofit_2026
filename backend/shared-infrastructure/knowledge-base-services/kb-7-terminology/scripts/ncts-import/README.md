# NCTS RF2 Import Automation Suite

Automated import and management of SNOMED CT-AU (NCTS) refset data into Neo4j.

## Features

- **Automatic RF2 Detection**: Scans ZIP archives for refset files
- **Version Tracking**: Prevents duplicate imports via `ImportMetadata` nodes
- **Pre-Import Backup**: Saves existing data before updates
- **Batch Processing**: Efficient large-scale imports with progress reporting
- **Rollback Capability**: Restore or delete refset data
- **Cron Automation**: Monthly automated update checking

## Quick Start

```bash
# 1. Configure environment
cp .env.template .env
# Edit .env with your Neo4j credentials

# 2. Run import
make import FILE=/path/to/SnomedCT_AU_20240930.zip

# 3. Verify import
make verify
```

## Prerequisites

- **Neo4j**: 4.4+ with APOC plugin
- **cypher-shell**: Neo4j command-line tool
- **unzip**: For extracting RF2 archives
- **Bash 4+**: Script compatibility

## Installation

```bash
# Make scripts executable
chmod +x *.sh

# Configure environment
cp .env.template .env
nano .env  # Edit with your settings

# Test connection
make test-connection
```

## Usage

### Standard Import

```bash
# Import with automatic version check and backup
make import FILE=/path/to/SnomedCT_AU_20240930.zip
```

### Dry Run (Validation Only)

```bash
# Validate without modifying data
make import-dry-run FILE=/path/to/ncts.zip
```

### Force Reimport

```bash
# Reimport even if same version exists
make import-force FILE=/path/to/ncts.zip
```

### View Import History

```bash
make list-versions
```

### Verify Import

```bash
make verify
make stats
```

### Rollback

```bash
# Interactive rollback (with confirmation)
make rollback

# Force delete all refset data
make delete-all
```

### Setup Automated Updates

```bash
# Create cron job for monthly checks
make setup-cron
```

## Directory Structure

```
ncts-import/
├── ncts_rf2_import.sh      # Main import script
├── ncts_rollback.sh        # Rollback/restore script
├── setup_cron.sh           # Cron automation setup
├── Makefile                # Command shortcuts
├── .env.template           # Configuration template
├── .env                    # Your configuration (gitignored)
├── cypher/
│   └── ncts_rf2_operations.cypher  # Manual Cypher scripts
├── backups/                # Pre-import backups
└── README.md               # This file
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NEO4J_URI` | `bolt://localhost:7687` | Neo4j bolt URI |
| `NEO4J_USER` | `neo4j` | Neo4j username |
| `NEO4J_PASSWORD` | (required) | Neo4j password |
| `NEO4J_DATABASE` | `neo4j` | Neo4j database |
| `BATCH_SIZE` | `10000` | Import batch size |
| `NCTS_DOWNLOAD_DIR` | `/path/to/downloads` | NCTS download location |

### .env Example

```bash
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=your_secure_password
NEO4J_DATABASE=snomed-au
BATCH_SIZE=10000
NCTS_DOWNLOAD_DIR=/data/ncts-downloads
```

## Neo4j Schema

### Relationship Types

| Type | Description |
|------|-------------|
| `:IN_REFSET` | Concept → Refset membership |
| `:REPLACED_BY` | Concept replacement (historical) |
| `:SAME_AS` | Equivalent concepts |
| `:MAPS_TO_ICD10` | ICD-10 mappings |

### Node Labels

| Label | Description |
|-------|-------------|
| `Refset` | Reference set definition |
| `ImportMetadata` | Version tracking |

### Indexes

```cypher
CREATE INDEX refset_id_idx FOR (r:Refset) ON (r.id);
CREATE INDEX import_metadata_idx FOR (m:ImportMetadata) ON (m.type, m.version);
CREATE CONSTRAINT refset_unique FOR (r:Refset) REQUIRE r.id IS UNIQUE;
```

## RF2 File Support

### Supported File Types

| Pattern | Type |
|---------|------|
| `der2_Refset_SimpleSnapshot*.txt` | Simple refsets |
| `der2_sRefset_SimpleSnapshot*.txt` | Simple refsets (alt) |
| `der2_cRefset_AssociationSnapshot*.txt` | Association refsets |
| `der2_cRefset_LanguageSnapshot*.txt` | Language refsets |
| `der2_sRefset_SimpleMapSnapshot*.txt` | Simple map refsets |

### RF2 Format

Tab-separated values with columns:
- `id`: Unique member UUID
- `effectiveTime`: YYYYMMDD format
- `active`: 0 or 1
- `moduleId`: Source module SCTID
- `refsetId`: Reference set SCTID
- `referencedComponentId`: Member concept SCTID

## Automated Monthly Updates

NCTS releases SNOMED CT-AU updates monthly (typically 1st-7th of month).

### Setup Cron

```bash
make setup-cron
```

This creates a cron job for the 15th of each month at 2 AM that:
1. Checks `NCTS_DOWNLOAD_DIR` for new ZIP files
2. Compares version against `ImportMetadata` in Neo4j
3. Imports if new version detected
4. Logs to `/var/log/ncts-import.log`

### Manual Cron Setup

```bash
# Edit crontab
crontab -e

# Add line (15th of month, 2 AM):
0 2 15 * * /path/to/ncts-import/ncts_cron_update.sh >> /var/log/ncts-import.log 2>&1
```

## API Integration

After import, refset data is available via KB-7 APIs:

```bash
# List all refsets
curl http://localhost:8087/v1/refsets

# Get refset members
curl http://localhost:8087/v1/refsets/32570581000036109/members

# Get concept's refsets
curl http://localhost:8087/v1/concepts/123456789/refsets

# Check membership
curl http://localhost:8087/v1/refsets/32570581000036109/contains/123456789
```

## Troubleshooting

### Connection Failed

```bash
# Test Neo4j connection
make test-connection

# Check Neo4j is running
neo4j status
```

### APOC Not Available

The scripts use APOC for batch operations. Ensure APOC is installed:

```cypher
RETURN apoc.version()
```

### Import Too Slow

Increase batch size for faster imports:

```bash
BATCH_SIZE=50000 make import FILE=/path/to/ncts.zip
```

### Out of Memory

Reduce batch size:

```bash
BATCH_SIZE=1000 make import FILE=/path/to/ncts.zip
```

## Module IDs

| ID | Module |
|----|--------|
| `32506021000036107` | SNOMED-AU |
| `900062011000036103` | AMT |
| `900000000000207008` | SNOMED-INT |

## License

Part of the CardioFit Clinical Synthesis Hub platform.
