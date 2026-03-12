# Scheduling Service

The Scheduling Service is a comprehensive microservice for managing healthcare appointments, provider schedules, and appointment slots within the Clinical Synthesis Hub. It provides FHIR-compliant scheduling functionality with Google Healthcare API integration, Supabase authentication, and Apollo Federation support.

## Features

### Core Functionality
- **Appointment Management**: Create, read, update, and cancel appointments
- **Provider Schedule Management**: Manage provider availability and schedules
- **Slot Management**: Create and manage available appointment slots
- **Multi-participant Support**: Handle appointments with multiple participants
- **FHIR Compliance**: Full FHIR R4 Appointment, Schedule, and Slot resource support
- **Google Healthcare API Integration**: Persistent storage using Google Cloud Healthcare API
- **Supabase Authentication**: JWT-based authentication with RBAC
- **Apollo Federation**: GraphQL federation support for distributed schemas

### Appointment Features
- ✅ Create, read, update, cancel appointments
- ✅ Multi-participant appointment support
- ✅ Status tracking and lifecycle management
- ✅ Reason codes and cancellation tracking
- ✅ Search by patient, practitioner, date, status

### Provider Schedule Management
- ✅ Create and manage provider schedules
- ✅ Actor-based schedule assignment
- ✅ Service category and specialty support
- ✅ Active/inactive status management

### Appointment Slot Management
- ✅ Create available appointment slots
- ✅ Status management (free, busy, etc.)
- ✅ Time-based slot searching
- ✅ Schedule association and validation

### Apollo Federation Support
- ✅ Patient entity extension with appointments
- ✅ Practitioner entity extension with appointments and schedules
- ✅ Cross-service query resolution
- ✅ Federation v2 directive support
- ✅ Shared type compatibility

### FHIR Compliance
- ✅ Complete FHIR Appointment resource
- ✅ FHIR Schedule resource support
- ✅ FHIR Slot resource support
- ✅ Standard FHIR REST API endpoints
- ✅ FHIR search parameter support

## Architecture

The service follows the established microservice architecture pattern:

```
API Gateway → Auth Service → Apollo Federation Gateway → Scheduling Service → Google Healthcare API
```

### Components:
1. **API Gateway** (Port 8005) - Routes requests and handles CORS
2. **Auth Service** (Port 8001) - Supabase JWT validation with RBAC
3. **Apollo Federation Gateway** (Port 4000) - GraphQL federation
4. **Scheduling Service** (Port 8014) - Appointment and schedule management
5. **Google Healthcare API** - FHIR resource storage

## Quick Start

### Prerequisites
1. Google Cloud Healthcare API setup
2. Supabase configuration with RBAC
3. Python 3.8+ with required dependencies

### Installation
```bash
# Navigate to service directory
cd backend/services/scheduling-service

# Install dependencies
pip install -r requirements.txt

# Set up Google credentials (place in credentials/ directory)
# Configure environment variables (see Configuration section)

# Run the service
python run_service.py
```

### Service URLs
- **REST API**: http://localhost:8014/api
- **GraphQL Federation**: http://localhost:8014/api/federation
- **API Documentation**: http://localhost:8014/docs
- **Health Check**: http://localhost:8014/health

## Configuration

### Environment Variables

```bash
# Service Configuration
SCHEDULING_SERVICE_PORT=8014
SCHEDULING_SERVICE_HOST=0.0.0.0
DEBUG=true

# Google Healthcare API
GOOGLE_CLOUD_PROJECT=cardiofit-905a8
GOOGLE_CLOUD_LOCATION=asia-south1
GOOGLE_CLOUD_DATASET=clinical-synthesis-hub
GOOGLE_CLOUD_FHIR_STORE=fhir-store
GOOGLE_APPLICATION_CREDENTIALS=credentials/google-credentials.json

# Authentication
AUTH_SERVICE_URL=http://localhost:8001
SUPABASE_URL=https://auugxeqzgrnknklgwqrh.supabase.co
SUPABASE_KEY=your_supabase_key
SUPABASE_JWT_SECRET=your_jwt_secret

# CORS
BACKEND_CORS_ORIGINS=http://localhost:3000,http://localhost:8000,http://localhost:8005
```

### Google Cloud Setup

