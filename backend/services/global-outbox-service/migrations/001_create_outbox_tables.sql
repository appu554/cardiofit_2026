-- =====================================================================
-- Global Outbox Service Database Schema
-- =====================================================================
-- This migration creates the partitioned outbox tables and supporting
-- infrastructure for the Global Outbox Service.
--
-- Key Features:
-- - Service-based partitioning for performance and isolation
-- - Optimized indexes for publisher polling (SELECT FOR UPDATE SKIP LOCKED)
-- - Dead letter queue for failed messages
-- - Comprehensive monitoring and debugging support
-- =====================================================================

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =====================================================================
-- Main Outbox Table (Partitioned by Service)
-- =====================================================================

-- Drop existing table if it exists (for development)
DROP TABLE IF EXISTS global_event_outbox CASCADE;

-- Create main outbox table with service-based partitioning
CREATE TABLE global_event_outbox (
    id UUID DEFAULT uuid_generate_v4(),

    -- Status tracking
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processing', 'published', 'failed', 'scheduled')),

    -- Event payload and routing
    event_payload BYTEA NOT NULL,
    kafka_topic VARCHAR(255) NOT NULL,
    kafka_key VARCHAR(255),
    event_type VARCHAR(100),

    -- Context and provenance
    origin_service VARCHAR(100) NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    correlation_id VARCHAR(255),
    causation_id VARCHAR(255),
    subject VARCHAR(255),

    -- Processing state
    retry_count INT NOT NULL DEFAULT 0,
    last_error TEXT,
    priority INT NOT NULL DEFAULT 1 CHECK (priority BETWEEN 0 AND 3),

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    scheduled_at TIMESTAMPTZ,

    -- Metadata
    metadata JSONB,

    -- Primary key must include partition key for partitioned tables
    PRIMARY KEY (id, origin_service),

    -- Unique constraint for client-side idempotency
    UNIQUE (origin_service, idempotency_key)
) PARTITION BY LIST (origin_service);

-- =====================================================================
-- Service-Specific Partitions
-- =====================================================================

-- Create partitions for existing microservices
CREATE TABLE outbox_patient_service 
    PARTITION OF global_event_outbox FOR VALUES IN ('patient-service');

CREATE TABLE outbox_observation_service 
    PARTITION OF global_event_outbox FOR VALUES IN ('observation-service');

CREATE TABLE outbox_condition_service 
    PARTITION OF global_event_outbox FOR VALUES IN ('condition-service');

CREATE TABLE outbox_medication_service 
    PARTITION OF global_event_outbox FOR VALUES IN ('medication-service');

CREATE TABLE outbox_encounter_service 
    PARTITION OF global_event_outbox FOR VALUES IN ('encounter-service');

CREATE TABLE outbox_timeline_service 
    PARTITION OF global_event_outbox FOR VALUES IN ('timeline-service');

CREATE TABLE outbox_workflow_engine_service 
    PARTITION OF global_event_outbox FOR VALUES IN ('workflow-engine-service');

CREATE TABLE outbox_order_management_service 
    PARTITION OF global_event_outbox FOR VALUES IN ('order-management-service');

CREATE TABLE outbox_scheduling_service 
    PARTITION OF global_event_outbox FOR VALUES IN ('scheduling-service');

CREATE TABLE outbox_organization_service 
    PARTITION OF global_event_outbox FOR VALUES IN ('organization-service');

CREATE TABLE outbox_device_data_ingestion_service 
    PARTITION OF global_event_outbox FOR VALUES IN ('device-data-ingestion-service');

CREATE TABLE outbox_lab_service 
    PARTITION OF global_event_outbox FOR VALUES IN ('lab-service');

CREATE TABLE outbox_fhir_service 
    PARTITION OF global_event_outbox FOR VALUES IN ('fhir-service');

-- Generic partition for new services
CREATE TABLE outbox_generic_service
    PARTITION OF global_event_outbox FOR VALUES IN ('generic-service');

