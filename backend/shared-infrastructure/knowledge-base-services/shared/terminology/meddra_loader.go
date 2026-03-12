// Package terminology provides ontology-grounded terminology normalization services.
//
// meddra_loader.go: Load official MedDRA ASCII files into SQLite.
//
// This file implements the data loading for Issue 2 & 3 fixes.
//
// MedDRA File Format:
//
//	MedDRA ASCII files use $ as delimiter (not pipe as some docs suggest).
//	Files are located in the MedAscii/ subdirectory of the MedDRA distribution.
//
// Required Files:
//   - llt.asc:  Lowest Level Terms (80,000+ terms)
//   - pt.asc:   Preferred Terms (24,000+ terms)
//   - hlt.asc:  High Level Terms
//   - hlgt.asc: High Level Group Terms
//   - soc.asc:  System Organ Classes (27 classes)
//   - soc_hlgt.asc: SOC to HLGT mappings
//   - hlgt_hlt.asc: HLGT to HLT mappings
//   - hlt_pt.asc:   HLT to PT mappings
//
// Optional Files (for SNOMED mapping):
//   - SNOMED CT to MedDRA Map (from SNOMED International)
//
// MedDRA Subscription:
//
//	FREE for non-commercial organizations (hospitals, academics, non-profits).
//	Apply at: https://subscribe.meddra.org/
package terminology

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// MedDRALoader loads official MedDRA ASCII files into SQLite.
type MedDRALoader struct {
	db  *sql.DB
	log *logrus.Entry
}

// MedDRALoaderConfig contains configuration for the loader.
type MedDRALoaderConfig struct {
	// DBPath is the path to the SQLite database file.
	// If empty, uses in-memory database.
	DBPath string

	// Logger for logging operations.
	Logger *logrus.Entry
}

