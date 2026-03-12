# Clinical Terminology Change Request

## 🏥 Clinical Impact Assessment

### Change Type
- [ ] **Clinical Data Model** - Changes to FHIR resources or clinical data structures
- [ ] **Terminology Mapping** - Updates to SNOMED, RxNorm, LOINC, ICD-10, or AMT mappings
- [ ] **API Contract** - Changes to service APIs or GraphQL schemas
- [ ] **Business Logic** - Updates to clinical decision rules or workflows
- [ ] **Configuration** - Changes to clinical policies or system configuration
- [ ] **Documentation** - Updates to clinical specifications or user guides

### Safety Classification
- [ ] **Patient Safety Critical** - Directly affects patient care decisions (medication dosing, contraindications, allergies)
- [ ] **Clinical Quality** - Affects clinical data quality or reporting accuracy
- [ ] **System Integration** - Changes service interfaces or data exchange protocols
- [ ] **Performance/Operational** - Infrastructure or performance improvements
- [ ] **Documentation Only** - No functional changes

### Affected Clinical Domains
- [ ] **Medication Management** - Drug dosing, interactions, contraindications
- [ ] **Allergy/Adverse Reactions** - Allergy management and drug reactions
- [ ] **Laboratory Results** - LOINC codes and lab value interpretations
- [ ] **Diagnosis Coding** - ICD-10/SNOMED diagnostic terminologies
- [ ] **Clinical Guidelines** - Evidence-based care protocols
- [ ] **Patient Demographics** - Patient identification and demographics

## 📋 Change Details

### Summary
<!-- Provide a clear, concise description of what changes are being made and why -->

### Clinical Rationale
<!-- Explain the clinical reasoning behind these changes -->
- **Clinical Problem**:
- **Evidence Base**:
- **Expected Outcome**:

### Technical Implementation
<!-- Describe the technical approach and implementation details -->

### Testing Strategy
<!-- Describe how these changes have been tested -->
- [ ] Unit tests updated/added
- [ ] Integration tests cover new functionality
- [ ] Clinical scenarios tested
- [ ] Performance impact assessed
- [ ] Backward compatibility verified

## 🔍 Clinical Review Requirements

### Clinical Validation
- [ ] **Clinical Accuracy**: Terminology mappings are clinically accurate
- [ ] **Standards Compliance**: Adheres to relevant clinical standards (HL7 FHIR, WHO, FDA)
- [ ] **Safety Impact**: No negative impact on patient safety
- [ ] **Quality Metrics**: Maintains or improves clinical data quality

### Regulatory Compliance
- [ ] **HIPAA Compliance**: PHI handling meets privacy requirements
- [ ] **FDA Validation**: Medical device software requirements considered
- [ ] **Clinical Guidelines**: Aligns with established clinical guidelines
- [ ] **Audit Requirements**: Changes support clinical audit needs

## 📊 Impact Analysis

### System Dependencies
<!-- List systems, services, or components that may be affected -->

### Performance Impact
<!-- Describe any performance implications -->
- **Database Impact**:
- **API Response Times**:
- **Memory/CPU Usage**:

### Rollback Plan
<!-- Describe how to rollback these changes if needed -->

## 🧪 Testing Evidence

### Test Coverage
- **Unit Test Coverage**: __%
- **Integration Test Coverage**: __%
- **Clinical Scenario Coverage**: __%

### Test Results
<!-- Attach or link to test results -->

### Clinical Validation Results
<!-- Results from clinical team review -->

## 📚 Documentation

### Updated Documentation
- [ ] API documentation updated
- [ ] Clinical specifications updated
- [ ] User guides updated
- [ ] Database schema documentation updated

### Training Requirements
- [ ] Clinical staff training required
- [ ] Technical team training required
- [ ] End-user documentation updated

## ✅ Pre-Submission Checklist

### Technical Requirements
- [ ] Code follows established patterns and conventions
- [ ] All tests pass (unit, integration, clinical scenarios)
- [ ] Documentation is updated and accurate
- [ ] Performance benchmarks meet requirements (<800ms response time)
- [ ] Security scan passes with no critical issues

### Clinical Requirements
- [ ] Clinical SME has reviewed and approved changes
- [ ] Terminology mappings verified against authoritative sources
- [ ] Clinical test scenarios cover edge cases
- [ ] Patient safety impact assessment completed
- [ ] Regulatory compliance verified

### Process Requirements
- [ ] Branch is up to date with target branch
- [ ] Commit messages follow conventional format
- [ ] PR description is complete and accurate
- [ ] Appropriate labels applied
- [ ] Clinical reviewers assigned (if required)

## 🔗 References

### Clinical References
<!-- Links to clinical guidelines, literature, or standards -->

### Technical References
<!-- Links to technical specifications, RFCs, or documentation -->

### Related Issues/PRs
<!-- Links to related GitHub issues or pull requests -->

---

## 📋 For Clinical Reviewers

### Clinical Review Checklist
- [ ] **Clinical Accuracy**: Terminology and mappings are medically accurate
- [ ] **Patient Safety**: No risk to patient safety or care quality
- [ ] **Clinical Workflow**: Integration with clinical workflows is appropriate
- [ ] **Standards Compliance**: Meets relevant clinical and regulatory standards
- [ ] **Evidence Base**: Changes are supported by clinical evidence
- [ ] **Documentation**: Clinical documentation is complete and accurate

### Clinical Approval
**Clinical Reviewer**: ____________
**Review Date**: ____________
**Clinical Risk Assessment**: Low / Medium / High
**Approval Status**: Approved / Approved with Conditions / Rejected

**Clinical Comments**:
<!-- Clinical reviewer comments and recommendations -->

---

*This template ensures compliance with clinical governance requirements and patient safety standards.*