-- Test partition for testing purposes
CREATE TABLE outbox_test_service
    PARTITION OF global_event_outbox FOR VALUES IN ('test-service');

-- =====================================================================
-- Performance Indexes
-- =====================================================================

-- CRITICAL: Index for publisher polling with SELECT FOR UPDATE SKIP LOCKED
-- This is the most important index for performance
CREATE INDEX idx_outbox_pending_poll 
    ON global_event_outbox (status, priority DESC, created_at ASC) 
    WHERE status = 'pending';

-- Index for scheduled events
CREATE INDEX idx_outbox_scheduled_poll
    ON global_event_outbox (status, scheduled_at ASC)
    WHERE status = 'scheduled';

-- Index for monitoring and debugging by service
CREATE INDEX idx_outbox_service_status 
    ON global_event_outbox (origin_service, status, created_at DESC);

-- Index for correlation tracking (debugging)
CREATE INDEX idx_outbox_correlation 
    ON global_event_outbox (correlation_id) 
    WHERE correlation_id IS NOT NULL;

-- Index for causation tracking (event sourcing)
CREATE INDEX idx_outbox_causation 
    ON global_event_outbox (causation_id) 
    WHERE causation_id IS NOT NULL;

-- Index for subject-based queries
CREATE INDEX idx_outbox_subject 
    ON global_event_outbox (subject, created_at DESC) 
    WHERE subject IS NOT NULL;

-- Index for failed events (monitoring)
CREATE INDEX idx_outbox_failed_events 
    ON global_event_outbox (status, retry_count, created_at DESC) 
    WHERE status = 'failed';

-- =====================================================================
-- Dead Letter Queue
-- =====================================================================

-- Drop existing table if it exists (for development)
DROP TABLE IF EXISTS global_dead_letter_queue CASCADE;

-- Create dead letter queue for messages that exceed retry limits
CREATE TABLE global_dead_letter_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Original event information
    original_outbox_id UUID,
    origin_service VARCHAR(100) NOT NULL,
    event_type VARCHAR(100),
    event_payload BYTEA NOT NULL,
    kafka_topic VARCHAR(255) NOT NULL,
    kafka_key VARCHAR(255),
    
    -- Context
    correlation_id VARCHAR(255),
    causation_id VARCHAR(255),
    subject VARCHAR(255),
    
    -- Failure information
    final_error TEXT NOT NULL,
    retry_count INT NOT NULL,
    original_created_at TIMESTAMPTZ NOT NULL,
    failed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Metadata
    metadata JSONB,
    
    -- Processing status for DLQ
    dlq_status VARCHAR(20) NOT NULL DEFAULT 'quarantined' 
        CHECK (dlq_status IN ('quarantined', 'investigating', 'resolved', 'discarded'))
);

-- Indexes for dead letter queue
CREATE INDEX idx_dlq_service_failed_at 
    ON global_dead_letter_queue (origin_service, failed_at DESC);

CREATE INDEX idx_dlq_correlation 
    ON global_dead_letter_queue (correlation_id) 
    WHERE correlation_id IS NOT NULL;

CREATE INDEX idx_dlq_status 
    ON global_dead_letter_queue (dlq_status, failed_at DESC);

-- =====================================================================
-- Monitoring and Statistics Views
-- =====================================================================

-- View for queue depths by service
CREATE OR REPLACE VIEW outbox_queue_depths AS
SELECT 
    origin_service,
    COUNT(*) as queue_depth,
    COUNT(*) FILTER (WHERE priority = 3) as critical_count,
    COUNT(*) FILTER (WHERE priority = 2) as high_count,
    COUNT(*) FILTER (WHERE priority = 1) as normal_count,
    COUNT(*) FILTER (WHERE priority = 0) as low_count,
    MIN(created_at) as oldest_event,
    MAX(created_at) as newest_event
