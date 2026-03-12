package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/models"
	"kb-7-terminology/internal/semantic"

	"github.com/sirupsen/logrus"
)

// ============================================================================
// Refset Service
// ============================================================================
// Service layer for NCTS reference set operations.
// Provides caching (L2 local + L2.5 Redis) and business logic.
// ============================================================================

// RefsetService handles reference set operations
type RefsetService struct {
	neo4jClient     *semantic.Neo4jClient
	redisClient     *cache.RedisClient
	logger          *logrus.Logger

	// L2 Local cache
	membershipCache *sync.Map // key: "refset:{refsetId}" -> []RefsetMember
	refsetCache     *sync.Map // key: "refsets:all" -> []Refset

	// Cache configuration
	localCacheTTL time.Duration
	redisCacheTTL time.Duration
}

// RefsetServiceConfig configures the RefsetService
type RefsetServiceConfig struct {
	LocalCacheTTL time.Duration
	RedisCacheTTL time.Duration
}

// DefaultRefsetServiceConfig returns default configuration
func DefaultRefsetServiceConfig() *RefsetServiceConfig {
	return &RefsetServiceConfig{
		LocalCacheTTL: 5 * time.Minute,
		RedisCacheTTL: 30 * time.Minute,
	}
}

// NewRefsetService creates a new RefsetService
func NewRefsetService(
	neo4jClient *semantic.Neo4jClient,
	redisClient *cache.RedisClient,
	logger *logrus.Logger,
) *RefsetService {
	config := DefaultRefsetServiceConfig()

	return &RefsetService{
		neo4jClient:     neo4jClient,
		redisClient:     redisClient,
		logger:          logger,
		membershipCache: &sync.Map{},
		refsetCache:     &sync.Map{},
		localCacheTTL:   config.LocalCacheTTL,
		redisCacheTTL:   config.RedisCacheTTL,
	}
}

// ============================================================================
// Refset Listing
// ============================================================================

// ListRefsets returns all available refsets
func (s *RefsetService) ListRefsets(ctx context.Context) (*models.RefsetListResponse, error) {
	start := time.Now()

	cypher := `
		MATCH (r:Refset)
		OPTIONAL MATCH (c:Class)-[m:IN_REFSET]->(r)
		WITH r, count(m) AS memberCount
		RETURN r.id AS id, r.name AS name, memberCount
		ORDER BY memberCount DESC
	`

	result, err := s.neo4jClient.ExecuteRead(ctx, cypher, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list refsets: %w", err)
	}

	refsets := make([]models.Refset, 0, len(result))
	for _, row := range result {
		refset := models.Refset{
			ID:          getStringValue(row, "id"),
			Name:        getStringValue(row, "name"),
			MemberCount: getIntValue(row, "memberCount"),
			Active:      true,
		}
		refsets = append(refsets, refset)
	}

	return &models.RefsetListResponse{
		Success:     true,
		Refsets:     refsets,
		TotalCount:  len(refsets),
		QueryTimeMs: float64(time.Since(start).Microseconds()) / 1000,
	}, nil
}

// GetRefset returns details for a specific refset
func (s *RefsetService) GetRefset(ctx context.Context, refsetID string) (*models.RefsetDetailResponse, error) {
	start := time.Now()

	cypher := `
		MATCH (r:Refset {id: $refsetId})
		OPTIONAL MATCH (c:Class)-[m:IN_REFSET]->(r)
		WITH r, count(m) AS memberCount
		RETURN r.id AS id, r.name AS name, memberCount
	`

	params := map[string]interface{}{
		"refsetId": refsetID,
	}

	result, err := s.neo4jClient.ExecuteRead(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get refset: %w", err)
	}

	if len(result) == 0 {
		return &models.RefsetDetailResponse{
			Success:     false,
			QueryTimeMs: float64(time.Since(start).Microseconds()) / 1000,
		}, nil
	}

	row := result[0]
	refset := &models.Refset{
		ID:          getStringValue(row, "id"),
		Name:        getStringValue(row, "name"),
		MemberCount: getIntValue(row, "memberCount"),
		Active:      true,
	}

	return &models.RefsetDetailResponse{
		Success:     true,
		Refset:      refset,
		MemberCount: refset.MemberCount,
		QueryTimeMs: float64(time.Since(start).Microseconds()) / 1000,
	}, nil
}

// ============================================================================
// Refset Members
// ============================================================================

