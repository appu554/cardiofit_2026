// Package graph provides Neo4j graph database operations for KB-3 Guidelines
package graph

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/kb-3-guidelines/pkg/models"
)

// Neo4jService provides graph database operations
type Neo4jService struct {
	driver neo4j.DriverWithContext
	logger *logrus.Logger
}

// NewNeo4jService creates a new Neo4j service
func NewNeo4jService(uri, username, password string) (*Neo4jService, error) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	// Verify connectivity
	ctx := context.Background()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("failed to verify Neo4j connectivity: %w", err)
	}

	logger.Info("Neo4j connection established")

	return &Neo4jService{
		driver: driver,
		logger: logger,
	}, nil
}

// Close closes the Neo4j driver
func (s *Neo4jService) Close(ctx context.Context) error {
	return s.driver.Close(ctx)
}

// Health checks Neo4j connectivity
func (s *Neo4jService) Health(ctx context.Context) error {
	return s.driver.VerifyConnectivity(ctx)
}

// ===== Guideline Graph Operations =====

// CreateGuidelineNode creates a guideline node in the graph
func (s *Neo4jService) CreateGuidelineNode(ctx context.Context, g *models.Guideline) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MERGE (g:Guideline {guidelineId: $guidelineId})
			SET g.name = $name,
			    g.source = $source,
			    g.version = $version,
			    g.domain = $domain,
			    g.evidenceGrade = $evidenceGrade,
			    g.active = $active,
			    g.updatedAt = datetime()
			RETURN g
		`
		params := map[string]interface{}{
			"guidelineId":   g.GuidelineID,
			"name":          g.Name,
			"source":        g.Source,
			"version":       g.Version,
			"domain":        g.Domain,
			"evidenceGrade": string(g.EvidenceGrade),
			"active":        g.Active,
		}
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})

	if err != nil {
		return fmt.Errorf("failed to create guideline node: %w", err)
	}
	return nil
}

// CreateRecommendationNodes creates recommendation nodes linked to a guideline
func (s *Neo4jService) CreateRecommendationNodes(ctx context.Context, guidelineID string, recommendations []models.Recommendation) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	for _, rec := range recommendations {
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
			query := `
				MATCH (g:Guideline {guidelineId: $guidelineId})
				MERGE (r:Recommendation {recommendationId: $recommendationId})
				SET r.action = $action,
				    r.strength = $strength,
				    r.evidenceQuality = $evidenceQuality
				MERGE (g)-[:HAS_RECOMMENDATION]->(r)
				RETURN r
			`
			params := map[string]interface{}{
				"guidelineId":      guidelineID,
				"recommendationId": rec.RecommendationID,
				"action":           rec.Action,
				"strength":         rec.Strength,
				"evidenceQuality":  rec.EvidenceQuality,
			}
			_, err := tx.Run(ctx, query, params)
			return nil, err
		})
		if err != nil {
			return fmt.Errorf("failed to create recommendation node: %w", err)
		}
	}
	return nil
}

// ===== Conflict Detection =====

// DetectConflicts finds potential conflicts between guidelines
func (s *Neo4jService) DetectConflicts(ctx context.Context, domain string) ([]models.Conflict, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MATCH (g1:Guideline)-[:HAS_RECOMMENDATION]->(r1:Recommendation)
			MATCH (g2:Guideline)-[:HAS_RECOMMENDATION]->(r2:Recommendation)
			WHERE g1.guidelineId < g2.guidelineId
			  AND g1.active = true AND g2.active = true
			  AND ($domain = '' OR g1.domain = $domain)
			  AND r1.action CONTAINS r2.action OR r2.action CONTAINS r1.action
			RETURN g1.guidelineId AS guideline1,
			       g2.guidelineId AS guideline2,
			       r1 AS recommendation1,
			       r2 AS recommendation2,
			       CASE
			         WHEN r1.strength <> r2.strength THEN 'evidence_disagreement'
			         ELSE 'target_difference'
			       END AS conflictType
			LIMIT 100
		`
		params := map[string]interface{}{
			"domain": domain,
		}
		records, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		var conflicts []models.Conflict
		for records.Next(ctx) {
			record := records.Record()
			conflict := models.Conflict{
				Guideline1ID: record.Values[0].(string),
				Guideline2ID: record.Values[1].(string),
				Type:         models.ConflictType(record.Values[4].(string)),
			}
			conflicts = append(conflicts, conflict)
		}
		return conflicts, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to detect conflicts: %w", err)
	}
	return result.([]models.Conflict), nil
}

