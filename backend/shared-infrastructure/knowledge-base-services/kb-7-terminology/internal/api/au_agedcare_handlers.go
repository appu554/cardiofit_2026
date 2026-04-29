// Package api: au_agedcare_handlers.go
//
// AU Aged Care terminology endpoints. These query the kb7_snomed_*
// and kb7_amt_pack tables loaded by scripts/load_snomed_au_rf2.py
// and scripts/load_amt.py — fresh from the NCTS 30-April-2026 release.
//
// These endpoints are READ-ONLY and Postgres-only (no Neo4j, no
// runtime materialization). Each handler is a single SQL query
// designed for ACOP / aged-care pharmacist consumption patterns.
//
// Route group: /v1/au/*
//
// Registered from server.go via SetAUAgedCareHandlers(...).

package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SNOMED type IDs we care about
const (
	snomedFSN           = 900000000000003001 // Fully Specified Name
	snomedSynonym       = 900000000000013009 // Synonym
	snomedIsA           = 116680003          // IS-A relationship type
	moduleIDInternational = 900000000000207008
	moduleIDSCTAU       = 32506021000036107
	moduleIDAMT         = 900062011000036103
)

// AUAgedCareHandlers exposes AU agedcare terminology endpoints.
type AUAgedCareHandlers struct {
	db     *sql.DB
	logger *logrus.Logger
}

func NewAUAgedCareHandlers(db *sql.DB, logger *logrus.Logger) *AUAgedCareHandlers {
	return &AUAgedCareHandlers{db: db, logger: logger}
}

// RegisterRoutes wires the AU agedcare routes onto the given gin group.
// The caller passes a /v1/au group constructed in server.go.
func (h *AUAgedCareHandlers) RegisterRoutes(g *gin.RouterGroup) {
	g.GET("/health", h.health)
	g.GET("/concepts/:code", h.lookupConcept)
	g.GET("/concepts/:code/children", h.conceptChildren)
	g.GET("/concepts/:code/parents", h.conceptParents)
	g.GET("/concepts/search", h.searchConcepts)
	g.GET("/amt/search", h.amtSearch)
	g.GET("/amt/substance/:mp_sctid", h.amtBySubstance)
	g.GET("/amt/pack/:ctpp_sctid", h.amtPackHierarchy)
}

// ---------- Helpers ----------

func (h *AUAgedCareHandlers) jsonError(c *gin.Context, status int, msg string, err error) {
	if err != nil {
		h.logger.WithError(err).Warn(msg)
	}
	c.JSON(status, gin.H{"error": msg})
}

// ---------- Handlers ----------

// GET /v1/au/health
// Reports row counts so consumers can see what AU data is loaded.
func (h *AUAgedCareHandlers) health(c *gin.Context) {
	type tableCount struct {
		Table string `json:"table"`
		Rows  int64  `json:"rows"`
	}
	tables := []string{
		"kb7_snomed_concept",
		"kb7_snomed_description",
		"kb7_snomed_relationship",
		"kb7_snomed_refset_simple",
		"kb7_amt_pack",
	}
	out := make([]tableCount, 0, len(tables))
	for _, t := range tables {
		var n int64
		row := h.db.QueryRow("SELECT count(*) FROM " + t)
		if err := row.Scan(&n); err != nil {
			h.jsonError(c, http.StatusInternalServerError, "row count failed for "+t, err)
			return
		}
		out = append(out, tableCount{Table: t, Rows: n})
	}
	c.JSON(http.StatusOK, gin.H{
		"status":           "healthy",
		"tables":           out,
		"snomed_au_module": moduleIDSCTAU,
		"amt_module":       moduleIDAMT,
	})
}

