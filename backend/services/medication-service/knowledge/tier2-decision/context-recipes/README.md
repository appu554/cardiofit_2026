# Context Service Recipe Book (CSRB)

## 🎯 **Purpose**
Defines exactly what clinical data is needed for each medication type, enabling optimized data gathering.

## 📋 **Files**

### **vancomycin_context.yaml**
- Required data for vancomycin dosing
- Renal function parameters
- Monitoring requirements
- Safety data needs

### **warfarin_context.yaml**
- Anticoagulation data requirements
- INR history and targets
- Bleeding risk factors
- Drug interaction checks

### **standard_context.yaml**
- Basic medication context
- Standard patient demographics
- Common safety parameters
- Default monitoring needs

## 🔗 **Integration with ORB**
1. **ORB generates Intent Manifest** with recipe_id
2. **Context Planner uses CSRB** to determine data_requirements
3. **Context Service fetches** exactly what's needed
4. **No wasted data gathering** - only relevant information

## 🎯 **Example Flow**
```
ORB: "Use vancomycin-renal recipe"
↓
CSRB: "vancomycin-renal needs: creatinine, weight, age"
↓
Context Service: "Fetch only those 3 data points"
↓
Result: Optimized, focused data gathering
```

**This enables the 2-hop architecture with precise data requirements.**