// GetRefsetMembers returns members of a refset with pagination
func (s *RefsetService) GetRefsetMembers(ctx context.Context, refsetID string, opts *models.RefsetQueryOptions) (*models.RefsetLookupResult, error) {
	start := time.Now()

	if opts == nil {
		opts = models.DefaultRefsetQueryOptions()
	}

	// Build query with pagination
	// Note: Uses Class nodes (from OWL import) with uri property containing SNOMED code
	cypher := `
		MATCH (c:Class)-[m:IN_REFSET]->(r:Refset {id: $refsetId})
		WITH c, m, r,
		     CASE WHEN c.uri STARTS WITH 'http://snomed.info/id/'
		          THEN substring(c.uri, 22)
		          ELSE c.uri END AS code
		RETURN code AS code, c.prefLabel AS label,
		       m.effectiveTime AS effectiveTime, m.memberId AS memberId,
		       m.moduleId AS moduleId
		ORDER BY c.prefLabel
		SKIP $offset LIMIT $limit
	`

	params := map[string]interface{}{
		"refsetId": refsetID,
		"offset":   opts.Offset,
		"limit":    opts.Limit,
	}

	result, err := s.neo4jClient.ExecuteRead(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get refset members: %w", err)
	}

	members := make([]models.RefsetMember, 0, len(result))
	for _, row := range result {
		member := models.RefsetMember{
			MemberID:              getStringValue(row, "memberId"),
			RefsetID:              refsetID,
			ReferencedComponentID: getStringValue(row, "code"),
			ModuleID:              getStringValue(row, "moduleId"),
			Active:                true,
			ConceptCode:           getStringValue(row, "code"),
			ConceptLabel:          getStringValue(row, "label"),
		}

		// Parse effective time if present
		if et, ok := row["effectiveTime"]; ok && et != nil {
			if t, ok := et.(time.Time); ok {
				member.EffectiveTime = t
			}
		}

		members = append(members, member)
	}

	// Get total count
	totalCount := 0
	if opts.IncludeCounts {
		countCypher := `
			MATCH (c:Class)-[m:IN_REFSET]->(r:Refset {id: $refsetId})
			RETURN count(m) AS total
		`
		countResult, err := s.neo4jClient.ExecuteRead(ctx, countCypher, map[string]interface{}{"refsetId": refsetID})
		if err == nil && len(countResult) > 0 {
			totalCount = getIntValue(countResult[0], "total")
		}
	}

	// Get refset name
	refsetName := ""
	nameCypher := `MATCH (r:Refset {id: $refsetId}) RETURN r.name AS name`
	nameResult, err := s.neo4jClient.ExecuteRead(ctx, nameCypher, map[string]interface{}{"refsetId": refsetID})
	if err == nil && len(nameResult) > 0 {
		refsetName = getStringValue(nameResult[0], "name")
	}

	return &models.RefsetLookupResult{
		Success:     true,
		RefsetID:    refsetID,
		RefsetName:  refsetName,
		Members:     members,
		MemberCount: len(members),
		TotalCount:  totalCount,
		Offset:      opts.Offset,
		Limit:       opts.Limit,
		QueryTimeMs: float64(time.Since(start).Microseconds()) / 1000,
	}, nil
}

// ============================================================================
// Concept Refsets (Reverse Lookup)
// ============================================================================

// GetConceptRefsets returns all refsets a concept belongs to
func (s *RefsetService) GetConceptRefsets(ctx context.Context, conceptCode string) (*models.ConceptRefsets, error) {
	start := time.Now()

	// Match by URI for Class nodes from OWL import
	cypher := `
		MATCH (c:Class {uri: 'http://snomed.info/id/' + $code})-[m:IN_REFSET]->(r:Refset)
		RETURN r.id AS refsetId, r.name AS refsetName,
		       m.effectiveTime AS effectiveTime, m.moduleId AS moduleId
		ORDER BY r.name
	`

	params := map[string]interface{}{
		"code": conceptCode,
	}

	result, err := s.neo4jClient.ExecuteRead(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get concept refsets: %w", err)
	}

	refsets := make([]models.Refset, 0, len(result))
	for _, row := range result {
		refset := models.Refset{
			ID:       getStringValue(row, "refsetId"),
			Name:     getStringValue(row, "refsetName"),
			ModuleID: getStringValue(row, "moduleId"),
			Active:   true,
		}
		refsets = append(refsets, refset)
	}

	return &models.ConceptRefsets{
		Success:     true,
		ConceptID:   conceptCode,
		ConceptCode: conceptCode,
		Refsets:     refsets,
		QueryTimeMs: float64(time.Since(start).Microseconds()) / 1000,
	}, nil
}

