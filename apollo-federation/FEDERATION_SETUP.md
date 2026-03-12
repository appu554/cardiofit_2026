# Apollo Federation Setup Documentation

## Overview

This document explains how Apollo Federation is connected to the Patient and Observation microservices, enabling a unified GraphQL API that combines schemas from multiple services.

## Architecture Flow

```
API Gateway (8005) → Apollo Federation Gateway (4000) → Microservices
                                                      ├── Patient Service (8003)
                                                      └── Observation Service (8007)
```

## How Apollo Federation Works

### 1. Federation Endpoints

Each microservice exposes a dedicated federation endpoint that:
- Bypasses authentication (for internal federation communication)
- Provides schema introspection with Federation directives
- Handles federated queries from the Apollo Gateway

#### Patient Service Federation Endpoint
- **URL**: `http://localhost:8003/api/federation`
- **File**: `backend/services/patient-service/app/api/endpoints/federation.py`
- **Key Features**:
  - Generates SDL with `@key(fields: "id")` directive for Patient type
  - Handles introspection queries for schema composition
  - Bypasses authentication for federation requests

#### Observation Service Federation Endpoint
- **URL**: `http://localhost:8007/api/federation`
- **File**: `backend/services/observation-service/app/api/endpoints/federation.py`
- **Key Features**:
  - Generates SDL with `@key(fields: "id")` directive for Observation type
  - Handles introspection queries for schema composition
  - Bypasses authentication for federation requests

### 2. Schema Generation with Federation Directives

Both services automatically add Federation directives to their schemas:

```python
def get_federation_sdl():
    """Generate the SDL with Federation directives."""
    # Get the base schema as string
    schema_str = print_schema(schema.graphql_schema)

    # Add @key directive to main type
    schema_str = schema_str.replace(
        "type Patient {",  # or "type Observation {"
        "type Patient @key(fields: \"id\") {"  # or "type Observation @key(fields: \"id\") {"
    )

    # Add Federation directives
    federation_sdl = f"""
    extend schema @link(url: "https://specs.apollo.dev/federation/v2.0", import: ["@key", "@shareable"])

    {schema_str}
    """

    return federation_sdl
```

### 3. Apollo Federation Gateway Configuration

The Apollo Federation Gateway is configured to discover and compose schemas from both services:

#### Service List Configuration
```javascript
const serviceList = [
  {
    name: 'patients',
    url: 'http://localhost:8003/api/federation'
  },
  {
    name: 'observations',
    url: 'http://localhost:8007/api/federation'
  },
  {
    name: 'medications',
    url: 'http://localhost:8009/api/federation'
  },
  {
    name: 'organizations',
    url: 'http://localhost:8012/api/federation'
  },
  {
    name: 'orders',
    url: 'http://localhost:8013/api/federation'
  },
  {
    name: 'scheduling',
    url: 'http://localhost:8014/api/federation'
  },
  {
    name: 'encounters',
    url: 'http://localhost:8020/api/federation'
  }
];
```

#### Files Updated:
- `apollo-federation/index.js`
- `apollo-federation/generate-supergraph.js`
- `apollo-federation/rover-gateway.js`
- `apollo-federation/supergraph.yaml`

### 4. Schema Composition Process

1. **Discovery**: Apollo Gateway calls each service's federation endpoint
2. **Introspection**: Services return their SDL with Federation directives
3. **Composition**: Gateway composes a unified supergraph schema
4. **Query Planning**: Gateway creates execution plans for federated queries

### 5. Authentication Handling

#### Regular GraphQL Requests
- Require authentication via Authorization header
- Token validation and user context

#### Federation Requests
- Bypass authentication (internal communication)
- Identified by `federation: true` flag in context
- Used only for schema introspection and federated query execution

```python
# Check if this is a federation request (bypass authentication)
is_federation = info.context.get("federation", False)
current_user = None

if not is_federation:
    # Normal authentication flow
    auth_header = request.headers.get("Authorization")
    # ... validate token
else:
    logger.info("Federation request detected, bypassing authentication")
```

## Testing the Federation Setup

### 1. Start the Services
```bash
# Terminal 1: Patient Service
cd backend/services/patient-service
python -m uvicorn app.main:app --host 0.0.0.0 --port 8003 --reload

# Terminal 2: Observation Service
cd backend/services/observation-service
python -m uvicorn app.main:app --host 0.0.0.0 --port 8007 --reload

# Terminal 3: Apollo Federation Gateway
cd apollo-federation
npm start
```

### 2. Test Federation Endpoints

#### Patient Service Federation
```bash
curl -X POST http://localhost:8003/api/federation \
  -H "Content-Type: application/json" \
  -d '{"query": "{ _service { sdl } }"}'
```

#### Observation Service Federation
```bash
curl -X POST http://localhost:8007/api/federation \
  -H "Content-Type: application/json" \
  -d '{"query": "{ _service { sdl } }"}'
```

### 3. Test Unified GraphQL API

Access the Apollo Federation Gateway at `http://localhost:4000/graphql` and run queries that span both services:

```graphql
query {
  patients(page: 1, limit: 5) {
    items {
      id
      name {
        given
        family
      }
    }
  }
  observations(page: 1, count: 5) {
    id
    status
    code {
      text
    }
    subject {
      reference
    }
  }
}
```

## Key Benefits

1. **Unified API**: Single GraphQL endpoint for multiple services
2. **Schema Composition**: Automatic merging of service schemas
3. **Type Federation**: Ability to extend types across services
4. **Independent Development**: Services can evolve independently
5. **Performance**: Efficient query planning and execution

## Troubleshooting

### Common Issues

1. **Service Discovery Failures**
   - Ensure federation endpoints are accessible
   - Check service URLs in configuration files

2. **Schema Composition Errors**
   - Verify Federation directives are properly added
   - Check for conflicting type definitions

3. **Authentication Issues**
   - Ensure federation requests bypass authentication
   - Verify regular requests still require authentication

### Logs to Check

- Apollo Gateway logs: Federation composition and query planning
- Service logs: Federation endpoint requests and schema generation
- Network logs: Service-to-service communication

## Next Steps

1. Add more microservices to the federation
2. Implement cross-service type extensions
3. Add subscription support for real-time updates
4. Implement distributed tracing for federated queries
