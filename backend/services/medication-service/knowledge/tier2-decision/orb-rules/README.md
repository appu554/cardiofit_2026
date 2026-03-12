# Orchestrator Rule Base (ORB) - THE BRAIN

## 🧠 **Purpose**
The **most critical component** of the entire system. The ORB makes intelligent routing decisions that transform generic medication requests into specific clinical recipes.

## 📋 **Files**

### **medication_routing_rules.yaml**
- Core routing logic for all medications
- Condition-based recipe selection
- Priority-based rule evaluation
- Intent Manifest generation rules

### **priority_matrix.yaml**
- Rule precedence and conflict resolution
- Emergency vs routine prioritization
- Patient-specific overrides
- Clinical urgency factors

### **exception_handlers.yaml**
- Edge case handling
- Fallback routing logic
- Error recovery procedures
- Unknown medication protocols

## 🎯 **How ORB Works**

### **Input**: Medication Request
```json
{
  "patient_id": "123",
  "medication_code": "vancomycin",
  "patient_conditions": ["chronic_kidney_disease"],
  "indication": "sepsis"
}
```

### **ORB Processing**: Rule Evaluation
1. **Match medication**: vancomycin
2. **Check conditions**: chronic_kidney_disease detected
3. **Apply rule**: vancomycin + renal_impairment → vancomycin-renal recipe
4. **Generate Intent Manifest**

### **Output**: Intent Manifest
```json
{
  "recipe_id": "vancomycin-renal-v2",
  "data_requirements": ["creatinine_clearance", "weight", "age"],
  "priority": "high",
  "rationale": "Renal adjustment required for vancomycin"
}
```

## 🚨 **Critical Importance**
- **Without ORB**: System is generic, unintelligent
- **With ORB**: System makes expert clinical decisions
- **ORB failure**: Entire architecture becomes meaningless
- **ORB success**: Transforms healthcare delivery

**The ORB is the difference between a calculator and clinical intelligence.**
