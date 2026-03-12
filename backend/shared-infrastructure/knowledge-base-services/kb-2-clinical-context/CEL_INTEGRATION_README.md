# CEL (Common Expression Language) Integration

This document describes the integration of Google CEL (Common Expression Language) into the KB-2 Clinical Context Go service for evaluating clinical phenotype expressions.

## Overview

The CEL integration provides a powerful, safe, and standardized way to evaluate complex clinical logic expressions for phenotype detection. This replaces the previous custom evaluation logic with a more flexible and maintainable solution.

## Key Features

- **Multiple Logic Engines**: Support for CEL, Rego, Python, and SQL engines (CEL implemented first)
- **Safe Expression Evaluation**: Built-in timeout and validation mechanisms
- **Clinical Context Variables**: Rich clinical data model exposed to expressions
- **Expression Caching**: Compiled expressions are cached for performance
- **YAML-based Configuration**: Phenotype definitions loaded from YAML files
- **Comprehensive Validation**: Expression validation before evaluation
- **Detailed Evidence**: Supporting evidence collection for phenotype matches

## Architecture

```
┌─────────────────────┐    ┌──────────────────────┐    ┌─────────────────────┐
│   Context Service   │───▶│   Phenotype Engine   │───▶│  Multi-Engine       │
│                     │    │                      │    │  Evaluator          │
└─────────────────────┘    └──────────────────────┘    └─────────────────────┘
                                      │                           │
                                      ▼                           ▼
                           ┌──────────────────────┐    ┌─────────────────────┐
                           │  Phenotype Loader    │    │    CEL Engine       │
                           │  (YAML)              │    │                     │
                           └──────────────────────┘    └─────────────────────┘
```

## Clinical Context Variables

The following variables are available in CEL expressions:

### Patient Demographics (`patient`)
```cel
patient.age >= 65
patient.sex == "M"
patient.has_diabetes
patient.has_ckd
patient.has_heart_failure
patient.has_atrial_fibrillation
```

### Blood Pressure (`bp`)
```cel
bp.systolic >= 140
bp.diastolic >= 90
```

### Laboratory Values (`labs`)
```cel
labs.total_cholesterol > 240
labs.hba1c >= 7.0
labs.creatinine > 1.5
labs.values["2093-3"] > 240  // Access by LOINC code
```

### Risk Scores (`risk`)
```cel
risk.ascvd_10yr >= 10
risk.cardiovascular >= 0.8
```

### Medications (`medications`)
```cel
"lisinopril" in medications.active_meds
medications.med_count > 5
medications.has_med["161"]  // Check by RxNorm code
```

### Conditions (`conditions`)
```cel
"diabetes" in conditions.active
conditions.has_condition["I10"]  // Check by ICD-10 code
conditions.condition_count > 3
```

### Vital Signs (`vitals`)
```cel
vitals.bmi >= 30
vitals.heart_rate > 100
```

## Example Phenotype Definitions

### Hypertension Stage 1 (High Risk)
```yaml
- id: "PHE-CV000001"
  name: "Hypertension Stage 1 High Risk"
  domain: "cardiovascular"
  status: "active"
  criteria:
    logic_engine: "cel"
    expression: |
      ((bp.systolic >= 130 && bp.systolic < 140) ||
       (bp.diastolic >= 80 && bp.diastolic < 90)) &&
      (risk.ascvd_10yr >= 10 || patient.has_diabetes || 
       patient.has_ckd || patient.age >= 65)
    data_requirements:
      - field: "bp.systolic"
        type: "integer"
        required: true
        time_window: "3m"
      - field: "bp.diastolic" 
        type: "integer"
        required: true
        time_window: "3m"
      - field: "patient.age"
        type: "integer"
        required: true
```

### Heart Failure with Reduced Ejection Fraction
```yaml
- id: "PHE-CV000003"
  name: "Heart Failure with Reduced Ejection Fraction"
  domain: "cardiovascular"
  status: "active"
  criteria:
    logic_engine: "cel"
    expression: |
      patient.has_heart_failure &&
      patient.lvef <= 40 &&
      (patient.bnp > 400 || patient.nt_probnp > 1800)
```

## API Endpoints

### Validate All Phenotypes
```http
GET /api/v1/phenotypes/validate
```
Validates all loaded phenotype expressions.

### Test Phenotype Expression
```http
POST /api/v1/phenotypes/test
Content-Type: application/json

{
  "expression": "bp.systolic >= 140 && patient.age >= 18",
  "logic_engine": "cel",
  "patient_data": {
    "patient_id": "test-001",
    "demographics": {
      "age_years": 72,
      "sex": "M"
    },
    "recent_labs": [...]
  }
}
```

