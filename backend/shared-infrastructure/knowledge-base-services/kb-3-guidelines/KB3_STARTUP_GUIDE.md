# KB3 Guidelines Service - Startup Guide

## Overview

The KB3 Guideline Evidence Service is a comprehensive clinical guideline management system that provides conflict resolution, safety overrides, and Apollo Federation integration. This service manages clinical guidelines with evidence-based recommendations and intelligent conflict resolution capabilities.

## Architecture

```
KB3 Service
├── PostgreSQL Database (port 5434) - Guideline storage
├── Neo4j Graph Database (port 7688) - Guideline relationships
├── Redis Cache (port 6380) - Performance optimization
├── GraphQL Federation (port 8085) - Apollo Federation subgraph
└── REST API (port 8084) - Direct HTTP endpoints
```

## Prerequisites

### System Requirements
- **Node.js**: >= 18.0.0
- **npm**: >= 9.0.0
- **TypeScript**: 5.2.2+
- **PostgreSQL**: 14+ (for guideline data)
- **Neo4j**: 4.4+ (for guideline relationships)
- **Redis**: 6.0+ (for caching)

### Database Setup

#### PostgreSQL (Port 5434)
Ensure PostgreSQL is running with the following configuration:
```bash
# Database: kb3_guidelines
# User: kb3admin
# Password: kb3_postgres_password
# Port: 5434
```

Required tables:
- `guidelines` - Core guideline data
- `recommendations` - Clinical recommendations
- `conflict_resolutions` - Conflict resolution records
- `safety_overrides` - Safety override rules
- `kb_linkages` - Cross-knowledge base references
- `audit_log` - System audit trail

#### Neo4j (Port 7688)
Configure Neo4j with:
```bash
# URI: bolt://localhost:7688
# Username: neo4j
# Password: kb3_neo4j_password
# Database: neo4j (default)
```

#### Redis (Port 6380)
Redis instance for caching:
```bash
# URL: redis://localhost:6380
```

## Installation

### 1. Navigate to Service Directory
```bash
cd backend/services/medication-service/knowledge-bases/kb-3-guidelines
```

### 2. Install Dependencies
```bash
npm install
```

### 3. Environment Configuration
Set the following environment variables:

```bash
# Database Configuration
export KB3_DB_HOST=localhost
export KB3_DB_PORT=5434
export KB3_DB_USER=kb3admin
export KB3_DB_PASSWORD=kb3_postgres_password
export KB3_DB_NAME=kb3_guidelines

# Neo4j Configuration
export KB3_NEO4J_URI=bolt://localhost:7688
export KB3_NEO4J_USER=neo4j
export KB3_NEO4J_PASSWORD=kb3_neo4j_password
export KB3_NEO4J_DATABASE=neo4j

# Redis Configuration
export KB3_REDIS_URL=redis://localhost:6380

# Service Configuration
export KB3_SERVICE_PORT=8084
export KB3_FEDERATION_PORT=8085
export NODE_ENV=development
```

## Startup Options

### Option 1: Development Mode (Recommended)
Start the service with auto-reload capabilities:

```bash
npm run dev
```

This starts the service on port 8084 with TypeScript compilation and hot reload.

### Option 2: GraphQL Federation Mode
Start as Apollo Federation subgraph:

```bash
npm run federation:dev
```

This starts the federation subgraph server on port 8085 for integration with Apollo Gateway.

### Option 3: Production Build
For production deployment:

```bash
npm run build
npm run start
```

### Option 4: Direct TypeScript Execution
Alternative development startup:

```bash
npx ts-node --transpile-only src/app.ts
```

## Environment Variables Reference

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `KB3_DB_HOST` | PostgreSQL host | localhost | Yes |
| `KB3_DB_PORT` | PostgreSQL port | 5434 | Yes |
| `KB3_DB_USER` | Database username | kb3admin | Yes |
| `KB3_DB_PASSWORD` | Database password | - | Yes |
| `KB3_DB_NAME` | Database name | kb3_guidelines | Yes |
| `KB3_NEO4J_URI` | Neo4j connection URI | bolt://localhost:7688 | Yes |
| `KB3_NEO4J_USER` | Neo4j username | neo4j | Yes |
| `KB3_NEO4J_PASSWORD` | Neo4j password | - | Yes |
| `KB3_NEO4J_DATABASE` | Neo4j database | neo4j | No |
| `KB3_REDIS_URL` | Redis connection URL | redis://localhost:6380 | No |
| `KB3_SERVICE_PORT` | Main service port | 8084 | No |
| `KB3_FEDERATION_PORT` | Federation port | 8085 | No |
| `PORT` | Service port override | 8084 | No |
| `NODE_ENV` | Environment mode | development | No |

## Service Endpoints

### Health Check
```bash
GET http://localhost:8084/health
```

### Metrics
```bash
GET http://localhost:8084/metrics
```

### GraphQL Federation (if enabled)
```bash
POST http://localhost:8085/graphql
```

### API Endpoints
```bash
POST /api/guidelines              # Search guidelines
POST /api/clinical-pathway        # Get clinical pathway
POST /api/guidelines/compare      # Compare guidelines
GET  /api/validate/cross-kb       # Validate KB links
GET  /api/conflicts/stats         # Conflict statistics
```

## Validation and Testing

### 1. Service Health Check
```bash
curl -f http://localhost:8084/health
```

