"""
Test configuration and fixtures for Workflow Engine Service.
"""
import asyncio
import os
import sys
from pathlib import Path
from typing import AsyncGenerator, Generator
from unittest.mock import AsyncMock, MagicMock

import pytest
import pytest_asyncio
from fastapi.testclient import TestClient
from sqlalchemy import create_engine
from sqlalchemy.ext.asyncio import AsyncSession, create_async_engine
from sqlalchemy.orm import sessionmaker

# Add the app directory to Python path
sys.path.insert(0, str(Path(__file__).parent.parent / "app"))

# Import core modules with error handling
try:
    from app.core.config import settings
except ImportError:
    # Create mock settings for testing
    class MockSettings:
        SERVICE_NAME = "workflow-engine-service"
        SERVICE_VERSION = "1.0.0"
        SERVICE_PORT = 8015
        DATABASE_URL = "sqlite+aiosqlite:///:memory:"
        SUPABASE_URL = "http://localhost:54321"
        SUPABASE_KEY = "test-key"
        USE_GOOGLE_HEALTHCARE_API = False
    settings = MockSettings()

try:
    from app.db.database import Base, get_db
except ImportError:
    # Create mock database components
    from sqlalchemy.ext.declarative import declarative_base
    Base = declarative_base()
    def get_db():
        pass

# Create a simple FastAPI app for testing (avoid GraphQL imports)
from fastapi import FastAPI
app = FastAPI(title="Workflow Engine Service Test", version="1.0.0")


# Test database URL (use in-memory SQLite for tests)
TEST_DATABASE_URL = "sqlite+aiosqlite:///:memory:"


@pytest.fixture(scope="session")
def event_loop():
    """Create an instance of the default event loop for the test session."""
    loop = asyncio.get_event_loop_policy().new_event_loop()
    yield loop
    loop.close()


@pytest.fixture(scope="session")
async def test_engine():
    """Create test database engine."""
    engine = create_async_engine(
        TEST_DATABASE_URL,
        echo=False,
        future=True
    )

    # Create all tables if Base is available
    if Base is not None:
        async with engine.begin() as conn:
            await conn.run_sync(Base.metadata.create_all)

    yield engine

    # Clean up
    await engine.dispose()


@pytest.fixture
async def test_session(test_engine) -> AsyncGenerator[AsyncSession, None]:
    """Create test database session."""
    async_session = sessionmaker(
        test_engine, class_=AsyncSession, expire_on_commit=False
    )
    
    async with async_session() as session:
        yield session


@pytest.fixture
def test_client(test_session):
    """Create test client with dependency overrides."""

    async def override_get_db():
        yield test_session

    # Only override if get_db is available
    if get_db is not None:
        app.dependency_overrides[get_db] = override_get_db

    with TestClient(app) as client:
        yield client

    # Clean up
    if hasattr(app, 'dependency_overrides'):
        app.dependency_overrides.clear()


@pytest.fixture
def mock_supabase_client():
    """Mock Supabase client."""
    mock_client = MagicMock()
    mock_client.table.return_value.select.return_value.execute.return_value.data = []
    mock_client.table.return_value.insert.return_value.execute.return_value.data = [{"id": "test-id"}]
    mock_client.table.return_value.update.return_value.eq.return_value.execute.return_value.data = [{"id": "test-id"}]
    mock_client.table.return_value.delete.return_value.eq.return_value.execute.return_value.data = []
    return mock_client


@pytest.fixture
def mock_google_fhir_client():
    """Mock Google FHIR client."""
    mock_client = MagicMock()
    mock_client.create_resource.return_value = {"id": "test-resource-id"}
    mock_client.get_resource.return_value = {"id": "test-resource-id", "resourceType": "Task"}
    mock_client.update_resource.return_value = {"id": "test-resource-id"}
    mock_client.delete_resource.return_value = True
    return mock_client


@pytest.fixture
def mock_camunda_client():
    """Mock Camunda client."""
    mock_client = MagicMock()
    mock_client.deploy_workflow.return_value = {"key": "test-workflow", "version": 1}
    mock_client.start_workflow_instance.return_value = {"workflowInstanceKey": "test-instance-123"}
    mock_client.publish_message.return_value = True
    mock_client.complete_job.return_value = True
    return mock_client


@pytest.fixture
def sample_workflow_definition():
    """Sample workflow definition for testing."""
    return {
        "id": "test-workflow-def",
        "name": "Test Workflow",
        "description": "A test workflow definition",
        "version": 1,
        "bpmn_xml": """<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="test-process" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:endEvent id="end"/>
    <bpmn:sequenceFlow id="flow" sourceRef="start" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>""",
        "category": "test",
        "is_active": True
    }


@pytest.fixture
def sample_workflow_instance():
    """Sample workflow instance for testing."""
    return {
        "id": "test-instance-123",
        "definition_id": "test-workflow-def",
        "patient_id": "test-patient-123",
        "status": "active",
        "variables": {"patientName": "John Doe"},
        "camunda_instance_key": "test-instance-123"
    }


@pytest.fixture
def sample_task():
    """Sample task for testing."""
    return {
        "id": "test-task-123",
        "workflow_instance_id": "test-instance-123",
        "name": "Review Patient Data",
        "description": "Review and validate patient data",
        "assignee_id": "test-user-123",
        "status": "ready",
        "form_data": {"patientId": "test-patient-123"},
        "fhir_task_id": "test-fhir-task-123"
    }


@pytest.fixture
def auth_headers():
    """Sample authentication headers."""
    return {
        "X-User-ID": "test-user-123",
        "X-User-Role": "doctor",
        "X-User-Roles": "doctor,admin",
        "X-User-Permissions": "patient:read,patient:write,task:read,task:write"
    }


# Test data factories
class WorkflowDefinitionFactory:
    """Factory for creating test workflow definitions."""
    
    @staticmethod
    def create(**kwargs):
        defaults = {
            "id": "test-workflow-def",
            "name": "Test Workflow",
            "description": "A test workflow definition",
            "version": 1,
            "bpmn_xml": "<bpmn:definitions></bpmn:definitions>",
            "category": "test",
            "is_active": True
        }
        defaults.update(kwargs)
        return defaults


class WorkflowInstanceFactory:
    """Factory for creating test workflow instances."""
    
    @staticmethod
    def create(**kwargs):
        defaults = {
            "id": "test-instance-123",
            "definition_id": "test-workflow-def",
            "patient_id": "test-patient-123",
            "status": "active",
            "variables": {},
            "camunda_instance_key": "test-instance-123"
        }
        defaults.update(kwargs)
        return defaults


class TaskFactory:
    """Factory for creating test tasks."""
    
    @staticmethod
    def create(**kwargs):
        defaults = {
            "id": "test-task-123",
            "workflow_instance_id": "test-instance-123",
            "name": "Test Task",
            "description": "A test task",
            "assignee_id": "test-user-123",
            "status": "ready",
            "form_data": {},
            "fhir_task_id": "test-fhir-task-123"
        }
        defaults.update(kwargs)
        return defaults
