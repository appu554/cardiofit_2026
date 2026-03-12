-- KB-7 Terminology Service Initial Schema
-- Creates the complete terminology management infrastructure

-- Create extension for UUID generation if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- For text search performance

-- Terminology Systems table - manages different code systems (SNOMED CT, ICD-10, etc.)
CREATE TABLE IF NOT EXISTS terminology_systems (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    system_uri VARCHAR(500) UNIQUE NOT NULL,
    system_name VARCHAR(255) NOT NULL,
    version VARCHAR(100) NOT NULL,
    description TEXT,
    publisher VARCHAR(255),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('draft', 'active', 'retired', 'unknown')),
    
    -- System metadata
    metadata JSONB DEFAULT '{}',
    /* Example structure:
    {
      "copyright": "Copyright notice",
      "jurisdiction": ["US", "CA"],
      "contact": [{
        "name": "Contact Name",
        "telecom": [{"system": "email", "value": "contact@example.com"}]
      }],
      "experimental": false,
      "properties": ["parent", "child", "synonym"]
    }
    */
    
    -- Regional support
    supported_regions TEXT[] DEFAULT ARRAY['US'],
    
    -- Content metadata
    concept_count INTEGER DEFAULT 0,
    hierarchy_meaning VARCHAR(50), -- is-a, part-of, classified-with
    compositional BOOLEAN DEFAULT FALSE,
    version_needed BOOLEAN DEFAULT TRUE,
    content VARCHAR(20) DEFAULT 'complete' CHECK (content IN ('not-present', 'example', 'fragment', 'complete', 'supplement')),
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(system_uri, version)
);

-- Terminology Concepts table - stores individual concepts from each system
CREATE TABLE IF NOT EXISTS terminology_concepts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    system_id UUID REFERENCES terminology_systems(id) ON DELETE CASCADE,
    code VARCHAR(255) NOT NULL,
    display VARCHAR(500) NOT NULL,
    definition TEXT,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'entered-in-error')),
    
    -- Hierarchy relationships (stored as arrays for performance)
    parent_codes TEXT[] DEFAULT ARRAY[]::TEXT[],
    child_codes TEXT[] DEFAULT ARRAY[]::TEXT[],
    
    -- Additional properties (flexible JSONB storage)
    properties JSONB DEFAULT '{}',
    /* Example structure:
    {
      "module": "900000000000207008",
      "effective_time": "20220131",
      "primitive": true,
      "relationships": [{
        "type": "116680003",
        "target": "404684003",
        "characteristicType": "900000000000011006"
      }]
    }
    */
    
    -- Alternative terms and translations
    designations JSONB DEFAULT '[]',
    /* Example structure:
    [{
      "language": "en",
      "use": {"system": "http://snomed.info/sct", "code": "900000000000013009"},
      "value": "Synonym term"
    }]
    */
    
    -- Clinical classification
    clinical_domain VARCHAR(100),
    specialty VARCHAR(100),
    
    -- Search optimization
    search_terms TSVECTOR,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(system_id, code)
);

