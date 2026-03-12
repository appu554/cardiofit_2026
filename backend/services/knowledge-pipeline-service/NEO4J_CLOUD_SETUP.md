# 🌐 Neo4j Cloud (AuraDB) Setup Guide

## 📋 **Overview**

Neo4j Cloud (AuraDB) provides a fully managed Neo4j database service that's perfect for the Clinical Knowledge Graph. This guide will help you set up and configure Neo4j Cloud for the knowledge pipeline.

## 🚀 **Benefits of Neo4j Cloud**

- **✅ Fully Managed**: No infrastructure management required
- **✅ Auto-Scaling**: Automatically scales based on demand
- **✅ High Availability**: Built-in redundancy and backup
- **✅ Enterprise Security**: Encryption, authentication, authorization
- **✅ Performance**: Optimized for graph queries and traversals
- **✅ Global Deployment**: Available in multiple regions

## 🔧 **Step 1: Create Neo4j Cloud Account**

1. **Visit Neo4j Cloud**: Go to [https://neo4j.com/cloud/aura/](https://neo4j.com/cloud/aura/)

2. **Sign Up**: Create a free account or sign in with existing credentials

3. **Choose Plan**:
   - **AuraDB Free**: Perfect for development and testing (up to 200k nodes)
   - **AuraDB Professional**: For production workloads with scaling
   - **AuraDB Enterprise**: For enterprise features and support

## 🏗️ **Step 2: Create Database Instance**

1. **Create New Instance**:
   - Click "Create Database"
   - Choose your preferred region (closest to your location)
   - Select instance size based on your data volume

2. **Database Configuration**:
   - **Database Name**: `clinical-knowledge-graph`
   - **Region**: Choose closest to your location
   - **Instance Size**: Start with smallest, can scale up later

3. **Security Settings**:
   - Set a strong password for the `neo4j` user
   - **IMPORTANT**: Save the password securely - you'll need it for configuration

4. **Network Access**:
   - Configure IP whitelist (or allow all IPs for development)
   - Note the connection URI (format: `neo4j+s://xxxxx.databases.neo4j.io`)

## ⚙️ **Step 3: Configure Knowledge Pipeline**

### Environment Variables

Create a `.env` file in the knowledge pipeline service directory:

```bash
# Database Configuration
DATABASE_TYPE=neo4j

# Neo4j Cloud Configuration
NEO4J_URI=neo4j+s://your-instance-id.databases.neo4j.io
NEO4J_USERNAME=neo4j
NEO4J_PASSWORD=your-secure-password
NEO4J_DATABASE=neo4j
```

### Configuration File

Update your `src/core/config.py` or set environment variables:

```python
# Database Type Selection
DATABASE_TYPE = "neo4j"  # Use Neo4j Cloud instead of GraphDB

# Neo4j Cloud Configuration
NEO4J_URI = "neo4j+s://your-instance-id.databases.neo4j.io"
NEO4J_USERNAME = "neo4j"
NEO4J_PASSWORD = "your-secure-password"
NEO4J_DATABASE = "neo4j"
```

## 🧪 **Step 4: Test Connection**

### Install Neo4j Driver

```bash
pip install neo4j
```

### Test Connection Script

```python
# test_neo4j_connection.py
import asyncio
from core.database_factory import validate_database_connection

async def test_connection():
    result = await validate_database_connection()
    print("Connection Status:", result["status"])
    print("Database Info:", result["database_info"])
    if "stats" in result:
        print("Database Stats:", result["stats"])

if __name__ == "__main__":
    asyncio.run(test_connection())
```

Run the test:
```bash
python test_neo4j_connection.py
```

## 🔄 **Step 5: Run Knowledge Pipeline with Neo4j Cloud**

### Validate Configuration

```bash
python validate_data_sources.py
```

### Run Pipeline

```bash
python start_pipeline.py --sources rxnorm snomed loinc
```

The pipeline will now use Neo4j Cloud instead of local GraphDB!

## 📊 **Step 6: Monitor and Query**

### Neo4j Browser

1. **Access Browser**: Go to your AuraDB instance in the Neo4j Cloud console
2. **Click "Open"**: Opens Neo4j Browser for your instance
3. **Run Queries**: Use Cypher queries to explore your clinical knowledge graph

### Example Queries

```cypher
// Count total nodes
MATCH (n) RETURN count(n) as total_nodes

// Count nodes by type
MATCH (n) RETURN labels(n) as node_type, count(n) as count ORDER BY count DESC

// Find drug interactions
MATCH (d1:Drug)-[r:INTERACTS_WITH]->(d2:Drug)
RETURN d1.name, r.severity, d2.name
LIMIT 10

// Find SNOMED concepts related to cardiovascular
MATCH (s:SNOMEDConcept)
WHERE s.fullySpecifiedName CONTAINS "cardiovascular"
RETURN s.conceptId, s.fullySpecifiedName
LIMIT 10

// Find LOINC codes for laboratory tests
MATCH (l:LOINCCode)
WHERE l.component CONTAINS "glucose"
RETURN l.loincNumber, l.longCommonName, l.component
LIMIT 10
```

## 🔧 **Performance Optimization**

### Indexes

The pipeline automatically creates indexes for:
- Drug RXCUIs and names
- SNOMED concept IDs and names
- LOINC codes and names
- Clinical entity codes

### Query Optimization

- Use `PROFILE` to analyze query performance
- Create additional indexes for frequently queried properties
- Use `EXPLAIN` to understand query execution plans

## 🔒 **Security Best Practices**

1. **Strong Passwords**: Use complex passwords for database access
2. **IP Whitelisting**: Restrict access to known IP addresses
3. **Regular Backups**: Neo4j Cloud provides automatic backups
4. **Monitor Access**: Review access logs regularly
5. **Rotate Credentials**: Change passwords periodically

## 💰 **Cost Management**

### Free Tier Limits
- **Nodes**: Up to 200,000 nodes
- **Relationships**: Up to 400,000 relationships
- **Storage**: Up to 1GB
- **Queries**: Unlimited

### Scaling Considerations
- Monitor node/relationship counts
- Upgrade to Professional when approaching limits
- Consider data archiving strategies for large datasets

## 🚨 **Troubleshooting**

### Common Issues

1. **Connection Timeout**:
   - Check network connectivity
   - Verify IP whitelist settings
   - Confirm URI format is correct

2. **Authentication Failed**:
   - Verify username/password
   - Check for special characters in password
   - Ensure credentials are properly encoded

3. **Query Performance**:
   - Add appropriate indexes
   - Optimize Cypher queries
   - Consider query result limits

4. **Memory Issues**:
   - Upgrade instance size
   - Optimize data model
   - Use batch processing for large operations

### Support Resources

- **Neo4j Documentation**: [https://neo4j.com/docs/](https://neo4j.com/docs/)
- **Community Forum**: [https://community.neo4j.com/](https://community.neo4j.com/)
- **AuraDB Support**: Available through Neo4j Cloud console

## 🎯 **Next Steps**

1. **✅ Set up Neo4j Cloud instance**
2. **✅ Configure knowledge pipeline**
3. **✅ Test connection**
4. **✅ Run pipeline with your clinical data**
5. **✅ Explore data with Neo4j Browser**
6. **✅ Integrate with CAE system**

## 📈 **Expected Results**

After running the pipeline with Neo4j Cloud, you'll have:

- **🏥 Clinical Knowledge Graph**: Comprehensive graph with drugs, conditions, observations
- **🔗 Rich Relationships**: Drug interactions, SNOMED hierarchies, LOINC mappings
- **⚡ Fast Queries**: Optimized graph traversals for clinical decision support
- **📊 Analytics Ready**: Graph algorithms for pattern discovery and insights
- **🌐 Cloud Scale**: Auto-scaling infrastructure for production workloads

Your CAE system can now leverage the power of Neo4j Cloud for enhanced clinical decision support!

## 🔄 **Migration from GraphDB**

If you're currently using GraphDB and want to migrate:

1. **Export Data**: Export RDF data from GraphDB
2. **Transform Data**: Convert RDF to Cypher statements
3. **Import to Neo4j**: Load data into Neo4j Cloud
4. **Update Queries**: Convert SPARQL queries to Cypher
5. **Test Integration**: Verify CAE system integration

The unified database adapter in the pipeline supports both GraphDB and Neo4j, making migration seamless!
