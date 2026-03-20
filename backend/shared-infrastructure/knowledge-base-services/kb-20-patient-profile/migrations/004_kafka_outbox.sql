-- Add Kafka delivery tracking to the event outbox.
-- The existing published_at column tracks in-memory subscriber delivery.
-- kafka_published_at tracks Kafka relay delivery independently.

ALTER TABLE event_outbox ADD COLUMN kafka_published_at TIMESTAMPTZ;

-- Partial index for efficient polling of Kafka-unpublished events
CREATE INDEX IF NOT EXISTS idx_event_outbox_kafka_unpublished
    ON event_outbox (created_at ASC)
    WHERE kafka_published_at IS NULL;