-- Value Sets table - collections of concepts for specific use cases
CREATE TABLE IF NOT EXISTS value_sets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    url VARCHAR(500) UNIQUE NOT NULL,
    identifier JSONB, -- Additional business identifiers
    version VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    title VARCHAR(500),
    description TEXT,
    status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'retired', 'unknown')),
    experimental BOOLEAN DEFAULT FALSE,
    
    -- Publication metadata
    date TIMESTAMPTZ,
    publisher VARCHAR(255),
    contact JSONB DEFAULT '[]',
    /* Example structure:
    [{
      "name": "Contact Name",
      "telecom": [{"system": "email", "value": "contact@example.com"}]
    }]
    */
    
    -- Use context and jurisdiction
    use_context JSONB DEFAULT '[]',
    /* Example structure:
    [{
      "code": {"system": "http://terminology.hl7.org/CodeSystem/usage-context-type", "code": "focus"},
      "valueCodeableConcept": {"coding": [{"system": "http://snomed.info/sct", "code": "87612001"}]}
    }]
    */
    jurisdiction JSONB DEFAULT '[]',
    
    -- Purpose and scope
    purpose TEXT,
    copyright TEXT,
    clinical_domain VARCHAR(100),
    
    -- Value set definition (compose rules)
    compose JSONB DEFAULT '{}',
    /* Example structure:
    {
      "lockedDate": "2022-01-01",
      "inactive": false,
      "include": [{
        "system": "http://snomed.info/sct",
        "version": "20220131",
        "concept": [{"code": "404684003", "display": "Clinical finding"}],
        "filter": [{
          "property": "concept",
          "op": "is-a",
          "value": "404684003"
        }]
      }],
      "exclude": []
    }
    */
    
    -- Computed expansion (snapshot of included concepts)
    expansion JSONB DEFAULT '{}',
    /* Example structure:
    {
      "identifier": "urn:uuid:12345",
      "timestamp": "2022-01-01T00:00:00Z",
      "total": 1500,
      "contains": [{
        "system": "http://snomed.info/sct",
        "code": "404684003",
        "display": "Clinical finding",
        "abstract": false
      }]
    }
    */
    
    -- Regional variations
    supported_regions TEXT[] DEFAULT ARRAY['US'],
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    expired_at TIMESTAMPTZ,
    
    UNIQUE(url, version)
);

-- Concept Mappings table - relationships between concepts in different systems
CREATE TABLE IF NOT EXISTS concept_mappings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Source concept
    source_system_id UUID REFERENCES terminology_systems(id) ON DELETE CASCADE,
    source_code VARCHAR(255) NOT NULL,
    
    -- Target concept
    target_system_id UUID REFERENCES terminology_systems(id) ON DELETE CASCADE,
    target_code VARCHAR(255) NOT NULL,
    
    -- Mapping relationship details
    equivalence VARCHAR(20) DEFAULT 'equivalent' CHECK (
        equivalence IN ('relatedto', 'equivalent', 'equal', 'wider', 'subsumes', 'narrower', 'specializes', 'inexact', 'unmatched', 'disjoint')
    ),
    mapping_type VARCHAR(50) DEFAULT 'manual', -- manual, automatic, reviewed
    confidence DECIMAL(3,2) DEFAULT 1.0 CHECK (confidence >= 0.0 AND confidence <= 1.0),
    
    -- Additional mapping information
    comment TEXT,
    mapped_by VARCHAR(100),
    evidence JSONB DEFAULT '{}',
    /* Example structure:
    {
      "algorithm": "lexical_similarity",
      "score": 0.95,
      "sources": ["manual_review", "automated_mapping"],
      "references": ["DOI:10.1000/xyz123"]
    }
    */
    
    -- Quality assurance
    verified BOOLEAN DEFAULT FALSE,
    verified_by VARCHAR(100),
    verified_at TIMESTAMPTZ,
    
    -- Usage statistics
    usage_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMPTZ,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(source_system_id, source_code, target_system_id, target_code)
);

-- Value Set Concepts - explicit membership for computed expansions
CREATE TABLE IF NOT EXISTS value_set_concepts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    value_set_id UUID REFERENCES value_sets(id) ON DELETE CASCADE,
    system_id UUID REFERENCES terminology_systems(id) ON DELETE CASCADE,
    concept_code VARCHAR(255) NOT NULL,
    display VARCHAR(500),
    
    -- Expansion metadata
    abstract BOOLEAN DEFAULT FALSE,
    inactive BOOLEAN DEFAULT FALSE,
    version VARCHAR(100),
    
    -- Additional properties from expansion
    properties JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(value_set_id, system_id, concept_code)
);

-- Create indexes for performance

