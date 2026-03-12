# SchedulingService Postman Collection

## Overview

This comprehensive Postman collection contains **50+ requests** covering all aspects of the SchedulingService functionality, including:

- **Appointment Management** (Create, Read, Update, Cancel)
- **Schedule Management** (Provider schedules and availability)
- **Slot Management** (Available appointment slots)
- **Federation Queries** (Cross-service data integration)
- **Complex Scenarios** (Real-world booking flows)
- **Error Handling** (Edge cases and validation)
- **Performance Testing** (Bulk operations and analytics)
- **FHIR Compliance** (Resource validation)

## Collection Structure

### 1. Health Check (2 requests)
- Service health verification
- Federation schema validation

### 2. Authentication (1 request)
- Doctor login with automatic token extraction
- Sets `auth_token` variable for subsequent requests

### 3. Appointment Management (8 requests)
- Create appointment with full FHIR compliance
- Get appointment by ID
- Search by patient, practitioner, status, date
- Update appointment details
- Cancel appointment with reason

### 4. Schedule Management (4 requests)
- Create provider schedules
- Get schedule details
- Search by actor and active status

### 5. Slot Management (5 requests)
- Create available time slots
- Search by schedule, status, date range
- Slot availability checking

### 6. Federation Queries (3 requests)
- Patient appointments (cross-service)
- User appointments (cross-service)
- User schedules (cross-service)

### 7. Complex Scenarios (3 requests)
- Complete appointment booking flow
- Waitlist management
- Resource availability checking

### 8. Error Handling & Edge Cases (2 requests)
- Non-existent resource handling
- Invalid data validation

### 9. Performance & Analytics (2 requests)
- Bulk data operations
- Provider utilization reports

### 10. FHIR Compliance Tests (3 requests)
- Full FHIR resource validation
- Standards compliance verification

## Variables

The collection uses the following variables:

| Variable | Description | Default Value |
|----------|-------------|---------------|
| `base_url` | SchedulingService URL | `http://localhost:8014` |
| `federation_url` | Apollo Federation Gateway URL | `http://localhost:8005/graphql` |
| `auth_token` | Authentication token | Auto-set by login request |
| `patient_id` | Test patient ID | `patient-123` |
| `practitioner_id` | Test practitioner ID | `practitioner-456` |
| `appointment_id` | Created appointment ID | Auto-set by create requests |
| `schedule_id` | Created schedule ID | Auto-set by create requests |
| `slot_id` | Created slot ID | Auto-set by create requests |

## Setup Instructions

### 1. Import Collection
1. Open Postman
2. Click "Import"
3. Select `SchedulingService_Collection.json`
4. Collection will be imported with all requests and variables

### 2. Environment Setup
1. Ensure all services are running:
   - Auth Service: `http://localhost:8001`
   - SchedulingService: `http://localhost:8014`
   - Apollo Federation Gateway: `http://localhost:8005`

2. Update variables if needed:
   - Right-click collection → "Edit"
   - Go to "Variables" tab
   - Update URLs and IDs as needed

### 3. Authentication
1. Run "Get Auth Token (Doctor)" request first
2. This will automatically set the `auth_token` variable
3. All subsequent requests will use this token

## Usage Workflow

### Basic Testing Flow:
1. **Health Check** → Verify services are running
2. **Authentication** → Get auth token
3. **Create Schedule** → Set up provider availability
4. **Create Slot** → Define available time slots
5. **Create Appointment** → Book appointments
6. **Federation Queries** → Test cross-service integration

### Advanced Testing:
1. **Complex Scenarios** → Test real-world workflows
2. **Error Handling** → Validate error responses
3. **FHIR Compliance** → Verify standards compliance
4. **Performance** → Test bulk operations

## Key Features

### Automatic Variable Management
- IDs are automatically extracted and stored
- Tokens are auto-refreshed
- Cross-request data flow

### FHIR Compliance
- All requests follow FHIR R4 standards
- Complete resource validation
- Proper coding systems and references

### Federation Integration
- Cross-service queries
- Patient and User extensions
- Distributed data access

### Real-world Scenarios
- Complete booking workflows
- Waitlist management
- Resource availability
- Provider utilization

## Testing Scenarios

### 1. Complete Appointment Booking
```
1. Create Schedule → 2. Create Slot → 3. Book Appointment → 4. Confirm Booking
```

### 2. Waitlist Management
```
1. Search Available Slots → 2. Create Waitlist Entry → 3. Monitor Availability
```

### 3. Provider Schedule Management
```
1. Create Schedule → 2. Add Multiple Slots → 3. Check Utilization
```

### 4. Cross-service Integration
```
1. Get Patient Data → 2. Book Appointment → 3. Verify Federation
```

## Error Scenarios

The collection includes tests for:
- Invalid appointment data
- Non-existent resources
- Authentication failures
- FHIR validation errors
- Federation query failures

## Performance Testing

Use the bulk operations to test:
- Large dataset handling
- Query performance
- Federation scalability
- Resource utilization

## Support

For issues or questions:
1. Check service logs
2. Verify all services are running
3. Ensure proper authentication
4. Validate FHIR resource structure