// ===== Clinical Pathway Graph Operations =====

// CreatePathwayGraph creates a clinical pathway in the graph
func (s *Neo4jService) CreatePathwayGraph(ctx context.Context, protocol *models.Protocol) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// Create protocol node
		protocolQuery := `
			MERGE (p:Protocol {protocolId: $protocolId})
			SET p.name = $name,
			    p.type = $type,
			    p.guidelineSource = $guidelineSource,
			    p.version = $version,
			    p.active = $active
			RETURN p
		`
		protocolParams := map[string]interface{}{
			"protocolId":      protocol.ProtocolID,
			"name":            protocol.Name,
			"type":            string(protocol.Type),
			"guidelineSource": protocol.GuidelineSource,
			"version":         protocol.Version,
			"active":          protocol.Active,
		}
		_, err := tx.Run(ctx, protocolQuery, protocolParams)
		if err != nil {
			return nil, err
		}

		// Create stage nodes
		var prevStageID string
		for _, stage := range protocol.Stages {
			stageQuery := `
				MATCH (p:Protocol {protocolId: $protocolId})
				MERGE (s:Stage {stageId: $stageId, protocolId: $protocolId})
				SET s.name = $name,
				    s.order = $order,
				    s.description = $description
				MERGE (p)-[:HAS_STAGE]->(s)
				RETURN s
			`
			stageParams := map[string]interface{}{
				"protocolId":  protocol.ProtocolID,
				"stageId":     stage.StageID,
				"name":        stage.Name,
				"order":       stage.Order,
				"description": stage.Description,
			}
			_, err := tx.Run(ctx, stageQuery, stageParams)
			if err != nil {
				return nil, err
			}

			// Link stages in order
			if prevStageID != "" {
				linkQuery := `
					MATCH (s1:Stage {stageId: $prevStageId, protocolId: $protocolId})
					MATCH (s2:Stage {stageId: $stageId, protocolId: $protocolId})
					MERGE (s1)-[:FOLLOWED_BY]->(s2)
				`
				linkParams := map[string]interface{}{
					"protocolId":  protocol.ProtocolID,
					"prevStageId": prevStageID,
					"stageId":     stage.StageID,
				}
				_, err := tx.Run(ctx, linkQuery, linkParams)
				if err != nil {
					return nil, err
				}
			}
			prevStageID = stage.StageID

			// Create action nodes
			for _, action := range stage.Actions {
				actionQuery := `
					MATCH (s:Stage {stageId: $stageId, protocolId: $protocolId})
					MERGE (a:Action {actionId: $actionId, stageId: $stageId})
					SET a.name = $name,
					    a.type = $type,
					    a.required = $required,
					    a.deadlineMinutes = $deadlineMinutes
					MERGE (s)-[:HAS_ACTION]->(a)
					RETURN a
				`
				actionParams := map[string]interface{}{
					"protocolId":      protocol.ProtocolID,
					"stageId":         stage.StageID,
					"actionId":        action.ActionID,
					"name":            action.Name,
					"type":            string(action.Type),
					"required":        action.Required,
					"deadlineMinutes": int(action.Deadline.Minutes()),
				}
				_, err := tx.Run(ctx, actionQuery, actionParams)
				if err != nil {
					return nil, err
				}
			}
		}

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("failed to create pathway graph: %w", err)
	}
	return nil
}

