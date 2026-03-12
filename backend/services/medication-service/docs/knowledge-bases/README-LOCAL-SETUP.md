# 🗄️ Local PostgreSQL Setup for KB-Drug-Rules

This guide helps you set up a local PostgreSQL database for the KB-Drug-Rules service without Docker.

## 🚀 Quick Setup

### **Option 1: Automated Setup (Recommended)**

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

### **Option 2: Manual Setup**

#### **1. Install PostgreSQL**

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install postgresql postgresql-contrib
```

**CentOS/RHEL:**
```bash
sudo yum install postgresql-server postgresql-contrib
sudo postgresql-setup initdb
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

**macOS:**
```bash
brew install postgresql
brew services start postgresql
```

**Windows:**
- Download from: https://www.postgresql.org/download/windows/
- Or use Chocolatey: `choco install postgresql`

#### **2. Start PostgreSQL Service**

**Linux:**
```bash
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

**macOS:**
```bash
brew services start postgresql
```

**Windows:**
- Start PostgreSQL service from Services panel
- Or run: `net start postgresql-x64-14` (adjust version)

#### **3. Create Database and User**

```bash
# Connect as postgres user
sudo -u postgres psql

# Or on Windows/macOS
psql -U postgres
```

Then run:
```sql
-- Create database and user
CREATE DATABASE kb_drug_rules;
CREATE USER kb_drug_rules_user WITH PASSWORD 'kb_password';
GRANT ALL PRIVILEGES ON DATABASE kb_drug_rules TO kb_drug_rules_user;

-- Exit
\q
```

#### **4. Run Setup Script**

```bash
cd backend/services/knowledge-base-services/scripts
psql -U postgres -f setup-local-postgres.sql
```

## 🧪 Test the Setup

### **1. Test Database Connection**
```bash
psql -U kb_drug_rules_user -h localhost -d kb_drug_rules
# Password: kb_password
```

### **2. Check Sample Data**
```sql
SELECT drug_id, version, signature_valid FROM drug_rule_packs;
```

Expected output:
```
  drug_id   | version | signature_valid 
-----------+---------+-----------------
 metformin | 1.0.0   | t
 lisinopril| 1.0.0   | t
 warfarin  | 1.0.0   | t
```

### **3. Start the KB-Drug-Rules Service**

```bash
cd backend/services/knowledge-base-services/kb-drug-rules

# Set environment variables
export DATABASE_URL="postgresql://kb_drug_rules_user:kb_password@localhost:5432/kb_drug_rules"
export PORT=8081
export DEBUG=true

# Install dependencies
go mod download

# Run the service
go run cmd/server/main.go
```

### **4. Test the API**

```bash
# Health check
curl http://localhost:8081/health

# Get drug rules
curl http://localhost:8081/v1/items/metformin

# Validate TOML rules
curl -X POST http://localhost:8081/v1/validate \
  -H "Content-Type: application/json" \
  -d '{
    "content": "[meta]\ndrug_name=\"Test Drug\"\ntherapeutic_class=[\"Test\"]\n[dose_calculation]\nbase_formula=\"100mg daily\"\nmax_daily_dose=200.0\nmin_daily_dose=50.0\n[safety_verification]\ncontraindications=[]\nwarnings=[]\nprecautions=[]\ninteraction_checks=[]\nlab_monitoring=[]\nmonitoring_requirements=[]\nregional_variations={}",
    "regions": ["US"]
  }'
```

## 📊 Database Schema

The setup creates these tables:

### **drug_rule_packs**
- Stores TOML rules as JSONB
- Includes versioning and digital signatures
- Regional support with arrays

### **governance_approvals**
- Tracks clinical and technical approvals
- Dual approval workflow support

### **audit_log**
- Complete audit trail for compliance
- Tracks all changes with user context

### **drug_latest_versions**
- Tracks latest version for each drug
- Optimizes version lookups

## 🔧 Configuration

### **Environment Variables**
```bash
# Database
DATABASE_URL=postgresql://kb_drug_rules_user:kb_password@localhost:5432/kb_drug_rules

# Redis (install separately)
REDIS_URL=redis://localhost:6379/0

# Server
PORT=8081
DEBUG=true

# Security (disabled for local development)
REQUIRE_APPROVAL=false
REQUIRE_SIGNATURE=false
```

### **Connection Details**
- **Host:** localhost
- **Port:** 5432
- **Database:** kb_drug_rules
- **Username:** kb_drug_rules_user
- **Password:** kb_password

## 🛠️ Troubleshooting

### **PostgreSQL Service Not Running**
```bash
# Linux
sudo systemctl status postgresql
sudo systemctl start postgresql

# macOS
brew services list | grep postgresql
brew services start postgresql

# Windows
net start postgresql-x64-14
```

### **Connection Refused**
1. Check if PostgreSQL is listening on port 5432:
   ```bash
   netstat -an | grep 5432
   ```

2. Check PostgreSQL configuration:
   ```bash
   # Find config file
   sudo -u postgres psql -c "SHOW config_file;"
   
   # Edit postgresql.conf
   listen_addresses = 'localhost'
   port = 5432
   ```

3. Check pg_hba.conf for authentication:
   ```
   # Add this line for local connections
   host    all             all             127.0.0.1/32            md5
   ```

### **Permission Denied**
```bash
# Grant permissions
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE kb_drug_rules TO kb_drug_rules_user;"
sudo -u postgres psql -d kb_drug_rules -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO kb_drug_rules_user;"
```

### **Go Module Issues**
```bash
cd backend/services/knowledge-base-services/kb-drug-rules
go mod tidy
go mod download
```

## 🎯 Next Steps

1. **Install Redis** for caching:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install redis-server
   
   # macOS
   brew install redis
   
   # Windows
   choco install redis-64
   ```

2. **Test Flow2 Integration**:
   - The KB service is now ready at `http://localhost:8081`
   - Your Flow2 orchestrator can call the API endpoints
   - Sample data is pre-loaded for testing

3. **Add More Drugs**:
   - Use the `/v1/validate` endpoint to validate new TOML rules
   - Use the `/v1/hotload` endpoint to add new drug rules

4. **Monitor Performance**:
   - Check `/metrics` endpoint for Prometheus metrics
   - Use `/health` endpoint for health monitoring

## 🎉 Success!

Your local PostgreSQL database is now ready for the KB-Drug-Rules service! The service provides:

- ✅ **3 sample drugs** (metformin, lisinopril, warfarin)
- ✅ **Complete TOML rule structures**
- ✅ **Regional variations** (US, EU, CA)
- ✅ **Safety contraindications**
- ✅ **Dose calculation formulas**
- ✅ **API endpoints** for Flow2 integration

The database is optimized for your Flow2 orchestrator to retrieve drug rules with sub-10ms performance! 🚀
