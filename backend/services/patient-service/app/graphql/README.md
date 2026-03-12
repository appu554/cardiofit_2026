# GraphQL Implementation for Patient Service

This directory contains the GraphQL implementation for the Patient Service. It provides a GraphQL API for accessing patient data using Graphene for automatic schema generation.

## Architecture

The GraphQL implementation follows a layered architecture:

1. **Schema Layer**: Defines the GraphQL schema using Graphene
2. **Resolver Layer**: Implements resolvers for queries and mutations
3. **Service Layer**: Provides business logic and validation
4. **Repository Layer**: Handles data access to MongoDB

```
GraphQL API → Service Layer → Repository Layer → MongoDB
```

## Components

### Types

The GraphQL types are defined in `types.py`. They mirror the FHIR resource types and provide methods for converting between FHIR and GraphQL formats:

- `Patient`: The main patient type
- `HumanName`: For patient names
- `Address`: For patient addresses
- `ContactPoint`: For patient contact information
- `Identifier`: For patient identifiers
- `CodeableConcept`: For coded concepts
- `Coding`: For individual codes

### Schema

The GraphQL schema is defined in `schema.py`. It consists of:

- **Queries**: Operations for retrieving data
  - `patient(id: String!)`: Get a patient by ID
  - `patients(...)`: Search for patients with various filters

- **Mutations**: Operations for modifying data
  - `createPatient(input: CreatePatientInput!)`: Create a new patient
  - `updatePatient(id: String!, input: UpdatePatientInput!)`: Update an existing patient
  - `deletePatient(id: String!)`: Delete a patient

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

## Usage

### GraphQL Endpoint

The GraphQL API is available at `/api/graphql`. You can use any GraphQL client to interact with it.

### GraphiQL Playground

A GraphiQL playground is available at `/api/graphql/playground`. It provides a web-based interface for exploring and testing the GraphQL API.

### Example Queries

```graphql
# Get a patient by ID
query GetPatient {
  patient(id: "123") {
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
query SearchPatients {
  patients(name: "Smith", gender: "male", page: 1, count: 10) {
    items {
      id
      name {
        family
        given
      }
      gender
      birthDate
    }
    total
    page
    count
  }
}
```

### Example Mutations

```graphql
# Create a new patient
mutation CreatePatient {
  createPatient(input: {
    name: [{
      family: "Smith",
      given: ["John"]
    }],
    gender: "male",
    birthDate: "1970-01-01"
  }) {
    patient {
      id
      name {
        family
        given
      }
    }
  }
}

# Update a patient
mutation UpdatePatient {
  updatePatient(
    id: "123",
    input: {
      name: [{
        family: "Smith",
        given: ["John", "Adam"]
      }]
    }
  ) {
    patient {
      id
      name {
        family
        given
      }
    }
  }
}

# Delete a patient
mutation DeletePatient {
  deletePatient(id: "123") {
    success
  }
}
```