Expected response:
```json
{
  "status": "healthy",
  "version": "3.0.0",
  "environment": "development",
  "services": {
    "database": {"status": "healthy"},
    "neo4j": {"status": "healthy"},
    "cache": {"status": "healthy"},
    "conflict_resolver": {"status": "healthy"}
  }
}
```

### 2. Database Schema Validation
The service automatically validates required database tables on startup:
- `guidelines`
- `recommendations`
- `conflict_resolutions`
- `safety_overrides`
- `kb_linkages`
- `audit_log`

### 3. Run Tests
```bash
npm test                    # Run all tests
npm run test:watch          # Watch mode
npm run test:coverage       # Coverage report
```

### 4. Type Checking
```bash
npm run typecheck
```

### 5. Linting
```bash
npm run lint
npm run lint:fix
```

## GraphQL Federation Integration

### Federation Schema
The service provides Apollo Federation v2.3 schema with:
- **Entities**: Patient, Medication, Observation (extended)
- **Native Types**: Guideline, ClinicalPathway, SafetyOverride
- **Directives**: @key, @external, @requires, @provides, @shareable

### Federation Startup
```bash
# Start federation subgraph
KB3_FEDERATION_PORT=8085 PORT=8085 npm run federation:dev
```

### Schema Introspection
```bash
curl -X POST http://localhost:8085/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "query { _service { sdl } }"}'
```

## Troubleshooting

### Common Issues

#### 1. Database Connection Failed
**Symptoms**: Service fails to start with database connection errors
**Solutions**:
- Verify PostgreSQL is running on port 5434
- Check database credentials
- Ensure `kb3_guidelines` database exists
- Validate network connectivity

#### 2. Neo4j Connection Failed
**Symptoms**: Neo4j service health check fails
**Solutions**:
- Verify Neo4j is running on port 7688
- Check Neo4j credentials
- Ensure bolt protocol is enabled
- Test connection: `cypher-shell -a bolt://localhost:7688`

#### 3. Redis Connection Issues
**Symptoms**: Cache operations fail or warning messages
**Solutions**:
- Verify Redis is running on port 6380
- Check Redis configuration
- Ensure Redis is accessible: `redis-cli -p 6380 ping`

#### 4. Port Conflicts
**Symptoms**: "Port already in use" errors
**Solutions**:
- Check for existing processes: `lsof -i :8084` or `lsof -i :8085`
- Stop conflicting services
- Use alternative ports via environment variables

#### 5. TypeScript Compilation Errors
**Symptoms**: Build or runtime TypeScript errors
**Solutions**:
- Run `npm run typecheck` for detailed errors
- Ensure all dependencies are installed
- Check TypeScript version compatibility

### Performance Optimization

#### 1. Cache Configuration
Adjust cache settings in environment:
```bash
export KB3_MEMORY_CACHE_SIZE=209715200  # 200MB
export KB3_CACHE_TTL=1800              # 30 minutes
```

#### 2. Database Pool Settings
Configure connection pooling for high-load scenarios in production.

#### 3. Neo4j Memory Settings
Ensure Neo4j has sufficient heap memory for graph operations.

## Development Commands

| Command | Description |
|---------|-------------|
| `npm run dev` | Start development server with hot reload |
| `npm run build` | Build TypeScript to JavaScript |
| `npm run start` | Start built application |
| `npm run test` | Run test suite |
| `npm run test:watch` | Run tests in watch mode |
| `npm run test:coverage` | Generate coverage report |
| `npm run lint` | Run ESLint |
| `npm run lint:fix` | Fix ESLint issues automatically |
| `npm run typecheck` | Type checking without emission |
| `npm run federation` | Start federation subgraph |
| `npm run federation:dev` | Start federation with hot reload |
| `npm run federation:build` | Build and start federation |

## Production Deployment

### Docker Support
Build and run with Docker:
```bash
npm run docker:build
npm run docker:run
```

### Kubernetes Deployment
Deploy to Kubernetes:
```bash
npm run k8s:deploy
```

### Health Monitoring
Use the health endpoint for load balancer health checks:
```bash
npm run health
```

## Integration with Other Services

### Apollo Federation Gateway
Register this subgraph with your Apollo Gateway:
```javascript
// In your gateway configuration
{
  name: 'kb3-guidelines',
  url: 'http://localhost:8085/graphql'
}
```

### Knowledge Base Integration
KB3 integrates with other knowledge bases through:
- KB1: Dosing references via `KBReferences.kb1_dosing`
- KB2: Drug interactions via `KBReferences.kb2_interactions`
- KB4: Monitoring via `KBReferences.kb4_monitoring`

### Clinical Workflow Integration
Use the clinical pathway endpoints for treatment decision support:
- Patient-specific guideline recommendations
- Conflict resolution for competing guidelines
- Safety override management
- Cross-knowledge base validation

## Support and Maintenance

### Logging
Application logs include:
- Service initialization
- Database connections
- GraphQL operations
- Conflict resolution activities
- Safety override applications

### Audit Trail
All clinical operations are logged to the audit_log table for compliance and troubleshooting.

### Version Information
- **Service Version**: 3.0.0
- **GraphQL Version**: 16.8.1
- **Apollo Federation**: 2.10.2
- **Node.js Requirement**: >= 18.0.0

---

**Last Updated**: January 2025
**Service Owner**: Clinical Synthesis Hub CardioFit Team