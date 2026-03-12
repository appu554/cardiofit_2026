package database

import (
	"fmt"
	"time"

	"github.com/clinical-synthesis-hub/workflow-engine/internal/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// NewConnection creates a new database connection pool
func NewConnection(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxConnections)
	db.SetMaxIdleConns(cfg.MinConnections)
	db.SetConnMaxLifetime(cfg.MaxLifetime)
	db.SetConnMaxIdleTime(cfg.IdleTimeout)

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// RunMigrations runs database migrations
func RunMigrations(db *sqlx.DB) error {
	migrations := []string{
		createInitialSchemaMigration,
		createIndexesMigration,
		createTriggersAndFunctionsMigration,
	}

	// Create migration tracking table if it doesn't exist
	if err := createMigrationTable(db); err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}

	for i, migration := range migrations {
		migrationID := fmt.Sprintf("%03d", i+1)
		if applied, err := isMigrationApplied(db, migrationID); err != nil {
			return fmt.Errorf("failed to check migration %s: %w", migrationID, err)
		} else if applied {
			continue
		}

		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migrationID, err)
		}

		if err := markMigrationApplied(db, migrationID); err != nil {
			return fmt.Errorf("failed to mark migration %s as applied: %w", migrationID, err)
		}
	}

	return nil
}

func createMigrationTable(db *sqlx.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ DEFAULT NOW()
		)
	`
	_, err := db.Exec(query)
	return err
}

func isMigrationApplied(db *sqlx.DB, migrationID string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE id = $1", migrationID).Scan(&count)
	return count > 0, err
}

func markMigrationApplied(db *sqlx.DB, migrationID string) error {
	_, err := db.Exec("INSERT INTO schema_migrations (id) VALUES ($1)", migrationID)
	return err
}

const createInitialSchemaMigration = `
-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Workflow definitions table
CREATE TABLE IF NOT EXISTS workflow_definitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    description TEXT,
    bpmn_data TEXT NOT NULL,
    variables JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    active BOOLEAN DEFAULT true,
    tags TEXT[] DEFAULT '{}',
    category VARCHAR(100),
    UNIQUE(name, version)
);

-- Workflow status enum
CREATE TYPE workflow_status AS ENUM (
    'pending', 'running', 'completed', 'failed', 'cancelled', 'suspended'
);

-- Workflow instances table
CREATE TABLE IF NOT EXISTS workflow_instances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    definition_id UUID REFERENCES workflow_definitions(id),
    patient_id VARCHAR(255) NOT NULL,
    status workflow_status DEFAULT 'pending',
    variables JSONB DEFAULT '{}',
    start_time TIMESTAMPTZ DEFAULT NOW(),
    end_time TIMESTAMPTZ,
    correlation_id VARCHAR(255) NOT NULL,
    snapshot_chain JSONB,
    parent_instance_id UUID REFERENCES workflow_instances(id),
    business_key VARCHAR(255),
    priority INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by VARCHAR(255),
    tags TEXT[] DEFAULT '{}'
);

-- Snapshot status enum
CREATE TYPE snapshot_status AS ENUM (
    'created', 'active', 'expired', 'archived', 'corrupted'
);

-- Workflow phase enum
CREATE TYPE workflow_phase AS ENUM (
    'calculate', 'validate', 'commit', 'override'
);

-- Snapshots table
CREATE TABLE IF NOT EXISTS snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    snapshot_id VARCHAR(255) UNIQUE NOT NULL,
    checksum VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    status snapshot_status DEFAULT 'created',
    phase_created workflow_phase NOT NULL,
    patient_id VARCHAR(255) NOT NULL,
    context_version VARCHAR(100) NOT NULL,
    metadata JSONB DEFAULT '{}',
    data JSONB NOT NULL
);

-- Task status enum
CREATE TYPE task_status AS ENUM (
    'created', 'assigned', 'in_progress', 'completed', 'cancelled', 'escalated'
);

-- Workflow tasks table
CREATE TABLE IF NOT EXISTS workflow_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_instance_id UUID REFERENCES workflow_instances(id),
    task_definition_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    assignee_id VARCHAR(255),
    candidate_groups TEXT[] DEFAULT '{}',
    status task_status DEFAULT 'created',
    variables JSONB DEFAULT '{}',
    form_key VARCHAR(255),
    due_date TIMESTAMPTZ,
    follow_up_date TIMESTAMPTZ,
    priority INTEGER DEFAULT 50,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    completed_by VARCHAR(255)
);

-- Workflow events table (audit trail)
CREATE TABLE IF NOT EXISTS workflow_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_instance_id UUID REFERENCES workflow_instances(id),
    task_id UUID REFERENCES workflow_tasks(id),
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB DEFAULT '{}',
    user_id VARCHAR(255),
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    correlation_id VARCHAR(255)
);

