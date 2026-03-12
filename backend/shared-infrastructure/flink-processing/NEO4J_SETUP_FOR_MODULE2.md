# Neo4j Setup for Module 2 - Quick Start Guide

## Current Neo4j Docker Status

Your Neo4j instance is running in Docker with these details:

**Container**: `neo4j` (cardiofit-neo4j)
**Image**: `neo4j:5.15-community`
**Ports**:
- Bolt: `localhost:55002` → `7687` (for connections)
- HTTP: `localhost:55001` → `7474` (web interface)
- HTTPS: `localhost:55000` → `7473`

---

## ⚠️ Password Change Required

The Neo4j instance is using the default password which **must be changed** before Module 2 can connect.

**Current Credentials**:
- Username: `neo4j`
- Password: `neo4j` (default, expired)

**Target Credentials** (configured in Module 2):
- Username: `neo4j`
- Password: `CardioFit2024!`

---

## 🔧 Setup Steps

### Step 1: Change Neo4j Password

You need to **manually** run this command in an interactive terminal:

```bash
docker exec -it neo4j cypher-shell -u neo4j -p neo4j
```

This will connect you to the Neo4j shell. You'll see a password change prompt. Then run:

```cypher
ALTER CURRENT USER SET PASSWORD FROM 'neo4j' TO 'CardioFit2024!';
```

Exit with `:exit`

### Step 2: Verify Connection

Test the new password works:

```bash
echo "RETURN 'Connection successful!' AS status;" | \
  docker exec -i neo4j cypher-shell -u neo4j -p 'CardioFit2024!'
```

Expected output:
```
+---------------------------+
| status                    |
+---------------------------+
| "Connection successful!"  |
+---------------------------+
```

### Step 3: Access Neo4j Browser (Optional)

Open the Neo4j browser interface to explore the database:

```bash
open http://localhost:55001
```

Login with:
- Connect URL: `bolt://localhost:55002`
- Username: `neo4j`
- Password: `CardioFit2024!`

---

## 📋 Module 2 Configuration

The following configuration has already been set in `KafkaConfigLoader.java`:

```java
// Neo4j Configuration (from docker-compose.hybrid-kafka.yml)
private static final String NEO4J_URI = "bolt://neo4j:7687"; // Internal Docker
private static final String NEO4J_EXTERNAL_URI = "bolt://localhost:55002"; // External (mapped port)
private static final String NEO4J_USERNAME = "neo4j";
private static final String NEO4J_PASSWORD = "CardioFit2024!"; // From docker-compose
```

**Environment Detection**:
- When running **locally** (from IDE): Uses `bolt://localhost:55002`
- When running **in Docker**: Uses `bolt://neo4j:7687`

---

## 🧪 Test Neo4j Connection from Java

Once the password is changed, you can test the connection:

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing

# Compile the project
mvn clean compile

# Run a simple Neo4j connection test (create this if needed)
```

---

## 🗄️ Initial Graph Schema for Module 2

After password setup, you can optionally create the initial schema for care networks:

```cypher
// Connect to Neo4j
docker exec -it neo4j cypher-shell -u neo4j -p 'CardioFit2024!'

// Create constraints
CREATE CONSTRAINT patient_id IF NOT EXISTS FOR (p:Patient) REQUIRE p.patientId IS UNIQUE;
CREATE CONSTRAINT provider_id IF NOT EXISTS FOR (pr:Provider) REQUIRE pr.providerId IS UNIQUE;
CREATE CONSTRAINT cohort_name IF NOT EXISTS FOR (c:Cohort) REQUIRE c.name IS UNIQUE;

// Create indexes
CREATE INDEX patient_name IF NOT EXISTS FOR (p:Patient) ON (p.lastName, p.firstName);
CREATE INDEX provider_specialty IF NOT EXISTS FOR (pr:Provider) ON (pr.specialty);