// GetPathwayGraph retrieves a clinical pathway from the graph
func (s *Neo4jService) GetPathwayGraph(ctx context.Context, protocolID string) (*models.Protocol, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MATCH (p:Protocol {protocolId: $protocolId})
			OPTIONAL MATCH (p)-[:HAS_STAGE]->(s:Stage)
			OPTIONAL MATCH (s)-[:HAS_ACTION]->(a:Action)
			RETURN p, collect(DISTINCT s) AS stages, collect(DISTINCT a) AS actions
		`
		params := map[string]interface{}{
			"protocolId": protocolID,
		}
		records, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		if records.Next(ctx) {
			record := records.Record()
			protocolNode := record.Values[0].(neo4j.Node)

			protocol := &models.Protocol{
				ProtocolID:      protocolNode.Props["protocolId"].(string),
				Name:            protocolNode.Props["name"].(string),
				Type:            models.ProtocolType(protocolNode.Props["type"].(string)),
				GuidelineSource: protocolNode.Props["guidelineSource"].(string),
			}
			if active, ok := protocolNode.Props["active"].(bool); ok {
				protocol.Active = active
			}

			return protocol, nil
		}
		return nil, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get pathway graph: %w", err)
	}
	if result == nil {
		return nil, nil
	}
	return result.(*models.Protocol), nil
}

// ===== Patient Pathway Instance Operations =====

// CreatePatientPathwayInstance creates a patient's pathway instance in the graph
func (s *Neo4jService) CreatePatientPathwayInstance(ctx context.Context, instance *models.PathwayInstance) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MATCH (p:Protocol {protocolId: $pathwayId})
			MERGE (patient:Patient {patientId: $patientId})
			MERGE (instance:PathwayInstance {instanceId: $instanceId})
			SET instance.currentStage = $currentStage,
			    instance.status = $status,
			    instance.startedAt = datetime($startedAt)
			MERGE (patient)-[:HAS_PATHWAY]->(instance)
			MERGE (instance)-[:FOLLOWS]->(p)
			RETURN instance
		`
		params := map[string]interface{}{
			"instanceId":   instance.InstanceID,
			"pathwayId":    instance.PathwayID,
			"patientId":    instance.PatientID,
			"currentStage": instance.CurrentStage,
			"status":       string(instance.Status),
			"startedAt":    instance.StartedAt.Format("2006-01-02T15:04:05Z"),
		}
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})

	if err != nil {
		return fmt.Errorf("failed to create patient pathway instance: %w", err)
	}
	return nil
}