-- Workflow timers table
CREATE TABLE IF NOT EXISTS workflow_timers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_instance_id UUID REFERENCES workflow_instances(id),
    timer_name VARCHAR(255) NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    executed_at TIMESTAMPTZ,
    configuration JSONB DEFAULT '{}',
    status VARCHAR(50) DEFAULT 'scheduled',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Recipe references table
CREATE TABLE IF NOT EXISTS recipe_references (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    recipe_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    resolved_at TIMESTAMPTZ DEFAULT NOW(),
    resolution_source VARCHAR(50) NOT NULL,
    metadata JSONB DEFAULT '{}'
);

-- Evidence envelopes table
CREATE TABLE IF NOT EXISTS evidence_envelopes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    evidence_id VARCHAR(255) UNIQUE NOT NULL,
    snapshot_id VARCHAR(255) NOT NULL,
    phase workflow_phase NOT NULL,
    evidence_type VARCHAR(100) NOT NULL,
    content JSONB NOT NULL,
    confidence_score NUMERIC(3,2) NOT NULL,
    generated_at TIMESTAMPTZ DEFAULT NOW(),
    source VARCHAR(100) NOT NULL
);

-- Clinical overrides table
CREATE TABLE IF NOT EXISTS clinical_overrides (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    override_id VARCHAR(255) UNIQUE NOT NULL,
    workflow_id VARCHAR(255) NOT NULL,
    snapshot_id VARCHAR(255) NOT NULL,
    override_type VARCHAR(100) NOT NULL,
    original_verdict VARCHAR(50) NOT NULL,
    overridden_to VARCHAR(50) NOT NULL,
    clinician_id VARCHAR(255) NOT NULL,
    justification TEXT NOT NULL,
    override_tokens TEXT[] DEFAULT '{}',
    override_timestamp TIMESTAMPTZ DEFAULT NOW(),
    patient_context JSONB DEFAULT '{}'
);

-- Workflow metrics table
CREATE TABLE IF NOT EXISTS workflow_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_instance_id UUID REFERENCES workflow_instances(id),
    metric_name VARCHAR(100) NOT NULL,
    metric_value NUMERIC NOT NULL,
    recorded_at TIMESTAMPTZ DEFAULT NOW(),
    correlation_id VARCHAR(255)
);
`

const createIndexesMigration = `
-- Indexes for workflow_instances
CREATE INDEX IF NOT EXISTS idx_workflow_instances_patient_id ON workflow_instances(patient_id);
CREATE INDEX IF NOT EXISTS idx_workflow_instances_status ON workflow_instances(status);
CREATE INDEX IF NOT EXISTS idx_workflow_instances_correlation_id ON workflow_instances(correlation_id);
CREATE INDEX IF NOT EXISTS idx_workflow_instances_created_at ON workflow_instances(created_at);
CREATE INDEX IF NOT EXISTS idx_workflow_instances_definition_id ON workflow_instances(definition_id);
CREATE INDEX IF NOT EXISTS idx_workflow_instances_business_key ON workflow_instances(business_key);

-- Indexes for workflow_definitions
CREATE INDEX IF NOT EXISTS idx_workflow_definitions_name ON workflow_definitions(name);
CREATE INDEX IF NOT EXISTS idx_workflow_definitions_active ON workflow_definitions(active);
CREATE INDEX IF NOT EXISTS idx_workflow_definitions_category ON workflow_definitions(category);

