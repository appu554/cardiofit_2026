-- KB-3 Guideline Evidence Database Schema
-- PostgreSQL Migration Script

-- Create extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create guideline_documents table
CREATE TABLE guideline_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    guideline_id VARCHAR(255) UNIQUE NOT NULL,
    source JSONB NOT NULL,
    version VARCHAR(50) NOT NULL,
    effective_date TIMESTAMP WITH TIME ZONE NOT NULL,
    superseded_date TIMESTAMP WITH TIME ZONE,
    supersedes VARCHAR(255),
    condition JSONB NOT NULL,
    publication JSONB,
    status VARCHAR(50) DEFAULT 'active' NOT NULL,
    is_active BOOLEAN DEFAULT true NOT NULL,
    digital_signature TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by VARCHAR(255),
    updated_by VARCHAR(255)
);

-- Create indexes for guideline_documents
CREATE INDEX idx_guideline_documents_guideline_id ON guideline_documents(guideline_id);
CREATE INDEX idx_guideline_documents_effective_date ON guideline_documents(effective_date);
CREATE INDEX idx_guideline_documents_status ON guideline_documents(status);
CREATE INDEX idx_guideline_documents_is_active ON guideline_documents(is_active);
CREATE INDEX idx_guideline_documents_deleted_at ON guideline_documents(deleted_at);

-- Create GIN indexes for JSONB fields
CREATE INDEX idx_guideline_documents_source ON guideline_documents USING GIN(source);
CREATE INDEX idx_guideline_documents_condition ON guideline_documents USING GIN(condition);

-- Full-text search index for condition search
CREATE INDEX idx_guideline_documents_condition_text ON guideline_documents 
USING GIN(to_tsvector('english', condition->>'primary'));

-- Create recommendations table
CREATE TABLE recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    guideline_id UUID NOT NULL REFERENCES guideline_documents(id) ON DELETE CASCADE,
    rec_id VARCHAR(255) UNIQUE NOT NULL,
    domain VARCHAR(100) NOT NULL,
    subdomain VARCHAR(100),
    recommendation TEXT NOT NULL,
    evidence_grade VARCHAR(50) NOT NULL,
    strength VARCHAR(50),
    class_of_recommendation VARCHAR(10),
    level_of_evidence VARCHAR(10),
    applicability JSONB,
    linked_kb_refs JSONB,
    metrics JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for recommendations
CREATE INDEX idx_recommendations_guideline_id ON recommendations(guideline_id);
CREATE INDEX idx_recommendations_rec_id ON recommendations(rec_id);
CREATE INDEX idx_recommendations_domain ON recommendations(domain);
CREATE INDEX idx_recommendations_evidence_grade ON recommendations(evidence_grade);
CREATE INDEX idx_recommendations_deleted_at ON recommendations(deleted_at);

-- Create GIN indexes for JSONB fields in recommendations
CREATE INDEX idx_recommendations_applicability ON recommendations USING GIN(applicability);
CREATE INDEX idx_recommendations_linked_kb_refs ON recommendations USING GIN(linked_kb_refs);
CREATE INDEX idx_recommendations_metrics ON recommendations USING GIN(metrics);

-- Full-text search index for recommendation text
CREATE INDEX idx_recommendations_text_search ON recommendations 
USING GIN(to_tsvector('english', recommendation));

-- Create regional_profiles table
CREATE TABLE regional_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region VARCHAR(10) UNIQUE NOT NULL,
    primary_sources JSONB NOT NULL,
    measurement_units JSONB NOT NULL,
    regulatory_framework VARCHAR(100),
    applicability VARCHAR(255),
    focus VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for regional_profiles
CREATE INDEX idx_regional_profiles_region ON regional_profiles(region);
CREATE INDEX idx_regional_profiles_deleted_at ON regional_profiles(deleted_at);

-- Create guideline_versions table for version tracking
CREATE TABLE guideline_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    guideline_id UUID NOT NULL REFERENCES guideline_documents(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    change_log TEXT,
    is_active BOOLEAN DEFAULT false,
    is_draft BOOLEAN DEFAULT true,
    published_at TIMESTAMP WITH TIME ZONE,
    deprecated_at TIMESTAMP WITH TIME ZONE,
    guideline_snapshot JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by VARCHAR(255),
    updated_by VARCHAR(255)
);