// GET /v1/au/concepts/:code
// Returns the concept + its preferred (FSN) and English synonyms.
func (h *AUAgedCareHandlers) lookupConcept(c *gin.Context) {
	code, err := strconv.ParseInt(c.Param("code"), 10, 64)
	if err != nil {
		h.jsonError(c, http.StatusBadRequest, "code must be a SCTID (integer)", err)
		return
	}

	var (
		id, moduleID, defStatus int64
		active                  int
	)
	row := h.db.QueryRow(`
		SELECT id, active, module_id, definition_status_id
		FROM kb7_snomed_concept
		WHERE id = $1`, code)
	if err := row.Scan(&id, &active, &moduleID, &defStatus); err != nil {
		if err == sql.ErrNoRows {
			h.jsonError(c, http.StatusNotFound, "concept not found", nil)
			return
		}
		h.jsonError(c, http.StatusInternalServerError, "concept query failed", err)
		return
	}

	rows, err := h.db.Query(`
		SELECT term, type_id, language_code
		FROM kb7_snomed_description
		WHERE concept_id = $1 AND active = 1
		ORDER BY (type_id = $2) DESC, term`, code, snomedFSN)
	if err != nil {
		h.jsonError(c, http.StatusInternalServerError, "descriptions query failed", err)
		return
	}
	defer rows.Close()

	type desc struct {
		Term     string `json:"term"`
		TypeID   int64  `json:"type_id"`
		Language string `json:"language"`
	}
	descs := []desc{}
	for rows.Next() {
		var d desc
		if err := rows.Scan(&d.Term, &d.TypeID, &d.Language); err != nil {
			h.jsonError(c, http.StatusInternalServerError, "description scan failed", err)
			return
		}
		descs = append(descs, d)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                   id,
		"active":               active == 1,
		"module_id":            moduleID,
		"definition_status_id": defStatus,
		"descriptions":         descs,
	})
}

// GET /v1/au/concepts/:code/children?limit=50
// Returns IS-A children (immediate subtypes) of the given concept.
func (h *AUAgedCareHandlers) conceptChildren(c *gin.Context) {
	h.subsumption(c, "source_id", "destination_id")
}

// GET /v1/au/concepts/:code/parents?limit=50
// Returns IS-A parents of the given concept.
func (h *AUAgedCareHandlers) conceptParents(c *gin.Context) {
	h.subsumption(c, "destination_id", "source_id")
}

// subsumption: shared logic for children/parents.
// resultCol is the relationship column whose value to RETURN.
// pivotCol is the column to filter ON (= input concept).
func (h *AUAgedCareHandlers) subsumption(c *gin.Context, resultCol, pivotCol string) {
	code, err := strconv.ParseInt(c.Param("code"), 10, 64)
	if err != nil {
		h.jsonError(c, http.StatusBadRequest, "code must be a SCTID (integer)", err)
		return
	}
	limit := parseLimit(c, 50, 500)

	q := `
		SELECT r.` + resultCol + ` AS rel_concept_id, d.term
		FROM kb7_snomed_relationship r
		JOIN kb7_snomed_description d
		  ON d.concept_id = r.` + resultCol + `
		 AND d.active = 1
		 AND d.type_id = $1
		WHERE r.` + pivotCol + ` = $2
		  AND r.type_id = $3
		  AND r.active = 1
		LIMIT $4`
	rows, err := h.db.Query(q, snomedFSN, code, snomedIsA, limit)
	if err != nil {
		h.jsonError(c, http.StatusInternalServerError, "subsumption query failed", err)
		return
	}
	defer rows.Close()

	type rel struct {
		ID   int64  `json:"id"`
		Term string `json:"term"`
	}
	out := []rel{}
	for rows.Next() {
		var r rel
		if err := rows.Scan(&r.ID, &r.Term); err != nil {
			h.jsonError(c, http.StatusInternalServerError, "subsumption scan failed", err)
			return
		}
		out = append(out, r)
	}
	c.JSON(http.StatusOK, gin.H{
		"of":      code,
		"results": out,
		"count":   len(out),
		"limit":   limit,
	})
}

