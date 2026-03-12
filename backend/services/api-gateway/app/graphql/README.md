# GraphQL Integration for Clinical Synthesis Hub

This directory contains the GraphQL implementation for the Clinical Synthesis Hub API Gateway. It provides a unified GraphQL API for accessing FHIR resources and other data from the various microservices.

## Architecture

The GraphQL implementation follows a layered architecture:

1. **Schema Layer**: Defines the GraphQL schema using Strawberry
2. **Resolver Layer**: Implements resolvers for queries and mutations
3. **Transformation Layer**: Converts between FHIR and GraphQL formats using the Data Transformation Layer (DTL)
4. **Service Layer**: Communicates with microservices via HTTP

```
Client → GraphQL API → Transformation Layer → Microservices
```

## Components

### Schema

The GraphQL schema is defined using Strawberry, a code-first GraphQL library for Python. The schema consists of:

- **Types**: GraphQL types that mirror FHIR resources
- **Queries**: Operations for retrieving data
- **Mutations**: Operations for creating, updating, and deleting data

### Resolvers

Resolvers are implemented as methods in the `Query` and `Mutation` classes. They:

1. Extract the authorization header from the request context
2. Call the appropriate microservice via HTTP
3. Transform the response data to GraphQL types
4. Return the result

### Data Transformation Layer (DTL)

The DTL is used to convert between FHIR and GraphQL formats. It provides:

- **FHIR to GraphQL Transformers**: Convert FHIR resources to GraphQL types
- **GraphQL to FHIR Transformers**: Convert GraphQL input types to FHIR resources

The DTL is implemented in the `shared/transformers` package and is used by the GraphQL resolvers.

## Usage

### Queries

```graphql
query GetPatient {
  patient(id: "123") {
    id
    name {
      family
      given
    }
    gender
    birthDate
  }
}
```

### Mutations

```graphql
mutation CreatePatient {
  createPatient(
    patientData: {
      name: [{ family: "Smith", given: ["John"] }]
      gender: "male"
      birthDate: "1970-01-01"
    }
  ) {
    id
    name {
      family
      given
    }
  }
}
```

## Integration with API Gateway

The GraphQL API is integrated with the API Gateway and mounted at the `/graphql` endpoint. It:

1. Shares the same authentication and authorization mechanisms as the REST API
2. Uses the same RBAC middleware for permission checks
3. Provides a unified API for accessing all microservices

## Integration with FHIR Service

The GraphQL resolvers communicate with the FHIR service to access FHIR resources. The flow is:

1. GraphQL resolver receives a request
2. Resolver calls the FHIR service with the appropriate parameters
3. FHIR service routes the request to the appropriate microservice
4. Microservice processes the request and returns the result
5. FHIR service returns the result to the resolver
6. Resolver transforms the result to GraphQL types and returns it

This ensures that the GraphQL API follows the same API Gateway > Auth > FHIR > Microservices flow as the REST API.

## Error Handling

The GraphQL implementation includes comprehensive error handling:

1. **Authentication Errors**: Return null for queries and mutations if the user is not authenticated
2. **Authorization Errors**: Return null for queries and mutations if the user doesn't have the required permissions
3. **Validation Errors**: Return appropriate error messages for invalid input
4. **Service Errors**: Handle errors from microservices gracefully

## Testing

The GraphQL API can be tested using the GraphQL Playground at `/graphql` when running in development mode.

## Future Enhancements

1. **Subscriptions**: Add support for real-time updates using GraphQL subscriptions
2. **DataLoader**: Implement DataLoader for efficient data fetching and batching
3. **Caching**: Add caching for frequently accessed data
4. **Field-Level Permissions**: Implement field-level permissions for fine-grained access control
