# Enhanced Scoring Engine Integration Guide

## 🎯 **Overview**

This guide demonstrates how to integrate and use the Enhanced Scoring Engine that combines the best features of both the Compare-and-Rank model and the comprehensive scoring engine from the provided `scoring_ranking_engine.go` file.

## 🏗️ **Architecture Integration**

### **Dual-Mode Scoring System**

The Enhanced Scoring Engine provides two operational modes:

1. **Enhanced Mode**: Uses the sophisticated Compare-and-Rank engine with phenotype-aware weights
2. **Traditional Mode**: Falls back to comprehensive multi-factor scoring with external data enrichment

```go
// Enhanced Mode (Preferred)
if e.config.Features.UseCompareAndRank && e.compareAndRankEngine != nil {
    return e.scoreWithCompareAndRank(ctx, proposals, patientContext, indication)
}

// Traditional Mode (Fallback)
return e.scoreWithTraditionalMethod(ctx, proposals, patientContext, indication)
```

## 📋 **Implementation Steps**

### **Step 1: Initialize the Enhanced Scoring Engine**

```go
// Create compare-and-rank engine
configPath := "internal/scoring/kb_config.yaml"
compareAndRankEngine := scoring.NewCompareAndRankEngine(configPath, logger)

// Create external service clients
efficacyService := NewEfficacyDataService()
costService := NewCostDataService()
historyService := NewPatientHistoryService()

// Configure enhanced scoring
config := &scoring.ScoringConfig{
    Weights: scoring.ScoringWeights{
        Safety:                 0.30,
        Efficacy:               0.25,
        Tolerability:           0.15,
        Convenience:            0.10,
        Cost:                   0.10,
        PatientPreference:      0.05,
        GuidelineAdherence:     0.03,
        DrugInteractionProfile: 0.02,
    },
    Features: scoring.ScoringFeatures{
        UseCompareAndRank:      true,  // Enable enhanced mode
        UsePatientHistory:      true,
        UsePopulationOutcomes:  true,
        UsePharmacogenomics:    true,
    },
}

// Create enhanced scoring engine
enhancedEngine := scoring.NewEnhancedScoringEngine(
    compareAndRankEngine,
    efficacyService,
    costService,
    historyService,
    config,
    logger,
)
```

### **Step 2: Integrate with Flow2 Orchestrator**

```go
// In orchestrator.go
type Orchestrator struct {
    // ... existing fields ...
    enhancedScoringEngine *scoring.EnhancedScoringEngine
}

// Update performMultiFactorScoring method
func (o *Orchestrator) performMultiFactorScoring(
    safetyVerified []*models.SafetyVerifiedProposal,
    clinicalContext *models.ClinicalContext,
    requestID string,
) []*models.EnhancedScoredProposal {
    
    // Use enhanced scoring engine
    scored, err := o.enhancedScoringEngine.ScoreAndRankProposals(
        context.Background(),
        safetyVerified,
        clinicalContext,
        "diabetes_type2", // indication
    )
    
    if err != nil {
        o.logger.WithError(err).Error("Enhanced scoring failed")
        return []*models.EnhancedScoredProposal{}
    }
    
    return scored
}
```

### **Step 3: Configure External Data Services**

#### **Efficacy Data Service Implementation**

