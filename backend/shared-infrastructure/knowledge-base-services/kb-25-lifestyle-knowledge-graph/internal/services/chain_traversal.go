package services

import (
	"context"
	"fmt"

	"kb-25-lifestyle-knowledge-graph/internal/graph"
	"kb-25-lifestyle-knowledge-graph/internal/models"

	"go.uber.org/zap"
)

type ChainTraversalService struct {
	graphClient graph.GraphClient
	logger      *zap.Logger
}

func NewChainTraversalService(client graph.GraphClient, logger *zap.Logger) *ChainTraversalService {
	return &ChainTraversalService{graphClient: client, logger: logger}
}

func (s *ChainTraversalService) GetChainsToTarget(ctx context.Context, targetCode string) ([]models.CausalChain, error) {
	records, err := s.graphClient.Run(ctx, graph.CypherGetAllChainsToTarget, map[string]any{
		"target_code": targetCode,
	})
	if err != nil {
		return nil, fmt.Errorf("chain traversal failed: %w", err)
	}

	var chains []models.CausalChain
	for _, record := range records {
		sourceCode, _ := record.Get("source_code")
		sourceType, _ := record.Get("source_type")
		edgeTypes, _ := record.Get("edge_types")
		effectSizes, _ := record.Get("effect_sizes")
		grades, _ := record.Get("grades")

		chain := models.CausalChain{
			Source:     fmt.Sprintf("%v", sourceCode),
			SourceType: fmt.Sprintf("%v", sourceType),
			Target:     targetCode,
		}

		etArr, _ := edgeTypes.([]interface{})
		esArr, _ := effectSizes.([]interface{})
		grArr, _ := grades.([]interface{})

		for i := range etArr {
			comp := models.ChainComponent{
				EdgeType: fmt.Sprintf("%v", etArr[i]),
			}
			if i < len(esArr) {
				if es, ok := esArr[i].(float64); ok {
					comp.Effect.EffectSize = es
				}
			}
			if i < len(grArr) {
				comp.Effect.EvidenceGrade = fmt.Sprintf("%v", grArr[i])
			}
			chain.Components = append(chain.Components, comp)
		}

		chain.PathLength = len(chain.Components)
		chain.NetEffect = ComputeNetEffect(chain.Components)
		chain.EvidenceGrade = chain.NetEffect.EvidenceGrade
		chains = append(chains, chain)
	}

	return chains, nil
}

func ComputeNetEffect(components []models.ChainComponent) models.EffectDescriptor {
	if len(components) == 0 {
		return models.EffectDescriptor{}
	}
	if len(components) == 1 {
		return components[0].Effect
	}

	netEffect := 1.0
	var grades []string
	var lastUnit string
	for _, c := range components {
		netEffect *= c.Effect.EffectSize
		grades = append(grades, c.Effect.EvidenceGrade)
		lastUnit = c.Effect.EffectUnit
	}

	return models.EffectDescriptor{
		EffectSize:    netEffect,
		EffectUnit:    lastUnit,
		EvidenceGrade: WeakestGrade(grades),
	}
}

func WeakestGrade(grades []string) string {
	gradeOrder := map[string]int{"A": 4, "B": 3, "C": 2, "D": 1}
	weakest := "A"
	weakestOrder := 4
	for _, g := range grades {
		if o, ok := gradeOrder[g]; ok && o < weakestOrder {
			weakest = g
			weakestOrder = o
		}
	}
	return weakest
}
