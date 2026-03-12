# Clinical Synthesis Hub - GraphQL Order Management Service

## 🏗️ Architecture Flow

```
Client → API Gateway (8005) → Auth Service (8001) → Apollo Federation Gateway (4000) → Order Management Service (8013) → Google Healthcare API
```

## Overview

This comprehensive Postman collection provides testing for the Order Management Service (CPOE Core) through the complete Clinical Synthesis Hub architecture flow using GraphQL. The collection includes all Order Management Service responsibilities and follows the established Clinical Synthesis Hub testing patterns.

## 📋 Order Management Service Responsibilities

### Core CPOE (Computerized Provider Order Entry) Features
- **Order Lifecycle Management**: Draft, active, hold, cancel, complete states
- **Clinical Decision Support Integration**: Drug interaction checking, allergy alerts, clinical guidelines
- **Order Sets & Protocols**: Standardized order sets for common conditions and procedures
- **FHIR-Compliant Data Models**: Full FHIR R4 ServiceRequest resource support

### Integration Capabilities
- **Apollo Federation**: Cross-service queries with Patient, User, and other entities
- **Google Healthcare API**: Persistent storage using Google Cloud Healthcare API
- **Authentication & RBAC**: JWT-based authentication with role-based access control

## Prerequisites

1. **Services Running**:
   - API Gateway: `http://localhost:8005`
   - Auth Service: `http://localhost:8001`
   - Apollo Federation Gateway: `http://localhost:4000`
   - Order Management Service: `http://localhost:8013`
   - Patient Service: `http://localhost:8003`
   - Organization Service: `http://localhost:8012`

2. **Authentication**:
   - Valid Supabase user credentials
   - Proper RBAC permissions for order management

## 📋 Collection Structure

### 🔐 Authentication
- **Login (Get Token)**: Authenticate and get JWT token
- **Verify Token**: Verify the authentication token

### 📋 Clinical Order Management (CPOE Core)
- **Create Clinical Order**: Create new clinical orders with full FHIR compliance
- **Get All Clinical Orders**: Retrieve all clinical orders
- **Get Orders by Patient**: Filter orders by patient ID
- **Get Single Order**: Retrieve specific order by ID

### 🩺 Clinical Decision Support
- **Create Order with Clinical Decision Support**: Order creation with CDS alerts
- **Check Drug Interactions**: Verify potential drug interactions

### 📋 Order Sets & Protocols
- **Create Order Set - Hypertension Protocol**: Standardized hypertension management
- **Create Order Set - Diabetes Management**: Standardized diabetes management

### 🔄 Order Lifecycle Management
- **Sign Order (Draft to Active)**: Activate draft orders
- **Hold Order**: Put orders on hold
- **Cancel Order**: Cancel active orders

### 🔗 Federation & Cross-Service Queries
- **Patient with Orders (Federation)**: Test Patient-Order federation
- **User with Orders (Federation)**: Test User-Order federation

### 🔍 Schema & Introspection
- **GraphQL Schema Introspection**: Explore available types and fields
- **Get Order Management Types**: List all Order Management types

### 🏥 Health Checks & Service Status
- **API Gateway Health**: Check API Gateway status
- **Order Management Service Health**: Check service status
- **Apollo Federation Gateway Health**: Check federation status
- **GraphQL Endpoint Connectivity**: Test endpoint connectivity

## Usage Steps

### Step 1: Authentication
1. **Authenticate**: Run the "Login (Get Token)" request first
2. **Verify**: Use "Verify Token" to confirm authentication

### Step 2: Health Checks
1. Run "API Gateway Health" to verify the gateway is running
2. Run "Order Management Service Health" to verify the service
3. Run "Apollo Federation Gateway Health" to verify federation
4. Run "GraphQL Endpoint Connectivity" to test the endpoint

### Step 3: Schema Exploration
1. Run "GraphQL Schema Introspection" to see all available types
2. Run "Get Order Management Types" to see Order Management specific types