```go
type EfficacyDataServiceImpl struct {
    clinicalTrialsDB *ClinicalTrialsDB
    metaAnalysisDB   *MetaAnalysisDB
    cache            *EfficacyCache
}

func (s *EfficacyDataServiceImpl) GetEfficacyData(
    ctx context.Context, 
    drugID string, 
    indication string,
) (*scoring.EfficacyData, error) {
    // Query clinical trials database
    trials, err := s.clinicalTrialsDB.GetTrialsByDrug(ctx, drugID, indication)
    if err != nil {
        return nil, err
    }
    
    // Query meta-analyses
    metaAnalyses, err := s.metaAnalysisDB.GetMetaAnalyses(ctx, drugID, indication)
    if err != nil {
        return nil, err
    }
    
    // Calculate composite efficacy score
    efficacyScore := s.calculateCompositeEfficacyScore(trials, metaAnalyses)
    
    return &scoring.EfficacyData{
        DrugID:         drugID,
        Indication:     indication,
        EfficacyScore:  efficacyScore,
        ClinicalTrials: convertTrials(trials),
        MetaAnalyses:   convertMetaAnalyses(metaAnalyses),
        TimeToEffect:   s.estimateTimeToEffect(trials),
    }, nil
}
```

#### **Cost Data Service Implementation**

```go
type CostDataServiceImpl struct {
    formularyDB     *FormularyDB
    pricingService  *PricingService
    insuranceDB     *InsuranceDB
}

func (s *CostDataServiceImpl) GetMedicationCost(
    ctx context.Context,
    drugID string,
    formularyID string,
) (*scoring.CostData, error) {
    // Get formulary information
    formularyInfo, err := s.formularyDB.GetFormularyInfo(ctx, drugID, formularyID)
    if err != nil {
        return nil, err
    }
    
    // Get current pricing
    pricing, err := s.pricingService.GetCurrentPricing(ctx, drugID)
    if err != nil {
        return nil, err
    }
    
    return &scoring.CostData{
        DrugID:               drugID,
        AWPPerMonth:          pricing.AWP * 30, // Assuming daily dosing
        PatientCopayPerMonth: formularyInfo.Copay,
        FormularyTier:        formularyInfo.Tier,
        GenericAvailable:     pricing.GenericAvailable,
        PatientAssistance:    formularyInfo.PatientAssistanceAvailable,
    }, nil
}
```

## 🔄 **Data Flow**

### **Enhanced Mode Flow**

1. **Input**: Safety-verified proposals + Clinical context
2. **Enrichment**: External data services provide efficacy, cost, and history data
3. **Conversion**: Create enhanced proposals with comprehensive data
4. **Risk Assessment**: Extract patient risk phenotype (ASCVD, HF, CKD, NONE)
5. **Compare-and-Rank**: Apply phenotype-aware weights and sophisticated ranking
6. **Output**: Ranked proposals with detailed explainability

### **Traditional Mode Flow**

1. **Input**: Safety-verified proposals + Clinical context
2. **Multi-Factor Scoring**: Apply comprehensive scoring across 8 dimensions
3. **Evidence Integration**: Include clinical trials, guidelines, and real-world evidence
4. **Ranking**: Sort by weighted total score with tie-breakers
5. **Output**: Scored proposals with clinical recommendations

## 📊 **Configuration Examples**

### **ASCVD Patient Configuration**

```yaml
# For patients with cardiovascular disease
patient_profile: "ASCVD"
weights:
  efficacy: 0.38      # Prioritize efficacy for CV outcomes
  safety: 0.22        # Important safety considerations
  cost: 0.08          # Lower cost priority for high-risk
features:
  use_compare_and_rank: true
  use_cv_outcome_data: true
  prefer_guideline_recommended: true
```

### **Budget-Conscious Configuration**

```yaml
# For cost-sensitive patients
patient_profile: "BUDGET_MODE"
weights:
  cost: 0.22          # High cost consideration
  availability: 0.16  # Formulary status important
  efficacy: 0.28      # Still maintain efficacy focus
features:
  prefer_generic: true
  max_acceptable_cost: 200.0
  cost_sensitivity: 0.8
```

## 🧪 **Testing and Validation**

### **Unit Tests**

