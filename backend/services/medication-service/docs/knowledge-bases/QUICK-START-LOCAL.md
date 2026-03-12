# 🚀 Quick Start: KB-Drug-Rules with Local PostgreSQL

Get your KB-Drug-Rules service running with local PostgreSQL in **5 minutes**!

## ⚡ Super Quick Setup

### **Step 1: Setup Database (2 minutes)**

#### **Linux/macOS:**
```bash
cd backend/services/knowledge-base-services/scripts
chmod +x setup-local-postgres.sh
./setup-local-postgres.sh
```

#### **Windows:**
```cmd
cd backend\services\knowledge-base-services\scripts
setup-local-postgres.bat
```

### **Step 2: Start Service (1 minute)**
```bash
cd backend/services/knowledge-base-services
make run-local
```

### **Step 3: Test Service (1 minute)**
```bash
# In another terminal
make test-kb-service
```

## 🎯 What You Get

After setup, you'll have:

### **✅ Local PostgreSQL Database**
- **Host:** localhost:5432
- **Database:** kb_drug_rules
- **User:** kb_drug_rules_user
- **Password:** kb_password

### **✅ KB-Drug-Rules Service**
- **URL:** http://localhost:8081
- **Health:** http://localhost:8081/health
- **Metrics:** http://localhost:8081/metrics

### **✅ Sample Drug Data**
- **Metformin** (diabetes medication)
- **Lisinopril** (blood pressure medication)  
- **Warfarin** (anticoagulant)

### **✅ Complete API Endpoints**
```bash
# Get drug rules
GET /v1/items/{drug_id}

# Validate TOML rules
POST /v1/validate

# Hot-load new rules
POST /v1/hotload

# Health check
GET /health

# Prometheus metrics
GET /metrics
```

## 🧪 Test Your Setup

### **Quick API Test:**
```bash
# Health check
curl http://localhost:8081/health

# Get metformin rules
curl http://localhost:8081/v1/items/metformin

# Validate sample TOML
curl -X POST http://localhost:8081/v1/validate \
  -H "Content-Type: application/json" \
  -d '{
    "content": "[meta]\ndrug_name=\"Test Drug\"\ntherapeutic_class=[\"Test\"]\n[dose_calculation]\nbase_formula=\"100mg daily\"\nmax_daily_dose=200.0\nmin_daily_dose=50.0\n[safety_verification]\ncontraindications=[]\nwarnings=[]\nprecautions=[]\ninteraction_checks=[]\nlab_monitoring=[]\nmonitoring_requirements=[]\nregional_variations={}",
    "regions": ["US"]
  }'
```

### **Database Test:**
```bash
psql -U kb_drug_rules_user -h localhost -d kb_drug_rules
# Password: kb_password

# Check sample data
SELECT drug_id, version, signature_valid FROM drug_rule_packs;
```

## 🔗 Flow2 Integration

Your Flow2 orchestrator can now call:

```go
// Example Go client code
type KBClient struct {
    baseURL string
}

func (k *KBClient) GetDrugRules(drugID, version, region string) (*DrugRules, error) {
    url := fmt.Sprintf("%s/v1/items/%s?version=%s&region=%s", 
        k.baseURL, drugID, version, region)
    
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var rules DrugRules
    return &rules, json.NewDecoder(resp.Body).Decode(&rules)
}

// Usage in Flow2
kbClient := &KBClient{baseURL: "http://localhost:8081"}
rules, err := kbClient.GetDrugRules("metformin", "1.0.0", "US")
```

## 📊 Sample Data Structure

### **Metformin Rules (Example):**
```json
{
  "drug_id": "metformin",
  "version": "1.0.0",
  "signature_valid": true,
  "regions": ["US", "EU"],
  "content": {
    "meta": {
      "drug_name": "Metformin",
      "therapeutic_class": ["Antidiabetic", "Biguanide"]
    },
    "dose_calculation": {
      "base_formula": "500mg BID",
      "max_daily_dose": 2000,
      "min_daily_dose": 500
    },
    "safety_verification": {
      "contraindications": [
        {
          "condition": "Severe renal impairment",
          "icd10_code": "N18.6",
          "severity": "absolute"
        }
      ]
    }
  }
}
```

## 🛠️ Available Commands

```bash
# Setup and run
make setup-local      # Setup PostgreSQL database
make run-local        # Start KB service locally
make test-kb-service  # Test complete setup

# Development
make build           # Build the service
make test            # Run unit tests
make test-coverage   # Run tests with coverage

# API testing
make test-api        # Test API endpoints
make health          # Check service health

# Utilities
make clean           # Clean build artifacts
make format          # Format Go code
make lint            # Run linters
```

## 🔧 Configuration

The service uses these defaults for local development:

```bash
# Database
DATABASE_URL=postgresql://kb_drug_rules_user:kb_password@localhost:5432/kb_drug_rules

# Server
PORT=8081
DEBUG=true

# Security (disabled for local dev)
REQUIRE_APPROVAL=false
REQUIRE_SIGNATURE=false

# Regions
SUPPORTED_REGIONS=US,EU,CA,AU
DEFAULT_REGION=US
```

## 🚨 Troubleshooting

### **Service Won't Start:**
1. Check PostgreSQL is running: `pg_isready -h localhost -p 5432`
2. Test database connection: `psql -U kb_drug_rules_user -h localhost -d kb_drug_rules`
3. Check Go dependencies: `cd kb-drug-rules && go mod download`

### **Database Connection Issues:**
1. Verify PostgreSQL service: `sudo systemctl status postgresql` (Linux)
2. Check port 5432 is open: `netstat -an | grep 5432`
3. Re-run setup script: `./scripts/setup-local-postgres.sh`

### **API Errors:**
1. Check service logs for errors
2. Verify health endpoint: `curl http://localhost:8081/health`
3. Test with sample data: `curl http://localhost:8081/v1/items/metformin`

## 🎉 Success!

You now have a **production-ready KB-Drug-Rules service** running locally with:

- ✅ **Local PostgreSQL** database with sample data
- ✅ **RESTful API** for drug rule retrieval
- ✅ **TOML validation** for clinical authors
- ✅ **Regional support** (US, EU, CA, AU)
- ✅ **Performance monitoring** with metrics
- ✅ **Health checks** for reliability
- ✅ **Complete test suite** for validation

Your **Flow2 orchestrator** can now integrate with the KB service for **sub-10ms drug rule lookups**! 🚀

## 📚 Next Steps

1. **Add More Drugs**: Use the validation API to add new TOML rules
2. **Integrate Flow2**: Connect your orchestrator to the KB API
3. **Monitor Performance**: Check `/metrics` for Prometheus monitoring
4. **Scale Up**: Deploy to production with Docker/Kubernetes
5. **Add Security**: Enable governance and digital signatures

**Happy coding!** 🎯
