-- KB-7 Terminology Service: Medical English Text Search Configuration
-- Phase 2: Enhanced Search Capabilities
-- This migration implements specialized text search configuration for medical terminology

-- Create medical English text search configuration (skip if already exists from migration 002)
-- Use exception handler rather than querying pg_catalog directly (avoids schema search path issues)
DO $$
BEGIN
    CREATE TEXT SEARCH CONFIGURATION medical_english (COPY = english);
    RAISE NOTICE 'Created medical_english text search configuration';
EXCEPTION
    WHEN duplicate_object THEN
        RAISE NOTICE 'medical_english text search config already exists';
    WHEN OTHERS THEN
        RAISE NOTICE 'Could not create medical_english text search config: %', SQLERRM;
END $$;

-- Create custom medical dictionaries for better stemming and synonyms
DO $$
BEGIN
    -- Create medical stem dictionary if not exists
    IF NOT EXISTS (SELECT 1 FROM pg_ts_dict WHERE dictname = 'medical_stem') THEN
        CREATE TEXT SEARCH DICTIONARY medical_stem (
            TEMPLATE = snowball,
            Language = english
        );
    END IF;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Could not create medical_stem dictionary: %', SQLERRM;
END $$;

-- Note: Synonym dictionary requires external file - skipping for now
-- CREATE TEXT SEARCH DICTIONARY medical_synonym (TEMPLATE = synonym, SYNONYMS = medical_synonyms);

-- Create temp table for medical synonyms data
CREATE TEMP TABLE temp_medical_synonyms (synonym_line TEXT);

INSERT INTO temp_medical_synonyms (synonym_line) VALUES
    -- Common medical abbreviations and synonyms
    ('hypertension,htn,high blood pressure'),
    ('diabetes mellitus,dm,diabetes'),
    ('myocardial infarction,mi,heart attack'),
    ('cerebrovascular accident,cva,stroke'),
    ('pneumonia,pna,lung infection'),
    ('urinary tract infection,uti,bladder infection'),
    ('gastroesophageal reflux,ger,gerd,acid reflux'),
    ('chronic obstructive pulmonary disease,copd'),
    ('congestive heart failure,chf,heart failure'),
    ('acute coronary syndrome,acs'),
    -- Medical prefixes and roots
    ('cardio,cardiac,heart'),
    ('pulmon,pulmonary,lung'),
    ('hepat,hepatic,liver'),
    ('nephr,nephric,renal,kidney'),
    ('gastr,gastric,stomach'),
    ('neuro,neural,nerve'),
    ('osteo,bone'),
    ('dermat,dermal,skin'),
    -- Common drug name variations
    ('acetaminophen,paracetamol,tylenol'),
    ('ibuprofen,advil,motrin'),
    ('aspirin,asa,acetylsalicylic acid'),
    ('metformin,glucophage'),
    ('lisinopril,prinivil,zestril'),
    ('atorvastatin,lipitor'),
    ('omeprazole,prilosec'),
    ('simvastatin,zocor'),
    ('amlodipine,norvasc'),
    ('metoprolol,lopressor,toprol');