// Sample data (optional - for testing)
CREATE (p:Patient {
  patientId: 'P12345',
  firstName: 'John',
  lastName: 'Doe',
  dateOfBirth: '1980-01-15',
  mrn: 'MRN-12345'
});

CREATE (pr:Provider {
  providerId: 'DR001',
  firstName: 'Jane',
  lastName: 'Smith',
  specialty: 'Cardiology',
  npi: 'NPI-1234567890'
});

CREATE (c:Cohort {
  name: 'Hypertension',
  description: 'Patients with hypertension diagnosis'
});

// Create relationships
MATCH (p:Patient {patientId: 'P12345'})
MATCH (pr:Provider {providerId: 'DR001'})
CREATE (p)-[:HAS_PROVIDER {role: 'Primary Care', since: datetime()}]->(pr);

MATCH (p:Patient {patientId: 'P12345'})
MATCH (c:Cohort {name: 'Hypertension'})
CREATE (p)-[:IN_COHORT {since: datetime(), riskLevel: 'Medium'}]->(c);
```

---

## 🔍 Verify Schema Created

```cypher
// List all constraints
SHOW CONSTRAINTS;

// List all indexes
SHOW INDEXES;

// Count nodes
MATCH (n) RETURN labels(n) AS NodeType, count(n) AS Count;

// View sample relationships
MATCH (p:Patient)-[r]->(x)
RETURN p.patientId, type(r), labels(x), x.name LIMIT 10;
```

---

## ✅ Verification Checklist

Before running Module 2 tests, verify:

- [ ] Neo4j container is running: `docker ps | grep neo4j`
- [ ] Password changed to `CardioFit2024!`
- [ ] Can connect via cypher-shell: `docker exec -i neo4j cypher-shell -u neo4j -p 'CardioFit2024!' <<< "RETURN 1;"`
- [ ] Can access browser: `http://localhost:55001` with credentials
- [ ] Initial schema created (constraints and indexes)
- [ ] Sample patient data loaded (optional)

---

## 🚨 Troubleshooting

### Issue: "credentials expired"
**Solution**: Follow Step 1 to change password interactively

### Issue: "Connection refused on port 55002"
**Solution**: Check Neo4j is running: `docker ps | grep neo4j`

### Issue: "Authentication failed"
**Solution**: Verify password was changed correctly

### Issue: Neo4j web browser won't load
**Solution**: Try `http://localhost:55001` instead of default port 7474

---

## 📚 Module 2 Usage

Once Neo4j is configured, Module 2 will:

1. **Query care network data** when a first-time patient event arrives:
   ```java
   CompletableFuture<GraphData> neo4jFuture = neo4jClient.queryGraphAsync(patientId);
   ```

2. **Populate PatientSnapshot** with graph data:
   - Care team (providers)
   - Risk cohorts (patient belongs to)
   - Care pathways (active clinical pathways)
   - Related patients (family/care network)

3. **Update graph on encounter closure**:
   ```java
   neo4jClient.updateCareNetwork(snapshot);
   ```

4. **Graceful degradation**: If Neo4j is unavailable, Module 2 continues processing without graph data

---

## 🔗 Related Files

- **Configuration**: `flink-processing/src/main/java/com/cardiofit/flink/utils/KafkaConfigLoader.java`
- **Client**: `flink-processing/src/main/java/com/cardiofit/flink/clients/Neo4jGraphClient.java`
- **Model**: `flink-processing/src/main/java/com/cardiofit/flink/models/GraphData.java`
- **Docker Compose**: `backend/shared-infrastructure/docker-compose.hybrid-kafka.yml`

---

## Next Steps

After completing Neo4j setup:

1. ✅ Change password to `CardioFit2024!`
2. ✅ Verify connection from command line
3. ✅ Create initial schema (constraints/indexes)
4. ✅ Load sample test data
5. 🔄 Run Module 2 Phase 5 testing with Neo4j integration