-- Create indexes for guideline_versions
CREATE INDEX idx_guideline_versions_guideline_id ON guideline_versions(guideline_id);
CREATE INDEX idx_guideline_versions_version ON guideline_versions(version);
CREATE INDEX idx_guideline_versions_is_active ON guideline_versions(is_active);
CREATE INDEX idx_guideline_versions_deleted_at ON guideline_versions(deleted_at);

-- Create cross_kb_validation table for tracking link validation
CREATE TABLE cross_kb_validation (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recommendation_id UUID NOT NULL REFERENCES recommendations(id) ON DELETE CASCADE,
    kb_name VARCHAR(10) NOT NULL,
    target_id VARCHAR(255) NOT NULL,
    link_type VARCHAR(50) NOT NULL,
    validation_status VARCHAR(50) DEFAULT 'pending',
    last_validated TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for cross_kb_validation
CREATE INDEX idx_cross_kb_validation_rec_id ON cross_kb_validation(recommendation_id);
CREATE INDEX idx_cross_kb_validation_kb_name ON cross_kb_validation(kb_name);
CREATE INDEX idx_cross_kb_validation_status ON cross_kb_validation(validation_status);

-- Insert default regional profiles
INSERT INTO regional_profiles (region, primary_sources, measurement_units, regulatory_framework) VALUES 
('US', 
 '["ADA", "ACC/AHA", "KDIGO"]'::jsonb,
 '{"glucose": "mg/dL", "hba1c": "%", "blood_pressure": "mmHg"}'::jsonb,
 'FDA'),
 
('EU',
 '["ESC/ESH", "EASD"]'::jsonb, 
 '{"glucose": "mmol/L", "hba1c": "mmol/mol", "blood_pressure": "mmHg"}'::jsonb,
 'EMA'),
 
('AU',
 '["NHFA", "ADS", "RACGP"]'::jsonb,
 '{"glucose": "mmol/L", "hba1c": "%", "blood_pressure": "mmHg"}'::jsonb,
 'TGA'),
 
('WHO',
 '["WHO"]'::jsonb,
 '{"glucose": "mmol/L", "hba1c": "%", "blood_pressure": "mmHg"}'::jsonb,
 'WHO');

-- Set the regional profile for WHO with special applicability
UPDATE regional_profiles 
SET applicability = 'resource_limited_settings', focus = 'essential_medicines' 
WHERE region = 'WHO';

-- Create constraints
ALTER TABLE guideline_documents ADD CONSTRAINT chk_status 
    CHECK (status IN ('draft', 'review', 'approved', 'active', 'deprecated'));

ALTER TABLE recommendations ADD CONSTRAINT chk_evidence_grade
    CHECK (evidence_grade IN ('A', 'B', 'C', 'D', 'Expert Opinion', 'Good Practice Point'));

ALTER TABLE recommendations ADD CONSTRAINT chk_strength
    CHECK (strength IS NULL OR strength IN ('Strong', 'Conditional', 'Weak', 'Against'));

ALTER TABLE recommendations ADD CONSTRAINT chk_class_recommendation
    CHECK (class_of_recommendation IS NULL OR class_of_recommendation IN ('I', 'IIa', 'IIb', 'III'));

ALTER TABLE recommendations ADD CONSTRAINT chk_level_evidence
    CHECK (level_of_evidence IS NULL OR level_of_evidence IN ('A', 'B-R', 'B-NR', 'C-LD', 'C-EO'));

ALTER TABLE cross_kb_validation ADD CONSTRAINT chk_kb_name
    CHECK (kb_name IN ('kb1', 'kb2', 'kb4', 'kb5', 'kb6', 'kb7'));

ALTER TABLE cross_kb_validation ADD CONSTRAINT chk_validation_status
    CHECK (validation_status IN ('pending', 'validated', 'broken', 'missing'));

-- Create composite unique constraint for guideline version tracking
ALTER TABLE guideline_versions ADD CONSTRAINT uq_guideline_version 
    UNIQUE (guideline_id, version);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_guideline_documents_updated_at 
    BEFORE UPDATE ON guideline_documents 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_recommendations_updated_at 
    BEFORE UPDATE ON recommendations 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_regional_profiles_updated_at 
    BEFORE UPDATE ON regional_profiles 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_guideline_versions_updated_at 
    BEFORE UPDATE ON guideline_versions 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_cross_kb_validation_updated_at 
    BEFORE UPDATE ON cross_kb_validation 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create views for common queries

-- View for active guidelines with their recommendation counts
CREATE VIEW active_guidelines_summary AS
SELECT 
    gd.id,
    gd.guideline_id,
    gd.source->>'organization' as organization,
    gd.source->>'region' as region,
    gd.condition->>'primary' as primary_condition,
    gd.version,
    gd.effective_date,
    gd.superseded_date,
    COUNT(r.id) as recommendation_count,
    gd.status,
    gd.created_at,
    gd.updated_at
FROM guideline_documents gd
LEFT JOIN recommendations r ON gd.id = r.guideline_id AND r.deleted_at IS NULL
WHERE gd.deleted_at IS NULL AND gd.is_active = true
GROUP BY gd.id, gd.guideline_id, gd.source, gd.condition, gd.version, 
         gd.effective_date, gd.superseded_date, gd.status, gd.created_at, gd.updated_at;

-- View for recommendations with cross-KB link counts
CREATE VIEW recommendations_with_links AS
SELECT 
    r.id,
    r.rec_id,
    r.domain,
    r.subdomain,
    r.recommendation,
    r.evidence_grade,
    r.strength,
    gd.guideline_id,
    gd.source->>'organization' as organization,
    gd.source->>'region' as region,
    COALESCE(jsonb_array_length(r.linked_kb_refs->'kb1_dosing'), 0) as kb1_links,
    COALESCE(jsonb_array_length(r.linked_kb_refs->'kb2_phenotypes'), 0) as kb2_links,
    COALESCE(jsonb_array_length(r.linked_kb_refs->'kb4_safety'), 0) as kb4_links,
    COALESCE(jsonb_array_length(r.linked_kb_refs->'kb5_interactions'), 0) as kb5_links,
    COALESCE(jsonb_array_length(r.linked_kb_refs->'kb6_formulary'), 0) as kb6_links,
    COALESCE(jsonb_array_length(r.linked_kb_refs->'kb7_terminology'), 0) as kb7_links,
    r.created_at,
    r.updated_at
FROM recommendations r
JOIN guideline_documents gd ON r.guideline_id = gd.id
WHERE r.deleted_at IS NULL AND gd.deleted_at IS NULL;

-- Create function for full-text search across guidelines and recommendations
CREATE OR REPLACE FUNCTION search_guidelines(search_query TEXT)
RETURNS TABLE (
    guideline_id VARCHAR(255),
    organization TEXT,
    region TEXT,
    primary_condition TEXT,
    recommendation_id VARCHAR(255),
    recommendation_text TEXT,
    evidence_grade VARCHAR(50),
    relevance_rank REAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT DISTINCT
        gd.guideline_id,
        gd.source->>'organization' as organization,
        gd.source->>'region' as region,
        gd.condition->>'primary' as primary_condition,
        r.rec_id as recommendation_id,
        r.recommendation as recommendation_text,
        r.evidence_grade,
        (ts_rank(
            to_tsvector('english', gd.condition->>'primary' || ' ' || r.recommendation),
            plainto_tsquery('english', search_query)
        )) as relevance_rank
    FROM guideline_documents gd
    JOIN recommendations r ON gd.id = r.guideline_id
    WHERE gd.deleted_at IS NULL 
      AND r.deleted_at IS NULL
      AND gd.is_active = true
      AND (
          to_tsvector('english', gd.condition->>'primary') @@ plainto_tsquery('english', search_query)
          OR to_tsvector('english', r.recommendation) @@ plainto_tsquery('english', search_query)
      )
    ORDER BY relevance_rank DESC;
END;
$$ LANGUAGE plpgsql;