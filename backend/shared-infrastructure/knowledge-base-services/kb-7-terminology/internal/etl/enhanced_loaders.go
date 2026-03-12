package etl

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/models"

	"github.com/lib/pq"
	"go.uber.org/zap"
)

// LoaderConfig contains configuration for data loaders
type LoaderConfig struct {
	DataDirectory    string
	BatchSize        int
	MaxWorkers       int
	EnableValidation bool
	EnableDebug      bool
	ValidateData     bool
}

// SNOMEDLoader handles loading SNOMED CT terminology data
type SNOMEDLoader struct {
	db     *sql.DB
	cache  cache.EnhancedCache
	logger *zap.Logger
	config LoaderConfig
}

// NewSNOMEDLoader creates a new SNOMED CT loader
func NewSNOMEDLoader(db *sql.DB, cache cache.EnhancedCache, logger *zap.Logger, config LoaderConfig) *SNOMEDLoader {
	return &SNOMEDLoader{
		db:     db,
		cache:  cache,
		logger: logger,
		config: config,
	}
}

// LoadSNOMEDData loads SNOMED CT data from RF2 files
func (s *SNOMEDLoader) LoadSNOMEDData(dataDirectory string) error {
	s.logger.Info("Starting SNOMED CT data loading", zap.String("directory", dataDirectory))
	
	// 1. Ensure SNOMED CT system is registered
	if err := s.ensureSNOMEDSystem(); err != nil {
		return fmt.Errorf("failed to ensure SNOMED system: %w", err)
	}
	
	// 2. Get system ID
	systemID, err := s.getSNOMEDSystemID()
	if err != nil {
		return fmt.Errorf("failed to get SNOMED system ID: %w", err)
	}
	
	// 3. Load RF2 files in order
	if err := s.loadConceptsFile(dataDirectory, systemID); err != nil {
		return fmt.Errorf("failed to load concepts: %w", err)
	}
	
	if err := s.loadDescriptionsFile(dataDirectory, systemID); err != nil {
		return fmt.Errorf("failed to load descriptions: %w", err)
	}
	
	if err := s.loadRelationshipsFile(dataDirectory, systemID); err != nil {
		return fmt.Errorf("failed to load relationships: %w", err)
	}
	
	s.logger.Info("SNOMED CT data loading completed successfully")
	return nil
}

func (s *SNOMEDLoader) getSNOMEDSystemID() (string, error) {
	var systemID string
	query := `SELECT id FROM terminology_systems WHERE system_uri = 'http://snomed.info/sct' LIMIT 1`
	err := s.db.QueryRow(query).Scan(&systemID)
	if err != nil {
		return "", fmt.Errorf("SNOMED system not found: %w", err)
	}
	return systemID, nil
}

func (s *SNOMEDLoader) ensureSNOMEDSystem() error {
	query := `
		INSERT INTO terminology_systems (
			id, system_uri, system_name, version, description, publisher, status,
			metadata, supported_regions, created_at, updated_at
		) VALUES (
			gen_random_uuid(), 'http://snomed.info/sct',
			'SNOMED Clinical Terms', $1, 'Systematically Organized Computer Processable Collection of Medical Terminology',
			'SNOMED International', 'active',
			$2, $3, NOW(), NOW()
		) ON CONFLICT (system_uri) DO UPDATE SET
			version = $1,
			metadata = $2,
			updated_at = NOW()
	`
	
	metadata := models.JSONB{
		"source":         "SNOMED CT International Edition",
		"license":        "SNOMED CT",
		"last_update":    time.Now().Format("2006-01-02"),
		"loader_version": "2.0.0",
		"rf2_format":     "snapshot",
	}
	
	supportedRegions := pq.Array([]string{"US", "UK", "EU", "CA", "AU", "GLOBAL"})
	
	_, err := s.db.Exec(query, "20240131", metadata, supportedRegions)
	return err
}

