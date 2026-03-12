# 2-Hop ORB-Driven Architecture

## Overview
The definitive 2-hop architecture that transforms generic medication requests into intelligent clinical decisions.

## Architecture Flow

### LOCAL DECISION (0ms - Sub-millisecond)
```
Medication Request → THE BRAIN (ORB) → Intent Manifest
```
- **Input**: Generic medication request
- **Process**: ORB rule evaluation (local, <1ms)
- **Output**: Intent Manifest with recipe_id + data_requirements + rationale

### NETWORK HOP 1: Context Service (Target: <500ms)
```
Intent Manifest → Context Planner → Context Service → Clinical Context
```
- **Input**: Intent Manifest with specific data requirements
- **Process**: Targeted data gathering (only what's needed)
- **Output**: Clinical context with exactly the required data

### NETWORK HOP 2: Rust Recipe Engine (Target: <200ms)
```
Recipe ID + Clinical Context → Rust Engine → Medication Proposal
```
- **Input**: Recipe ID + Clinical context
- **Process**: Pure calculation and clinical intelligence
- **Output**: Complete medication proposal with dosing, safety, monitoring

## Key Benefits

### 1. Intelligence First
- **OLD**: "Gather everything, then decide"
- **NEW**: "Decide first, then get exactly what's needed"

### 2. Minimal Network Traffic
- Only 2 network calls (vs 5-10 in parallel approach)
- Targeted data gathering (vs comprehensive context assembly)
- Recipe-specific execution (vs generic processing)

### 3. Sub-Second Performance
- Local ORB decision: <1ms
- Context fetch: <500ms
- Recipe execution: <200ms
- **Total target: <700ms**

### 4. Clinical Intelligence
- Every decision backed by clinical knowledge
- Evidence-based routing rules
- Safety-first approach

## Implementation Details

### Step 1: ORB Evaluation
```go
intentManifest, err := o.orb.ExecuteLocal(ctx, medicationRequest)
```

### Step 2: Context Planning & Fetch
```go
contextRequest := o.contextPlanner.PlanDataRequirements(intentManifest)
clinicalContext, err := o.contextServiceClient.FetchContext(ctx, contextRequest)
```

### Step 3: Recipe Execution
```go
recipeRequest := &models.RecipeExecutionRequest{
    RecipeID:        intentManifest.RecipeID,
    ClinicalContext: clinicalContext,
}
medicationProposal, err := o.rustRecipeClient.ExecuteRecipe(ctx, recipeRequest)
```

## Performance Characteristics

| Component | Target Time | Actual Implementation |
|-----------|-------------|----------------------|
| ORB Evaluation | <1ms | Sub-millisecond local |
| Context Fetch | <500ms | Single targeted call |
| Recipe Execution | <200ms | Rust engine |
| **Total** | **<700ms** | **2-hop architecture** |

## Error Handling

### ORB Evaluation Failure
- Fallback to unknown medication recipe
- Manual review required
- Full audit trail

### Context Service Failure
- Proceed with available data
- Log missing requirements
- Adjust confidence levels

### Recipe Engine Failure
- Return safety-first response
- Recommend manual review
- Preserve clinical context

## Monitoring & Metrics

### Performance Metrics
- ORB evaluation time
- Context fetch time
- Recipe execution time
- End-to-end latency
- Network hop count (always 2)

### Clinical Metrics
- Rule match rate
- Data completeness
- Safety alert rate
- Clinical accuracy

### System Metrics
- Cache hit rates
- Error rates by component
- Throughput (requests/second)

## Comparison: Old vs New

### Old Generic Approach
```
Request → [Parallel Context Assembly] → [Generic Processing] → Response
         ↓
    Patient Service
    Lab Service  
    Medication Service
    Allergy Service
    Condition Service
    (5-10 parallel calls)
```

### New ORB-Driven Approach
```
Request → ORB → Context Service → Rust Engine → Response
         <1ms      <500ms         <200ms
         LOCAL     HOP 1          HOP 2
```

## Clinical Intelligence Examples

### Example 1: Vancomycin + Kidney Disease
```
Input: "Vancomycin for sepsis, patient has CKD"
↓
ORB: Matches "vancomycin-renal-impairment" rule
↓
Context: Fetch only ["creatinine_clearance", "weight", "age", "dialysis_status"]
↓
Rust: Execute "vancomycin-renal-v2" recipe
↓
Output: Renal-adjusted dosing with monitoring plan
```

### Example 2: Warfarin + Genetic Testing
```
Input: "Warfarin for AFib, genetic testing available"
↓
ORB: Matches "warfarin-initiation-genetic" rule
↓
Context: Fetch ["age", "weight", "cyp2c9_genotype", "vkorc1_genotype"]
↓
Rust: Execute "warfarin-initiation-v2" recipe
↓
Output: Pharmacogenetic-guided dosing
```

## Future Enhancements

### Phase 2: Advanced Intelligence
- Machine learning rule optimization
- Outcome-based rule refinement
- Predictive context requirements

### Phase 3: Real-time Adaptation
- Dynamic rule updates
- A/B testing of clinical algorithms
- Continuous performance optimization

---

**This 2-hop architecture represents the definitive approach to clinical decision support: intelligent, fast, and clinically sound.**
