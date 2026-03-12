# Clinical Recipe Book (CRB)

## 🎯 **Purpose**
Clinical algorithms and procedures that define how medications should be calculated, validated, and monitored.

## 📋 **Recipe Files**

### **vancomycin_renal.yaml**
- Kidney-adjusted vancomycin dosing
- Creatinine clearance calculations
- Trough level monitoring
- Nephrotoxicity prevention

### **warfarin_initiation.yaml**
- Anticoagulation initiation protocols
- INR-based dose adjustments
- Genetic factor considerations
- Bleeding risk assessments

### **acetaminophen_standard.yaml**
- Standard analgesic dosing
- Weight-based calculations
- Hepatotoxicity monitoring
- Maximum daily dose limits

### **insulin_sliding_scale.yaml**
- Blood glucose management
- Sliding scale protocols
- Hypoglycemia prevention
- Monitoring requirements

## 🔗 **Integration Points**
- **Rust Recipe Engine**: Primary consumer for calculations
- **ORB Engine**: References for recipe selection
- **Monitoring Engine**: Uses for surveillance protocols
- **Safety Engine**: Uses for validation rules

**These recipes contain the actual clinical intelligence that drives medication decisions.**