// loadConceptsFile loads the RF2 concepts file
func (s *SNOMEDLoader) loadConceptsFile(dataDirectory, systemID string) error {
	filePath := filepath.Join(dataDirectory, "Terminology", "sct2_Concept_Snapshot_*.txt")
	files, err := filepath.Glob(filePath)
	if err != nil || len(files) == 0 {
		return fmt.Errorf("concepts file not found at pattern: %s", filePath)
	}
	
	s.logger.Info("Loading SNOMED concepts", zap.String("file", files[0]))
	
	file, err := os.Open(files[0])
	if err != nil {
		return err
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.LazyQuotes = true
	
	// Skip header
	if _, err := reader.Read(); err != nil {
		return err
	}
	
	// Prepare batch insert
	txn, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()
	
	stmt, err := txn.Prepare(`
		INSERT INTO concepts (system, code, preferred_term, active, version, properties, created_at)
		VALUES ('SNOMED', $1, $2, $3, '20240131', $4, NOW())
		ON CONFLICT (system, code, version) DO UPDATE SET
			preferred_term = $2, active = $3, properties = $4, updated_at = NOW()
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	
	batchCount := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		
		if len(record) < 5 {
			continue
		}
		
		// RF2 Concept format: id, effectiveTime, active, moduleId, definitionStatusId
		conceptID := record[0]
		active := record[2] == "1"
		
		properties := models.JSONB{
			"effective_time":       record[1],
			"module_id":            record[3],
			"definition_status_id": record[4],
			"source_file":          "concepts",
		}
		
		_, err = stmt.Exec(conceptID, conceptID, active, properties)
		if err != nil {
			s.logger.Error("Failed to insert concept", zap.String("concept_id", conceptID), zap.Error(err))
			continue
		}
		
		batchCount++
		if batchCount%s.config.BatchSize == 0 {
			s.logger.Info("Processed concepts", zap.Int("count", batchCount))
		}
	}
	
	return txn.Commit()
}

// loadDescriptionsFile loads the RF2 descriptions file
func (s *SNOMEDLoader) loadDescriptionsFile(dataDirectory, systemID string) error {
	filePath := filepath.Join(dataDirectory, "Terminology", "sct2_Description_Snapshot-en_*.txt")
	files, err := filepath.Glob(filePath)
	if err != nil || len(files) == 0 {
		return fmt.Errorf("descriptions file not found at pattern: %s", filePath)
	}
	
	s.logger.Info("Loading SNOMED descriptions", zap.String("file", files[0]))
	
	file, err := os.Open(files[0])
	if err != nil {
		return err
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.LazyQuotes = true
	
	// Skip header
	if _, err := reader.Read(); err != nil {
		return err
	}
	
	txn, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()
	
	// Update concepts with preferred terms
	updateStmt, err := txn.Prepare(`
		UPDATE concepts SET preferred_term = $1, properties = properties || $2
		WHERE system = 'SNOMED' AND code = $3
	`)
	if err != nil {
		return err
	}
	defer updateStmt.Close()
	
	batchCount := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		
		if len(record) < 8 {
			continue
		}
		
		// RF2 Description format: id, effectiveTime, active, moduleId, conceptId, languageCode, typeId, term, caseSignificanceId
		conceptID := record[4]
		typeID := record[6]
		term := record[7]
		active := record[2] == "1"
		
		// Only process active FSN (Fully Specified Name) or Preferred terms
		if !active || (typeID != "900000000000003001" && typeID != "900000000000013009") {
			continue
		}
		
		properties := models.JSONB{
			"description_type": typeID,
			"language_code":   record[5],
			"case_significance": record[8],
		}
		
		_, err = updateStmt.Exec(term, properties, conceptID)
		if err != nil {
			s.logger.Error("Failed to update concept description", zap.String("concept_id", conceptID), zap.Error(err))
			continue
		}
		
		batchCount++
		if batchCount%s.config.BatchSize == 0 {
			s.logger.Info("Processed descriptions", zap.Int("count", batchCount))
		}
	}
	
	return txn.Commit()
}

// loadRelationshipsFile loads the RF2 relationships file
func (s *SNOMEDLoader) loadRelationshipsFile(dataDirectory, systemID string) error {
	filePath := filepath.Join(dataDirectory, "Terminology", "sct2_Relationship_Snapshot_*.txt")
	files, err := filepath.Glob(filePath)
	if err != nil || len(files) == 0 {
		return fmt.Errorf("relationships file not found at pattern: %s", filePath)
	}
	
	s.logger.Info("Loading SNOMED relationships", zap.String("file", files[0]))
	
	file, err := os.Open(files[0])
	if err != nil {
		return err
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.LazyQuotes = true
	
	// Skip header
	if _, err := reader.Read(); err != nil {
		return err
	}
	
	txn, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()
	
	stmt, err := txn.Prepare(`
		INSERT INTO concept_relationships (
			source_concept_id, target_concept_id, relationship_type, 
			active, properties, created_at
		) 
		SELECT c1.concept_uuid, c2.concept_uuid, $1, $2, $3, NOW()
		FROM concepts c1, concepts c2
		WHERE c1.system = 'SNOMED' AND c1.code = $4
		  AND c2.system = 'SNOMED' AND c2.code = $5
		ON CONFLICT (source_concept_id, target_concept_id, relationship_type) 
		DO UPDATE SET active = $2, properties = $3, updated_at = NOW()
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	
	batchCount := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		
		if len(record) < 10 {
			continue
		}
		
		// RF2 Relationship format: id, effectiveTime, active, moduleId, sourceId, destinationId, relationshipGroup, typeId, characteristicTypeId, modifierId
		sourceID := record[4]
		destinationID := record[5]
		relationshipType := record[7]
		active := record[2] == "1"
		
		if !active {
			continue
		}
		
		properties := models.JSONB{
			"relationship_group":    record[6],
			"characteristic_type":  record[8],
			"modifier_id":          record[9],
			"effective_time":       record[1],
		}
		
		_, err = stmt.Exec(relationshipType, active, properties, sourceID, destinationID)
		if err != nil {
			s.logger.Error("Failed to insert relationship", 
				zap.String("source", sourceID), 
				zap.String("target", destinationID),
				zap.Error(err))
			continue
		}
		
		batchCount++
		if batchCount%s.config.BatchSize == 0 {
			s.logger.Info("Processed relationships", zap.Int("count", batchCount))
		}
	}
	
	return txn.Commit()
}

// ICD10Loader handles loading ICD-10 terminology data
type ICD10Loader struct {
	db     *sql.DB
	cache  cache.EnhancedCache
	logger *zap.Logger
	config LoaderConfig
}

// NewICD10Loader creates a new ICD-10 loader
func NewICD10Loader(db *sql.DB, cache cache.EnhancedCache, logger *zap.Logger, config LoaderConfig) *ICD10Loader {
	return &ICD10Loader{
		db:     db,
		cache:  cache,
		logger: logger,
		config: config,
	}
}

// LoadICD10Data loads ICD-10 data from XML/CSV files
func (i *ICD10Loader) LoadICD10Data(dataDirectory string) error {
	i.logger.Info("Starting ICD-10 data loading", zap.String("directory", dataDirectory))
	
	// 1. Ensure ICD-10 system is registered
	if err := i.ensureICD10System(); err != nil {
		return fmt.Errorf("failed to ensure ICD-10 system: %w", err)
	}
	
	// 2. Get system ID
	systemID, err := i.getICD10SystemID()
	if err != nil {
		return fmt.Errorf("failed to get ICD-10 system ID: %w", err)
	}
	
	// 3. Load ICD-10 data from tabular format
	if err := i.loadICD10FromCSV(dataDirectory, systemID); err != nil {
		return fmt.Errorf("failed to load ICD-10 data: %w", err)
	}
	
	i.logger.Info("ICD-10 data loading completed successfully")
	return nil
}

func (i *ICD10Loader) ensureICD10System() error {
	query := `
		INSERT INTO terminology_systems (
			id, system_uri, system_name, version, description, publisher, status,
			metadata, supported_regions, created_at, updated_at
		) VALUES (
			gen_random_uuid(), 'http://hl7.org/fhir/sid/icd-10-cm',
			'ICD-10-CM', $1, 'International Classification of Diseases, 10th Revision, Clinical Modification',
			'World Health Organization', 'active',
			$2, $3, NOW(), NOW()
		) ON CONFLICT (system_uri) DO UPDATE SET
			version = $1,
			metadata = $2,
			updated_at = NOW()
	`
	
	metadata := models.JSONB{
		"source":         "ICD-10-CM Official Guidelines",
		"license":        "Public Domain",
		"last_update":    time.Now().Format("2006-01-02"),
		"loader_version": "2.0.0",
	}
	
	supportedRegions := pq.Array([]string{"US"})
	
	_, err := i.db.Exec(query, "2024", metadata, supportedRegions)
	return err
}

// getICD10SystemID gets the ICD-10 system ID
func (i *ICD10Loader) getICD10SystemID() (string, error) {
	var systemID string
	query := `SELECT id FROM terminology_systems WHERE system_uri = 'http://hl7.org/fhir/sid/icd-10-cm' LIMIT 1`
	err := i.db.QueryRow(query).Scan(&systemID)
	if err != nil {
		return "", fmt.Errorf("ICD-10 system not found: %w", err)
	}
	return systemID, nil
}

// loadICD10FromCSV loads ICD-10 data from CSV tabular format
func (i *ICD10Loader) loadICD10FromCSV(dataDirectory, systemID string) error {
	// Look for ICD-10 CSV files (common formats: icd10cm_codes_YYYY.txt, icd10cm_order_YYYY.txt)
	patterns := []string{
		filepath.Join(dataDirectory, "icd10cm_codes_*.txt"),
		filepath.Join(dataDirectory, "icd10cm_order_*.txt"),
		filepath.Join(dataDirectory, "*.csv"),
	}
	
	var filePath string
	for _, pattern := range patterns {
		files, err := filepath.Glob(pattern)
		if err == nil && len(files) > 0 {
			filePath = files[0]
			break
		}
	}
	
	if filePath == "" {
		return fmt.Errorf("no ICD-10 data file found in directory: %s", dataDirectory)
	}
	
	i.logger.Info("Loading ICD-10 data", zap.String("file", filePath))
	
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	
	// Prepare batch insert
	txn, err := i.db.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()
	
	stmt, err := txn.Prepare(`
		INSERT INTO concepts (system, code, preferred_term, active, version, properties, created_at)
		VALUES ($1, 'ICD-10-CM', $2, $3, $4, '2024', $5, NOW())
		ON CONFLICT (system, code, version) DO UPDATE SET
			preferred_term = $3, active = $4, properties = $5, updated_at = NOW()
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	
	batchCount := 0
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		
		// Detect format: either tab-delimited or space-delimited
		var code, description string
		if strings.Contains(line, "\t") {
			// Tab-delimited format
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) >= 2 {
				code = strings.TrimSpace(parts[0])
				description = strings.TrimSpace(parts[1])
			}
		} else {
			// Space-delimited format - code is typically first 3-7 chars
			if len(line) >= 8 {
				code = strings.TrimSpace(line[:7])
				description = strings.TrimSpace(line[7:])
			}
		}
		
		if code == "" || description == "" {
			continue
		}
		
		// Basic validation for ICD-10 code format
		if !strings.HasPrefix(code, "A") && !strings.HasPrefix(code, "B") && 
		   !strings.HasPrefix(code, "C") && !strings.HasPrefix(code, "D") &&
		   !strings.HasPrefix(code, "E") && !strings.HasPrefix(code, "F") &&
		   !strings.HasPrefix(code, "G") && !strings.HasPrefix(code, "H") &&
		   !strings.HasPrefix(code, "I") && !strings.HasPrefix(code, "J") &&
		   !strings.HasPrefix(code, "K") && !strings.HasPrefix(code, "L") &&
		   !strings.HasPrefix(code, "M") && !strings.HasPrefix(code, "N") &&
		   !strings.HasPrefix(code, "O") && !strings.HasPrefix(code, "P") &&
		   !strings.HasPrefix(code, "Q") && !strings.HasPrefix(code, "R") &&
		   !strings.HasPrefix(code, "S") && !strings.HasPrefix(code, "T") &&
		   !strings.HasPrefix(code, "U") && !strings.HasPrefix(code, "V") &&
		   !strings.HasPrefix(code, "W") && !strings.HasPrefix(code, "X") &&
		   !strings.HasPrefix(code, "Y") && !strings.HasPrefix(code, "Z") {
			continue
		}
		
		properties := models.JSONB{
			"category": string(code[0]),
			"source_file": "icd10_codes",
		}
		
		_, err = stmt.Exec(code, description, true, properties)
		if err != nil {
			i.logger.Error("Failed to insert ICD-10 concept", zap.String("code", code), zap.Error(err))
			continue
		}
		
		batchCount++
		if batchCount%i.config.BatchSize == 0 {
			i.logger.Info("Processed ICD-10 concepts", zap.Int("count", batchCount))
		}
	}
	
	if err := scanner.Err(); err != nil {
		return err
	}
	
	return txn.Commit()
}

// LOINCLoader handles loading LOINC terminology data
type LOINCLoader struct {
	db     *sql.DB
	cache  cache.EnhancedCache
	logger *zap.Logger
	config LoaderConfig
}

// NewLOINCLoader creates a new LOINC loader
func NewLOINCLoader(db *sql.DB, cache cache.EnhancedCache, logger *zap.Logger, config LoaderConfig) *LOINCLoader {
	return &LOINCLoader{
		db:     db,
		cache:  cache,
		logger: logger,
		config: config,
	}
}

// LoadLOINCData loads LOINC data from CSV files
func (l *LOINCLoader) LoadLOINCData(dataDirectory string) error {
	l.logger.Info("Starting LOINC data loading", zap.String("directory", dataDirectory))
	
	// 1. Ensure LOINC system is registered
	if err := l.ensureLOINCSystem(); err != nil {
		return fmt.Errorf("failed to ensure LOINC system: %w", err)
	}
	
	// 2. Get system ID
	systemID, err := l.getLOINCSystemID()
	if err != nil {
		return fmt.Errorf("failed to get LOINC system ID: %w", err)
	}
	
	// 3. Load LOINC data from CSV
	if err := l.loadLOINCFromCSV(dataDirectory, systemID); err != nil {
		return fmt.Errorf("failed to load LOINC data: %w", err)
	}
	
	l.logger.Info("LOINC data loading completed successfully")
	return nil
}

func (l *LOINCLoader) ensureLOINCSystem() error {
	query := `
		INSERT INTO terminology_systems (
			id, system_uri, system_name, version, description, publisher, status,
			metadata, supported_regions, created_at, updated_at
		) VALUES (
			gen_random_uuid(), 'http://loinc.org',
			'LOINC', $1, 'Logical Observation Identifiers Names and Codes',
			'Regenstrief Institute', 'active',
			$2, $3, NOW(), NOW()
		) ON CONFLICT (system_uri) DO UPDATE SET
			version = $1,
			metadata = $2,
			updated_at = NOW()
	`
	
	metadata := models.JSONB{
		"source":         "LOINC Database",
		"license":        "LOINC License",
		"last_update":    time.Now().Format("2006-01-02"),
		"loader_version": "2.0.0",
	}
	
	supportedRegions := pq.Array([]string{"US", "EU", "CA", "AU", "GLOBAL"})
	
	_, err := l.db.Exec(query, "2.76", metadata, supportedRegions)
	return err
}

// getLOINCSystemID gets the LOINC system ID
func (l *LOINCLoader) getLOINCSystemID() (string, error) {
	var systemID string
	query := `SELECT id FROM terminology_systems WHERE system_uri = 'http://loinc.org' LIMIT 1`
	err := l.db.QueryRow(query).Scan(&systemID)
	if err != nil {
		return "", fmt.Errorf("LOINC system not found: %w", err)
	}
	return systemID, nil
}

// loadLOINCFromCSV loads LOINC data from SNOMED format or CSV files
func (l *LOINCLoader) loadLOINCFromCSV(dataDirectory, systemID string) error {
	// First try SNOMED format (LOINC in SNOMED CT format)
	snomedPattern := filepath.Join(dataDirectory, "sct2_Concept_Snapshot_*.txt")
	snomedFiles, err := filepath.Glob(snomedPattern)
	if err == nil && len(snomedFiles) > 0 {
		l.logger.Info("Loading LOINC from SNOMED format", zap.String("file", snomedFiles[0]))
		return l.loadLOINCFromSNOMEDFormat(snomedFiles[0], systemID)
	}

	// Fall back to traditional LOINC CSV file (typically LoincTable/Loinc.csv)
	patterns := []string{
		filepath.Join(dataDirectory, "LoincTable", "Loinc.csv"),
		filepath.Join(dataDirectory, "Loinc.csv"),
		filepath.Join(dataDirectory, "loinc.csv"),
	}

	var filePath string
	for _, pattern := range patterns {
		if _, err := os.Stat(pattern); err == nil {
			filePath = pattern
			break
		}
	}

	if filePath == "" {
		return fmt.Errorf("no LOINC CSV or SNOMED format file found in directory: %s", dataDirectory)
	}
	
	l.logger.Info("Loading LOINC data", zap.String("file", filePath))
	
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	reader.Comma = ','
	reader.LazyQuotes = true
	
	// Read header to understand field positions
	header, err := reader.Read()
	if err != nil {
		return err
	}
	
	// Map header positions
	fieldMap := make(map[string]int)
	for i, field := range header {
		fieldMap[field] = i
	}
	
	// Check for required fields
	requiredFields := []string{"LOINC_NUM", "LONG_COMMON_NAME", "STATUS"}
	for _, field := range requiredFields {
		if _, exists := fieldMap[field]; !exists {
			return fmt.Errorf("required field '%s' not found in LOINC CSV header", field)
		}
	}
	
	// Prepare batch insert
	txn, err := l.db.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()
	
	stmt, err := txn.Prepare(`
		INSERT INTO concepts (system, code, preferred_term, active, version, properties, created_at)
		VALUES ($1, 'LOINC', $2, $3, $4, '2.76', $5, NOW())
		ON CONFLICT (system, code, version) DO UPDATE SET
			preferred_term = $3, active = $4, properties = $5, updated_at = NOW()
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	
	batchCount := 0
	
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		
		if len(record) < len(header) {
			continue
		}
		
		// Extract required fields
		loincNum := record[fieldMap["LOINC_NUM"]]
		longName := record[fieldMap["LONG_COMMON_NAME"]]
		status := record[fieldMap["STATUS"]]
		
		if loincNum == "" || longName == "" {
			continue
		}
		
		active := status == "ACTIVE"
		
		// Extract additional properties if available
		properties := models.JSONB{
			"status": status,
			"source_file": "Loinc.csv",
		}
		
		// Add optional fields if present
		if component, exists := fieldMap["COMPONENT"]; exists && component < len(record) {
			properties["component"] = record[component]
		}
		if property, exists := fieldMap["PROPERTY"]; exists && property < len(record) {
			properties["property"] = record[property]
		}
		if timeAspct, exists := fieldMap["TIME_ASPCT"]; exists && timeAspct < len(record) {
			properties["time_aspect"] = record[timeAspct]
		}
		if system, exists := fieldMap["SYSTEM"]; exists && system < len(record) {
			properties["measurement_system"] = record[system]
		}
		if scaleTyp, exists := fieldMap["SCALE_TYP"]; exists && scaleTyp < len(record) {
			properties["scale_type"] = record[scaleTyp]
		}
		if methodTyp, exists := fieldMap["METHOD_TYP"]; exists && methodTyp < len(record) {
			properties["method_type"] = record[methodTyp]
		}
		if classField, exists := fieldMap["CLASS"]; exists && classField < len(record) {
			properties["class"] = record[classField]
		}
		
		_, err = stmt.Exec(loincNum, longName, active, properties)
		if err != nil {
			l.logger.Error("Failed to insert LOINC concept", zap.String("loinc_num", loincNum), zap.Error(err))
			continue
		}
		
		batchCount++
		if batchCount%l.config.BatchSize == 0 {
			l.logger.Info("Processed LOINC concepts", zap.Int("count", batchCount))
		}
	}
	
	return txn.Commit()
}

// loadLOINCFromSNOMEDFormat loads LOINC data from SNOMED CT format files
func (l *LOINCLoader) loadLOINCFromSNOMEDFormat(filePath, systemID string) error {
	l.logger.Info("Loading LOINC from SNOMED format file", zap.String("file", filePath))

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Skip header line
	if !scanner.Scan() {
		return fmt.Errorf("empty LOINC SNOMED file")
	}

	// Prepare batch insert with smaller transaction batches
	var txn *sql.Tx
	var stmt *sql.Stmt

	batchSize := 500
	commitEvery := batchSize

	startNewBatch := func() error {
		if txn != nil {
			stmt.Close()
			if err := txn.Commit(); err != nil {
				txn.Rollback()
				return fmt.Errorf("failed to commit batch: %w", err)
			}
		}

		var err error
		txn, err = l.db.Begin()
		if err != nil {
			return err
		}

		stmt, err = txn.Prepare(`
			INSERT INTO concepts (system, code, preferred_term, active, version, properties, created_at)
			VALUES ('LOINC', $1, $2, $3, '2.76', $4, NOW())
			ON CONFLICT (system, code, version) DO UPDATE SET
				preferred_term = $2, active = $3, properties = $4, updated_at = NOW()
		`)
		return err
	}

	// Start first batch
	if err := startNewBatch(); err != nil {
		return err
	}

	defer func() {
		if stmt != nil {
			stmt.Close()
		}
		if txn != nil {
			txn.Rollback()
		}
	}()

	batchCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "\t")

		if len(fields) < 5 {
			continue
		}

		// SNOMED format: id, effectiveTime, active, moduleId, definitionStatusId
		// For LOINC in SNOMED format: conceptId, effectiveTime, active, moduleId, definitionStatusId
		conceptId := fields[0]
		activeStr := fields[2]

		active := activeStr == "1"

		// Get preferred term from descriptions (for now use concept ID as term)
		preferredTerm := conceptId

		properties := models.JSONB{
			"source":            "LOINC_SNOMED",
			"source_file":       "sct2_Concept_Snapshot",
			"module_id":         fields[3],
			"definition_status": fields[4],
		}

		// Validate and truncate data if necessary
		if len(preferredTerm) > 250 {
			preferredTerm = preferredTerm[:250]
		}

		_, err = stmt.Exec(conceptId, preferredTerm, active, properties)
		if err != nil {
			l.logger.Warn("Failed to insert LOINC concept, skipping", zap.String("concept_id", conceptId), zap.Error(err))
			// If transaction is aborted, start a new one
			if strings.Contains(err.Error(), "aborted") {
				if err := startNewBatch(); err != nil {
					return fmt.Errorf("failed to restart batch after error: %w", err)
				}
			}
			continue
		}

		batchCount++
		if batchCount%1000 == 0 {
			l.logger.Info("Processed LOINC concepts", zap.Int("count", batchCount))
		}

		// Commit every commitEvery records to avoid large transactions
		if batchCount%commitEvery == 0 {
			if err := startNewBatch(); err != nil {
				return fmt.Errorf("failed to commit and start new batch: %w", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Commit final batch
	if txn != nil && stmt != nil {
		stmt.Close()
		stmt = nil
		if err := txn.Commit(); err != nil {
			txn.Rollback()
			return fmt.Errorf("failed to commit final batch: %w", err)
		}
		txn = nil
	}

	return nil
}

// EnhancedRxNormLoader extends the basic RxNorm loader with additional features
type EnhancedRxNormLoader struct {
	db     *sql.DB
	cache  cache.EnhancedCache
	logger *zap.Logger
	config LoaderConfig
}

// NewEnhancedRxNormLoader creates a new enhanced RxNorm loader
func NewEnhancedRxNormLoader(db *sql.DB, cache cache.EnhancedCache, logger *zap.Logger, config LoaderConfig) *EnhancedRxNormLoader {
	return &EnhancedRxNormLoader{
		db:     db,
		cache:  cache,
		logger: logger,
		config: config,
	}
}

// LoadRxNormData loads RxNorm data with enhanced features
func (r *EnhancedRxNormLoader) LoadRxNormData(dataDirectory string) error {
	r.logger.Info("Starting enhanced RxNorm data loading", zap.String("directory", dataDirectory))
	
	// 1. Ensure RxNorm system is registered
	if err := r.ensureRxNormSystem(); err != nil {
		return fmt.Errorf("failed to ensure RxNorm system: %w", err)
	}
	
	// 2. Get system ID
	systemID, err := r.getRxNormSystemID()
	if err != nil {
		return fmt.Errorf("failed to get RxNorm system ID: %w", err)
	}
	
	// 3. Load RRF files in order
	if err := r.loadConceptsFromRRF(dataDirectory, systemID); err != nil {
		return fmt.Errorf("failed to load RxNorm concepts: %w", err)
	}
	
	if err := r.loadRelationshipsFromRRF(dataDirectory, systemID); err != nil {
		return fmt.Errorf("failed to load RxNorm relationships: %w", err)
	}
	
	// 4. Add enhanced features
	if err := r.loadDrugSpecificData(dataDirectory); err != nil {
		return fmt.Errorf("failed to load drug-specific data: %w", err)
	}
	
	r.logger.Info("Enhanced RxNorm data loading completed successfully")
	return nil
}

// loadDrugSpecificData loads additional drug-specific information
func (r *EnhancedRxNormLoader) loadDrugSpecificData(dataDirectory string) error {
	r.logger.Info("Loading drug-specific data")
	
	// Load drug concepts into specialized table
	query := `
		INSERT INTO drug_concepts (
			rxnorm_cui, ingredient, brand_names, is_generic, atc_codes
		)
		SELECT DISTINCT
			c.code,
			c.preferred_term,
			ARRAY[c.preferred_term],
			CASE 
				WHEN c.properties->>'term_type' IN ('IN', 'PIN') THEN true
				ELSE false
			END,
			ARRAY[]::TEXT[]
		FROM concepts c
		WHERE c.system = 'RxNorm'
		AND c.properties->>'term_type' IN ('IN', 'PIN', 'BN', 'SCD', 'SBD')
		ON CONFLICT (rxnorm_cui) DO NOTHING
	`
	
	_, err := r.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to populate drug_concepts: %w", err)
	}
	
	return nil
}

// ensureRxNormSystem ensures RxNorm system is registered
func (r *EnhancedRxNormLoader) ensureRxNormSystem() error {
	query := `
		INSERT INTO terminology_systems (
			id, system_uri, system_name, version, description, publisher, status,
			metadata, supported_regions, created_at, updated_at
		) VALUES (
			gen_random_uuid(), 'http://www.nlm.nih.gov/research/umls/rxnorm',
			'RxNorm', $1, 'Normalized names for clinical drugs',
			'National Library of Medicine', 'active',
			$2, $3, NOW(), NOW()
		) ON CONFLICT (system_uri) DO UPDATE SET
			version = $1,
			metadata = $2,
			updated_at = NOW()
	`
	
	metadata := models.JSONB{
		"source":         "RxNorm Monthly Release",
		"license":        "UMLS License",
		"last_update":    time.Now().Format("2006-01-02"),
		"loader_version": "2.0.0",
		"rrf_format":     "pipe_delimited",
	}
	
	supportedRegions := pq.Array([]string{"US"})
	
	_, err := r.db.Exec(query, "20240101", metadata, supportedRegions)
	return err
}

// getRxNormSystemID gets the RxNorm system ID
func (r *EnhancedRxNormLoader) getRxNormSystemID() (string, error) {
	var systemID string
	query := `SELECT id FROM terminology_systems WHERE system_uri = 'http://www.nlm.nih.gov/research/umls/rxnorm' LIMIT 1`
	err := r.db.QueryRow(query).Scan(&systemID)
	if err != nil {
		return "", fmt.Errorf("RxNorm system not found: %w", err)
	}
	return systemID, nil
}

// loadConceptsFromRRF loads RxNorm concepts from RXNCONSO.RRF file
func (r *EnhancedRxNormLoader) loadConceptsFromRRF(dataDirectory, systemID string) error {
	filePath := filepath.Join(dataDirectory, "rrf", "RXNCONSO.RRF")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("RXNCONSO.RRF file not found at: %s", filePath)
	}
	
	r.logger.Info("Loading RxNorm concepts from RRF", zap.String("file", filePath))
	
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	
	// Prepare batch insert with smaller transaction batches
	var txn *sql.Tx
	var stmt *sql.Stmt

	batchSize := 500 // Smaller batches to avoid transaction issues
	commitEvery := batchSize

	startNewBatch := func() error {
		if txn != nil {
			stmt.Close()
			if err := txn.Commit(); err != nil {
				txn.Rollback()
				return fmt.Errorf("failed to commit batch: %w", err)
			}
		}

		var err error
		txn, err = r.db.Begin()
		if err != nil {
			return err
		}

		stmt, err = txn.Prepare(`
			INSERT INTO concepts (system, code, preferred_term, active, version, properties, created_at)
			VALUES ('RxNorm', $1, $2, $3, '20240101', $4, NOW())
			ON CONFLICT (system, code, version) DO UPDATE SET
				preferred_term = $2, active = $3, properties = $4, updated_at = NOW()
		`)
		return err
	}

	// Start first batch
	if err := startNewBatch(); err != nil {
		return err
	}

	defer func() {
		if stmt != nil {
			stmt.Close()
		}
		if txn != nil {
			txn.Rollback()
		}
	}()
	
	batchCount := 0
	processedCUIs := make(map[string]bool)
	
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "|")
		
		if len(fields) < 18 {
			continue
		}
		
		// RXNCONSO format: RXCUI|LAT|TS|LUI|STT|SUI|ISPREF|RXAUI|SAUI|SCUI|SDUI|SAB|TTY|CODE|STR|SRL|SUPPRESS|CVF
		rxcui := fields[0]
		lat := fields[1]   // Language
		ts := fields[2]    // Term status
		sab := fields[11]  // Source abbreviation
		tty := fields[12]  // Term type
		str := fields[14]  // String/term
		suppress := fields[16] // Suppress flag
		
		// Only process English terms from RxNorm source that are not suppressed
		if lat != "ENG" || sab != "RXNORM" || suppress == "Y" {
			continue
		}
		
		// Skip if we've already processed this RXCUI
		if processedCUIs[rxcui] {
			continue
		}
		processedCUIs[rxcui] = true
		
		// Only process preferred terms or brand names for concepts
		if tty != "PSN" && tty != "PIN" && tty != "BN" && tty != "SCD" && tty != "SBD" && tty != "IN" {
			continue
		}
		
		active := ts == "P" // P = Preferred, N = Non-preferred
		
		properties := models.JSONB{
			"term_type":       tty,
			"term_status":     ts,
			"source":          sab,
			"suppress_flag":   suppress,
			"source_file":     "RXNCONSO",
		}
		
		// Validate and truncate data if necessary
		if len(str) > 250 {
			str = str[:250] // Truncate very long terms to avoid 255-byte limit
		}

		_, err = stmt.Exec(rxcui, str, active, properties)
		if err != nil {
			r.logger.Warn("Failed to insert RxNorm concept, skipping", zap.String("rxcui", rxcui), zap.String("term", str), zap.Error(err))
			// If transaction is aborted, start a new one
			if strings.Contains(err.Error(), "aborted") {
				if err := startNewBatch(); err != nil {
					return fmt.Errorf("failed to restart batch after error: %w", err)
				}
			}
			continue
		}

		batchCount++
		if batchCount%r.config.BatchSize == 0 {
			r.logger.Info("Processed RxNorm concepts", zap.Int("count", batchCount))
		}

		// Commit every commitEvery records to avoid large transactions
		if batchCount%commitEvery == 0 {
			if err := startNewBatch(); err != nil {
				return fmt.Errorf("failed to commit and start new batch: %w", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Commit final batch
	if txn != nil && stmt != nil {
		stmt.Close()
		stmt = nil
		if err := txn.Commit(); err != nil {
			txn.Rollback()
			return fmt.Errorf("failed to commit final batch: %w", err)
		}
		txn = nil
	}

	return nil
}

// loadRelationshipsFromRRF loads RxNorm relationships from RXNREL.RRF file
func (r *EnhancedRxNormLoader) loadRelationshipsFromRRF(dataDirectory, systemID string) error {
	filePath := filepath.Join(dataDirectory, "rrf", "RXNREL.RRF")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("RXNREL.RRF file not found at: %s", filePath)
	}
	
	r.logger.Info("Loading RxNorm relationships from RRF", zap.String("file", filePath))
	
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	
	// Prepare batch insert
	txn, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()
	
	stmt, err := txn.Prepare(`
		INSERT INTO concept_relationships (
			source_concept_id, target_concept_id, relationship_type, 
			active, properties, created_at
		) 
		SELECT c1.concept_uuid, c2.concept_uuid, $1, $2, $3, NOW()
		FROM concepts c1, concepts c2
		WHERE c1.system = 'RxNorm' AND c1.code = $4
		  AND c2.system = 'RxNorm' AND c2.code = $5
		ON CONFLICT (source_concept_id, target_concept_id, relationship_type) 
		DO UPDATE SET active = $2, properties = $3, updated_at = NOW()
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	
	batchCount := 0
	
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, "|")
		
		if len(fields) < 16 {
			continue
		}
		
		// RXNREL format: RXCUI1|RXAUI1|STYPE1|REL|RXCUI2|RXAUI2|STYPE2|RELA|RUI|SRUI|SAB|SL|DIR|RG|SUPPRESS|CVF
		rxcui1 := fields[0]
		rel := fields[3]      // Relationship
		rxcui2 := fields[4]
		rela := fields[7]     // Additional relationship attribute
		sab := fields[10]     // Source abbreviation
		suppress := fields[14] // Suppress flag
		
		// Only process relationships from RxNorm source that are not suppressed
		if sab != "RXNORM" || suppress == "Y" {
			continue
		}
		
		// Skip self-relationships
		if rxcui1 == rxcui2 {
			continue
		}
		
		// Map relationship types to standard terms
		relationshipType := rel
		if rela != "" {
			relationshipType = rela // Use more specific relationship if available
		}
		
		properties := models.JSONB{
			"relationship":    rel,
			"relationship_attribute": rela,
			"source":          sab,
			"suppress_flag":   suppress,
			"source_file":     "RXNREL",
		}
		
		_, err = stmt.Exec(relationshipType, true, properties, rxcui1, rxcui2)
		if err != nil {
			r.logger.Error("Failed to insert RxNorm relationship", 
				zap.String("source", rxcui1), 
				zap.String("target", rxcui2),
				zap.Error(err))
			continue
		}
		
		batchCount++
		if batchCount%r.config.BatchSize == 0 {
			r.logger.Info("Processed RxNorm relationships", zap.Int("count", batchCount))
		}
	}
	
	if err := scanner.Err(); err != nil {
		return err
	}
	
	return txn.Commit()
}