# Order Management Service (CPOE Core)

A comprehensive microservice for managing clinical orders with CPOE (Computerized Provider Order Entry) functionality. This service integrates with Google Healthcare API for FHIR-compliant data storage and supports Apollo Federation for GraphQL operations.

## Features

- **FHIR-Compliant Order Management**: Implements FHIR ServiceRequest and related resources
- **CPOE Core Functionality**: Complete order lifecycle management
- **Clinical Decision Support**: Drug interaction checking and clinical validation
- **Order Sets & Protocols**: Pre-defined order templates and clinical pathways
- **Apollo Federation**: GraphQL federation support for distributed architecture
- **Google Healthcare API**: FHIR-compliant data storage
- **Authentication & Authorization**: RBAC with role-based permissions
- **Order Lifecycle Management**: Draft → Active → Completed workflows

## Architecture

```
API Gateway (8005) → Auth Service (8001) → Apollo Federation Gateway (4000) → Order Management Service (8013) → Google Healthcare API
```

## Prerequisites

- Python 3.12+
- Google Cloud Healthcare API access
- Service account credentials for Google Cloud
- Access to Clinical Synthesis Hub shared modules

## Installation

1. Install dependencies:
```bash
pip install -r requirements.txt
```

2. Set up Google Cloud credentials:
   - Place your service account key in `credentials/google-credentials.json`
   - Or use `credentials/service-account-key.json`

3. Configure environment variables (handled by `run_service.py`):
   - `GOOGLE_CLOUD_PROJECT=cardiofit-905a8`
   - `GOOGLE_CLOUD_LOCATION=asia-south1`
   - `GOOGLE_CLOUD_DATASET=clinical-synthesis-hub`
   - `GOOGLE_CLOUD_FHIR_STORE=fhir-store`

## Running the Service

### Using the Service Runner (Recommended)

```bash
python run_service.py
```

### Using Uvicorn Directly

```bash
uvicorn app.main:app --host 0.0.0.0 --port 8013 --reload
```

The service will be available at http://localhost:8013.

## API Endpoints

### REST API Endpoints

- `GET /api/orders`: List orders
- `POST /api/orders`: Create a new order
- `GET /api/orders/{id}`: Get order details
- `PUT /api/orders/{id}`: Update an order
- `DELETE /api/orders/{id}`: Delete an order

### Order Management Endpoints

- `POST /api/order-management/sign/{id}`: Sign an order
- `POST /api/order-management/cosign/{id}`: Co-sign an order
- `POST /api/order-management/cancel/{id}`: Cancel an order
- `POST /api/order-management/hold/{id}`: Put order on hold
- `POST /api/order-management/release/{id}`: Release order from hold
- `POST /api/order-management/discontinue/{id}`: Discontinue an order

### Order Sets Endpoints

- `GET /api/order-sets`: List order sets
- `POST /api/order-sets`: Create order set
- `GET /api/order-sets/{id}`: Get order set details

### GraphQL Endpoints

- `/graphql`: Direct GraphQL access
- `/api/federation`: Apollo Federation schema endpoint

### Other Endpoints

- `GET /`: Service welcome message
- `GET /health`: Health check endpoint
- `GET /api/docs`: API documentation

## Order Types Supported

1. **Medication Orders**: Prescriptions, dosage, administration
2. **Laboratory Orders**: Lab tests, specimen collection
3. **Imaging Orders**: Radiology, diagnostic imaging
4. **Procedure Orders**: Clinical procedures
5. **Consultation Orders**: Specialist referrals
6. **Nursing Orders**: Care instructions

## FHIR Resources

The service implements the following FHIR R4 resources:

- **ServiceRequest**: Core order resource
- **MedicationRequest**: Medication orders
- **DiagnosticReport**: Lab/imaging orders
- **Task**: Order workflow management
- **RequestGroup**: Order sets

## Authentication

The service uses JWT tokens for authentication through the HeaderAuthMiddleware. Required permissions:

- `order:create` - Create new orders
- `order:read` - View orders
- `order:update` - Modify orders
- `order:sign` - Sign orders
- `order:cosign` - Co-sign orders
- `order:cancel` - Cancel orders
- `order:manage_sets` - Manage order sets

## Development Status

### Phase 0: Foundation Setup ✅ COMPLETED
- [x] Basic service structure
- [x] Configuration and authentication setup
- [x] Service runner and requirements
- [x] Placeholder API endpoints

### Phase 1: Core Data Models (IN PROGRESS)
- [ ] FHIR-compliant order models
- [ ] Service layer implementation
- [ ] Google Healthcare API integration

### Phase 2: GraphQL Schema & Federation
- [ ] GraphQL types and operations
- [ ] Apollo Federation integration

### Phase 3: Testing & Integration
- [ ] Postman collection
- [ ] Apollo Federation gateway update

## Integration

This service is designed to work with:

- **API Gateway**: Routes requests and handles authentication
- **Auth Service**: Provides JWT token validation
- **Apollo Federation Gateway**: GraphQL federation
- **Patient Service**: Patient context and references
- **Medication Service**: Drug interaction checking
- **Organization Service**: Provider and facility context

## Next Steps

1. Implement FHIR-compliant data models
2. Create Google Healthcare API integration
3. Develop GraphQL schema with federation support
4. Add comprehensive testing
5. Create Postman collection for API testing

## Port Assignment

**Order Management Service Port: 8013**

This service follows the established microservice architecture pattern of the Clinical Synthesis Hub.
