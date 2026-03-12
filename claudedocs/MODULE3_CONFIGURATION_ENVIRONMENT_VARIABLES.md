# Module 3 Configuration Environment Variables

## Overview
All hardcoded configuration values have been externalized to environment variables with sensible fallback defaults. This allows for flexible deployment across different environments without code changes.

## Environment Variables

### Kafka Configuration

#### KAFKA_BOOTSTRAP_SERVERS
- **Description**: Kafka cluster bootstrap servers
- **Default**: `localhost:9092`
- **Example**: `localhost:9092` or `pkc-xxxxx.us-east-1.aws.confluent.cloud:9092`
- **Used by**: All Kafka sources and sinks

#### MODULE3_INPUT_TOPIC
- **Description**: Input topic for enriched patient context events from Module 2
- **Default**: `clinical-patterns.v1`
- **Example**: `clinical-patterns.v1` or `enriched-patient-events-v1`
- **Used by**: Module 3 Kafka source

#### MODULE3_OUTPUT_TOPIC
- **Description**: Output topic for comprehensive CDS events
- **Default**: `comprehensive-cds-events.v1`
- **Example**: `comprehensive-cds-events.v1` or `cds-events-output`
- **Used by**: Module 3 Kafka sink

---

### PubMed API Configuration

#### PUBMED_API_URL
- **Description**: NCBI eUtils ESummary API endpoint
- **Default**: `https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esummary.fcgi`
- **Example**: `https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esummary.fcgi`
- **Used by**: PubMedClient for citation metadata retrieval

#### PUBMED_API_KEY
- **Description**: NCBI API key for increased rate limits (10 requests/second vs 3/second)
- **Default**: `3ddce7afddefb52bd45a79f3a4416dabaf0a`
- **Example**: Get your API key from https://www.ncbi.nlm.nih.gov/account/settings/
- **Used by**: PubMedClient for authenticated API requests

#### PUBMED_EMAIL
- **Description**: Contact email required by NCBI API policy
- **Default**: `noreply@cardiofit.health`
- **Example**: `your-email@example.com`
- **Used by**: PubMedClient to identify the application

#### PUBMED_MAX_PMIDS_PER_REQUEST
- **Description**: Maximum number of PMIDs to fetch in a single API request
- **Default**: `200`
- **Example**: `200` (NCBI recommended maximum)
- **Used by**: PubMedClient batching logic

#### PUBMED_REQUEST_TIMEOUT_SECONDS
- **Description**: HTTP request timeout in seconds
- **Default**: `10`
- **Example**: `10` or `30` for slower networks
- **Used by**: PubMedClient HTTP client

---

## Configuration Examples

### Local Development
```bash
# Use defaults - no environment variables needed
export KAFKA_BOOTSTRAP_SERVERS=localhost:9092
export MODULE3_INPUT_TOPIC=clinical-patterns.v1
export MODULE3_OUTPUT_TOPIC=comprehensive-cds-events.v1
```

### Production (Confluent Cloud)
```bash
export KAFKA_BOOTSTRAP_SERVERS=pkc-xxxxx.us-east-1.aws.confluent.cloud:9092
export MODULE3_INPUT_TOPIC=prod-clinical-patterns-v1
export MODULE3_OUTPUT_TOPIC=prod-cds-events-v1

export PUBMED_API_KEY=your-actual-api-key
export PUBMED_EMAIL=production@cardiofit.health
export PUBMED_REQUEST_TIMEOUT_SECONDS=30
```

### Docker Compose
```yaml
services:
  flink-taskmanager:
    environment:
      - KAFKA_BOOTSTRAP_SERVERS=kafka:9092
      - MODULE3_INPUT_TOPIC=clinical-patterns.v1
      - MODULE3_OUTPUT_TOPIC=comprehensive-cds-events.v1
      - PUBMED_API_KEY=${PUBMED_API_KEY}
      - PUBMED_EMAIL=noreply@cardiofit.health
```

### Flink Cluster (flink-conf.yaml)
```yaml
env.java.opts: >-
  -DKAFKA_BOOTSTRAP_SERVERS=localhost:9092
  -DMODULE3_INPUT_TOPIC=clinical-patterns.v1
  -DMODULE3_OUTPUT_TOPIC=comprehensive-cds-events.v1
  -DPUBMED_API_KEY=3ddce7afddefb52bd45a79f3a4416dabaf0a
  -DPUBMED_EMAIL=noreply@cardiofit.health
```

---

## Testing Configuration Override

To test with different configuration:

```bash
# Set environment variables before submitting Flink job
export MODULE3_INPUT_TOPIC=test-clinical-patterns
export MODULE3_OUTPUT_TOPIC=test-cds-events
export PUBMED_REQUEST_TIMEOUT_SECONDS=5

# Submit job
curl -X POST 'http://localhost:8081/jars/<jar-id>/run' \
  -H "Content-Type: application/json" \
  -d '{"entryClass":"com.cardiofit.flink.operators.Module3_ComprehensiveCDS","parallelism":2}'
```

---

## Validation

To verify configuration is being read correctly, check Flink logs:

```bash
# Check TaskManager logs for configuration values
docker logs flink-taskmanager 2>&1 | grep -E "Loading|Phase|PubMed|Kafka"
```

You should see log entries showing:
- Kafka topics being used
- PubMed API configuration (without exposing full API key)
- Drug interaction analyzer statistics
- Protocol matcher initialization

---

## Migration Notes

### Before (Hardcoded)
```java
.setTopic("comprehensive-cds-events.v1")  // ❌ Hardcoded
private static final String API_KEY = "3ddce7afddefb52bd45a79f3a4416dabaf0a";  // ❌ Hardcoded
```

### After (Configurable)
```java
.setTopic(getTopicName("MODULE3_OUTPUT_TOPIC", "comprehensive-cds-events.v1"))  // ✅ Configurable
private static final String API_KEY = getEnvOrDefault(
    "PUBMED_API_KEY",
    "3ddce7afddefb52bd45a79f3a4416dabaf0a");  // ✅ Configurable with fallback
```

---

## Best Practices

1. **API Keys**: Always override `PUBMED_API_KEY` in production with your own NCBI API key
2. **Email**: Set `PUBMED_EMAIL` to a monitored email address for NCBI communication
3. **Topics**: Use environment-specific topic names (e.g., `dev-`, `staging-`, `prod-` prefixes)
4. **Timeouts**: Adjust `PUBMED_REQUEST_TIMEOUT_SECONDS` based on network latency
5. **Secrets Management**: Use secret management tools (AWS Secrets Manager, HashiCorp Vault) for API keys in production

---

## Related Files

- **PubMedClient.java**: `/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/clients/PubMedClient.java`
- **Module3_ComprehensiveCDS.java**: `/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module3_ComprehensiveCDS.java`

---

## Session Information

**Completed**: Session 2025-10-29 (continuation session)
**Changes**: Externalized all hardcoded values to environment variables
**Job ID**: ec645a26204404524feca7d07ddfe25d
**Status**: ✅ RUNNING successfully with configurable topics and PubMed settings