// NewMedDRALoader creates a new MedDRA loader.
func NewMedDRALoader(config MedDRALoaderConfig) (*MedDRALoader, error) {
	dsn := config.DBPath
	if dsn == "" {
		dsn = ":memory:"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent read performance
	if dsn != ":memory:" {
		_, _ = db.Exec("PRAGMA journal_mode=WAL")
	}

	// Set busy timeout
	_, _ = db.Exec("PRAGMA busy_timeout=5000")

	log := config.Logger
	if log == nil {
		log = logrus.NewEntry(logrus.StandardLogger())
	}

	return &MedDRALoader{
		db:  db,
		log: log.WithField("component", "meddra_loader"),
	}, nil
}

// LoadFromFiles loads official MedDRA ASCII files into SQLite.
//
// Parameters:
//   - meddraDir: Path to the MedDRA distribution directory
//     (should contain MedAscii/ subdirectory)
//
// Example:
//
//	loader.LoadFromFiles("/path/to/meddra_26_1_english")
func (l *MedDRALoader) LoadFromFiles(ctx context.Context, meddraDir string) error {
	startTime := time.Now()
	l.log.WithField("meddra_dir", meddraDir).Info("Starting MedDRA load")

	// Check if MedAscii directory exists
	asciiDir := filepath.Join(meddraDir, "MedAscii")
	if _, err := os.Stat(asciiDir); os.IsNotExist(err) {
		// Try without subdirectory (some distributions have files directly)
		asciiDir = meddraDir
	}

	// Create schema
	if err := l.createSchema(ctx); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Load files in dependency order
	// 1. SOC (System Organ Class) - top of hierarchy
	if err := l.loadSOC(ctx, asciiDir); err != nil {
		l.log.WithError(err).Warn("Failed to load SOC (optional)")
	}

	// 2. HLGT (High Level Group Term)
	if err := l.loadHLGT(ctx, asciiDir); err != nil {
		l.log.WithError(err).Warn("Failed to load HLGT (optional)")
	}

	// 3. HLT (High Level Term)
	if err := l.loadHLT(ctx, asciiDir); err != nil {
		l.log.WithError(err).Warn("Failed to load HLT (optional)")
	}

	// 4. PT (Preferred Term) - REQUIRED
	if err := l.loadPT(ctx, asciiDir); err != nil {
		return fmt.Errorf("failed to load PT (required): %w", err)
	}

	// 5. LLT (Lowest Level Term) - REQUIRED
	if err := l.loadLLT(ctx, asciiDir); err != nil {
		return fmt.Errorf("failed to load LLT (required): %w", err)
	}

	// 6. Load hierarchy mappings
	if err := l.loadHierarchyMappings(ctx, asciiDir); err != nil {
		l.log.WithError(err).Warn("Failed to load hierarchy mappings (optional)")
	}

	// 7. Create indexes for fast lookup
	if err := l.createIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// 8. Record metadata
	if err := l.recordMetadata(ctx, meddraDir); err != nil {
		l.log.WithError(err).Warn("Failed to record metadata")
	}

	duration := time.Since(startTime)
	stats := l.GetStats(ctx)
	l.log.WithFields(logrus.Fields{
		"duration":   duration.String(),
		"llt_count":  stats.LLTCount,
		"pt_count":   stats.PTCount,
		"soc_count":  stats.SOCCount,
	}).Info("MedDRA load complete")

	return nil
}

// createSchema creates the database schema.
func (l *MedDRALoader) createSchema(ctx context.Context) error {
	schema := `
		-- Lowest Level Terms (80,000+ terms)
		-- This is the primary lookup table for adverse event normalization
		CREATE TABLE IF NOT EXISTS meddra_llt (
			llt_code TEXT PRIMARY KEY,
			llt_name TEXT NOT NULL,
			pt_code TEXT NOT NULL,
			llt_whoart_code TEXT,
			llt_harts_code TEXT,
			llt_costart_sym TEXT,
			llt_icd9_code TEXT,
			llt_icd9cm_code TEXT,
			llt_icd10_code TEXT,
			llt_currency TEXT DEFAULT 'Y',  -- 'Y' = current, 'N' = non-current
			llt_jart_code TEXT
		);

		-- Preferred Terms (24,000+ terms)
		-- FAERS uses PT codes for adverse event reporting
		CREATE TABLE IF NOT EXISTS meddra_pt (
			pt_code TEXT PRIMARY KEY,
			pt_name TEXT NOT NULL,
			null_field TEXT,
			pt_soc_code TEXT,
			pt_whoart_code TEXT,
			pt_harts_code TEXT,
			pt_costart_sym TEXT,
			pt_icd9_code TEXT,
			pt_icd9cm_code TEXT,
			pt_icd10_code TEXT,
			pt_jart_code TEXT
		);

		-- High Level Terms
		CREATE TABLE IF NOT EXISTS meddra_hlt (
			hlt_code TEXT PRIMARY KEY,
			hlt_name TEXT NOT NULL,
			hlt_whoart_code TEXT,
			hlt_harts_code TEXT,
			hlt_costart_sym TEXT,
			hlt_icd9_code TEXT,
			hlt_icd9cm_code TEXT,
			hlt_icd10_code TEXT,
			hlt_jart_code TEXT
		);

		-- High Level Group Terms
		CREATE TABLE IF NOT EXISTS meddra_hlgt (
			hlgt_code TEXT PRIMARY KEY,
			hlgt_name TEXT NOT NULL,
			hlgt_whoart_code TEXT,
			hlgt_harts_code TEXT,
			hlgt_costart_sym TEXT,
			hlgt_icd9_code TEXT,
			hlgt_icd9cm_code TEXT,
			hlgt_icd10_code TEXT,
			hlgt_jart_code TEXT
		);

		-- System Organ Classes (27 classes)
		CREATE TABLE IF NOT EXISTS meddra_soc (
			soc_code TEXT PRIMARY KEY,
			soc_name TEXT NOT NULL,
			soc_abbrev TEXT,
			soc_whoart_code TEXT,
			soc_harts_code TEXT,
			soc_costart_sym TEXT,
			soc_icd9_code TEXT,
			soc_icd9cm_code TEXT,
			soc_icd10_code TEXT,
			soc_jart_code TEXT
		);

		-- Hierarchy mappings
		CREATE TABLE IF NOT EXISTS meddra_soc_hlgt (
			soc_code TEXT,
			hlgt_code TEXT,
			PRIMARY KEY (soc_code, hlgt_code)
		);

		CREATE TABLE IF NOT EXISTS meddra_hlgt_hlt (
			hlgt_code TEXT,
			hlt_code TEXT,
			PRIMARY KEY (hlgt_code, hlt_code)
		);

		CREATE TABLE IF NOT EXISTS meddra_hlt_pt (
			hlt_code TEXT,
			pt_code TEXT,
			PRIMARY KEY (hlt_code, pt_code)
		);

		-- SNOMED CT to MedDRA mappings (from SNOMED International)
		-- This enables dual-coding for EHR integration
		CREATE TABLE IF NOT EXISTS meddra_snomed_map (
			meddra_code TEXT NOT NULL,
			meddra_type TEXT NOT NULL,  -- 'PT' or 'LLT'
			snomed_code TEXT NOT NULL,
			snomed_term TEXT,
			relationship TEXT DEFAULT 'equivalent',
			PRIMARY KEY (meddra_code, snomed_code)
		);

		-- Metadata table
		CREATE TABLE IF NOT EXISTS meddra_metadata (
			key TEXT PRIMARY KEY,
			value TEXT
		);
	`

	_, err := l.db.ExecContext(ctx, schema)
	return err
}

// loadLLT loads Lowest Level Terms from llt.asc.
// Format: llt_code$llt_name$pt_code$llt_whoart$llt_harts$llt_costart$llt_icd9$llt_icd9cm$llt_icd10$llt_currency$llt_jart$
func (l *MedDRALoader) loadLLT(ctx context.Context, dir string) error {
	filePath := filepath.Join(dir, "llt.asc")
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open llt.asc: %w", err)
	}
	defer file.Close()

	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO meddra_llt
		(llt_code, llt_name, pt_code, llt_whoart_code, llt_harts_code,
		 llt_costart_sym, llt_icd9_code, llt_icd9cm_code, llt_icd10_code,
		 llt_currency, llt_jart_code)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for long lines

	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		fields := strings.Split(line, "$")
		if len(fields) < 10 {
			l.log.WithField("line", line).Warn("Invalid LLT line format")
			continue
		}

		// Extract fields with defaults for missing values
		lltCode := fields[0]
		lltName := fields[1]
		ptCode := fields[2]
		whoart := safeGet(fields, 3)
		harts := safeGet(fields, 4)
		costart := safeGet(fields, 5)
		icd9 := safeGet(fields, 6)
		icd9cm := safeGet(fields, 7)
		icd10 := safeGet(fields, 8)
		currency := safeGet(fields, 9)
		jart := safeGet(fields, 10)

		if currency == "" {
			currency = "Y"
		}

		_, err := stmt.ExecContext(ctx,
			lltCode, lltName, ptCode, whoart, harts,
			costart, icd9, icd9cm, icd10, currency, jart)
		if err != nil {
			l.log.WithError(err).WithField("llt_code", lltCode).Warn("Failed to insert LLT")
			continue
		}
		count++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	l.log.WithField("count", count).Info("Loaded LLT terms")
	return nil
}