1. **Create Service Account**: Create a service account in Google Cloud Console
2. **Grant Permissions**: Assign Healthcare API permissions
3. **Download Credentials**: Save as `credentials/google-credentials.json`
4. **Configure FHIR Store**: Ensure the FHIR store exists in your dataset

## API Endpoints

### REST API

#### Appointments
- `POST /api/appointments` - Create appointment
- `GET /api/appointments/{id}` - Get appointment by ID
- `GET /api/appointments` - Search appointments
- `PUT /api/appointments/{id}` - Update appointment
- `DELETE /api/appointments/{id}` - Delete appointment

#### Schedules
- `POST /api/schedules` - Create schedule
- `GET /api/schedules/{id}` - Get schedule by ID
- `GET /api/schedules` - Search schedules
- `PUT /api/schedules/{id}` - Update schedule
- `DELETE /api/schedules/{id}` - Delete schedule

#### Slots
- `POST /api/slots` - Create slot
- `GET /api/slots/{id}` - Get slot by ID
- `GET /api/slots` - Search slots
- `PUT /api/slots/{id}` - Update slot
- `DELETE /api/slots/{id}` - Delete slot

### GraphQL Federation

The service provides a federated GraphQL schema accessible at `/api/federation` with:

- **Appointment**: Complete appointment management
- **Schedule**: Provider schedule management
- **Slot**: Appointment slot management
- **Patient Extension**: Appointments for patients
- **Practitioner Extension**: Appointments and schedules for practitioners

## Data Models

### FHIR Resources

#### Appointment
- Complete FHIR R4 Appointment resource
- Status tracking (proposed, booked, fulfilled, cancelled, etc.)
- Multi-participant support
- Reason codes and instructions

#### Schedule
- FHIR R4 Schedule resource
- Actor-based (practitioner/resource) scheduling
- Service categories and specialties
- Planning horizon support

#### Slot
- FHIR R4 Slot resource
- Time-based availability
- Status management (free, busy, etc.)
- Schedule association

## Testing

### Health Check
```bash
curl http://localhost:8014/health
```

### Create Appointment
```bash
curl -X POST http://localhost:8014/api/appointments \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "status": "proposed",
    "description": "Regular checkup",
    "start": "2024-01-15T10:00:00Z",
    "end": "2024-01-15T10:30:00Z",
    "minutes_duration": 30
  }'
```

### GraphQL Federation Query
```graphql
query GetPatientAppointments($patientId: ID!) {
  patient(id: $patientId) {
    id
    appointments {
      id
      status
      start
      end
      description
    }
  }
}
```

## Development

### Project Structure
```
backend/services/scheduling-service/
├── app/
│   ├── api/                # REST API endpoints
│   ├── graphql/           # GraphQL federation schema
│   ├── services/          # Business logic and FHIR integration
│   ├── core/              # Configuration
│   └── main.py            # FastAPI application
├── credentials/           # Google Cloud credentials
├── requirements.txt       # Dependencies
├── run_service.py        # Service runner
└── README.md             # This file
```

### Adding New Features

1. **REST Endpoints**: Add to `app/api/endpoints/`
2. **GraphQL Types**: Extend `app/graphql/federation_schema.py`
3. **Business Logic**: Add to `app/services/`
4. **FHIR Operations**: Extend `app/services/google_fhir_service.py`

## Integration

### Apollo Federation

The service is designed to work with Apollo Federation Gateway. To include in your supergraph:

1. **Start the service** on port 8014
2. **Configure federation** to include the scheduling service endpoint
3. **Generate supergraph** with the scheduling schema

### Authentication

The service uses HeaderAuthMiddleware to extract user information from headers set by the API Gateway after JWT validation.

## Monitoring and Logging

- **Structured Logging**: Uses Python logging with structured output
- **Health Checks**: Available at `/health` endpoint
- **Error Handling**: Comprehensive error handling with proper HTTP status codes
- **FHIR Compliance**: Validates against FHIR R4 specifications

## Security

- **JWT Authentication**: Via Supabase integration
- **RBAC**: Role-based access control
- **CORS**: Configurable cross-origin resource sharing
- **Input Validation**: Pydantic model validation
- **Secure Headers**: Proper security headers

The Scheduling Service is now ready for integration and testing! 🎉