FROM global_event_outbox 
WHERE status = 'pending'
GROUP BY origin_service;

-- View for service statistics
CREATE OR REPLACE VIEW outbox_service_stats AS
SELECT 
    origin_service,
    COUNT(*) as total_events,
    COUNT(*) FILTER (WHERE status = 'published') as published_count,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
    COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
    ROUND(
        (COUNT(*) FILTER (WHERE status = 'published')::DECIMAL / 
         NULLIF(COUNT(*) FILTER (WHERE status IN ('published', 'failed')), 0)) * 100, 
        2
    ) as success_rate_percent,
    AVG(EXTRACT(EPOCH FROM (processed_at - created_at)) * 1000) 
        FILTER (WHERE processed_at IS NOT NULL) as avg_processing_time_ms
FROM global_event_outbox 
WHERE created_at > NOW() - INTERVAL '24 hours'
GROUP BY origin_service;

-- =====================================================================
-- Utility Functions
-- =====================================================================

-- Function to add new service partition
CREATE OR REPLACE FUNCTION add_service_partition(service_name TEXT)
RETURNS VOID AS $$
DECLARE
    partition_name TEXT;
BEGIN
    partition_name := 'outbox_' || replace(service_name, '-', '_');
    
    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF global_event_outbox FOR VALUES IN (%L)',
                   partition_name, service_name);
    
    RAISE NOTICE 'Created partition % for service %', partition_name, service_name;
END;
$$ LANGUAGE plpgsql;

-- Function to get outbox statistics
CREATE OR REPLACE FUNCTION get_outbox_statistics()
RETURNS JSON AS $$
DECLARE
    result JSON;
BEGIN
    SELECT json_build_object(
        'total_pending', (SELECT COUNT(*) FROM global_event_outbox WHERE status = 'pending'),
        'total_processing', (SELECT COUNT(*) FROM global_event_outbox WHERE status = 'processing'),
        'total_published_24h', (SELECT COUNT(*) FROM global_event_outbox 
                               WHERE status = 'published' AND processed_at > NOW() - INTERVAL '24 hours'),
        'total_failed', (SELECT COUNT(*) FROM global_event_outbox WHERE status = 'failed'),
        'dead_letter_count', (SELECT COUNT(*) FROM global_dead_letter_queue),
        'queue_depths', (SELECT json_object_agg(origin_service, queue_depth) 
                        FROM outbox_queue_depths),
        'service_stats', (SELECT json_agg(row_to_json(outbox_service_stats.*)) 
                         FROM outbox_service_stats),
        'timestamp', NOW()
    ) INTO result;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- =====================================================================
-- Comments and Documentation
-- =====================================================================

COMMENT ON TABLE global_event_outbox IS 'Main outbox table for guaranteed event delivery, partitioned by service';
COMMENT ON TABLE global_dead_letter_queue IS 'Dead letter queue for events that failed after maximum retries';
COMMENT ON COLUMN global_event_outbox.status IS 'Event status: pending, processing, published, failed, scheduled';
COMMENT ON COLUMN global_event_outbox.priority IS 'Event priority: 0=low, 1=normal, 2=high, 3=critical';
COMMENT ON INDEX idx_outbox_pending_poll IS 'CRITICAL: Optimized for SELECT FOR UPDATE SKIP LOCKED publisher polling';

-- =====================================================================
-- Migration Complete
-- =====================================================================

-- Log successful migration
DO $$
BEGIN
    RAISE NOTICE '✅ Global Outbox Service database schema created successfully';
    RAISE NOTICE '   - Main outbox table with % partitions created', 
        (SELECT COUNT(*) FROM information_schema.tables 
         WHERE table_name LIKE 'outbox_%' AND table_type = 'BASE TABLE');
    RAISE NOTICE '   - Dead letter queue created';
    RAISE NOTICE '   - Performance indexes created';
    RAISE NOTICE '   - Monitoring views and functions created';
END $$;