-- Terminology systems indexes
CREATE INDEX IF NOT EXISTS idx_terminology_systems_uri ON terminology_systems(system_uri);
CREATE INDEX IF NOT EXISTS idx_terminology_systems_status ON terminology_systems(status);
CREATE INDEX IF NOT EXISTS idx_terminology_systems_created_at ON terminology_systems(created_at DESC);

-- Terminology concepts indexes
CREATE INDEX IF NOT EXISTS idx_terminology_concepts_system_id ON terminology_concepts(system_id);
CREATE INDEX IF NOT EXISTS idx_terminology_concepts_code ON terminology_concepts(code);
CREATE INDEX IF NOT EXISTS idx_terminology_concepts_system_code ON terminology_concepts(system_id, code);
CREATE INDEX IF NOT EXISTS idx_terminology_concepts_status ON terminology_concepts(status);
CREATE INDEX IF NOT EXISTS idx_terminology_concepts_domain ON terminology_concepts(clinical_domain);
CREATE INDEX IF NOT EXISTS idx_terminology_concepts_display ON terminology_concepts USING gin(display gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_terminology_concepts_search ON terminology_concepts USING gin(search_terms);
CREATE INDEX IF NOT EXISTS idx_terminology_concepts_parents ON terminology_concepts USING gin(parent_codes);
CREATE INDEX IF NOT EXISTS idx_terminology_concepts_children ON terminology_concepts USING gin(child_codes);

-- Value sets indexes
CREATE INDEX IF NOT EXISTS idx_value_sets_url ON value_sets(url);
CREATE INDEX IF NOT EXISTS idx_value_sets_status ON value_sets(status);
CREATE INDEX IF NOT EXISTS idx_value_sets_domain ON value_sets(clinical_domain);
CREATE INDEX IF NOT EXISTS idx_value_sets_created_at ON value_sets(created_at DESC);

-- Concept mappings indexes
CREATE INDEX IF NOT EXISTS idx_concept_mappings_source ON concept_mappings(source_system_id, source_code);
CREATE INDEX IF NOT EXISTS idx_concept_mappings_target ON concept_mappings(target_system_id, target_code);
CREATE INDEX IF NOT EXISTS idx_concept_mappings_equivalence ON concept_mappings(equivalence);
CREATE INDEX IF NOT EXISTS idx_concept_mappings_verified ON concept_mappings(verified);
CREATE INDEX IF NOT EXISTS idx_concept_mappings_usage ON concept_mappings(usage_count DESC);

-- Value set concepts indexes
CREATE INDEX IF NOT EXISTS idx_value_set_concepts_value_set ON value_set_concepts(value_set_id);
CREATE INDEX IF NOT EXISTS idx_value_set_concepts_system ON value_set_concepts(system_id);
CREATE INDEX IF NOT EXISTS idx_value_set_concepts_code ON value_set_concepts(concept_code);

-- GIN indexes for JSONB columns
CREATE INDEX IF NOT EXISTS idx_terminology_systems_metadata_gin ON terminology_systems USING gin(metadata);
CREATE INDEX IF NOT EXISTS idx_terminology_concepts_properties_gin ON terminology_concepts USING gin(properties);
CREATE INDEX IF NOT EXISTS idx_terminology_concepts_designations_gin ON terminology_concepts USING gin(designations);
CREATE INDEX IF NOT EXISTS idx_value_sets_compose_gin ON value_sets USING gin(compose);
CREATE INDEX IF NOT EXISTS idx_value_sets_expansion_gin ON value_sets USING gin(expansion);
CREATE INDEX IF NOT EXISTS idx_concept_mappings_evidence_gin ON concept_mappings USING gin(evidence);

-- Functions for text search optimization
CREATE OR REPLACE FUNCTION update_concept_search_terms()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_terms := 
        setweight(to_tsvector('english', COALESCE(NEW.display, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.definition, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.code, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update search terms
CREATE TRIGGER trigger_update_concept_search_terms
    BEFORE INSERT OR UPDATE ON terminology_concepts
    FOR EACH ROW
    EXECUTE FUNCTION update_concept_search_terms();

-- Function to update terminology system concept count
CREATE OR REPLACE FUNCTION update_system_concept_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE terminology_systems 
        SET concept_count = concept_count + 1,
            updated_at = NOW()
        WHERE id = NEW.system_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE terminology_systems 
        SET concept_count = concept_count - 1,
            updated_at = NOW()
        WHERE id = OLD.system_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Triggers to maintain concept counts
CREATE TRIGGER trigger_update_system_concept_count_insert
    AFTER INSERT ON terminology_concepts
    FOR EACH ROW
    EXECUTE FUNCTION update_system_concept_count();

CREATE TRIGGER trigger_update_system_concept_count_delete
    AFTER DELETE ON terminology_concepts
    FOR EACH ROW
    EXECUTE FUNCTION update_system_concept_count();

-- Function to update timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers to automatically update updated_at timestamps
CREATE TRIGGER trigger_terminology_systems_updated_at
    BEFORE UPDATE ON terminology_systems
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_terminology_concepts_updated_at
    BEFORE UPDATE ON terminology_concepts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_value_sets_updated_at
    BEFORE UPDATE ON value_sets
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_concept_mappings_updated_at
    BEFORE UPDATE ON concept_mappings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Insert initial terminology systems
INSERT INTO terminology_systems (system_uri, system_name, version, description, publisher, status, supported_regions) VALUES
('http://snomed.info/sct', 'SNOMED Clinical Terms', '20220131', 'Systematized Nomenclature of Medicine Clinical Terms', 'SNOMED International', 'active', ARRAY['US', 'EU', 'CA', 'AU']),
('http://hl7.org/fhir/sid/icd-10-cm', 'ICD-10-CM', '2022', 'International Classification of Diseases, 10th Revision, Clinical Modification', 'World Health Organization', 'active', ARRAY['US']),
('http://www.nlm.nih.gov/research/umls/rxnorm', 'RxNorm', '2022-01', 'RxNorm provides normalized names for clinical drugs', 'National Library of Medicine', 'active', ARRAY['US']),
('http://loinc.org', 'LOINC', '2.72', 'Logical Observation Identifiers Names and Codes', 'Regenstrief Institute', 'active', ARRAY['US', 'EU', 'CA', 'AU']),
('http://hl7.org/fhir/sid/icd-10', 'ICD-10', '2019', 'International Statistical Classification of Diseases and Related Health Problems 10th Revision', 'World Health Organization', 'active', ARRAY['US', 'EU', 'CA', 'AU'])
ON CONFLICT (system_uri, version) DO NOTHING;

-- Comments for documentation
COMMENT ON TABLE terminology_systems IS 'Stores metadata about different terminology systems (SNOMED CT, ICD-10, RxNorm, etc.)';
COMMENT ON TABLE terminology_concepts IS 'Stores individual concepts from terminology systems with hierarchical relationships';
COMMENT ON TABLE value_sets IS 'Collections of concepts for specific clinical use cases with compose and expansion rules';
COMMENT ON TABLE concept_mappings IS 'Mappings and relationships between concepts in different terminology systems';
COMMENT ON TABLE value_set_concepts IS 'Explicit membership table for value set expansions';

COMMENT ON COLUMN terminology_concepts.search_terms IS 'Automatically maintained tsvector for full-text search performance';
COMMENT ON COLUMN terminology_concepts.parent_codes IS 'Array of parent concept codes for hierarchy traversal';
COMMENT ON COLUMN terminology_concepts.child_codes IS 'Array of child concept codes for hierarchy traversal';
COMMENT ON COLUMN concept_mappings.equivalence IS 'FHIR ConceptMap equivalence values for mapping relationships';
COMMENT ON COLUMN concept_mappings.confidence IS 'Confidence score (0.0-1.0) for the mapping accuracy';