// ============================================================================
// Membership Check
// ============================================================================

// IsConceptInRefset checks if a concept is a member of a refset (O(1) lookup)
func (s *RefsetService) IsConceptInRefset(ctx context.Context, conceptCode, refsetID string) (*models.RefsetMembershipCheck, error) {
	start := time.Now()

	// Match by URI for Class nodes from OWL import
	cypher := `
		MATCH (c:Class {uri: 'http://snomed.info/id/' + $code})-[m:IN_REFSET]->(r:Refset {id: $refsetId})
		RETURN m.memberId AS memberId
		LIMIT 1
	`

	params := map[string]interface{}{
		"code":     conceptCode,
		"refsetId": refsetID,
	}

	result, err := s.neo4jClient.ExecuteRead(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to check membership: %w", err)
	}

	isMember := len(result) > 0
	memberID := ""
	if isMember {
		memberID = getStringValue(result[0], "memberId")
	}

	return &models.RefsetMembershipCheck{
		Success:     true,
		ConceptCode: conceptCode,
		RefsetID:    refsetID,
		IsMember:    isMember,
		MemberID:    memberID,
		QueryTimeMs: float64(time.Since(start).Microseconds()) / 1000,
	}, nil
}

// ============================================================================
// Import Status
// ============================================================================

// GetImportStatus returns the current import status and history
func (s *RefsetService) GetImportStatus(ctx context.Context) (*models.ImportStatusResponse, error) {
	start := time.Now()

	// Get current version
	cypher := `
		MATCH (m:ImportMetadata {type: 'NCTS_REFSET'})
		RETURN m.version AS version, m.importedAt AS importedAt,
		       m.fileCount AS fileCount, m.relationshipCount AS relationshipCount
		ORDER BY m.importedAt DESC
	`

	result, err := s.neo4jClient.ExecuteRead(ctx, cypher, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get import status: %w", err)
	}

	if len(result) == 0 {
		return &models.ImportStatusResponse{
			Success:     true,
			QueryTimeMs: float64(time.Since(start).Microseconds()) / 1000,
		}, nil
	}

	// Build history
	history := make([]models.ImportMetadata, 0, len(result))
	for _, row := range result {
		meta := models.ImportMetadata{
			Type:              "NCTS_REFSET",
			Version:           getStringValue(row, "version"),
			FileCount:         getIntValue(row, "fileCount"),
			RelationshipCount: getIntValue(row, "relationshipCount"),
		}

		if t, ok := row["importedAt"]; ok && t != nil {
			if ts, ok := t.(time.Time); ok {
				meta.ImportedAt = ts
			}
		}

		history = append(history, meta)
	}

	// Get current (first in history)
	current := history[0]

	// Get refset type counts
	refsetTypes := make(map[string]int)
	countCypher := `
		MATCH ()-[m:IN_REFSET]->()
		RETURN count(m) AS total
	`
	countResult, err := s.neo4jClient.ExecuteRead(ctx, countCypher, nil)
	if err == nil && len(countResult) > 0 {
		refsetTypes["total"] = getIntValue(countResult[0], "total")
	}

	return &models.ImportStatusResponse{
		Success:           true,
		CurrentVersion:    current.Version,
		ImportedAt:        current.ImportedAt,
		FileCount:         current.FileCount,
		RelationshipCount: current.RelationshipCount,
		RefsetTypes:       refsetTypes,
		History:           history,
		QueryTimeMs:       float64(time.Since(start).Microseconds()) / 1000,
	}, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func getStringValue(row map[string]interface{}, key string) string {
	if val, ok := row[key]; ok && val != nil {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func getIntValue(row map[string]interface{}, key string) int {
	if val, ok := row[key]; ok && val != nil {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return 0
}

// ============================================================================
// Cache Invalidation
// ============================================================================

// InvalidateCache clears all cached refset data
func (s *RefsetService) InvalidateCache() {
	s.membershipCache = &sync.Map{}
	s.refsetCache = &sync.Map{}
	s.logger.Info("Refset cache invalidated")
}
