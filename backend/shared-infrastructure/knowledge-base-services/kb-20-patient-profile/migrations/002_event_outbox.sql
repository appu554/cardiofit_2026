-- 002_event_outbox.sql
-- Transactional outbox table for durable event delivery (G-03 remediation).
-- Events are written atomically in the same DB transaction as the data change,
-- then a background poller publishes and marks them as delivered.

CREATE TABLE IF NOT EXISTS event_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL,
    patient_id TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

-- Partial index for efficient polling of unpublished events
CREATE INDEX IF NOT EXISTS idx_event_outbox_unpublished
    ON event_outbox (created_at ASC)
    WHERE published_at IS NULL;

-- Index for event history queries by patient
CREATE INDEX IF NOT EXISTS idx_event_outbox_patient
    ON event_outbox (patient_id, created_at DESC);