### Step 4: Clinical Order Management
1. **Create Orders**: Use "Create Clinical Order" to create new orders
2. **Query Orders**: Use "Get All Clinical Orders" to see all orders
3. **Filter Orders**: Use "Get Orders by Patient" to filter by patient
4. **Single Order**: Use "Get Single Order" to retrieve specific orders

### Step 5: Clinical Decision Support
1. **CDS Integration**: Use "Create Order with Clinical Decision Support"
2. **Drug Interactions**: Use "Check Drug Interactions" for safety checks

### Step 6: Order Sets & Protocols
1. **Hypertension Protocol**: Use "Create Order Set - Hypertension Protocol"
2. **Diabetes Management**: Use "Create Order Set - Diabetes Management"

### Step 7: Order Lifecycle Management
1. **Sign Orders**: Use "Sign Order (Draft to Active)" to activate orders
2. **Hold Orders**: Use "Hold Order" to pause orders
3. **Cancel Orders**: Use "Cancel Order" to cancel orders

### Step 8: Federation Testing
1. **Patient Federation**: Use "Patient with Orders (Federation)"
2. **User Federation**: Use "User with Orders (Federation)"

## Configuration

The collection includes pre-configured variables:

| Variable | Default Value | Description |
|----------|---------------|-------------|
| `api_gateway_url` | `http://localhost:8005` | API Gateway URL |
| `username` | `admin` | Username for authentication |
| `password` | `password` | Password for authentication |
| `auth_token` | (auto-populated) | JWT token from login |
| `patient_id` | `patient-123` | Patient ID for testing |
| `practitioner_id` | `practitioner-456` | Practitioner ID for testing |
| `order_id` | (auto-populated) | Order ID from creation |
| `encounter_id` | `encounter-789` | Encounter ID for testing |
| `medication_id` | `medication-101` | Medication ID for testing |

## Sample GraphQL Queries

### Basic Order Query
```graphql
{
  orders {
    id
    status
    description
    patientId
  }
}
```

### Create Order Mutation
```graphql
mutation CreateOrder($description: String!, $patientId: String!) {
  createOrder(description: $description, patientId: $patientId) {
    id
    status
    description
    patientId
  }
}
```

### Federation Query (Patient with Orders)
```graphql
query PatientWithOrders($id: ID!) {
  patient(id: $id) {
    id
    name {
      given
      family
    }
    orders {
      id
      status
      description
    }
  }
}
```

## Expected Responses

### Successful Order Query
```json
{
  "data": {
    "orders": [
      {
        "id": "order-1",
        "status": "active",
        "description": "Sample order 1",
        "patientId": "patient-123"
      }
    ]
  }
}
```

### Successful Order Creation
```json
{
  "data": {
    "createOrder": {
      "id": "new-order",
      "status": "draft",
      "description": "Blood pressure medication",
      "patientId": "patient-123"
    }
  }
}
```

## Troubleshooting

### Common Issues

1. **Service Not Running**
   - Error: Connection refused
   - Solution: Start the Order Management Service

2. **GraphQL Endpoint Not Found**
   - Error: 404 Not Found
   - Solution: Verify the service is running and GraphQL is mounted

3. **Federation Errors**
   - Error: Cannot query field on type
   - Solution: Ensure all federated services are running

4. **Authentication Errors**
   - Error: Unauthorized
   - Solution: Set valid JWT token in auth_token variable

### Debug Steps

1. Check service health endpoint: `GET /health`
2. Verify GraphQL schema: `POST /api/federation` with introspection query
3. Check service logs for errors
4. Verify environment variables are set correctly

## Advanced Testing

### Performance Testing
- Use Postman's collection runner for load testing
- Monitor response times for GraphQL queries
- Test with large datasets

### Error Handling
- Test with invalid patient IDs
- Test with malformed GraphQL queries
- Test authentication failures

### Federation Integration
- Test cross-service queries
- Verify data consistency across services
- Test federation error handling

## Next Steps

1. Extend the collection with more complex queries
2. Add automated tests with Postman scripts
3. Integrate with CI/CD pipeline
4. Add performance benchmarks