// GetPatientPathwayInstances retrieves all pathway instances for a patient
func (s *Neo4jService) GetPatientPathwayInstances(ctx context.Context, patientID string) ([]*models.PathwayInstance, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MATCH (patient:Patient {patientId: $patientId})-[:HAS_PATHWAY]->(instance:PathwayInstance)
			OPTIONAL MATCH (instance)-[:FOLLOWS]->(p:Protocol)
			RETURN instance, p.protocolId AS pathwayId
			ORDER BY instance.startedAt DESC
		`
		params := map[string]interface{}{
			"patientId": patientID,
		}
		records, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		var instances []*models.PathwayInstance
		for records.Next(ctx) {
			record := records.Record()
			node := record.Values[0].(neo4j.Node)
			pathwayID := record.Values[1].(string)

			instance := &models.PathwayInstance{
				InstanceID:   node.Props["instanceId"].(string),
				PathwayID:    pathwayID,
				PatientID:    patientID,
				CurrentStage: node.Props["currentStage"].(string),
				Status:       models.PathwayStatus(node.Props["status"].(string)),
			}
			instances = append(instances, instance)
		}
		return instances, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get patient pathway instances: %w", err)
	}
	return result.([]*models.PathwayInstance), nil
}

// ===== Analytics Queries =====

// GetProtocolComplianceStats returns compliance statistics for a protocol
func (s *Neo4jService) GetProtocolComplianceStats(ctx context.Context, protocolID string) (map[string]interface{}, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MATCH (p:Protocol {protocolId: $protocolId})<-[:FOLLOWS]-(instance:PathwayInstance)
			WITH p,
			     count(instance) AS totalInstances,
			     count(CASE WHEN instance.status = 'completed' THEN 1 END) AS completedInstances,
			     count(CASE WHEN instance.status = 'active' THEN 1 END) AS activeInstances
			RETURN {
				protocolId: p.protocolId,
				protocolName: p.name,
				totalInstances: totalInstances,
				completedInstances: completedInstances,
				activeInstances: activeInstances,
				completionRate: CASE WHEN totalInstances > 0
				                     THEN toFloat(completedInstances) / totalInstances
				                     ELSE 0.0 END
			} AS stats
		`
		params := map[string]interface{}{
			"protocolId": protocolID,
		}
		records, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		if records.Next(ctx) {
			record := records.Record()
			return record.Values[0].(map[string]interface{}), nil
		}
		return map[string]interface{}{}, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get protocol compliance stats: %w", err)
	}
	return result.(map[string]interface{}), nil
}

// FindRelatedGuidelines finds guidelines related to a given guideline
func (s *Neo4jService) FindRelatedGuidelines(ctx context.Context, guidelineID string) ([]models.Guideline, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MATCH (g1:Guideline {guidelineId: $guidelineId})
			MATCH (g2:Guideline)
			WHERE g1 <> g2
			  AND g2.active = true
			  AND (g1.domain = g2.domain OR
			       EXISTS((g1)-[:HAS_RECOMMENDATION]->(:Recommendation)-[:RELATED_TO]-(:Recommendation)<-[:HAS_RECOMMENDATION]-(g2)))
			RETURN g2
			LIMIT 10
		`
		params := map[string]interface{}{
			"guidelineId": guidelineID,
		}
		records, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		var guidelines []models.Guideline
		for records.Next(ctx) {
			record := records.Record()
			node := record.Values[0].(neo4j.Node)

			g := models.Guideline{
				GuidelineID: node.Props["guidelineId"].(string),
				Name:        node.Props["name"].(string),
			}
			if domain, ok := node.Props["domain"].(string); ok {
				g.Domain = domain
			}
			if source, ok := node.Props["source"].(string); ok {
				g.Source = source
			}
			guidelines = append(guidelines, g)
		}
		return guidelines, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to find related guidelines: %w", err)
	}
	return result.([]models.Guideline), nil
}

// GetConditionProtocols returns protocols applicable to a condition
func (s *Neo4jService) GetConditionProtocols(ctx context.Context, condition string) ([]models.Protocol, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MATCH (c:Condition {name: $condition})-[:TREATED_BY]->(p:Protocol)
			WHERE p.active = true
			RETURN p
			UNION
			MATCH (p:Protocol)
			WHERE p.active = true AND toLower(p.name) CONTAINS toLower($condition)
			RETURN p
		`
		params := map[string]interface{}{
			"condition": condition,
		}
		records, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		var protocols []models.Protocol
		for records.Next(ctx) {
			record := records.Record()
			node := record.Values[0].(neo4j.Node)

			p := models.Protocol{
				ProtocolID: node.Props["protocolId"].(string),
				Name:       node.Props["name"].(string),
			}
			if pType, ok := node.Props["type"].(string); ok {
				p.Type = models.ProtocolType(pType)
			}
			if source, ok := node.Props["guidelineSource"].(string); ok {
				p.GuidelineSource = source
			}
			protocols = append(protocols, p)
		}
		return protocols, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get condition protocols: %w", err)
	}
	return result.([]models.Protocol), nil
}
