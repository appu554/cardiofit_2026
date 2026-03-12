# Phase 6: Testing & Documentation - Implementation Guide

## Overview

Phase 6 focuses on comprehensive testing and documentation for the Workflow Engine Service. This phase ensures the service is production-ready with thorough test coverage, API documentation, and user guides.

## Implementation Summary

### ✅ Completed Components

#### 1. **Comprehensive Test Suite**
- **Unit Tests**: Complete test coverage for all service classes
  - `tests/unit/test_workflow_definition_service.py`
  - `tests/unit/test_workflow_instance_service.py`
  - `tests/unit/test_task_service.py`
- **Integration Tests**: End-to-end workflow testing
  - `tests/integration/test_graphql_federation.py`
  - `tests/integration/test_end_to_end_workflow.py`
- **Test Configuration**: Proper pytest setup with fixtures
  - `pytest.ini`: Test configuration and markers
  - `tests/conftest.py`: Shared fixtures and test utilities

#### 2. **Postman API Collection**
- **File**: `postman/Workflow_Engine_Service_API.postman_collection.json`
- **Features**:
  - Complete GraphQL query and mutation examples
  - Authentication setup with headers
  - Environment variables for easy testing
  - Health check endpoints
  - Workflow definition management
  - Workflow instance operations
  - Task management operations

#### 3. **API Documentation**
- **File**: `docs/API_DOCUMENTATION.md`
- **Content**:
  - Complete GraphQL schema documentation
  - Query and mutation examples
  - Authentication requirements
  - Error handling guidelines
  - Federation integration details

#### 4. **Workflow Modeling Guide**
- **File**: `docs/WORKFLOW_MODELING_GUIDE.md`
- **Content**:
  - BPMN 2.0 fundamentals
  - Clinical workflow patterns
  - Task assignment strategies
  - Variable management
  - Error handling best practices
  - Deployment procedures

#### 5. **Test Runner Script**
- **File**: `run_tests.py`
- **Features**:
  - Automated test execution
  - Coverage reporting
  - Multiple test categories
  - Dependency installation
  - Comprehensive reporting

## Test Structure

### Directory Layout
```
tests/
├── __init__.py
├── conftest.py                 # Shared fixtures and configuration
├── unit/                       # Unit tests
│   ├── __init__.py
│   ├── test_workflow_definition_service.py
│   ├── test_workflow_instance_service.py
│   └── test_task_service.py
└── integration/                # Integration tests
    ├── __init__.py
    ├── test_graphql_federation.py
    └── test_end_to_end_workflow.py
```

### Test Categories

#### Unit Tests (`@pytest.mark.unit`)
- Test individual service methods in isolation
- Mock external dependencies (Supabase, Google FHIR, Camunda)
- Verify business logic and error handling
- Fast execution with comprehensive coverage

#### Integration Tests (`@pytest.mark.integration`)
- Test GraphQL federation integration
- Verify end-to-end workflow execution
- Test service interactions
- Database and external service integration

#### Federation Tests (`@pytest.mark.federation`)
- Apollo Federation schema validation
- Cross-service entity resolution
- Federation directive testing
- Gateway integration verification

#### Workflow Tests (`@pytest.mark.workflow`)
- Complete workflow lifecycle testing
- Parallel task execution
- Error handling and recovery
- Timeout and escalation scenarios

## Running Tests

### Prerequisites
```bash
# Install dependencies
pip install -r requirements.txt

# Install additional test dependencies
pip install pytest-cov pytest-mock pytest-xdist factory-boy
```

### Test Execution Options

#### 1. Run All Tests
```bash
python run_tests.py
```

#### 2. Run Specific Test Categories
```bash
# Unit tests only
python run_tests.py --unit

# Integration tests only
python run_tests.py --integration

# Federation tests only
python run_tests.py --federation

# Workflow tests only
python run_tests.py --workflow

# Legacy tests only
python run_tests.py --legacy
```

#### 3. Run with Coverage
```bash
python run_tests.py --coverage
```

#### 4. Verbose Output
```bash
python run_tests.py --verbose
```

#### 5. Install Dependencies and Run
```bash
python run_tests.py --install-deps
```

### Direct pytest Commands
```bash
# Run all tests
pytest

# Run unit tests with coverage
pytest tests/unit/ --cov=app --cov-report=html

# Run specific test file
pytest tests/unit/test_workflow_definition_service.py -v

# Run tests with specific markers
pytest -m "unit and not slow"
```

## Test Fixtures and Utilities

### Key Fixtures (from `conftest.py`)

#### Database Fixtures
- `test_engine`: In-memory SQLite database for testing
- `test_session`: Database session for each test
- `test_client`: FastAPI test client with dependency overrides

#### Mock Fixtures
- `mock_supabase_client`: Mocked Supabase client
- `mock_google_fhir_client`: Mocked Google FHIR client
- `mock_camunda_client`: Mocked Camunda client

#### Data Fixtures
- `sample_workflow_definition`: Test workflow definition data
- `sample_workflow_instance`: Test workflow instance data
- `sample_task`: Test task data
- `auth_headers`: Authentication headers for testing

#### Factory Classes
- `WorkflowDefinitionFactory`: Create test workflow definitions
- `WorkflowInstanceFactory`: Create test workflow instances
- `TaskFactory`: Create test tasks

