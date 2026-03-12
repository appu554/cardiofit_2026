-- KB-7 Terminology Releases Outbox Table
-- This table acts as the CDC event source for terminology updates
-- Debezium monitors this table and publishes changes to Kafka
--
-- IMPORTANT: Only INSERT into this table AFTER GraphDB is fully loaded and verified
-- This implements the "Commit-Last" strategy to prevent race conditions in EDA

-- Create the kb_releases table (Outbox pattern)
CREATE TABLE IF NOT EXISTS kb_releases (
    id SERIAL PRIMARY KEY,

    -- Version identification
    version_id VARCHAR(50) UNIQUE NOT NULL,  -- e.g., "20251203" or "latest"

    -- Timestamps
    release_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    graphdb_load_started_at TIMESTAMP WITH TIME ZONE,
    graphdb_load_completed_at TIMESTAMP WITH TIME ZONE,

    -- Source terminology versions
    snomed_version VARCHAR(50),      -- e.g., "2024-09"
    rxnorm_version VARCHAR(50),      -- e.g., "12012025"
    loinc_version VARCHAR(50),       -- e.g., "2.77"

    -- Content metrics
    triple_count BIGINT,             -- Total triples in GraphDB
    concept_count INTEGER,           -- Total concepts loaded
    snomed_concept_count INTEGER,
    rxnorm_concept_count INTEGER,
    loinc_concept_count INTEGER,

    -- File information
    kernel_file_size_bytes BIGINT,
    kernel_checksum VARCHAR(64),     -- SHA-256 hash
    gcs_uri VARCHAR(500),            -- gs://bucket/version/kb7-kernel.ttl

    -- GraphDB information
    graphdb_repository VARCHAR(100) DEFAULT 'kb7-terminology',
    graphdb_endpoint VARCHAR(500),

    -- Status tracking
    status VARCHAR(20) DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'LOADING', 'ACTIVE', 'ARCHIVED', 'FAILED')),
    error_message TEXT,

    -- Metadata
    created_by VARCHAR(100) DEFAULT 'kb-factory-pipeline',
    notes TEXT
);

-- Enable CDC on this table (required for Debezium)
ALTER TABLE kb_releases REPLICA IDENTITY FULL;

-- Create index for common queries
CREATE INDEX IF NOT EXISTS idx_kb_releases_version ON kb_releases(version_id);
CREATE INDEX IF NOT EXISTS idx_kb_releases_status ON kb_releases(status);
CREATE INDEX IF NOT EXISTS idx_kb_releases_date ON kb_releases(release_date DESC);

-- Create a view for the current active release
CREATE OR REPLACE VIEW current_kb_release AS
SELECT * FROM kb_releases
WHERE status = 'ACTIVE'
ORDER BY release_date DESC
LIMIT 1;

-- Function to archive previous active releases when a new one becomes active
CREATE OR REPLACE FUNCTION archive_previous_releases()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'ACTIVE' THEN
        UPDATE kb_releases
        SET status = 'ARCHIVED'
        WHERE id != NEW.id AND status = 'ACTIVE';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-archive previous releases
DROP TRIGGER IF EXISTS trigger_archive_previous_releases ON kb_releases;
CREATE TRIGGER trigger_archive_previous_releases
    AFTER UPDATE ON kb_releases
    FOR EACH ROW
    WHEN (NEW.status = 'ACTIVE')
    EXECUTE FUNCTION archive_previous_releases();

-- Grant permissions for CDC
GRANT SELECT ON kb_releases TO debezium;
GRANT SELECT ON current_kb_release TO debezium;

-- Comments for documentation
COMMENT ON TABLE kb_releases IS 'Outbox table for KB-7 terminology releases - monitored by Debezium CDC';
COMMENT ON COLUMN kb_releases.version_id IS 'Unique version identifier (e.g., 20251203)';
COMMENT ON COLUMN kb_releases.status IS 'PENDING=Not started, LOADING=GraphDB import in progress, ACTIVE=Ready for use, ARCHIVED=Superseded, FAILED=Error';
COMMENT ON COLUMN kb_releases.triple_count IS 'Total RDF triples loaded into GraphDB';