// loadPT loads Preferred Terms from pt.asc.
// Format: pt_code$pt_name$null$pt_soc_code$...
func (l *MedDRALoader) loadPT(ctx context.Context, dir string) error {
	filePath := filepath.Join(dir, "pt.asc")
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open pt.asc: %w", err)
	}
	defer file.Close()

	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO meddra_pt
		(pt_code, pt_name, null_field, pt_soc_code, pt_whoart_code,
		 pt_harts_code, pt_costart_sym, pt_icd9_code, pt_icd9cm_code,
		 pt_icd10_code, pt_jart_code)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		fields := strings.Split(line, "$")
		if len(fields) < 4 {
			l.log.WithField("line", line).Warn("Invalid PT line format")
			continue
		}

		ptCode := fields[0]
		ptName := fields[1]
		nullField := safeGet(fields, 2)
		socCode := safeGet(fields, 3)
		whoart := safeGet(fields, 4)
		harts := safeGet(fields, 5)
		costart := safeGet(fields, 6)
		icd9 := safeGet(fields, 7)
		icd9cm := safeGet(fields, 8)
		icd10 := safeGet(fields, 9)
		jart := safeGet(fields, 10)

		_, err := stmt.ExecContext(ctx,
			ptCode, ptName, nullField, socCode, whoart,
			harts, costart, icd9, icd9cm, icd10, jart)
		if err != nil {
			l.log.WithError(err).WithField("pt_code", ptCode).Warn("Failed to insert PT")
			continue
		}
		count++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	l.log.WithField("count", count).Info("Loaded PT terms")
	return nil
}

