# Organization Management Service

The Organization Management Service is a comprehensive microservice for managing healthcare organizations within the Clinical Synthesis Hub. It provides FHIR-compliant organization management with Google Healthcare API integration, Supabase authentication, and Apollo Federation support.

## Features

### Core Functionality
- **Organization CRUD Operations**: Create, read, update, and delete healthcare organizations
- **FHIR Compliance**: Full FHIR R4 Organization resource support
- **Google Healthcare API Integration**: Persistent storage using Google Cloud Healthcare API
- **Supabase Authentication**: JWT-based authentication with RBAC
- **Apollo Federation**: GraphQL federation support for distributed schemas

### Organization Management
- **Organization Registration**: Register new healthcare organizations
- **Verification Workflow**: Multi-step verification process for organizations
- **Hierarchy Management**: Parent-child organization relationships
- **Settings Management**: Organization-specific configuration settings
- **User Association**: Link users to organizations with roles and permissions

### Advanced Features
- **Professional Credentials**: Track licenses and certifications
- **Document Management**: Handle verification documents
- **Audit Trail**: Comprehensive logging of all changes
- **Search and Filtering**: Advanced organization search capabilities

## Architecture

```
API Gateway > Auth > Apollo Federation Gateway > Organization Service > Google Healthcare API
```

### Components
- **FastAPI Application**: REST API endpoints
- **GraphQL Federation**: Apollo Federation schema and resolvers
- **Google Healthcare FHIR Service**: FHIR resource operations
- **Organization Management Service**: Business logic layer
- **Authentication Middleware**: Supabase JWT validation with RBAC

## Installation

### Prerequisites
- Python 3.8+
- Google Cloud Healthcare API access
- Supabase account and configuration
- Required Python packages (see requirements.txt)

### Setup Steps

1. **Install Dependencies**
   ```bash
   pip install -r requirements.txt
   ```

2. **Configure Google Healthcare API**
   - Set up Google Cloud project
   - Enable Healthcare API
   - Create service account and download credentials
   - Place credentials in `credentials/service-account-key.json`

3. **Configure Environment Variables**
   ```bash
   export GOOGLE_CLOUD_PROJECT="your-project-id"
   export GOOGLE_CLOUD_LOCATION="us-central1"
   export GOOGLE_CLOUD_DATASET="clinical-synthesis-hub"
   export GOOGLE_CLOUD_FHIR_STORE="organization-store"
   export GOOGLE_APPLICATION_CREDENTIALS="path/to/credentials.json"
   
   export SUPABASE_URL="your-supabase-url"
   export SUPABASE_KEY="your-supabase-key"
   export SUPABASE_JWT_SECRET="your-jwt-secret"
   
   export AUTH_SERVICE_URL="http://localhost:8001"
   ```

4. **Run the Service**
   ```bash
   python run_service.py
   ```

## API Endpoints

### REST API

#### Organizations
- `POST /api/organizations` - Create organization
- `GET /api/organizations/{id}` - Get organization by ID
- `PUT /api/organizations/{id}` - Update organization
- `DELETE /api/organizations/{id}` - Delete organization
- `GET /api/organizations` - Search organizations

#### Verification
- `POST /api/organizations/{id}/verify` - Submit for verification
- `POST /api/organizations/{id}/approve` - Approve organization

### GraphQL Federation

#### Queries
```graphql
query GetOrganization($id: ID!) {
  organization(id: $id) {
    id
    name
    organizationType
    status
    telecom {
      system
      value
      use
    }
    address {
      line
      city
      state
      postalCode
      country
    }
  }
}

query SearchOrganizations($name: String, $type: OrganizationType) {
  organizations(name: $name, organizationType: $type) {
    organizations {
      id
      name
      organizationType
      status
    }
    totalCount
    hasMore
  }
}
```

#### Mutations
```graphql
mutation CreateOrganization($data: OrganizationInput!) {
  createOrganization(organizationData: $data) {
    id
    name
    status
  }
}

mutation UpdateOrganization($id: ID!, $data: OrganizationUpdateInput!) {
  updateOrganization(id: $id, updateData: $data) {
    id
    name
    status
  }
}
```

## Data Models

### Organization
FHIR-compliant organization resource with extensions for:
- Legal and trading names
- Tax ID and license numbers
- Verification status and documents
- Custom organization types
- Audit trail information

### Organization Settings
Key-value configuration storage for organization-specific settings:
- Billing configurations
- UI preferences
- Workflow settings
- Integration parameters

### Organization Relationships
Inter-organization relationships:
- Parent-child hierarchies
- Partnerships and affiliations
- Service relationships
- Contract references

### User-Organization Access
User association with organizations:
- Role assignments
- Professional credentials
- Employment details
- Access periods

## Permissions

The service uses the following RBAC permissions:
- `organization:read` - Read organization information
- `organization:write` - Create and update organizations
- `organization:delete` - Delete organizations
- `organization:verify` - Submit for verification
- `organization:approve` - Approve verification
- `organization:manage_users` - Manage organization users
- `organization:manage_settings` - Manage settings

## Federation Integration

### Apollo Federation Setup
The service provides a federation endpoint at `/api/federation` that exposes the GraphQL schema for Apollo Federation Gateway integration.

### Service Registration
Add to Apollo Federation Gateway service list:
```javascript
{
  name: 'organizations',
  url: 'http://localhost:8012/api/federation'
}
```

## Development

### Running in Development Mode
```bash
python run_service.py
```

### Testing
```bash
pytest tests/
```

### API Documentation
- Swagger UI: http://localhost:8012/docs
- ReDoc: http://localhost:8012/redoc
- GraphQL Playground: http://localhost:8012/api/federation

## Configuration

### Environment Variables
- `ORGANIZATION_SERVICE_PORT`: Service port (default: 8012)
- `ORGANIZATION_SERVICE_HOST`: Service host (default: 0.0.0.0)
- `DEBUG`: Enable debug mode (default: true)
- `GOOGLE_CLOUD_PROJECT`: Google Cloud project ID
- `GOOGLE_CLOUD_LOCATION`: Google Cloud region
- `GOOGLE_CLOUD_DATASET`: Healthcare dataset name
- `GOOGLE_CLOUD_FHIR_STORE`: FHIR store name
- `GOOGLE_APPLICATION_CREDENTIALS`: Path to service account credentials
- `SUPABASE_URL`: Supabase project URL
- `SUPABASE_KEY`: Supabase API key
- `SUPABASE_JWT_SECRET`: JWT secret for token validation
- `AUTH_SERVICE_URL`: Authentication service URL

## Monitoring and Logging

The service provides comprehensive logging and monitoring:
- Structured logging with timestamps
- Health check endpoint at `/health`
- Error tracking and reporting
- Performance metrics (TODO)

## Security

- JWT-based authentication with Supabase
- Role-based access control (RBAC)
- Input validation and sanitization
- Secure credential management
- CORS configuration for web clients

## Troubleshooting

### Common Issues
1. **Google Healthcare API Connection**: Verify credentials and project configuration
2. **Authentication Failures**: Check Supabase JWT secret and token format
3. **Federation Issues**: Ensure schema compatibility and endpoint accessibility
4. **Permission Errors**: Verify user roles and permissions in Supabase

### Logs
Check service logs for detailed error information:
```bash
tail -f organization_service.log
```