-- Write synonyms to file (simulated - in real deployment this would be a separate file)
-- For now we'll create a simple synonym mapping table
CREATE TABLE IF NOT EXISTS medical_synonyms (
    id SERIAL PRIMARY KEY,
    term VARCHAR(255) NOT NULL,
    synonyms TEXT[] NOT NULL,
    category VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert medical synonyms from temp table
INSERT INTO medical_synonyms (term, synonyms, category)
SELECT 
    split_part(synonym_line, ',', 1) as term,
    string_to_array(synonym_line, ',') as synonyms,
    CASE 
        WHEN synonym_line LIKE '%hypertension%' OR synonym_line LIKE '%diabetes%' 
             OR synonym_line LIKE '%infarction%' OR synonym_line LIKE '%pneumonia%'
             OR synonym_line LIKE '%infection%' OR synonym_line LIKE '%reflux%'
             OR synonym_line LIKE '%disease%' OR synonym_line LIKE '%failure%'
             OR synonym_line LIKE '%syndrome%' THEN 'condition'
        WHEN synonym_line LIKE '%cardio%' OR synonym_line LIKE '%pulmon%'
             OR synonym_line LIKE '%hepat%' OR synonym_line LIKE '%nephr%'
             OR synonym_line LIKE '%gastr%' OR synonym_line LIKE '%neuro%'
             OR synonym_line LIKE '%osteo%' OR synonym_line LIKE '%dermat%' THEN 'anatomy'
        ELSE 'medication'
    END as category
FROM temp_medical_synonyms;

-- Create phonetic matching support for medical terms (optional - graceful degradation)
DO $$
BEGIN
    CREATE EXTENSION IF NOT EXISTS fuzzystrmatch;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'fuzzystrmatch extension not available - phonetic matching disabled';
END $$;

-- Create medical term phonetic index for soundex matching
-- Note: Foreign key references partitioned concepts table via concept_uuid
CREATE TABLE IF NOT EXISTS concept_phonetics (
    concept_id UUID,
    term_soundex VARCHAR(10),
    term_metaphone VARCHAR(20),
    term_dmetaphone VARCHAR(20),
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (concept_id, term_soundex)
);

-- Populate phonetic data for existing concepts (only if fuzzystrmatch available and concepts exist)
DO $$
BEGIN
    INSERT INTO concept_phonetics (concept_id, term_soundex, term_metaphone, term_dmetaphone)
    SELECT
        concept_uuid,
        soundex(preferred_term),
        metaphone(preferred_term, 8),
        dmetaphone(preferred_term)
    FROM concepts
    WHERE preferred_term IS NOT NULL
    ON CONFLICT (concept_id, term_soundex) DO NOTHING;
EXCEPTION WHEN undefined_function THEN
    RAISE NOTICE 'Phonetic functions not available - skipping phonetics population';
WHEN undefined_table THEN
    RAISE NOTICE 'Concepts table empty - skipping phonetics population';
END $$;

-- Create full-text search indexes with medical configuration (skip if already exists from migration 002)
DO $$
BEGIN
    ALTER TABLE concepts ADD COLUMN IF NOT EXISTS search_vector tsvector;
EXCEPTION WHEN duplicate_column THEN
    RAISE NOTICE 'search_vector column already exists';
END $$;

-- Update search vectors with medical English configuration (only if data exists)
DO $$
BEGIN
    UPDATE concepts SET search_vector =
        setweight(to_tsvector('medical_english', COALESCE(preferred_term, '')), 'A') ||
        setweight(to_tsvector('medical_english', COALESCE(code, '')), 'B') ||
        setweight(to_tsvector('medical_english', COALESCE(properties->>'synonyms', '')), 'C')
    WHERE EXISTS (SELECT 1 FROM concepts LIMIT 1);
EXCEPTION WHEN undefined_table THEN
    RAISE NOTICE 'Concepts table does not exist yet - skipping search vector update';
END $$;

-- Create GIN index for fast full-text search
CREATE INDEX IF NOT EXISTS idx_concepts_search_vector 
ON concepts USING GIN(search_vector);

-- Create trigger to maintain search vectors automatically
-- Uses 'english' config as fallback if 'medical_english' is not available
CREATE OR REPLACE FUNCTION update_concept_search_vector()
RETURNS TRIGGER AS $$
DECLARE
    ts_config TEXT := 'english';  -- Default fallback
BEGIN
    -- Check if medical_english config exists, use it if available
    BEGIN
        PERFORM to_tsvector('medical_english', 'test');
        ts_config := 'medical_english';
    EXCEPTION WHEN undefined_object THEN
        ts_config := 'english';
    END;

    -- Update search vector using the appropriate config
    NEW.search_vector :=
        setweight(to_tsvector(ts_config::regconfig, COALESCE(NEW.preferred_term, '')), 'A') ||
        setweight(to_tsvector(ts_config::regconfig, COALESCE(NEW.code, '')), 'B') ||
        setweight(to_tsvector(ts_config::regconfig, COALESCE(NEW.properties->>'synonyms', '')), 'C');

    -- Update phonetic data if term changed (only if fuzzystrmatch functions available)
    IF OLD IS NULL OR OLD.preferred_term IS DISTINCT FROM NEW.preferred_term THEN
        BEGIN
            DELETE FROM concept_phonetics WHERE concept_id = NEW.concept_uuid;
            INSERT INTO concept_phonetics (concept_id, term_soundex, term_metaphone, term_dmetaphone)
            VALUES (
                NEW.concept_uuid,
                soundex(NEW.preferred_term),
                metaphone(NEW.preferred_term, 8),
                dmetaphone(NEW.preferred_term)
            ) ON CONFLICT (concept_id, term_soundex) DO NOTHING;
        EXCEPTION WHEN undefined_function THEN
            -- Phonetic functions not available, skip phonetics update
            NULL;
        END;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Attach trigger to concepts table
DROP TRIGGER IF EXISTS trigger_update_concept_search_vector ON concepts;
CREATE TRIGGER trigger_update_concept_search_vector
    BEFORE INSERT OR UPDATE ON concepts
    FOR EACH ROW EXECUTE FUNCTION update_concept_search_vector();

-- Create enhanced search functions for medical terminology
-- Uses 'english' as fallback if 'medical_english' is not available
CREATE OR REPLACE FUNCTION search_medical_concepts(
    search_term TEXT,
    target_system TEXT DEFAULT NULL,
    use_phonetic BOOLEAN DEFAULT FALSE,
    max_results INTEGER DEFAULT 50
) RETURNS TABLE (
    concept_uuid UUID,
    system VARCHAR(20),
    code VARCHAR(255),
    preferred_term VARCHAR(500),
    rank REAL,
    match_type TEXT
) AS $$
DECLARE
    ts_config regconfig := 'english';  -- Default fallback
BEGIN
    -- Check if medical_english config exists
    BEGIN
        PERFORM to_tsvector('medical_english', 'test');
        ts_config := 'medical_english';
    EXCEPTION WHEN undefined_object THEN
        ts_config := 'english';
    END;

    -- Direct text search with medical configuration
    RETURN QUERY
    SELECT
        c.concept_uuid,
        c.system,
        c.code,
        c.preferred_term,
        ts_rank(c.search_vector, plainto_tsquery(ts_config, search_term)) as rank,
        'text_search'::TEXT as match_type
    FROM concepts c
    WHERE c.search_vector @@ plainto_tsquery(ts_config, search_term)
      AND (target_system IS NULL OR c.system = target_system)
      AND c.active = true
    ORDER BY rank DESC
    LIMIT max_results;

    -- If phonetic search is requested, add phonetic matches (with graceful degradation)
    IF use_phonetic THEN
        BEGIN
            RETURN QUERY
            SELECT DISTINCT
                c.concept_uuid,
                c.system,
                c.code,
                c.preferred_term,
                0.5::REAL as rank,  -- Lower rank for phonetic matches
                'phonetic'::TEXT as match_type
            FROM concepts c
            JOIN concept_phonetics cp ON c.concept_uuid = cp.concept_id
            WHERE (
                cp.term_soundex = soundex(search_term) OR
                cp.term_metaphone = metaphone(search_term, 8) OR
                cp.term_dmetaphone = dmetaphone(search_term)
            )
            AND (target_system IS NULL OR c.system = target_system)
            AND c.active = true
            AND NOT EXISTS (
                SELECT 1 FROM concepts c2
                WHERE c2.concept_uuid = c.concept_uuid
                AND c2.search_vector @@ plainto_tsquery(ts_config, search_term)
            )
            LIMIT (max_results / 2);
        EXCEPTION WHEN undefined_function THEN
            -- Phonetic functions not available, skip phonetic search
            NULL;
        END;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Create synonym expansion function
CREATE OR REPLACE FUNCTION expand_medical_synonyms(search_term TEXT)
RETURNS TEXT[] AS $$
DECLARE
    expanded_terms TEXT[];
BEGIN
    -- Find synonyms for the search term
    SELECT ARRAY(
        SELECT DISTINCT unnest(synonyms)
        FROM medical_synonyms 
        WHERE term ILIKE '%' || search_term || '%' 
           OR search_term = ANY(synonyms)
    ) INTO expanded_terms;
    
    -- If no synonyms found, return original term
    IF array_length(expanded_terms, 1) IS NULL THEN
        expanded_terms := ARRAY[search_term];
    END IF;
    
    RETURN expanded_terms;
END;
$$ LANGUAGE plpgsql;

-- Create comprehensive medical search function with synonym expansion
CREATE OR REPLACE FUNCTION comprehensive_medical_search(
    search_term TEXT,
    target_system TEXT DEFAULT NULL,
    expand_synonyms BOOLEAN DEFAULT TRUE,
    use_phonetic BOOLEAN DEFAULT FALSE,
    max_results INTEGER DEFAULT 50
) RETURNS TABLE (
    concept_uuid UUID,
    system VARCHAR(20),
    code VARCHAR(255),
    preferred_term VARCHAR(500),
    rank REAL,
    match_type TEXT
) AS $$
DECLARE
    expanded_terms TEXT[];
    term TEXT;
BEGIN
    -- Expand synonyms if requested
    IF expand_synonyms THEN
        expanded_terms := expand_medical_synonyms(search_term);
    ELSE
        expanded_terms := ARRAY[search_term];
    END IF;
    
    -- Search with each expanded term
    FOREACH term IN ARRAY expanded_terms
    LOOP
        RETURN QUERY
        SELECT * FROM search_medical_concepts(term, target_system, use_phonetic, max_results);
    END LOOP;
    
    RETURN;
END;
$$ LANGUAGE plpgsql;

-- Create search performance statistics table
CREATE TABLE IF NOT EXISTS search_statistics (
    id SERIAL PRIMARY KEY,
    search_term TEXT,
    target_system VARCHAR(20),
    result_count INTEGER,
    search_duration_ms NUMERIC,
    used_synonyms BOOLEAN,
    used_phonetic BOOLEAN,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create index on search statistics for analytics
CREATE INDEX IF NOT EXISTS idx_search_statistics_created_at 
ON search_statistics(created_at);

CREATE INDEX IF NOT EXISTS idx_search_statistics_term 
ON search_statistics(search_term);

-- Note: Cannot directly INSERT into pg_ts_config_map (system catalog)
-- To add custom dictionary to medical_english config, use ALTER TEXT SEARCH CONFIGURATION:
-- ALTER TEXT SEARCH CONFIGURATION medical_english
--     ALTER MAPPING FOR asciiword, asciihword, hword_asciipart, word, hword, hword_part
--     WITH medical_stem;
-- This is skipped for now since medical_stem dictionary is optional

-- Update statistics (wrapped in exception handler since tables may be empty)
DO $$
BEGIN
    ANALYZE concepts;
EXCEPTION WHEN undefined_table THEN
    RAISE NOTICE 'concepts table not ready for ANALYZE';
END $$;

DO $$
BEGIN
    ANALYZE concept_phonetics;
EXCEPTION WHEN undefined_table THEN
    RAISE NOTICE 'concept_phonetics table not ready for ANALYZE';
END $$;

DO $$
BEGIN
    ANALYZE medical_synonyms;
EXCEPTION WHEN undefined_table THEN
    RAISE NOTICE 'medical_synonyms table not ready for ANALYZE';
END $$;

-- Create materialized view for common medical term searches
CREATE MATERIALIZED VIEW IF NOT EXISTS common_medical_searches AS
SELECT 
    search_term,
    target_system,
    COUNT(*) as search_count,
    AVG(search_duration_ms) as avg_duration_ms,
    MAX(created_at) as last_searched
FROM search_statistics
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY search_term, target_system
HAVING COUNT(*) >= 5
ORDER BY search_count DESC;

-- Create index on materialized view
CREATE UNIQUE INDEX IF NOT EXISTS idx_common_medical_searches_unique
ON common_medical_searches(search_term, COALESCE(target_system, ''));

-- Schedule refresh of materialized view (requires pg_cron extension)
-- SELECT cron.schedule('refresh-medical-searches', '0 1 * * *', 'REFRESH MATERIALIZED VIEW CONCURRENTLY common_medical_searches;');

-- Migration completion log (create table if not exists)
DO $$
BEGIN
    -- Create migration_log table if it doesn't exist
    CREATE TABLE IF NOT EXISTS migration_log (
        id SERIAL PRIMARY KEY,
        migration_name VARCHAR(255) NOT NULL UNIQUE,
        status VARCHAR(50) NOT NULL DEFAULT 'completed',
        completed_at TIMESTAMP DEFAULT NOW()
    );

    -- Log this migration
    INSERT INTO migration_log (migration_name, status, completed_at)
    VALUES ('003_medical_text_search', 'completed', NOW())
    ON CONFLICT (migration_name) DO UPDATE SET
        status = 'completed',
        completed_at = NOW();
END $$;

-- Performance validation queries
-- These can be run to validate the search performance improvements

/*
-- Test medical text search performance
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM comprehensive_medical_search('hypertension', 'SNOMED', true, false, 25);

-- Test phonetic matching
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM search_medical_concepts('diabetis', 'SNOMED', true, 25);

-- Test synonym expansion
SELECT expand_medical_synonyms('heart attack');
SELECT expand_medical_synonyms('diabetes');
SELECT expand_medical_synonyms('high blood pressure');

-- Performance metrics
SELECT
    COUNT(*) as total_concepts,
    COUNT(*) FILTER (WHERE search_vector IS NOT NULL) as indexed_concepts,
    AVG(length(preferred_term)) as avg_term_length
FROM concepts;
*/

-- Note: No explicit COMMIT needed - migration runner handles transaction management