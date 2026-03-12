# KB-Drug-Rules Microservice - TOML API Guide

## 🚀 **Overview**

The KB-Drug-Rules microservice provides complete TOML format support for drug rule management with automatic parsing, validation, format conversion, and enhanced database storage.

### **Key Features**
- ✅ **Complete TOML Workflow**: TOML parsing → validation → conversion → database storage
- ✅ **Enhanced Database Schema**: Stores both TOML and JSON formats with integrity verification
- ✅ **Real-time Processing**: Automatic TOML to JSON conversion
- ✅ **Version Management**: Complete audit trails and version history
- ✅ **Production-Ready**: Comprehensive error handling and logging

---

## 🌐 **Server Information**

- **Default Port**: `8081`
- **Base URL**: `http://localhost:8081`
- **Health Check**: `GET /health`
- **Readiness Check**: `GET /ready`

### **Start the Server**
```bash
cd backend/services/knowledge-base-services/kb-drug-rules
go run run_server.go
```

---

## 📋 **API Endpoints**

### **1. Complete TOML Workflow**
**Endpoint**: `POST /v1/toml/process`  
**Purpose**: Complete TOML workflow (parsing → validation → conversion → storage)

#### **Request Body**:
```json
{
  "drug_id": "metformin_example",
  "version": "1.0.0",
  "toml_content": "[meta]\ndrug_id = \"metformin_example\"\nname = \"Metformin Example\"\nversion = \"1.0.0\"\nclinical_reviewer = \"Dr. Example\"\ntherapeutic_class = \"Antidiabetic\"\n\n[dose_calculation]\nbase_dose_mg = 500.0\nmax_daily_dose_mg = 2000.0\ntitration_interval_days = 7\n\n[safety_verification]\ncontraindications = [\"Severe renal impairment\", \"Metabolic acidosis\"]\nmonitoring_requirements = [\"Renal function\", \"Vitamin B12 levels\"]\n\n[drug_interactions]\nmajor = [\"Contrast agents\", \"Alcohol\"]\nmoderate = [\"Furosemide\", \"Nifedipine\"]",
  "clinical_reviewer": "Dr. Example",
  "signed_by": "example_user",
  "regions": ["US", "EU"],
  "tags": ["example", "antidiabetic"],
  "notes": "Example drug rule"
}
```

#### **PowerShell Example**:
```powershell
$body = @{
    drug_id = "metformin_example"
    version = "1.0.0"
    toml_content = "[meta]`ndrug_id = `"metformin_example`"`nname = `"Metformin Example`"`nversion = `"1.0.0`"`nclinical_reviewer = `"Dr. Example`"`ntherapeutic_class = `"Antidiabetic`"`n`n[dose_calculation]`nbase_dose_mg = 500.0`nmax_daily_dose_mg = 2000.0`ntitration_interval_days = 7`n`n[safety_verification]`ncontraindications = [`"Severe renal impairment`", `"Metabolic acidosis`"]`nmonitoring_requirements = [`"Renal function`", `"Vitamin B12 levels`"]`n`n[drug_interactions]`nmajor = [`"Contrast agents`", `"Alcohol`"]`nmoderate = [`"Furosemide`", `"Nifedipine`"]"
    clinical_reviewer = "Dr. Example"
    signed_by = "example_user"
    regions = @("US", "EU")
    tags = @("example", "antidiabetic")
    notes = "Example drug rule"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8081/v1/toml/process" -Method POST -Body $body -ContentType "application/json"
