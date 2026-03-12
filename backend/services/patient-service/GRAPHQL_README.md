# GraphQL Implementation for Patient Service

This document provides an overview of the GraphQL implementation for the Patient Service in the Clinical Synthesis Hub.

## Overview

The GraphQL implementation follows a federated approach with a service layer pattern. It uses Graphene for automatic schema generation while maintaining the existing architecture flow.

## Architecture

The architecture follows the pattern:

```
Client → API Gateway (GraphQL) → Auth → FHIR → Microservices (GraphQL Service Layer)
```

The existing REST API remains intact:

```
Client → API Gateway (REST) → Auth → FHIR → Microservices
```

## Components

### Service Layer

The service layer is implemented in `app/services/patient_service.py`. It provides:

- Business logic for patient operations
- Validation using the FHIR service
- Integration with the repository layer

### Repository Layer

The repository layer is implemented in `app/repositories/patient_repository.py`. It provides:

- Data access to MongoDB
- Query building for search operations
- CRUD operations for patient data

### GraphQL Schema

The GraphQL schema is defined in `app/graphql/schema.py` using Graphene. It consists of:

- **Types**: GraphQL types that mirror FHIR resources
- **Queries**: Operations for retrieving data
- **Mutations**: Operations for creating, updating, and deleting data

### GraphQL Endpoint

The GraphQL API is available at `/api/graphql`. It is implemented using Starlette's GraphQL support.

## Installation

To install the required dependencies, run:

```bash
pip install graphene>=3.2.2 starlette-graphene3>=0.6.0
```

Or use the provided batch file:

```bash
./install_graphql_deps.bat
```

## Usage

### GraphQL Endpoint

The GraphQL API is available at `/api/graphql`. You can use any GraphQL client to interact with it.

### GraphiQL Playground

A GraphiQL playground is available at `/api/graphql/playground`. It provides a web-based interface for exploring and testing the GraphQL API.

### Postman Collection

A Postman collection is provided in `postman/patient_service_graphql.postman_collection.json` for testing the GraphQL API.

## Example Queries

```graphql
# Get a patient by ID
query GetPatient($id: String!) {
  patient(id: $id) {
    id
    resourceType
    name {
      family
      given
      use
    }
    gender
    birthDate
    active
  }
}

# Search for patients
query SearchPatients($name: String, $gender: String, $page: Int, $count: Int) {
  patients(name: $name, gender: $gender, page: $page, count: $count) {
    items {
      id
      resourceType
      name {
        family
        given
        use
      }
      gender
      birthDate
      active
    }
    total
    page
    count
  }
}
```

## Example Mutations

```graphql
# Create a new patient
mutation CreatePatient($input: CreatePatientInput!) {
  createPatient(input: $input) {
    patient {
      id
      resourceType
      name {
        family
        given
      }
      gender
      birthDate
      active
    }
  }
}

# Update a patient
mutation UpdatePatient($id: String!, $input: UpdatePatientInput!) {
  updatePatient(id: $id, input: $input) {
    patient {
      id
      resourceType
      name {
        family
        given
      }
      gender
      birthDate
      active
    }
  }
}

# Delete a patient
mutation DeletePatient($id: String!) {
  deletePatient(id: $id) {
    success
  }
}
```

## Testing

To test the GraphQL API, run:

```bash
pytest tests/test_graphql.py
```

## Integration with API Gateway

The Patient Service GraphQL API can be integrated with the API Gateway using federation. This allows the API Gateway to stitch together the GraphQL schemas from multiple microservices into a unified GraphQL API.

## Next Steps

1. Implement GraphQL in other microservices (Observation, Condition, etc.)
2. Implement federation in the API Gateway
3. Add authentication and authorization to the GraphQL API
4. Add more complex queries and mutations
5. Add subscription support for real-time updates