// GET /v1/au/concepts/search?q=foo&limit=20&module=au
// Case-insensitive term search across SNOMED descriptions.
// module=au filters to AU-extension concepts only.
func (h *AUAgedCareHandlers) searchConcepts(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		h.jsonError(c, http.StatusBadRequest, "q parameter required", nil)
		return
	}
	limit := parseLimit(c, 20, 200)

	module := c.Query("module") // "au" filters; otherwise no filter
	moduleFilter := ""
	args := []interface{}{"%" + q + "%", limit}
	if module == "au" {
		moduleFilter = " AND c.module_id = $3"
		args = append(args, moduleIDSCTAU)
	}

	rows, err := h.db.Query(`
		SELECT DISTINCT d.concept_id, d.term, c.module_id, c.active
		FROM kb7_snomed_description d
		JOIN kb7_snomed_concept c ON c.id = d.concept_id
		WHERE lower(d.term) LIKE lower($1)
		  AND d.active = 1`+moduleFilter+`
		ORDER BY d.term
		LIMIT $2`, args...)
	if err != nil {
		h.jsonError(c, http.StatusInternalServerError, "search query failed", err)
		return
	}
	defer rows.Close()

	type hit struct {
		ID       int64  `json:"id"`
		Term     string `json:"term"`
		ModuleID int64  `json:"module_id"`
		Active   bool   `json:"active"`
	}
	out := []hit{}
	for rows.Next() {
		var hh hit
		var active int
		if err := rows.Scan(&hh.ID, &hh.Term, &hh.ModuleID, &active); err != nil {
			h.jsonError(c, http.StatusInternalServerError, "search scan failed", err)
			return
		}
		hh.Active = active == 1
		out = append(out, hh)
	}
	c.JSON(http.StatusOK, gin.H{
		"q":       q,
		"results": out,
		"count":   len(out),
		"limit":   limit,
		"module":  module,
	})
}

// GET /v1/au/amt/search?q=metformin&level=mp&limit=20
// level: mp (substance) | tp (brand) | mpuu | tpuu | ctpp; default mp
func (h *AUAgedCareHandlers) amtSearch(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		h.jsonError(c, http.StatusBadRequest, "q parameter required", nil)
		return
	}
	limit := parseLimit(c, 20, 200)
	level := c.DefaultQuery("level", "mp")

	var ptCol, idCol string
	switch level {
	case "mp":
		ptCol, idCol = "mp_pt", "mp_sctid"
	case "tp":
		ptCol, idCol = "tpp_tp_pt", "tpp_tp_sctid"
	case "mpuu":
		ptCol, idCol = "mpuu_pt", "mpuu_sctid"
	case "tpuu":
		ptCol, idCol = "tpuu_pt", "tpuu_sctid"
	case "ctpp":
		ptCol, idCol = "ctpp_pt", "ctpp_sctid"
	default:
		h.jsonError(c, http.StatusBadRequest,
			"level must be one of: mp, tp, mpuu, tpuu, ctpp", nil)
		return
	}

	rows, err := h.db.Query(`
		SELECT DISTINCT `+idCol+`, `+ptCol+`
		FROM kb7_amt_pack
		WHERE lower(`+ptCol+`) LIKE lower($1)
		ORDER BY `+ptCol+`
		LIMIT $2`, "%"+q+"%", limit)
	if err != nil {
		h.jsonError(c, http.StatusInternalServerError, "AMT search failed", err)
		return
	}
	defer rows.Close()

	type hit struct {
		ID   int64  `json:"id"`
		Term string `json:"term"`
	}
	out := []hit{}
	for rows.Next() {
		var hh hit
		if err := rows.Scan(&hh.ID, &hh.Term); err != nil {
			h.jsonError(c, http.StatusInternalServerError, "AMT search scan failed", err)
			return
		}
		out = append(out, hh)
	}
	c.JSON(http.StatusOK, gin.H{
		"q":       q,
		"level":   level,
		"results": out,
		"count":   len(out),
		"limit":   limit,
	})
}