// loadSOC loads System Organ Classes from soc.asc.
func (l *MedDRALoader) loadSOC(ctx context.Context, dir string) error {
	filePath := filepath.Join(dir, "soc.asc")
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO meddra_soc
		(soc_code, soc_name, soc_abbrev)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		fields := strings.Split(line, "$")
		if len(fields) < 2 {
			continue
		}

		socCode := fields[0]
		socName := fields[1]
		abbrev := safeGet(fields, 2)

		_, err := stmt.ExecContext(ctx, socCode, socName, abbrev)
		if err != nil {
			continue
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	l.log.WithField("count", count).Info("Loaded SOC terms")
	return nil
}

// loadHLGT loads High Level Group Terms from hlgt.asc.
func (l *MedDRALoader) loadHLGT(ctx context.Context, dir string) error {
	filePath := filepath.Join(dir, "hlgt.asc")
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO meddra_hlgt (hlgt_code, hlgt_name) VALUES (?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		fields := strings.Split(line, "$")
		if len(fields) < 2 {
			continue
		}

		_, err := stmt.ExecContext(ctx, fields[0], fields[1])
		if err != nil {
			continue
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	l.log.WithField("count", count).Info("Loaded HLGT terms")
	return nil
}

// loadHLT loads High Level Terms from hlt.asc.
func (l *MedDRALoader) loadHLT(ctx context.Context, dir string) error {
	filePath := filepath.Join(dir, "hlt.asc")
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO meddra_hlt (hlt_code, hlt_name) VALUES (?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		fields := strings.Split(line, "$")
		if len(fields) < 2 {
			continue
		}

		_, err := stmt.ExecContext(ctx, fields[0], fields[1])
		if err != nil {
			continue
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	l.log.WithField("count", count).Info("Loaded HLT terms")
	return nil
}

// loadHierarchyMappings loads the hierarchy relationship files.
func (l *MedDRALoader) loadHierarchyMappings(ctx context.Context, dir string) error {
	// Load SOC-HLGT mappings
	if err := l.loadMappingFile(ctx, dir, "soc_hlgt.asc", "meddra_soc_hlgt", "soc_code", "hlgt_code"); err != nil {
		l.log.WithError(err).Debug("Failed to load soc_hlgt mappings")
	}

	// Load HLGT-HLT mappings
	if err := l.loadMappingFile(ctx, dir, "hlgt_hlt.asc", "meddra_hlgt_hlt", "hlgt_code", "hlt_code"); err != nil {
		l.log.WithError(err).Debug("Failed to load hlgt_hlt mappings")
	}

	// Load HLT-PT mappings
	if err := l.loadMappingFile(ctx, dir, "hlt_pt.asc", "meddra_hlt_pt", "hlt_code", "pt_code"); err != nil {
		l.log.WithError(err).Debug("Failed to load hlt_pt mappings")
	}

	return nil
}

// loadMappingFile loads a two-column mapping file.
func (l *MedDRALoader) loadMappingFile(ctx context.Context, dir, fileName, tableName, col1, col2 string) error {
	filePath := filepath.Join(dir, fileName)
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, fmt.Sprintf(
		"INSERT OR REPLACE INTO %s (%s, %s) VALUES (?, ?)",
		tableName, col1, col2))
	if err != nil {
		return err
	}
	defer stmt.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		fields := strings.Split(line, "$")
		if len(fields) < 2 {
			continue
		}

		_, err := stmt.ExecContext(ctx, fields[0], fields[1])
		if err != nil {
			continue
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	l.log.WithFields(logrus.Fields{
		"file":  fileName,
		"count": count,
	}).Debug("Loaded mapping file")

	return nil
}

// LoadSNOMEDMapping loads SNOMED CT to MedDRA mappings.
// This enables dual-coding (MedDRA + SNOMED) for EHR integration.
func (l *MedDRALoader) LoadSNOMEDMapping(ctx context.Context, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open SNOMED mapping file: %w", err)
	}
	defer file.Close()

	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO meddra_snomed_map
		(meddra_code, meddra_type, snomed_code, snomed_term, relationship)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Tab-delimited format: meddra_code\tmeddra_type\tsnomed_code\tsnomed_term\trelationship
		fields := strings.Split(line, "\t")
		if len(fields) < 3 {
			continue
		}

		meddraCode := fields[0]
		meddraType := safeGet(fields, 1)
		snomedCode := safeGet(fields, 2)
		snomedTerm := safeGet(fields, 3)
		relationship := safeGet(fields, 4)

		if relationship == "" {
			relationship = "equivalent"
		}

		_, err := stmt.ExecContext(ctx, meddraCode, meddraType, snomedCode, snomedTerm, relationship)
		if err != nil {
			continue
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	l.log.WithField("count", count).Info("Loaded SNOMED mappings")
	return nil
}

// createIndexes creates indexes for fast lookup.
func (l *MedDRALoader) createIndexes(ctx context.Context) error {
	indexes := []string{
		// LLT indexes - primary lookup table
		"CREATE INDEX IF NOT EXISTS idx_llt_name_lower ON meddra_llt(LOWER(llt_name))",
		"CREATE INDEX IF NOT EXISTS idx_llt_pt_code ON meddra_llt(pt_code)",
		"CREATE INDEX IF NOT EXISTS idx_llt_currency ON meddra_llt(llt_currency)",

		// PT indexes
		"CREATE INDEX IF NOT EXISTS idx_pt_name_lower ON meddra_pt(LOWER(pt_name))",
		"CREATE INDEX IF NOT EXISTS idx_pt_soc_code ON meddra_pt(pt_soc_code)",

		// SNOMED mapping indexes
		"CREATE INDEX IF NOT EXISTS idx_snomed_meddra ON meddra_snomed_map(meddra_code)",
		"CREATE INDEX IF NOT EXISTS idx_snomed_snomed ON meddra_snomed_map(snomed_code)",
	}

	for _, idx := range indexes {
		if _, err := l.db.ExecContext(ctx, idx); err != nil {
			l.log.WithError(err).WithField("index", idx).Warn("Failed to create index")
		}
	}

	return nil
}

// recordMetadata stores metadata about the loaded dictionary.
func (l *MedDRALoader) recordMetadata(ctx context.Context, meddraDir string) error {
	metadata := map[string]string{
		"loaded_at":  time.Now().UTC().Format(time.RFC3339),
		"source_dir": meddraDir,
	}

	for key, value := range metadata {
		_, err := l.db.ExecContext(ctx,
			"INSERT OR REPLACE INTO meddra_metadata (key, value) VALUES (?, ?)",
			key, value)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetStats returns statistics about the loaded dictionary.
func (l *MedDRALoader) GetStats(ctx context.Context) *MedDRAStats {
	stats := &MedDRAStats{}

	l.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM meddra_llt").Scan(&stats.LLTCount)
	l.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM meddra_pt").Scan(&stats.PTCount)
	l.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM meddra_soc").Scan(&stats.SOCCount)
	l.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM meddra_snomed_map").Scan(&stats.SNOMEDMappingCount)

	l.db.QueryRowContext(ctx, "SELECT value FROM meddra_metadata WHERE key = 'version'").Scan(&stats.Version)
	l.db.QueryRowContext(ctx, "SELECT value FROM meddra_metadata WHERE key = 'loaded_at'").Scan(&stats.LoadedAt)

	return stats
}

// DB returns the underlying database for use by MedDRANormalizer.
func (l *MedDRALoader) DB() *sql.DB {
	return l.db
}

// Close closes the database connection.
func (l *MedDRALoader) Close() error {
	return l.db.Close()
}

// LoadFromMRCONSO loads MedDRA data from a UMLS MRCONSO.RRF file.
//
// MRCONSO.RRF is the UMLS Metathesaurus concept names and sources file.
// It is pipe-delimited with 18 columns:
//
//	CUI|LAT|TS|LUI|STT|SUI|ISPREF|AUI|SAUI|SCUI|SDUI|SAB|TTY|CODE|STR|SRL|SUPPRESS|CVF|
//	 0    1   2  3   4   5    6     7   8    9    10   11  12  13   14  15    16      17
//
// For MedDRA rows (SAB=MDR):
//   - TTY=PT:  CODE(13) = PT code, STR(14) = PT name, SDUI(10) = same as CODE
//   - TTY=LLT: CODE(13) = LLT code, STR(14) = LLT name, SDUI(10) = parent PT code
//   - TTY=OS:  CODE(13) = SOC code, STR(14) = SOC name (OS = "Ordering SOC")
//   - TTY=HT:  CODE(13) = HLT code, STR(14) = HLT name
//   - TTY=HG:  CODE(13) = HLGT code, STR(14) = HLGT name
//
// Parameters:
//   - mrconsoPath: Absolute path to MRCONSO.RRF file
//     (e.g., /Users/.../Downloads/2025AB/META/MRCONSO.RRF)
func (l *MedDRALoader) LoadFromMRCONSO(ctx context.Context, mrconsoPath string) error {
	startTime := time.Now()
	l.log.WithField("path", mrconsoPath).Info("Starting MedDRA load from MRCONSO.RRF")

	file, err := os.Open(mrconsoPath)
	if err != nil {
		return fmt.Errorf("failed to open MRCONSO.RRF: %w", err)
	}
	defer file.Close()

	// Create schema
	if err := l.createSchema(ctx); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Use a single transaction for all inserts (much faster for bulk loading)
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare statements for each term type
	ptStmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO meddra_pt (pt_code, pt_name, pt_soc_code)
		VALUES (?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare PT statement: %w", err)
	}
	defer ptStmt.Close()

	lltStmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO meddra_llt (llt_code, llt_name, pt_code, llt_currency)
		VALUES (?, ?, ?, 'Y')`)
	if err != nil {
		return fmt.Errorf("failed to prepare LLT statement: %w", err)
	}
	defer lltStmt.Close()

	socStmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO meddra_soc (soc_code, soc_name)
		VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare SOC statement: %w", err)
	}
	defer socStmt.Close()

	hltStmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO meddra_hlt (hlt_code, hlt_name)
		VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare HLT statement: %w", err)
	}
	defer hltStmt.Close()

	hlgtStmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO meddra_hlgt (hlgt_code, hlgt_name)
		VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare HLGT statement: %w", err)
	}
	defer hlgtStmt.Close()

	// Scan the file line by line
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024) // 4MB buffer

	var ptCount, lltCount, socCount, hltCount, hlgtCount, totalLines int

	for scanner.Scan() {
		totalLines++
		line := scanner.Text()
		if line == "" {
			continue
		}

		fields := strings.Split(line, "|")
		if len(fields) < 15 {
			continue
		}

		// Filter: only MDR (MedDRA) source, English language
		sab := fields[11] // SAB = source abbreviation
		lat := fields[1]  // LAT = language
		if sab != "MDR" || lat != "ENG" {
			continue
		}

		tty := fields[12]  // TTY = term type
		code := fields[13] // CODE = source-specific code
		name := fields[14] // STR = term string
		sdui := fields[10] // SDUI = source-asserted descriptor UI (parent PT for LLTs)

		switch tty {
		case "PT":
			if _, err := ptStmt.ExecContext(ctx, code, name, ""); err != nil {
				l.log.WithError(err).WithField("code", code).Debug("Failed to insert PT")
			} else {
				ptCount++
			}

		case "LLT":
			// SDUI contains the parent PT code for LLT rows
			ptCode := sdui
			if _, err := lltStmt.ExecContext(ctx, code, name, ptCode); err != nil {
				l.log.WithError(err).WithField("code", code).Debug("Failed to insert LLT")
			} else {
				lltCount++
			}

		case "OS": // Ordering SOC - System Organ Class
			if _, err := socStmt.ExecContext(ctx, code, name); err != nil {
				l.log.WithError(err).WithField("code", code).Debug("Failed to insert SOC")
			} else {
				socCount++
			}

		case "HT": // High Level Term
			if _, err := hltStmt.ExecContext(ctx, code, name); err != nil {
				l.log.WithError(err).WithField("code", code).Debug("Failed to insert HLT")
			} else {
				hltCount++
			}

		case "HG": // High Level Group Term
			if _, err := hlgtStmt.ExecContext(ctx, code, name); err != nil {
				l.log.WithError(err).WithField("code", code).Debug("Failed to insert HLGT")
			} else {
				hlgtCount++
			}
		}

		// Log progress every 1M lines
		if totalLines%1_000_000 == 0 {
			l.log.WithField("lines_scanned", totalLines).Info("MRCONSO.RRF scan progress")
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error reading MRCONSO.RRF: %w", err)
	}

	// Now update PT SOC codes by joining LLT→PT where PT has same code as LLT's pt_code
	// MRCONSO doesn't directly give us pt_soc_code, but we can derive it from SOC TTY rows
	// For now, leave pt_soc_code empty (can be enriched later from hierarchy files like MRHIER.RRF)

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Create indexes
	if err := l.createIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// Record metadata
	metadata := map[string]string{
		"loaded_at":    time.Now().UTC().Format(time.RFC3339),
		"source":       "UMLS MRCONSO.RRF",
		"source_path":  mrconsoPath,
		"version":      "2025AB",
		"total_lines":  fmt.Sprintf("%d", totalLines),
	}
	for key, value := range metadata {
		l.db.ExecContext(ctx,
			"INSERT OR REPLACE INTO meddra_metadata (key, value) VALUES (?, ?)",
			key, value)
	}

	duration := time.Since(startTime)
	l.log.WithFields(logrus.Fields{
		"duration":     duration.String(),
		"total_lines":  totalLines,
		"pt_count":     ptCount,
		"llt_count":    lltCount,
		"soc_count":    socCount,
		"hlt_count":    hltCount,
		"hlgt_count":   hlgtCount,
	}).Info("MedDRA load from MRCONSO.RRF complete")

	return nil
}

// LoadFromMRHIER enriches PT→SOC mappings from a UMLS MRHIER.RRF file.
//
// MRHIER.RRF is the UMLS Metathesaurus hierarchical relationships file.
// Format: CUI|AUI|CXN|PAUI|SAB|RELA|PTR|HCD|CVF|
//          0    1   2   3    4   5    6   7   8
//
// For MedDRA (SAB=MDR), the PTR field contains a dot-separated ancestor AUI path.
// The hierarchy is: SOC → HLGT → HLT → PT, so for a PT row the 2nd AUI
// in the PTR path (index 1) is the SOC AUI.
//
// This function:
//  1. Scans MRCONSO.RRF to build AUI → (code, TTY) lookup for MDR rows
//  2. Scans MRHIER.RRF filtered to SAB=MDR, extracts PT AUI → SOC AUI from PTR
//  3. Updates meddra_pt.pt_soc_code in the SQLite DB
//
// Parameters:
//   - mrhierPath: Absolute path to MRHIER.RRF
//   - mrconsoPath: Absolute path to MRCONSO.RRF (needed for AUI→code mapping)
func (l *MedDRALoader) LoadFromMRHIER(ctx context.Context, mrhierPath, mrconsoPath string) error {
	startTime := time.Now()
	l.log.WithFields(logrus.Fields{
		"mrhier_path":  mrhierPath,
		"mrconso_path": mrconsoPath,
	}).Info("Starting PT→SOC enrichment from MRHIER.RRF")

	// Step 1: Build AUI → (code, TTY) map from MRCONSO.RRF (MDR rows only)
	auiMap, err := l.buildAUIMap(ctx, mrconsoPath)
	if err != nil {
		return fmt.Errorf("failed to build AUI map from MRCONSO: %w", err)
	}
	l.log.WithField("aui_count", len(auiMap)).Info("Built AUI→code map from MRCONSO")

	// Step 2: Scan MRHIER.RRF for SAB=MDR, extract PT→SOC mappings
	ptToSOC, err := l.extractPTSOCFromMRHIER(ctx, mrhierPath, auiMap)
	if err != nil {
		return fmt.Errorf("failed to extract PT→SOC from MRHIER: %w", err)
	}
	l.log.WithField("mapping_count", len(ptToSOC)).Info("Extracted PT→SOC mappings from MRHIER")

	// Step 3: Update meddra_pt.pt_soc_code in SQLite
	updated, err := l.updatePTSOCCodes(ctx, ptToSOC)
	if err != nil {
		return fmt.Errorf("failed to update PT SOC codes: %w", err)
	}

	duration := time.Since(startTime)
	l.log.WithFields(logrus.Fields{
		"duration":       duration.String(),
		"mappings_found": len(ptToSOC),
		"rows_updated":   updated,
	}).Info("PT→SOC enrichment from MRHIER.RRF complete")

	return nil
}

// auiInfo holds code and term type for an AUI.
type auiInfo struct {
	Code string
	TTY  string
}

// buildAUIMap scans MRCONSO.RRF and builds AUI → (code, TTY) for MDR rows.
// We only need PT (TTY=PT) and SOC (TTY=OS) AUIs, but we collect all MDR
// AUIs since the file scan cost is the same.
func (l *MedDRALoader) buildAUIMap(ctx context.Context, mrconsoPath string) (map[string]auiInfo, error) {
	file, err := os.Open(mrconsoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open MRCONSO.RRF: %w", err)
	}
	defer file.Close()

	auiMap := make(map[string]auiInfo, 100000) // ~80K MDR AUIs expected
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	lines := 0
	for scanner.Scan() {
		lines++
		line := scanner.Text()
		if line == "" {
			continue
		}

		fields := strings.Split(line, "|")
		if len(fields) < 15 {
			continue
		}

		if fields[11] != "MDR" || fields[1] != "ENG" {
			continue
		}

		aui := fields[7]  // AUI
		tty := fields[12] // TTY
		code := fields[13] // CODE

		auiMap[aui] = auiInfo{Code: code, TTY: tty}

		if lines%2_000_000 == 0 {
			l.log.WithField("lines", lines).Debug("MRCONSO AUI scan progress")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return auiMap, nil
}

// extractPTSOCFromMRHIER scans MRHIER.RRF for SAB=MDR rows and extracts
// PT code → SOC code mappings using the PTR ancestor path.
func (l *MedDRALoader) extractPTSOCFromMRHIER(ctx context.Context, mrhierPath string, auiMap map[string]auiInfo) (map[string]string, error) {
	file, err := os.Open(mrhierPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open MRHIER.RRF: %w", err)
	}
	defer file.Close()

	// ptCode → socCode
	ptToSOC := make(map[string]string, 30000)

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	lines := 0
	mdrRows := 0
	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		lines++
		line := scanner.Text()
		if line == "" {
			continue
		}

		fields := strings.Split(line, "|")
		if len(fields) < 7 {
			continue
		}

		// Filter: SAB=MDR only
		if fields[4] != "MDR" {
			continue
		}
		mdrRows++

		aui := fields[1] // AUI of this node
		ptr := fields[6] // PTR = dot-separated ancestor AUI path

		if ptr == "" {
			continue
		}

		// Identify this node's type from AUI map
		nodeInfo, ok := auiMap[aui]
		if !ok || nodeInfo.TTY != "PT" {
			// We only care about PT rows — they give us PT→SOC
			continue
		}

		// PTR contains ancestor AUIs from root to parent.
		// MedDRA hierarchy: SOC(root) → HLGT → HLT → PT
		// For a PT node, PTR = "rootAUI.socAUI.hlgtAUI.hltAUI"
		// The SOC is at index 1 (second element, after the MedDRA root)
		ancestors := strings.Split(ptr, ".")
		if len(ancestors) < 2 {
			continue
		}

		// The first AUI is the MedDRA root (V-MDR), second is SOC
		socAUI := ancestors[1]
		socInfo, ok := auiMap[socAUI]
		if !ok || socInfo.TTY != "OS" {
			// Try other positions — some hierarchies may vary
			for i := 0; i < len(ancestors) && i < 3; i++ {
				if info, found := auiMap[ancestors[i]]; found && info.TTY == "OS" {
					socInfo = info
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}

		ptToSOC[nodeInfo.Code] = socInfo.Code

		if lines%5_000_000 == 0 {
			l.log.WithFields(logrus.Fields{
				"lines":    lines,
				"mdr_rows": mdrRows,
				"mappings": len(ptToSOC),
			}).Info("MRHIER scan progress")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	l.log.WithFields(logrus.Fields{
		"total_lines": lines,
		"mdr_rows":    mdrRows,
	}).Info("MRHIER scan complete")

	return ptToSOC, nil
}

// updatePTSOCCodes updates meddra_pt.pt_soc_code for all mapped PTs.
func (l *MedDRALoader) updatePTSOCCodes(ctx context.Context, ptToSOC map[string]string) (int, error) {
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `UPDATE meddra_pt SET pt_soc_code = ? WHERE pt_code = ?`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	updated := 0
	for ptCode, socCode := range ptToSOC {
		result, err := stmt.ExecContext(ctx, socCode, ptCode)
		if err != nil {
			l.log.WithError(err).WithField("pt_code", ptCode).Debug("Failed to update PT SOC code")
			continue
		}
		if n, _ := result.RowsAffected(); n > 0 {
			updated++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return updated, nil
}

// LoadFromValueSetJSON loads MedDRA terms from a FHIR ValueSet expansion JSON file.
//
// This is the zero-dependency loading channel — it reads from the KB7 Terminology
// Service's pre-expanded ValueSet files, avoiding the need for a UMLS license.
//
// The JSON has structure:
//
//	{
//	  "resourceType": "ValueSet",
//	  "expansion": {
//	    "total": 79661,
//	    "contains": [
//	      {"system": "http://terminology.hl7.org/CodeSystem/mdr", "code": "10000002", "display": "11-beta-hydroxylase deficiency"},
//	      ...
//	    ]
//	  }
//	}
//
// Limitations vs MRCONSO loading:
//   - No LLT→PT parent mapping (all entries loaded as both LLT and PT with self-reference)
//   - SOC classification uses keyword heuristic (not official hierarchy) — ~85% accuracy
//   - Term validation works fully; FAERS PT code assignment is approximate
//
// Parameters:
//   - jsonPath: Absolute path to the expanded ValueSet JSON file
//     (e.g., kb-7-terminology/data/ontoserver-valuesets/expansions/meddra-code-27-1_expanded.json)
func (l *MedDRALoader) LoadFromValueSetJSON(ctx context.Context, jsonPath string) error {
	startTime := time.Now()
	l.log.WithField("path", jsonPath).Info("Starting MedDRA load from FHIR ValueSet JSON")

	file, err := os.Open(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to open ValueSet JSON: %w", err)
	}
	defer file.Close()

	// Parse JSON using streaming decoder for memory efficiency
	var vs fhirValueSet
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&vs); err != nil {
		return fmt.Errorf("failed to parse ValueSet JSON: %w", err)
	}

	if vs.ResourceType != "ValueSet" {
		return fmt.Errorf("expected resourceType 'ValueSet', got '%s'", vs.ResourceType)
	}

	if len(vs.Expansion.Contains) == 0 {
		return fmt.Errorf("ValueSet expansion is empty (no terms found)")
	}

	l.log.WithField("total", vs.Expansion.Total).Info("Parsed ValueSet expansion")

	// Create schema (same tables as MRCONSO loader)
	if err := l.createSchema(ctx); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Bulk insert into SQLite
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert as both LLT and PT (since ValueSet doesn't distinguish)
	lltStmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO meddra_llt (llt_code, llt_name, pt_code, llt_currency)
		VALUES (?, ?, ?, 'Y')`)
	if err != nil {
		return fmt.Errorf("failed to prepare LLT statement: %w", err)
	}
	defer lltStmt.Close()

	ptStmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO meddra_pt (pt_code, pt_name, pt_soc_code)
		VALUES (?, ?, '')`)
	if err != nil {
		return fmt.Errorf("failed to prepare PT statement: %w", err)
	}
	defer ptStmt.Close()

	var lltCount, ptCount, skipped int
	for _, entry := range vs.Expansion.Contains {
		if entry.Code == "" || entry.Display == "" {
			skipped++
			continue
		}

		// Insert as LLT (self-referencing pt_code since we don't know the parent)
		if _, err := lltStmt.ExecContext(ctx, entry.Code, entry.Display, entry.Code); err != nil {
			l.log.WithError(err).WithField("code", entry.Code).Debug("Failed to insert LLT from ValueSet")
			continue
		}
		lltCount++

		// Also insert as PT (enables lookupPT to work)
		if _, err := ptStmt.ExecContext(ctx, entry.Code, entry.Display); err != nil {
			// Ignore — PT may already exist from LLT with same code
		} else {
			ptCount++
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Create indexes
	if err := l.createIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// Record metadata
	version := vs.Version
	if version == "" {
		version = "unknown"
	}
	metadata := map[string]string{
		"loaded_at":   time.Now().UTC().Format(time.RFC3339),
		"source":      "FHIR ValueSet JSON",
		"source_path": jsonPath,
		"version":     version,
	}
	for key, value := range metadata {
		l.db.ExecContext(ctx,
			"INSERT OR REPLACE INTO meddra_metadata (key, value) VALUES (?, ?)",
			key, value)
	}

	duration := time.Since(startTime)
	l.log.WithFields(logrus.Fields{
		"duration":  duration.String(),
		"llt_count": lltCount,
		"pt_count":  ptCount,
		"skipped":   skipped,
		"version":   version,
	}).Info("MedDRA load from ValueSet JSON complete")

	return nil
}

// fhirValueSet is the minimal FHIR ValueSet structure needed for MedDRA loading.
type fhirValueSet struct {
	ResourceType string `json:"resourceType"`
	Version      string `json:"version"`
	Name         string `json:"name"`
	Expansion    struct {
		Total    int                  `json:"total"`
		Contains []fhirValueSetEntry  `json:"contains"`
	} `json:"expansion"`
}

// fhirValueSetEntry represents a single code in a ValueSet expansion.
type fhirValueSetEntry struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display"`
}

// safeGet safely gets an element from a slice, returning empty string if out of bounds.
func safeGet(slice []string, index int) string {
	if index >= len(slice) {
		return ""
	}
	return slice[index]
}