-- Indexes for snapshots
CREATE INDEX IF NOT EXISTS idx_snapshots_snapshot_id ON snapshots(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_snapshots_patient_id ON snapshots(patient_id);
CREATE INDEX IF NOT EXISTS idx_snapshots_expires_at ON snapshots(expires_at);
CREATE INDEX IF NOT EXISTS idx_snapshots_status ON snapshots(status);
CREATE INDEX IF NOT EXISTS idx_snapshots_phase_created ON snapshots(phase_created);

-- Indexes for workflow_tasks
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_workflow_instance_id ON workflow_tasks(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_assignee_id ON workflow_tasks(assignee_id);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_status ON workflow_tasks(status);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_due_date ON workflow_tasks(due_date);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_priority ON workflow_tasks(priority);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_candidate_groups ON workflow_tasks USING GIN(candidate_groups);

-- Indexes for workflow_events
CREATE INDEX IF NOT EXISTS idx_workflow_events_workflow_instance_id ON workflow_events(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_workflow_events_task_id ON workflow_events(task_id);
CREATE INDEX IF NOT EXISTS idx_workflow_events_event_type ON workflow_events(event_type);
CREATE INDEX IF NOT EXISTS idx_workflow_events_timestamp ON workflow_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_workflow_events_correlation_id ON workflow_events(correlation_id);

-- Indexes for workflow_timers
CREATE INDEX IF NOT EXISTS idx_workflow_timers_workflow_instance_id ON workflow_timers(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_workflow_timers_scheduled_at ON workflow_timers(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_workflow_timers_status ON workflow_timers(status);

-- Indexes for workflow_metrics
CREATE INDEX IF NOT EXISTS idx_workflow_metrics_workflow_instance_id ON workflow_metrics(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_workflow_metrics_correlation_id ON workflow_metrics(correlation_id);
CREATE INDEX IF NOT EXISTS idx_workflow_metrics_metric_name ON workflow_metrics(metric_name);
CREATE INDEX IF NOT EXISTS idx_workflow_metrics_recorded_at ON workflow_metrics(recorded_at);

-- Indexes for evidence_envelopes
CREATE INDEX IF NOT EXISTS idx_evidence_envelopes_snapshot_id ON evidence_envelopes(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_evidence_envelopes_phase ON evidence_envelopes(phase);
CREATE INDEX IF NOT EXISTS idx_evidence_envelopes_evidence_type ON evidence_envelopes(evidence_type);

-- Indexes for clinical_overrides
CREATE INDEX IF NOT EXISTS idx_clinical_overrides_workflow_id ON clinical_overrides(workflow_id);
CREATE INDEX IF NOT EXISTS idx_clinical_overrides_snapshot_id ON clinical_overrides(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_clinical_overrides_clinician_id ON clinical_overrides(clinician_id);
CREATE INDEX IF NOT EXISTS idx_clinical_overrides_override_timestamp ON clinical_overrides(override_timestamp);
`

const createTriggersAndFunctionsMigration = `
-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updating updated_at columns
CREATE TRIGGER update_workflow_definitions_updated_at
    BEFORE UPDATE ON workflow_definitions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workflow_instances_updated_at
    BEFORE UPDATE ON workflow_instances
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workflow_tasks_updated_at
    BEFORE UPDATE ON workflow_tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to automatically expire snapshots
CREATE OR REPLACE FUNCTION expire_old_snapshots()
RETURNS void AS $$
BEGIN
    UPDATE snapshots
    SET status = 'expired'
    WHERE status = 'active' AND expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- Function to clean up old workflow events (keep last 30 days)
CREATE OR REPLACE FUNCTION cleanup_old_workflow_events()
RETURNS void AS $$
BEGIN
    DELETE FROM workflow_events
    WHERE timestamp < NOW() - INTERVAL '30 days';
END;
$$ LANGUAGE plpgsql;

-- Function to calculate workflow metrics
CREATE OR REPLACE FUNCTION calculate_workflow_duration()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.end_time IS NOT NULL AND OLD.end_time IS NULL THEN
        INSERT INTO workflow_metrics (workflow_instance_id, metric_name, metric_value, correlation_id)
        VALUES (
            NEW.id,
            'duration_seconds',
            EXTRACT(EPOCH FROM (NEW.end_time - NEW.start_time)),
            NEW.correlation_id
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically calculate duration when workflow completes
CREATE TRIGGER calculate_workflow_duration_trigger
    AFTER UPDATE ON workflow_instances
    FOR EACH ROW
    EXECUTE FUNCTION calculate_workflow_duration();

-- Function to validate snapshot consistency
CREATE OR REPLACE FUNCTION validate_snapshot_consistency()
RETURNS TRIGGER AS $$
DECLARE
    existing_patient_id VARCHAR(255);
    existing_context_version VARCHAR(100);
BEGIN
    -- Check if there are existing snapshots for this workflow
    SELECT DISTINCT s.patient_id, s.context_version
    INTO existing_patient_id, existing_context_version
    FROM snapshots s
    WHERE s.snapshot_id IN (
        SELECT DISTINCT jsonb_array_elements_text(
            COALESCE(
                NEW.snapshot_chain->'calculate_snapshot'->>'snapshot_id',
                NEW.snapshot_chain->'validate_snapshot'->>'snapshot_id',
                NEW.snapshot_chain->'commit_snapshot'->>'snapshot_id'
            )::jsonb
        )
    )
    LIMIT 1;

    -- If existing snapshots found, validate consistency
    IF existing_patient_id IS NOT NULL THEN
        IF NEW.patient_id != existing_patient_id THEN
            RAISE EXCEPTION 'Snapshot consistency violation: patient_id mismatch';
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to validate snapshot consistency on workflow instances
CREATE TRIGGER validate_snapshot_consistency_trigger
    BEFORE INSERT OR UPDATE ON workflow_instances
    FOR EACH ROW
    WHEN (NEW.snapshot_chain IS NOT NULL)
    EXECUTE FUNCTION validate_snapshot_consistency();
`