### Get Engine Statistics
```http
GET /api/v1/phenotypes/engine/stats
```
Returns statistics about the phenotype engines.

### Reload Phenotypes
```http
POST /api/v1/phenotypes/reload
```
Reloads all phenotype definitions from YAML files.

## Configuration

### Environment Variables
```bash
# Phenotype definitions directory
PHENOTYPE_DIR=/app/phenotypes

# CEL engine settings
CEL_MAX_EVALUATION_TIME=5s
CEL_MAX_EXPRESSION_LENGTH=10000
CEL_ENABLE_DETAILED_LOGGING=false

# Cache settings
PHENOTYPE_CACHE_TTL=15m
```

### go.mod Dependencies
```go
require (
    github.com/google/cel-go v0.21.0
    google.golang.org/genproto/googleapis/api/expr/v1alpha1 v0.0.0-20240903143218-8af14fe29dc1
    gopkg.in/yaml.v3 v3.0.1
    go.uber.org/zap v1.27.0
)
```

## Usage Examples

### Basic Usage
```go
// Initialize CEL engine
celEngine, err := engines.NewCELEngine(logger)
if err != nil {
    return err
}

// Evaluate expression
matched, confidence, err := celEngine.EvaluateExpression(
    "bp.systolic >= 140 && patient.age >= 18",
    patientContext,
)
```

### Multi-Engine Usage
```go
// Initialize multi-engine evaluator
multiEngine, err := engines.NewMultiEngineEvaluator(logger)
if err != nil {
    return err
}

// Evaluate phenotype
result, err := multiEngine.EvaluatePhenotype(phenotypeDef, patientContext)
if err != nil {
    return err
}

if result.Matched {
    fmt.Printf("Phenotype matched with confidence: %f", result.Confidence)
}
```

## Testing

### Run Tests
```bash
cd tests
go test -v ./...
```

### Run Benchmarks
```bash
go test -bench=. -v
```

### Test Coverage
```bash
go test -cover ./...
```

## Performance Considerations

### Expression Caching
- Compiled CEL expressions are automatically cached
- Cache improves performance for repeated evaluations
- Use `ClearCache()` to reset if needed

### Timeout Management
- Default evaluation timeout: 5 seconds
- Configurable per engine instance
- Prevents infinite loops or slow expressions

### Memory Management
- CEL engine uses minimal memory overhead
- Patient context converted to structured data
- Evidence collection is lightweight

## Security Features

### Expression Validation
- Syntax validation before evaluation
- Type checking (expressions must return boolean)
- Maximum expression length limits

### Safe Evaluation
- Sandboxed expression execution
- No access to system functions
- No file system or network access
- Memory and CPU bounded

### Data Access Control
- Only predefined clinical variables exposed
- No access to sensitive system data
- Structured data model prevents injection

## Troubleshooting

### Common Issues

1. **Expression Compilation Errors**
   ```
   Error: CEL compilation error: undeclared reference to 'unknown_var'
   ```
   Solution: Use only defined clinical context variables.

2. **Type Errors**
   ```
   Error: expression must return boolean, got int
   ```
   Solution: Ensure expressions return boolean values.

3. **Timeout Errors**
   ```
   Error: expression evaluation timeout after 5s
   ```
   Solution: Simplify complex expressions or increase timeout.

### Debug Mode
Enable detailed logging for debugging:
```go
config := CELEngineConfig{
    EnableDetailedLogging: true,
}
```

### Validation
Always validate expressions before deployment:
```bash
curl -X GET http://localhost:8082/api/v1/phenotypes/validate
```

## Future Enhancements

### Planned Features
- **Rego Engine**: OPA policy evaluation
- **Python Engine**: Python expression evaluation
- **SQL Engine**: Database query-based evaluation
- **ML Model Integration**: Machine learning model calls
- **Custom Functions**: Domain-specific CEL functions

### Performance Improvements
- **Parallel Evaluation**: Multi-threaded expression evaluation
- **Pre-compilation**: Ahead-of-time expression compilation
- **Result Caching**: Cache evaluation results with TTL

## Related Documentation

- [Google CEL Documentation](https://github.com/google/cel-go)
- [CEL Language Definition](https://github.com/google/cel-spec)
- [Clinical Phenotype Standards](../phenotypes/README.md)
- [API Documentation](../api/README.md)

## Support

For issues related to CEL integration:
1. Check the troubleshooting section above
2. Review the test files for examples
3. Validate expressions using the API endpoints
4. Enable debug logging for detailed error information