## API Testing with Postman

### Collection Features

#### Environment Variables
```json
{
  "base_url": "http://localhost:8015",
  "federation_url": "http://localhost:4000",
  "auth_token": "your-auth-token",
  "user_id": "test-user-123",
  "user_role": "doctor",
  "patient_id": "test-patient-123"
}
```

#### Pre-request Scripts
- Automatic header injection for authentication
- User context setup
- Environment variable management

#### Test Categories
1. **Health Check**: Service status verification
2. **Workflow Definitions**: CRUD operations for workflow definitions
3. **Workflow Instances**: Workflow execution management
4. **Tasks**: Task management and completion

### Usage Instructions

1. **Import Collection**: Import the JSON file into Postman
2. **Set Environment**: Configure environment variables
3. **Authentication**: Set up auth token in environment
4. **Run Tests**: Execute individual requests or entire collection
5. **Automation**: Use Postman Runner for automated testing

## Documentation Structure

### API Documentation (`API_DOCUMENTATION.md`)
- **GraphQL Schema**: Complete type definitions
- **Queries**: All available queries with examples
- **Mutations**: All mutations with parameters
- **Federation**: Cross-service integration details
- **Authentication**: Security requirements
- **Error Handling**: Error response formats

### Workflow Modeling Guide (`WORKFLOW_MODELING_GUIDE.md`)
- **BPMN Basics**: Core concepts and elements
- **Design Patterns**: Common workflow patterns
- **Clinical Examples**: Healthcare-specific workflows
- **Best Practices**: Design and implementation guidelines
- **Deployment**: Step-by-step deployment process

## Quality Assurance

### Test Coverage Goals
- **Unit Tests**: >90% code coverage
- **Integration Tests**: All major workflows covered
- **Error Scenarios**: All error paths tested
- **Edge Cases**: Boundary conditions verified

### Code Quality
- **Linting**: flake8 compliance
- **Type Hints**: Comprehensive type annotations
- **Documentation**: Docstrings for all public methods
- **Error Handling**: Graceful error management

### Performance Testing
- **Load Testing**: High-volume workflow execution
- **Stress Testing**: Resource limitation scenarios
- **Concurrency Testing**: Parallel workflow execution
- **Memory Testing**: Memory leak detection

## Continuous Integration

### GitHub Actions (Recommended)
```yaml
name: Workflow Engine Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Set up Python
        uses: actions/setup-python@v2
        with:
          python-version: 3.11
      - name: Install dependencies
        run: |
          pip install -r requirements.txt
          pip install pytest-cov pytest-mock pytest-xdist
      - name: Run tests
        run: python run_tests.py --coverage
      - name: Upload coverage
        uses: codecov/codecov-action@v1
```

## Monitoring and Alerting

### Test Metrics
- **Test Execution Time**: Monitor for performance regression
- **Test Success Rate**: Track test reliability
- **Coverage Trends**: Ensure coverage doesn't decrease
- **Flaky Tests**: Identify and fix unstable tests

### Production Monitoring
- **Health Checks**: Automated service health monitoring
- **Performance Metrics**: Response time and throughput
- **Error Rates**: Track and alert on error increases
- **Resource Usage**: Memory and CPU monitoring

## Next Steps

### Phase 7: Deployment Preparation
1. **Docker Configuration**: Container setup for deployment
2. **Environment Configuration**: Production environment setup
3. **Security Hardening**: Security best practices implementation
4. **Performance Optimization**: Production performance tuning

### Phase 8: Production Deployment
1. **Infrastructure Setup**: Production infrastructure provisioning
2. **Deployment Pipeline**: CI/CD pipeline implementation
3. **Monitoring Setup**: Production monitoring configuration
4. **Documentation Finalization**: Complete user and admin guides

## Troubleshooting

### Common Test Issues

#### 1. Database Connection Errors
```bash
# Check database configuration
python test_db_connection.py

# Reset test database
rm test.db
pytest tests/unit/test_workflow_definition_service.py
```

#### 2. Mock Configuration Issues
```python
# Verify mock setup in conftest.py
# Check patch decorators in test files
# Ensure mock return values match expected format
```

#### 3. Federation Test Failures
```bash
# Check Apollo Federation Gateway is running
curl http://localhost:4000/health

# Verify federation schema
python test_federation.py
```

#### 4. Async Test Issues
```python
# Ensure proper async/await usage
# Check pytest-asyncio configuration
# Verify event loop setup
```

### Performance Issues
- **Slow Tests**: Optimize database operations and mocks
- **Memory Leaks**: Check for proper cleanup in fixtures
- **Timeout Issues**: Increase timeout values for slow operations

## Success Criteria

Phase 6 is considered complete when:

✅ **Test Coverage**: >90% code coverage achieved
✅ **Test Reliability**: All tests pass consistently
✅ **Documentation**: Complete API and user documentation
✅ **Postman Collection**: Comprehensive API testing collection
✅ **Automation**: Automated test execution pipeline
✅ **Quality Gates**: Code quality standards met
✅ **Performance**: Tests execute within acceptable time limits
✅ **Integration**: Federation and service integration verified

## Conclusion

Phase 6 establishes a robust testing and documentation foundation for the Workflow Engine Service. The comprehensive test suite ensures reliability and maintainability, while the detailed documentation enables effective usage and integration by development teams and end users.
