"""
Minimal workflow service for testing basic functionality
"""
import strawberry
from fastapi import FastAPI
from strawberry.fastapi import GraphQLRouter
from typing import List, Optional

# Simple types for testing
@strawberry.type
class WorkflowDefinition:
    id: str
    name: str
    version: str
    status: str
    description: Optional[str] = None

@strawberry.type
class Task:
    id: str
    description: str
    status: str
    priority: str

@strawberry.type
class Query:
    @strawberry.field
    def hello(self) -> str:
        return "Hello from Workflow Service!"
    
    @strawberry.field
    def workflow_definitions(self) -> List[WorkflowDefinition]:
        """Get sample workflow definitions."""
        return [
            WorkflowDefinition(
                id="patient-admission-workflow",
                name="Patient Admission Workflow",
                version="1.0",
                status="active",
                description="Complete patient admission process"
            ),
            WorkflowDefinition(
                id="medication-review-workflow", 
                name="Medication Review Workflow",
                version="1.0",
                status="active",
                description="Review patient medications for interactions"
            )
        ]
    
    @strawberry.field
    def tasks(self, assignee: Optional[str] = None) -> List[Task]:
        """Get sample tasks."""
        return [
            Task(
                id="task-1",
                description="Review patient admission data",
                status="ready",
                priority="high"
            ),
            Task(
                id="task-2", 
                description="Assign room to patient",
                status="ready",
                priority="normal"
            )
        ]

@strawberry.type
class Mutation:
    @strawberry.field
    def start_workflow(self, definition_id: str, patient_id: str) -> str:
        """Start a workflow (mock implementation)."""
        return f"Started workflow {definition_id} for patient {patient_id}"

# Create schema
schema = strawberry.Schema(query=Query, mutation=Mutation)

# Create FastAPI app
app = FastAPI(
    title="Minimal Workflow Service",
    description="Basic workflow service for testing",
    version="1.0.0"
)

# Add GraphQL router
graphql_router = GraphQLRouter(schema)
app.include_router(graphql_router, prefix="/api/federation")

@app.get("/")
async def root():
    return {
        "message": "Minimal Workflow Service",
        "endpoints": {
            "graphql": "/api/federation",
            "health": "/health"
        }
    }

@app.get("/health")
async def health():
    return {
        "status": "healthy",
        "service": "minimal-workflow-service",
        "version": "1.0.0"
    }

if __name__ == "__main__":
    import uvicorn
    print("🚀 Starting Minimal Workflow Service...")
    print("📍 GraphQL endpoint: http://localhost:8015/api/federation")
    print("🏥 Health check: http://localhost:8015/health")
    
    uvicorn.run(
        "minimal_service:app",
        host="0.0.0.0",
        port=8015,
        reload=True
    )