```go
func TestEnhancedScoringEngine_ASCVD_Patient(t *testing.T) {
    // Setup
    engine := setupEnhancedScoringEngine()
    proposals := createASCVDTestProposals()
    context := createASCVDPatientContext()
    
    // Execute
    scored, err := engine.ScoreAndRankProposals(
        context.Background(),
        proposals,
        context,
        "diabetes_type2",
    )
    
    // Validate
    require.NoError(t, err)
    assert.NotEmpty(t, scored)
    
    // Top recommendation should have CV benefit
    topRanked := scored[0]
    assert.True(t, topRanked.SubScores.Efficacy.CVBenefit)
    assert.Greater(t, topRanked.FinalScore, 0.7)
}
```

### **Integration Tests**

```go
func TestEnhancedScoringEngine_WithExternalServices(t *testing.T) {
    // Test with real external service calls
    engine := setupEnhancedScoringEngineWithRealServices()
    
    // Test data enrichment
    proposals := createRealWorldProposals()
    scored, err := engine.ScoreAndRankProposals(
        context.Background(),
        proposals,
        createPatientContext(),
        "diabetes_type2",
    )
    
    // Validate external data integration
    require.NoError(t, err)
    for _, proposal := range scored {
        assert.NotEmpty(t, proposal.SubScores.Efficacy.EvidenceLevel)
        assert.Greater(t, proposal.SubScores.Cost.MonthlyEstimate, 0.0)
    }
}
```

## 📈 **Performance Monitoring**

### **Key Metrics**

```go
// Performance metrics to track
type ScoringMetrics struct {
    ResponseTime        time.Duration
    ExternalServiceCalls int
    CacheHitRate        float64
    ScoringAccuracy     float64
    ClinicalAppropriateness float64
}

// Monitoring implementation
func (e *EnhancedScoringEngine) recordMetrics(
    startTime time.Time,
    proposalCount int,
    method string,
) {
    metrics := ScoringMetrics{
        ResponseTime: time.Since(startTime),
        ExternalServiceCalls: e.getServiceCallCount(),
        CacheHitRate: e.getCacheHitRate(),
    }
    
    e.metricsCollector.Record("enhanced_scoring", metrics)
}
```

## 🚀 **Deployment Considerations**

### **Configuration Management**

1. **Environment-Specific Configs**: Different weights for dev/staging/prod
2. **Feature Flags**: Enable/disable enhanced features per environment
3. **Service Discovery**: Dynamic configuration of external service endpoints
4. **Circuit Breakers**: Fallback mechanisms for external service failures

### **Monitoring and Alerting**

1. **Response Time**: Alert if scoring takes >500ms
2. **Error Rate**: Alert if error rate >1%
3. **External Service Health**: Monitor dependency availability
4. **Clinical Accuracy**: Track clinician feedback and outcomes

### **Scaling Considerations**

1. **Caching Strategy**: Redis for efficacy and cost data caching
2. **Async Processing**: Queue-based processing for non-urgent requests
3. **Load Balancing**: Distribute scoring load across multiple instances
4. **Database Optimization**: Optimize queries for external data services

## 🎯 **Benefits of Enhanced Integration**

### **Clinical Benefits**

- **Personalized Recommendations**: Risk phenotype-aware scoring
- **Evidence-Based Decisions**: Integration of latest clinical data
- **Transparent Rationale**: Full explainability for clinical review
- **Consistent Quality**: Standardized decision-making process

### **Technical Benefits**

- **Flexible Architecture**: Dual-mode operation with graceful fallbacks
- **Extensible Design**: Easy to add new scoring dimensions
- **Performance Optimized**: Caching and parallel processing
- **Production Ready**: Comprehensive error handling and monitoring

### **Operational Benefits**

- **Configuration Driven**: Clinical governance can adjust without code changes
- **Audit Compliant**: Complete decision trail for regulatory requirements
- **Scalable**: Handles high-volume medication recommendation requests
- **Maintainable**: Clean separation of concerns and modular design

This enhanced integration provides a robust, scalable, and clinically intelligent medication scoring system that combines the best of both sophisticated algorithmic ranking and comprehensive clinical evidence integration.
