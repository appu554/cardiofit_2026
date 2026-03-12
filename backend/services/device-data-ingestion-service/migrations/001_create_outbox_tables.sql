-- Transactional Outbox Pattern Database Migration
-- Creates per-vendor outbox tables for true fault isolation
-- Date: 2025-06-27
-- Version: 1.0

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =====================================================
-- VENDOR REGISTRY TABLE
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
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_outbox_table UNIQUE (outbox_table_name),
    CONSTRAINT unique_dead_letter_table UNIQUE (dead_letter_table_name)
);

-- Create indexes for vendor registry
CREATE INDEX IF NOT EXISTS idx_vendor_registry_active ON vendor_outbox_registry (is_active);
CREATE INDEX IF NOT EXISTS idx_vendor_registry_updated ON vendor_outbox_registry (updated_at);

-- =====================================================
-- FITBIT OUTBOX TABLES
-- =====================================================
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
    trace_id VARCHAR(255),
    
    -- Performance indexes
    CONSTRAINT fitbit_outbox_status_check CHECK (status IN ('pending', 'processing', 'completed', 'failed'))
);

-- Optimized indexes for Fitbit outbox
CREATE INDEX IF NOT EXISTS idx_fitbit_outbox_status_created ON fitbit_outbox (status, created_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_fitbit_outbox_device ON fitbit_outbox (device_id);
CREATE INDEX IF NOT EXISTS idx_fitbit_outbox_correlation ON fitbit_outbox (correlation_id);
CREATE INDEX IF NOT EXISTS idx_fitbit_outbox_processing ON fitbit_outbox (status, processed_at) WHERE status = 'processing';

-- Fitbit dead letter table
CREATE TABLE IF NOT EXISTS fitbit_dead_letter (
    id UUID PRIMARY KEY,
    device_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    event_payload JSONB NOT NULL,
    kafka_topic VARCHAR(255) NOT NULL,
    kafka_key VARCHAR(255),
    original_created_at TIMESTAMPTZ NOT NULL,
    failed_at TIMESTAMPTZ DEFAULT NOW(),
    final_error TEXT NOT NULL,
    retry_count INT NOT NULL,
    correlation_id UUID,
    trace_id VARCHAR(255),
    failure_reason VARCHAR(255) DEFAULT 'max_retries_exceeded'
);

-- Indexes for Fitbit dead letter
CREATE INDEX IF NOT EXISTS idx_fitbit_dead_letter_failed ON fitbit_dead_letter (failed_at);
CREATE INDEX IF NOT EXISTS idx_fitbit_dead_letter_correlation ON fitbit_dead_letter (correlation_id);
CREATE INDEX IF NOT EXISTS idx_fitbit_dead_letter_device ON fitbit_dead_letter (device_id);

-- =====================================================
-- GARMIN OUTBOX TABLES
-- =====================================================
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

-- Optimized indexes for Garmin outbox
CREATE INDEX IF NOT EXISTS idx_garmin_outbox_status_created ON garmin_outbox (status, created_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_garmin_outbox_device ON garmin_outbox (device_id);
CREATE INDEX IF NOT EXISTS idx_garmin_outbox_correlation ON garmin_outbox (correlation_id);
CREATE INDEX IF NOT EXISTS idx_garmin_outbox_processing ON garmin_outbox (status, processed_at) WHERE status = 'processing';

-- Garmin dead letter table
CREATE TABLE IF NOT EXISTS garmin_dead_letter (
    id UUID PRIMARY KEY,
    device_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    event_payload JSONB NOT NULL,
    kafka_topic VARCHAR(255) NOT NULL,
    kafka_key VARCHAR(255),
    original_created_at TIMESTAMPTZ NOT NULL,
    failed_at TIMESTAMPTZ DEFAULT NOW(),
    final_error TEXT NOT NULL,
    retry_count INT NOT NULL,
    correlation_id UUID,
    trace_id VARCHAR(255),
    failure_reason VARCHAR(255) DEFAULT 'max_retries_exceeded'
);

-- Indexes for Garmin dead letter
CREATE INDEX IF NOT EXISTS idx_garmin_dead_letter_failed ON garmin_dead_letter (failed_at);
CREATE INDEX IF NOT EXISTS idx_garmin_dead_letter_correlation ON garmin_dead_letter (correlation_id);
CREATE INDEX IF NOT EXISTS idx_garmin_dead_letter_device ON garmin_dead_letter (device_id);

-- =====================================================
-- APPLE HEALTH OUTBOX TABLES
-- =====================================================
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

-- Optimized indexes for Apple Health outbox
CREATE INDEX IF NOT EXISTS idx_apple_health_outbox_status_created ON apple_health_outbox (status, created_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_apple_health_outbox_device ON apple_health_outbox (device_id);
CREATE INDEX IF NOT EXISTS idx_apple_health_outbox_correlation ON apple_health_outbox (correlation_id);
CREATE INDEX IF NOT EXISTS idx_apple_health_outbox_processing ON apple_health_outbox (status, processed_at) WHERE status = 'processing';

-- Apple Health dead letter table
CREATE TABLE IF NOT EXISTS apple_health_dead_letter (
    id UUID PRIMARY KEY,
    device_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    event_payload JSONB NOT NULL,
    kafka_topic VARCHAR(255) NOT NULL,
    kafka_key VARCHAR(255),
    original_created_at TIMESTAMPTZ NOT NULL,
    failed_at TIMESTAMPTZ DEFAULT NOW(),
    final_error TEXT NOT NULL,
    retry_count INT NOT NULL,
    correlation_id UUID,
    trace_id VARCHAR(255),
    failure_reason VARCHAR(255) DEFAULT 'max_retries_exceeded'
);

-- Indexes for Apple Health dead letter
CREATE INDEX IF NOT EXISTS idx_apple_health_dead_letter_failed ON apple_health_dead_letter (failed_at);
CREATE INDEX IF NOT EXISTS idx_apple_health_dead_letter_correlation ON apple_health_dead_letter (correlation_id);
CREATE INDEX IF NOT EXISTS idx_apple_health_dead_letter_device ON apple_health_dead_letter (device_id);

-- =====================================================
-- ADDITIONAL MEDICAL DEVICE VENDOR TABLES
-- =====================================================

-- Medical Device outbox table (for clinical-grade devices)
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

CREATE INDEX IF NOT EXISTS idx_medical_device_outbox_status_created ON medical_device_outbox (status, created_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_medical_device_outbox_device ON medical_device_outbox (device_id);
CREATE INDEX IF NOT EXISTS idx_medical_device_outbox_correlation ON medical_device_outbox (correlation_id);

-- Medical Device dead letter table
CREATE TABLE IF NOT EXISTS medical_device_dead_letter (
    id UUID PRIMARY KEY,
    device_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    event_payload JSONB NOT NULL,
    kafka_topic VARCHAR(255) NOT NULL,
    kafka_key VARCHAR(255),
    original_created_at TIMESTAMPTZ NOT NULL,
    failed_at TIMESTAMPTZ DEFAULT NOW(),
    final_error TEXT NOT NULL,
    retry_count INT NOT NULL,
    correlation_id UUID,
    trace_id VARCHAR(255),
    failure_reason VARCHAR(255) DEFAULT 'max_retries_exceeded'
);

CREATE INDEX IF NOT EXISTS idx_medical_device_dead_letter_failed ON medical_device_dead_letter (failed_at);
CREATE INDEX IF NOT EXISTS idx_medical_device_dead_letter_correlation ON medical_device_dead_letter (correlation_id);

-- Generic Device outbox table (for unknown/new devices)
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

CREATE INDEX IF NOT EXISTS idx_generic_device_outbox_status_created ON generic_device_outbox (status, created_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_generic_device_outbox_device ON generic_device_outbox (device_id);
CREATE INDEX IF NOT EXISTS idx_generic_device_outbox_correlation ON generic_device_outbox (correlation_id);

-- Generic Device dead letter table
CREATE TABLE IF NOT EXISTS generic_device_dead_letter (
    id UUID PRIMARY KEY,
    device_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    event_payload JSONB NOT NULL,
    kafka_topic VARCHAR(255) NOT NULL,
    kafka_key VARCHAR(255),
    original_created_at TIMESTAMPTZ NOT NULL,
    failed_at TIMESTAMPTZ DEFAULT NOW(),
    final_error TEXT NOT NULL,
    retry_count INT NOT NULL,
    correlation_id UUID,
    trace_id VARCHAR(255),
    failure_reason VARCHAR(255) DEFAULT 'max_retries_exceeded'
);

CREATE INDEX IF NOT EXISTS idx_generic_device_dead_letter_failed ON generic_device_dead_letter (failed_at);
CREATE INDEX IF NOT EXISTS idx_generic_device_dead_letter_correlation ON generic_device_dead_letter (correlation_id);

-- =====================================================
-- POPULATE VENDOR REGISTRY
-- =====================================================
-- Enhanced vendor registry data with medical device support
INSERT INTO vendor_outbox_registry (vendor_id, vendor_name, outbox_table_name, dead_letter_table_name) VALUES
('fitbit', 'Fitbit', 'fitbit_outbox', 'fitbit_dead_letter'),
('garmin', 'Garmin', 'garmin_outbox', 'garmin_dead_letter'),
('apple_health', 'Apple Health', 'apple_health_outbox', 'apple_health_dead_letter'),
('samsung_health', 'Samsung Health', 'samsung_health_outbox', 'samsung_health_dead_letter'),
('withings', 'Withings', 'withings_outbox', 'withings_dead_letter'),
('omron', 'Omron', 'omron_outbox', 'omron_dead_letter'),
('polar', 'Polar', 'polar_outbox', 'polar_dead_letter'),
('suunto', 'Suunto', 'suunto_outbox', 'suunto_dead_letter'),
('medical_device', 'Medical Device', 'medical_device_outbox', 'medical_device_dead_letter'),
('generic_device', 'Generic Device', 'generic_device_outbox', 'generic_device_dead_letter')
ON CONFLICT (vendor_id) DO NOTHING;

-- =====================================================
-- UTILITY FUNCTIONS
-- =====================================================

-- Function to get outbox table name for a vendor
CREATE OR REPLACE FUNCTION get_outbox_table_name(p_vendor_id VARCHAR)
RETURNS VARCHAR AS $$
DECLARE
    table_name VARCHAR;
BEGIN
    SELECT outbox_table_name INTO table_name
    FROM vendor_outbox_registry
    WHERE vendor_id = p_vendor_id AND is_active = true;
    
    IF table_name IS NULL THEN
        RAISE EXCEPTION 'Unknown or inactive vendor: %', p_vendor_id;
    END IF;
    
    RETURN table_name;
END;
$$ LANGUAGE plpgsql;

-- Function to get dead letter table name for a vendor
CREATE OR REPLACE FUNCTION get_dead_letter_table_name(p_vendor_id VARCHAR)
RETURNS VARCHAR AS $$
DECLARE
    table_name VARCHAR;
BEGIN
    SELECT dead_letter_table_name INTO table_name
    FROM vendor_outbox_registry
    WHERE vendor_id = p_vendor_id AND is_active = true;
    
    IF table_name IS NULL THEN
        RAISE EXCEPTION 'Unknown or inactive vendor: %', p_vendor_id;
    END IF;
    
    RETURN table_name;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- MONITORING VIEWS
-- =====================================================

-- View for outbox queue depths per vendor
CREATE OR REPLACE VIEW outbox_queue_depths AS
SELECT 
    'fitbit' as vendor_id,
    COUNT(*) as pending_messages,
    MIN(created_at) as oldest_message,
    MAX(created_at) as newest_message
FROM fitbit_outbox WHERE status = 'pending'
UNION ALL
SELECT 
    'garmin' as vendor_id,
    COUNT(*) as pending_messages,
    MIN(created_at) as oldest_message,
    MAX(created_at) as newest_message
FROM garmin_outbox WHERE status = 'pending'
UNION ALL
SELECT 
    'apple_health' as vendor_id,
    COUNT(*) as pending_messages,
    MIN(created_at) as oldest_message,
    MAX(created_at) as newest_message
FROM apple_health_outbox WHERE status = 'pending';

-- View for dead letter statistics
CREATE OR REPLACE VIEW dead_letter_stats AS
SELECT 
    'fitbit' as vendor_id,
    COUNT(*) as dead_letter_count,
    COUNT(CASE WHEN failed_at > NOW() - INTERVAL '1 hour' THEN 1 END) as recent_failures
FROM fitbit_dead_letter
UNION ALL
SELECT 
    'garmin' as vendor_id,
    COUNT(*) as dead_letter_count,
    COUNT(CASE WHEN failed_at > NOW() - INTERVAL '1 hour' THEN 1 END) as recent_failures
FROM garmin_dead_letter
UNION ALL
SELECT 
    'apple_health' as vendor_id,
    COUNT(*) as dead_letter_count,
    COUNT(CASE WHEN failed_at > NOW() - INTERVAL '1 hour' THEN 1 END) as recent_failures
FROM apple_health_dead_letter;

-- Migration completed successfully
-- Run this script in your Supabase SQL editor or via psql