// GET /v1/au/amt/substance/:mp_sctid?limit=50
// Returns all CTPP packs that contain the given MP (substance).
func (h *AUAgedCareHandlers) amtBySubstance(c *gin.Context) {
	mp, err := strconv.ParseInt(c.Param("mp_sctid"), 10, 64)
	if err != nil {
		h.jsonError(c, http.StatusBadRequest, "mp_sctid must be an integer SCTID", err)
		return
	}
	limit := parseLimit(c, 50, 500)

	rows, err := h.db.Query(`
		SELECT ctpp_sctid, ctpp_pt, tpp_tp_pt, mpuu_pt, mp_pt, artg_id
		FROM kb7_amt_pack
		WHERE mp_sctid = $1
		ORDER BY ctpp_pt
		LIMIT $2`, mp, limit)
	if err != nil {
		h.jsonError(c, http.StatusInternalServerError, "amt by substance query failed", err)
		return
	}
	defer rows.Close()

	type pack struct {
		CTPPSCTID int64          `json:"ctpp_sctid"`
		CTPPPT    string         `json:"ctpp_pt"`
		Brand     string         `json:"brand"`
		MPUU      string         `json:"mpuu_pt"`
		MP        string         `json:"mp_pt"`
		ARTGID    sql.NullInt64  `json:"-"`
	}
	type wirePack struct {
		CTPPSCTID int64  `json:"ctpp_sctid"`
		CTPPPT    string `json:"ctpp_pt"`
		Brand     string `json:"brand"`
		MPUU      string `json:"mpuu_pt"`
		MP        string `json:"mp_pt"`
		ARTGID    *int64 `json:"artg_id,omitempty"`
	}
	out := []wirePack{}
	for rows.Next() {
		var p pack
		if err := rows.Scan(&p.CTPPSCTID, &p.CTPPPT, &p.Brand, &p.MPUU, &p.MP, &p.ARTGID); err != nil {
			h.jsonError(c, http.StatusInternalServerError, "amt by substance scan failed", err)
			return
		}
		w := wirePack{p.CTPPSCTID, p.CTPPPT, p.Brand, p.MPUU, p.MP, nil}
		if p.ARTGID.Valid {
			v := p.ARTGID.Int64
			w.ARTGID = &v
		}
		out = append(out, w)
	}
	c.JSON(http.StatusOK, gin.H{
		"mp_sctid": mp,
		"packs":    out,
		"count":    len(out),
		"limit":    limit,
	})
}

// GET /v1/au/amt/pack/:ctpp_sctid
// Returns the full AMT hierarchy walk for the given CTPP. A combination
// product (multi-chamber pack) returns multiple rows — one per substance.
func (h *AUAgedCareHandlers) amtPackHierarchy(c *gin.Context) {
	ctpp, err := strconv.ParseInt(c.Param("ctpp_sctid"), 10, 64)
	if err != nil {
		h.jsonError(c, http.StatusBadRequest, "ctpp_sctid must be an integer SCTID", err)
		return
	}

	rows, err := h.db.Query(`
		SELECT ctpp_sctid, ctpp_pt, artg_id,
		       tpp_sctid, tpp_pt,
		       tpuu_sctid, tpuu_pt,
		       tpp_tp_sctid, tpp_tp_pt,
		       mpp_sctid, mpp_pt,
		       mpuu_sctid, mpuu_pt,
		       mp_sctid, mp_pt
		FROM kb7_amt_pack
		WHERE ctpp_sctid = $1`, ctpp)
	if err != nil {
		h.jsonError(c, http.StatusInternalServerError, "amt pack query failed", err)
		return
	}
	defer rows.Close()

	type level struct {
		ID   int64  `json:"id"`
		Term string `json:"term"`
	}
	type packRow struct {
		CTPP      level         `json:"ctpp"`
		TPP       level         `json:"tpp"`
		TPUU      level         `json:"tpuu"`
		TP        level         `json:"tp"`
		MPP       level         `json:"mpp"`
		MPUU      level         `json:"mpuu"`
		MP        level         `json:"mp"`
		ARTGID    *int64        `json:"artg_id,omitempty"`
	}
	out := []packRow{}
	for rows.Next() {
		var (
			p              packRow
			artg           sql.NullInt64
			ctppID         int64
			ctppPT         string
		)
		if err := rows.Scan(
			&ctppID, &ctppPT, &artg,
			&p.TPP.ID, &p.TPP.Term,
			&p.TPUU.ID, &p.TPUU.Term,
			&p.TP.ID, &p.TP.Term,
			&p.MPP.ID, &p.MPP.Term,
			&p.MPUU.ID, &p.MPUU.Term,
			&p.MP.ID, &p.MP.Term,
		); err != nil {
			h.jsonError(c, http.StatusInternalServerError, "amt pack scan failed", err)
			return
		}
		p.CTPP = level{ID: ctppID, Term: ctppPT}
		if artg.Valid {
			v := artg.Int64
			p.ARTGID = &v
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		h.jsonError(c, http.StatusNotFound, "ctpp_sctid not found", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ctpp_sctid":            ctpp,
		"hierarchies":           out,
		"is_combination_product": len(out) > 1,
		"count":                 len(out),
	})
}

// ---------- Utility ----------

func parseLimit(c *gin.Context, def, max int) int {
	limStr := c.Query("limit")
	if limStr == "" {
		return def
	}
	n, err := strconv.Atoi(limStr)
	if err != nil || n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}
