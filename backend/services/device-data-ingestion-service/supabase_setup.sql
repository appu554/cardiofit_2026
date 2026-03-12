-- =====================================================
-- SUPABASE SETUP FOR OUTBOX PATTERN
-- =====================================================
-- Run this script in Supabase Dashboard → SQL Editor
-- This will create all necessary tables and configurations

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =====================================================
-- 1. VENDOR REGISTRY TABLE
-- =====================================================
CREATE TABLE IF NOT EXISTS vendor_outbox_registry (
    vendor_id VARCHAR(100) PRIMARY KEY,
    vendor_name VARCHAR(255) NOT NULL,
    outbox_table_name VARCHAR(255) NOT NULL,
    dead_letter_table_name VARCHAR(255) NOT NULL,
    kafka_topic VARCHAR(255) NOT NULL DEFAULT 'raw-device-data.v1',
    max_retries INT DEFAULT 3,
    retry_backoff_seconds INT DEFAULT 60,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- 2. OUTBOX TABLES FOR EACH VENDOR
-- =====================================================

-- Fitbit Outbox
CREATE TABLE IF NOT EXISTS fitbit_outbox (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(50) NOT NULL DEFAULT 'device_reading',
    event_payload JSONB NOT NULL,
    kafka_topic VARCHAR(255) NOT NULL DEFAULT 'raw-device-data.v1',
    kafka_key VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    last_error TEXT,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    correlation_id UUID,
    trace_id VARCHAR(255)
);

-- Garmin Outbox
CREATE TABLE IF NOT EXISTS garmin_outbox (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(50) NOT NULL DEFAULT 'device_reading',
    event_payload JSONB NOT NULL,
    kafka_topic VARCHAR(255) NOT NULL DEFAULT 'raw-device-data.v1',
    kafka_key VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    last_error TEXT,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    correlation_id UUID,
    trace_id VARCHAR(255)
);

-- Apple Health Outbox
CREATE TABLE IF NOT EXISTS apple_health_outbox (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(50) NOT NULL DEFAULT 'device_reading',
    event_payload JSONB NOT NULL,
    kafka_topic VARCHAR(255) NOT NULL DEFAULT 'raw-device-data.v1',
    kafka_key VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    last_error TEXT,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    correlation_id UUID,
    trace_id VARCHAR(255)
);

-- Medical Device Outbox
CREATE TABLE IF NOT EXISTS medical_device_outbox (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(50) NOT NULL DEFAULT 'device_reading',
    event_payload JSONB NOT NULL,
    kafka_topic VARCHAR(255) NOT NULL DEFAULT 'raw-device-data.v1',
    kafka_key VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    last_error TEXT,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    correlation_id UUID,
    trace_id VARCHAR(255)
);

-- Generic Device Outbox
CREATE TABLE IF NOT EXISTS generic_device_outbox (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(50) NOT NULL DEFAULT 'device_reading',
    event_payload JSONB NOT NULL,
    kafka_topic VARCHAR(255) NOT NULL DEFAULT 'raw-device-data.v1',
    kafka_key VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    last_error TEXT,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    correlation_id UUID,
    trace_id VARCHAR(255)
);

-- =====================================================
-- 3. CREATE INDEXES FOR PERFORMANCE
-- =====================================================

-- Fitbit indexes
CREATE INDEX IF NOT EXISTS idx_fitbit_outbox_status_created ON fitbit_outbox (status, created_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_fitbit_outbox_device ON fitbit_outbox (device_id);
CREATE INDEX IF NOT EXISTS idx_fitbit_outbox_correlation ON fitbit_outbox (correlation_id);

-- Garmin indexes
CREATE INDEX IF NOT EXISTS idx_garmin_outbox_status_created ON garmin_outbox (status, created_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_garmin_outbox_device ON garmin_outbox (device_id);
CREATE INDEX IF NOT EXISTS idx_garmin_outbox_correlation ON garmin_outbox (correlation_id);

-- Apple Health indexes
CREATE INDEX IF NOT EXISTS idx_apple_health_outbox_status_created ON apple_health_outbox (status, created_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_apple_health_outbox_device ON apple_health_outbox (device_id);
CREATE INDEX IF NOT EXISTS idx_apple_health_outbox_correlation ON apple_health_outbox (correlation_id);

-- Medical Device indexes
CREATE INDEX IF NOT EXISTS idx_medical_device_outbox_status_created ON medical_device_outbox (status, created_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_medical_device_outbox_device ON medical_device_outbox (device_id);
CREATE INDEX IF NOT EXISTS idx_medical_device_outbox_correlation ON medical_device_outbox (correlation_id);

-- Generic Device indexes
CREATE INDEX IF NOT EXISTS idx_generic_device_outbox_status_created ON generic_device_outbox (status, created_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_generic_device_outbox_device ON generic_device_outbox (device_id);
CREATE INDEX IF NOT EXISTS idx_generic_device_outbox_correlation ON generic_device_outbox (correlation_id);

-- =====================================================
-- 4. DISABLE ROW LEVEL SECURITY FOR SERVICE ACCESS
-- =====================================================
-- This allows your backend service to access the tables

ALTER TABLE vendor_outbox_registry DISABLE ROW LEVEL SECURITY;
ALTER TABLE fitbit_outbox DISABLE ROW LEVEL SECURITY;
ALTER TABLE garmin_outbox DISABLE ROW LEVEL SECURITY;
ALTER TABLE apple_health_outbox DISABLE ROW LEVEL SECURITY;
ALTER TABLE medical_device_outbox DISABLE ROW LEVEL SECURITY;
ALTER TABLE generic_device_outbox DISABLE ROW LEVEL SECURITY;

-- =====================================================
-- 5. POPULATE VENDOR REGISTRY
-- =====================================================
INSERT INTO vendor_outbox_registry (vendor_id, vendor_name, outbox_table_name, dead_letter_table_name) VALUES
('fitbit', 'Fitbit', 'fitbit_outbox', 'fitbit_dead_letter'),
('garmin', 'Garmin', 'garmin_outbox', 'garmin_dead_letter'),
('apple_health', 'Apple Health', 'apple_health_outbox', 'apple_health_dead_letter'),
('medical_device', 'Medical Device', 'medical_device_outbox', 'medical_device_dead_letter'),
('generic_device', 'Generic Device', 'generic_device_outbox', 'generic_device_dead_letter')
ON CONFLICT (vendor_id) DO NOTHING;

-- =====================================================
-- 6. VERIFICATION QUERIES
-- =====================================================
-- Run these to verify everything was created correctly

-- Check tables were created
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public' 
AND table_name LIKE '%outbox%'
ORDER BY table_name;

-- Check vendor registry
SELECT vendor_id, vendor_name, outbox_table_name, is_active 
FROM vendor_outbox_registry 
ORDER BY vendor_id;

-- Check indexes
SELECT schemaname, tablename, indexname 
FROM pg_indexes 
WHERE tablename LIKE '%outbox%' 
ORDER BY tablename, indexname;

-- =====================================================
-- SUCCESS MESSAGE
-- =====================================================
-- If you see results from the verification queries above,
-- your Supabase database is ready for the outbox pattern!
--
-- Next steps:
-- 1. Update your service configuration
-- 2. Start the device ingestion service
-- 3. Test the endpoints
