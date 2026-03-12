# Apollo Federation Implementation Guide

This guide explains how to implement and test the complete flow for GraphQL:

```
API Gateway > Auth > Apollo Federation Gateway > Microservices > Google Healthcare API
```

## Changes Made

1. **Apollo Federation Gateway**:
   - Removed mock authentication
   - Implemented proper authentication header forwarding
   - Configured to use IntrospectAndCompose for federation
   - Updated context builder to extract user information from headers
   - Added proper health check endpoint

2. **Patient Service**:
   - Enhanced federation endpoint to handle entity references
   - Added support for `__resolveReference` resolver

## Testing the Implementation

### 1. Start the Services

Start all services in the following order:

```bash
# 1. Start Auth Service
cd backend/services/auth-service
python main.py

# 2. Start API Gateway
cd backend/services/api-gateway
python main.py

# 3. Start Apollo Federation Gateway
cd apollo-federation
npm start

# 4. Start Patient Service
cd backend/services/patient-service
python main.py

# 5. Start other microservices as needed
cd backend/services/observation-service
python main.py
# Repeat for other services
```

### 2. Test Authentication Flow

1. Get a valid authentication token:
   ```bash
   curl -X POST http://localhost:8001/api/auth/token -d '{"username":"test@example.com","password":"password123"}'
   ```

2. Use the token to make a GraphQL request through the API Gateway:
   ```bash
   curl -X POST http://localhost:8000/api/graphql \
     -H "Authorization: Bearer YOUR_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"query":"{ patients(limit: 10) { items { id name { family given } } total } }"}'
   ```

### 3. Test Federation

1. Check the health of the Federation Gateway:
   ```bash
   curl http://localhost:4000/health
   ```

2. Make a direct request to the Federation Gateway:
   ```bash
   curl -X POST http://localhost:4000/graphql \
     -H "Content-Type: application/json" \
     -d '{"query":"{ _service { sdl } }"}'
   ```

3. Test entity references:
   ```bash
   curl -X POST http://localhost:4000/graphql \
     -H "Content-Type: application/json" \
     -d '{"query":"{ _entities(representations: [{__typename: \"Patient\", id: \"PATIENT_ID\"}]) { ... on Patient { id name { family given } } } }"}'
   ```

## Troubleshooting

### Common Issues

1. **Authentication Errors**:
   - Check that the token is valid
   - Verify that the API Gateway is forwarding the authentication headers
   - Check the logs of the Auth Service

2. **Federation Errors**:
   - Verify that all services have implemented the federation endpoint
   - Check that the federation endpoint is accessible without authentication
   - Look for schema composition errors in the Federation Gateway logs

3. **Missing Headers**:
   - Ensure that the API Gateway is adding the X-User-* headers
   - Verify that the Federation Gateway is forwarding these headers to microservices

### Logs to Check

- API Gateway logs for authentication and header forwarding
- Apollo Federation Gateway logs for schema composition and request forwarding
- Microservice logs for request handling and Google Healthcare API interaction

## Next Steps

1. **Implement Federation for Other Microservices**:
   - Add federation endpoints to all microservices
   - Implement `__resolveReference` resolvers for all entity types
   - Update the Federation Gateway to include all services

2. **Enhance Security**:
   - Add rate limiting
   - Implement more granular permission checks
   - Add request validation

3. **Monitoring and Observability**:
   - Add metrics collection
   - Implement distributed tracing
   - Set up alerting for service health
