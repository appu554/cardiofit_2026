# Patient Service - Startup Guide

## Overview
The Patient Service is a Python FastAPI-based microservice that handles FHIR-compliant patient data management. It provides RESTful APIs for patient operations and integrates with shared authentication and FHIR resources.

## Prerequisites

### Required Software
- **Python 3.11+** - Runtime environment
- **pip3** - Package manager
- **Virtual Environment** - Isolated Python environment (recommended)

### System Dependencies
- Access to shared Python modules in `backend/shared/`
- Network access for external API integrations

## Quick Start

### 1. Navigate to Service Directory
```bash
cd backend/services/patient-service
```

### 2. Set Up Python Virtual Environment (Recommended)
```bash
# Create virtual environment
python3 -m venv venv

# Activate virtual environment
# On macOS/Linux:
source venv/bin/activate
# On Windows:
# venv\Scripts\activate
```

### 3. Install Dependencies
```bash
# Install required packages
pip3 install -r requirements.txt

# Install additional dependencies if needed
pip3 install requests fhir.resources eval_type_backport
```

### 4. Start the Service
```bash
# Use the provided startup script (recommended)
python3 run_service.py

# Alternative: Direct uvicorn (not recommended)
# uvicorn main:app --host 0.0.0.0 --port 8003
```

## Service Configuration

### Default Settings
- **Port**: 8003
- **Host**: 0.0.0.0 (all interfaces)
- **Environment**: Development
- **Shared Modules**: Auto-configured via `run_service.py`

### Environment Variables
```bash
export PATIENT_SERVICE_PORT=8003
export ENVIRONMENT=development
export LOG_LEVEL=info
```

## Health Check Verification

### Test Service Health
```bash
# Check if service is running
curl http://localhost:8003/health

# Expected response:
# {"status":"healthy"}
```

### Test Service Endpoints
```bash
# Test patient endpoints
curl http://localhost:8003/api/v1/patients

# Test specific patient data
curl http://localhost:8003/api/v1/patients/{patient-id}
```

## Service Architecture

### Key Components
- **FastAPI Application**: Modern Python web framework
- **FHIR Resources**: Healthcare data standards compliance
- **Shared Modules**: Authentication, validation, and common utilities
- **Request Handling**: RESTful API endpoints for patient operations

### Shared Module Integration
The service automatically includes:
- `backend/shared/` - Common utilities and FHIR models
- Authentication middleware
- Request validation
- Error handling patterns

## Troubleshooting

### Common Issues

#### **Missing Dependencies**
```bash
# Error: Module not found
# Solution: Install missing packages
pip3 install requests fhir.resources eval_type_backport
```

#### **Port Already in Use**
```bash
# Error: Address already in use
# Solution: Kill existing process or change port
lsof -ti:8003 | xargs kill -9
# OR
export PATIENT_SERVICE_PORT=8004
```

#### **Shared Module Import Error**
```bash
# Error: Cannot import shared modules
# Solution: Use run_service.py (not direct uvicorn)
python3 run_service.py
```

#### **Virtual Environment Issues**
```bash
# Recreate virtual environment
rm -rf venv
python3 -m venv venv
source venv/bin/activate
pip3 install -r requirements.txt
```

## Development Workflow

### Local Development
```bash
# 1. Activate virtual environment
source venv/bin/activate

# 2. Start service with auto-reload
python3 run_service.py

# 3. Test changes
curl http://localhost:8003/health
```

### Integration Testing
```bash
# Test with other services
curl http://localhost:8003/health  # Patient Service
curl http://localhost:8005/health  # Medication Service
curl http://localhost:8117/health  # Context Gateway
```

## Service Endpoints

### Health & Status
- **GET** `/health` - Service health check
- **GET** `/` - Service information

### Patient Operations
- **GET** `/api/v1/patients` - List patients
- **GET** `/api/v1/patients/{id}` - Get specific patient
- **POST** `/api/v1/patients` - Create new patient
- **PUT** `/api/v1/patients/{id}` - Update patient
- **DELETE** `/api/v1/patients/{id}` - Delete patient

## Integration Notes

### With Other Services
- **Authentication**: Uses shared authentication middleware
- **FHIR Compliance**: Integrates with FHIR resource validation
- **API Gateway**: Ready for Apollo Federation integration
- **Monitoring**: Health endpoints for service discovery

### Database Integration
- Connects to MongoDB for patient data persistence
- Uses shared database configuration patterns
- Implements FHIR-compliant data models

## Production Considerations

### Deployment
```bash
# Use production WSGI server
pip3 install gunicorn
gunicorn -w 4 -k uvicorn.workers.UvicornWorker main:app --bind 0.0.0.0:8003
```

### Security
- Enable authentication middleware in production
- Configure CORS policies appropriately
- Use HTTPS for all external communications

### Monitoring
- Monitor `/health` endpoint for service availability
- Implement logging for audit trails
- Set up performance metrics collection

## Support

### Logs Location
- **Development**: Console output
- **Production**: Configure file-based logging

### Common Commands
```bash
# Start service
python3 run_service.py

# Check process
ps aux | grep "python3 run_service.py"

# Stop service
pkill -f "python3 run_service.py"

# View logs (if file-based)
tail -f logs/patient-service.log
```