```

#### **Response**:
```json
{
  "success": true,
  "drug_id": "metformin_example",
  "version": "1.0.0",
  "message": "TOML workflow completed successfully",
  "stored_id": "12345678-1234-5678-9012-123456789012",
  "json_content": "{ ... converted JSON structure ... }",
  "workflow_steps": [
    "✅ TOML parsing and validation",
    "✅ Format conversion (TOML → JSON)",
    "✅ Database storage with enhanced schema"
  ]
}
```

---

### **2. Retrieve Drug Rule (Full JSON)**
**Endpoint**: `GET /v1/items/:drug_id`  
**Purpose**: Get complete drug rule with converted JSON content

#### **PowerShell Example**:
```powershell
Invoke-RestMethod -Uri "http://localhost:8081/v1/items/metformin_example" -Method GET
```

#### **Response**:
```json
{
  "success": true,
  "drug_id": "metformin_example",
  "version": "1.0.0",
  "original_format": "toml",
  "clinical_reviewer": "Dr. Example",
  "content": "{ ... complete converted JSON structure ... }",
  "regions": ["US", "EU"],
  "tags": ["example", "antidiabetic"],
  "created_at": "2025-08-22T15:30:00Z",
  "updated_at": "2025-08-22T15:30:00Z"
}
```

---

### **3. Retrieve TOML Content**
**Endpoint**: `GET /v1/toml/rules/:drug_id`  
**Purpose**: Get original TOML content

#### **PowerShell Example**:
```powershell
Invoke-RestMethod -Uri "http://localhost:8081/v1/toml/rules/metformin_example" -Method GET
```

#### **Response**:
```json
{
  "success": true,
  "drug_id": "metformin_example",
  "version": "1.0.0",
  "original_format": "toml",
  "has_toml": true,
  "toml_content": "[meta]\ndrug_id = \"metformin_example\"\n...",
  "toml_length": 512,
  "created_at": "2025-08-22T15:30:00Z",
  "updated_at": "2025-08-22T15:30:00Z"
}
```

---

### **4. Retrieve JSON Structure**
**Endpoint**: `GET /v1/json/rules/:drug_id`  
**Purpose**: Get converted JSON structure

#### **PowerShell Example**:
```powershell
Invoke-RestMethod -Uri "http://localhost:8081/v1/json/rules/metformin_example" -Method GET
```

---

### **5. TOML Validation Only**
**Endpoint**: `POST /v1/toml/validate`  
**Purpose**: Validate TOML content without storing

#### **Request Body**:
```json
{
  "content": "[meta]\ndrug_id = \"test\"\nname = \"Test Drug\"\n\n[dose_calculation]\nbase_dose_mg = 500.0"
}
```

#### **PowerShell Example**:
```powershell
$validationBody = @{
    content = "[meta]`ndrug_id = `"test`"`nname = `"Test Drug`"`n`n[dose_calculation]`nbase_dose_mg = 500.0"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8081/v1/toml/validate" -Method POST -Body $validationBody -ContentType "application/json"
```

---

### **6. Format Conversion Only**
**Endpoint**: `POST /v1/toml/convert`  
**Purpose**: Convert TOML to JSON without storing

#### **Request Body**:
```json
{
  "toml_content": "[meta]\ndrug_id = \"conversion_test\"\nname = \"Conversion Test\""
}
```

#### **PowerShell Example**:
```powershell
$conversionBody = @{
    toml_content = "[meta]`ndrug_id = `"conversion_test`"`nname = `"Conversion Test`""
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8081/v1/toml/convert" -Method POST -Body $conversionBody -ContentType "application/json"
```

---

### **7. Service Statistics**
**Endpoint**: `GET /v1/stats`  
**Purpose**: Get service statistics including TOML rule counts

#### **PowerShell Example**:
```powershell
Invoke-RestMethod -Uri "http://localhost:8081/v1/stats" -Method GET
```

#### **Response**:
```json
{
  "success": true,
  "total_rules": 15,
  "toml_rules": 8,
  "json_rules": 7,
  "service": "kb-drug-rules",
  "version": "1.0.0-toml",
  "features": ["TOML workflow", "Enhanced database", "Version management"]
}
```

---

## 📊 **TOML Structure Example**

### **Complete Drug Rule TOML**:
```toml
[meta]
drug_id = "comprehensive_example"
name = "Comprehensive Drug Example"
version = "2.0.0"
clinical_reviewer = "Dr. Comprehensive"
therapeutic_class = "Example Class"
evidence_grade = "A"

[indications]
primary = "Primary indication"
secondary = ["Secondary indication 1", "Secondary indication 2"]

[dose_calculation]
base_dose_mg = 500.0
max_daily_dose_mg = 2000.0
min_dose_mg = 250.0
titration_interval_days = 7
administration_frequency = "twice_daily"

[dose_calculation.renal_adjustment]
egfr_30_45 = "reduce_by_50_percent"
egfr_below_30 = "contraindicated"

[safety_verification]
contraindications = [
    "Severe renal impairment",
    "Metabolic acidosis",
    "Severe hepatic impairment"
]
warnings = [
    "Risk of lactic acidosis",
    "Vitamin B12 deficiency with long-term use"
]
precautions = [
    "Monitor renal function regularly",
    "Assess for signs of lactic acidosis"
]

[monitoring_requirements]
baseline = ["Complete blood count", "Comprehensive metabolic panel", "HbA1c"]
ongoing = ["HbA1c every 3-6 months", "Renal function every 6-12 months"]

[drug_interactions]
major = ["Contrast agents", "Alcohol", "Cimetidine"]
moderate = ["Furosemide", "Nifedipine", "Vancomycin"]
minor = ["Thiazide diuretics", "Corticosteroids"]

[adverse_effects]
common = ["Gastrointestinal upset", "Metallic taste", "Decreased appetite"]
serious = ["Lactic acidosis", "Vitamin B12 deficiency"]

[clinical_protocols]
initiation = "Start with 500mg once daily with evening meal"
titration_schedule = [
    { week = 1, dose_mg = 500, frequency = "once_daily" },
    { week = 2, dose_mg = 500, frequency = "twice_daily" }
]
```

---

## 🔄 **Complete Workflow Example**

### **1. Submit TOML Drug Rule**
```powershell
# Submit a comprehensive drug rule
$comprehensiveRule = @{
    drug_id = "workflow_example"
    version = "1.0.0"
    toml_content = "... (TOML content from above) ..."
    clinical_reviewer = "Dr. Workflow"
    signed_by = "workflow_user"
    regions = @("US", "EU", "CA")
    tags = @("workflow", "example", "comprehensive")
    notes = "Complete workflow demonstration"
} | ConvertTo-Json

$result = Invoke-RestMethod -Uri "http://localhost:8081/v1/toml/process" -Method POST -Body $comprehensiveRule -ContentType "application/json"
Write-Host "✅ Rule stored with ID: $($result.stored_id)"
```

### **2. Retrieve and Verify**
```powershell
# Get the converted JSON
$jsonRule = Invoke-RestMethod -Uri "http://localhost:8081/v1/items/workflow_example" -Method GET
Write-Host "📄 JSON Content Length: $($jsonRule.content.Length) characters"

# Get the original TOML
$tomlRule = Invoke-RestMethod -Uri "http://localhost:8081/v1/toml/rules/workflow_example" -Method GET
Write-Host "📄 TOML Content Length: $($tomlRule.toml_length) characters"

# Check service stats
$stats = Invoke-RestMethod -Uri "http://localhost:8081/v1/stats" -Method GET
Write-Host "📊 Total Rules: $($stats.total_rules), TOML Rules: $($stats.toml_rules)"
```

---

## 🎯 **Key Benefits**

1. **Automatic Processing**: TOML → JSON conversion happens automatically
2. **Dual Storage**: Both TOML and JSON formats preserved
3. **Content Integrity**: SHA256 hashing for verification
4. **Audit Trail**: Complete tracking with clinical reviewer signatures
5. **Version Management**: Enhanced database schema with version history
6. **Production Ready**: Comprehensive error handling and logging

---

## 🚀 **Getting Started**

1. **Start the server**: `go run run_server.go`
2. **Check health**: `curl http://localhost:8081/health`
3. **Submit your first TOML rule** using the examples above
4. **Retrieve and verify** the stored content

Your KB-Drug-Rules microservice is ready for production use with complete TOML format support! 🎉
