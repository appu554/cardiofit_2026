"""
Enhanced Workflow Service with Patient, Vitals, and Encounter Integration
"""
import strawberry
from fastapi import FastAPI
from strawberry.fastapi import GraphQLRouter
from typing import List, Optional
from datetime import datetime, timedelta
import json

# Enhanced types with patient integration
@strawberry.type
class Patient:
    id: str
    name: str
    gender: str
    birth_date: str
    active: bool

@strawberry.type
class Vital:
    id: str
    patient_id: str
    code: str
    display: str
    value: float
    unit: str
    recorded_time: str
    status: str

@strawberry.type
class Encounter:
    id: str
    patient_id: str
    status: str
    encounter_class: str
    type_display: str
    start_time: str
    end_time: Optional[str] = None
    location: Optional[str] = None

@strawberry.type
class WorkflowDefinition:
    id: str
    name: str
    version: str
    status: str
    category: str
    description: Optional[str] = None
    triggers: List[str]

@strawberry.type
class Task:
    id: str
    workflow_instance_id: str
    patient_id: str
    description: str
    status: str
    priority: str
    assignee: Optional[str] = None
    due_date: Optional[str] = None
    created_at: str
    context: Optional[str] = None

@strawberry.type
class WorkflowInstance:
    id: str
    definition_id: str
    patient_id: str
    status: str
    start_time: str
    end_time: Optional[str] = None
    variables: Optional[str] = None
    current_step: str

# Sample data
SAMPLE_PATIENTS = [
    Patient(
        id="patient-001",
        name="John Doe",
        gender="male",
        birth_date="1985-03-15",
        active=True
    ),
    Patient(
        id="patient-002", 
        name="Jane Smith",
        gender="female",
        birth_date="1992-07-22",
        active=True
    )
]

SAMPLE_VITALS = [
    Vital(
        id="vital-001",
        patient_id="patient-001",
        code="8480-6",
        display="Systolic Blood Pressure",
        value=140.0,
        unit="mmHg",
        recorded_time="2024-01-15T10:30:00Z",
        status="final"
    ),
    Vital(
        id="vital-002",
        patient_id="patient-001", 
        code="8462-4",
        display="Diastolic Blood Pressure",
        value=90.0,
        unit="mmHg",
        recorded_time="2024-01-15T10:30:00Z",
        status="final"
    ),
    Vital(
        id="vital-003",
        patient_id="patient-001",
        code="8867-4",
        display="Heart Rate",
        value=85.0,
        unit="beats/min",
        recorded_time="2024-01-15T10:30:00Z",
        status="final"
    )
]

SAMPLE_ENCOUNTERS = [
    Encounter(
        id="encounter-001",
        patient_id="patient-001",
        status="in-progress",
        encounter_class="inpatient",
        type_display="Emergency Department Visit",
        start_time="2024-01-15T09:00:00Z",
        location="Emergency Department - Room 3"
    )
]

SAMPLE_WORKFLOWS = [
    WorkflowDefinition(
        id="patient-admission-workflow",
        name="Patient Admission Workflow",
        version="1.0",
        status="active",
        category="admission",
        description="Complete patient admission process with vitals monitoring",
        triggers=["patient-arrival", "emergency-admission"]
    ),
    WorkflowDefinition(
        id="vitals-monitoring-workflow",
        name="Vitals Monitoring Workflow", 
        version="1.0",
        status="active",
        category="monitoring",
        description="Continuous monitoring of patient vital signs",
        triggers=["abnormal-vitals", "high-risk-patient"]
    ),
    WorkflowDefinition(
        id="discharge-planning-workflow",
        name="Discharge Planning Workflow",
        version="1.0", 
        status="active",
        category="discharge",
        description="Coordinate patient discharge with medication reconciliation",
        triggers=["discharge-order", "recovery-complete"]
    )
]

SAMPLE_TASKS = [
    Task(
        id="task-001",
        workflow_instance_id="workflow-inst-001",
        patient_id="patient-001",
        description="Review abnormal blood pressure readings",
        status="ready",
        priority="high",
        assignee="doctor-123",
        due_date="2024-01-15T12:00:00Z",
        created_at="2024-01-15T10:35:00Z",
        context="BP: 140/90 mmHg - requires immediate attention"
    ),
    Task(
        id="task-002",
        workflow_instance_id="workflow-inst-001", 
        patient_id="patient-001",
        description="Assign ICU bed for monitoring",
        status="ready",
        priority="urgent",
        assignee="nurse-456",
        due_date="2024-01-15T11:00:00Z",
        created_at="2024-01-15T10:40:00Z",
        context="Patient requires continuous monitoring due to hypertension"
    )
]

SAMPLE_WORKFLOW_INSTANCES = [
    WorkflowInstance(
        id="workflow-inst-001",
        definition_id="vitals-monitoring-workflow",
        patient_id="patient-001",
        status="active",
        start_time="2024-01-15T10:35:00Z",
        variables='{"trigger": "abnormal-vitals", "priority": "high"}',
        current_step="medical-review"
    )
]

@strawberry.type
class Query:
    @strawberry.field
    def hello(self) -> str:
        return "Enhanced Workflow Service with Patient Integration!"
    
    @strawberry.field
    def patients(self) -> List[Patient]:
        """Get all patients."""
        return SAMPLE_PATIENTS
    
    @strawberry.field
    def patient(self, id: str) -> Optional[Patient]:
        """Get patient by ID."""
        return next((p for p in SAMPLE_PATIENTS if p.id == id), None)
    
    @strawberry.field
    def vitals(self, patient_id: Optional[str] = None) -> List[Vital]:
        """Get vitals, optionally filtered by patient."""
        if patient_id:
            return [v for v in SAMPLE_VITALS if v.patient_id == patient_id]
        return SAMPLE_VITALS
    
    @strawberry.field
    def encounters(self, patient_id: Optional[str] = None) -> List[Encounter]:
        """Get encounters, optionally filtered by patient."""
        if patient_id:
            return [e for e in SAMPLE_ENCOUNTERS if e.patient_id == patient_id]
        return SAMPLE_ENCOUNTERS
    
    @strawberry.field
    def workflow_definitions(self, category: Optional[str] = None) -> List[WorkflowDefinition]:
        """Get workflow definitions, optionally filtered by category."""
        if category:
            return [w for w in SAMPLE_WORKFLOWS if w.category == category]
        return SAMPLE_WORKFLOWS
    
    @strawberry.field
    def tasks(self, patient_id: Optional[str] = None, assignee: Optional[str] = None, status: Optional[str] = None) -> List[Task]:
        """Get tasks with optional filters."""
        tasks = SAMPLE_TASKS
        if patient_id:
            tasks = [t for t in tasks if t.patient_id == patient_id]
        if assignee:
            tasks = [t for t in tasks if t.assignee == assignee]
        if status:
            tasks = [t for t in tasks if t.status == status]
        return tasks
    
    @strawberry.field
    def workflow_instances(self, patient_id: Optional[str] = None, status: Optional[str] = None) -> List[WorkflowInstance]:
        """Get workflow instances with optional filters."""
        instances = SAMPLE_WORKFLOW_INSTANCES
        if patient_id:
            instances = [i for i in instances if i.patient_id == patient_id]
        if status:
            instances = [i for i in instances if i.status == status]
        return instances
    
    @strawberry.field
    def patient_summary(self, patient_id: str) -> Optional[str]:
        """Get comprehensive patient summary with workflows, vitals, and encounters."""
        patient = next((p for p in SAMPLE_PATIENTS if p.id == patient_id), None)
        if not patient:
            return None
        
        vitals = [v for v in SAMPLE_VITALS if v.patient_id == patient_id]
        encounters = [e for e in SAMPLE_ENCOUNTERS if e.patient_id == patient_id]
        tasks = [t for t in SAMPLE_TASKS if t.patient_id == patient_id]
        workflows = [w for w in SAMPLE_WORKFLOW_INSTANCES if w.patient_id == patient_id]
        
        summary = {
            "patient": {
                "name": patient.name,
                "id": patient.id,
                "gender": patient.gender
            },
            "current_encounter": encounters[0].__dict__ if encounters else None,
            "latest_vitals": [v.__dict__ for v in vitals],
            "active_tasks": [t.__dict__ for t in tasks if t.status == "ready"],
            "active_workflows": [w.__dict__ for w in workflows if w.status == "active"]
        }
        
        return json.dumps(summary, indent=2)

@strawberry.type  
class Mutation:
    @strawberry.field
    def start_workflow(self, definition_id: str, patient_id: str, trigger: Optional[str] = None) -> str:
        """Start a workflow for a patient."""
        workflow = next((w for w in SAMPLE_WORKFLOWS if w.id == definition_id), None)
        patient = next((p for p in SAMPLE_PATIENTS if p.id == patient_id), None)
        
        if not workflow or not patient:
            return "Error: Workflow or patient not found"
        
        instance_id = f"workflow-inst-{len(SAMPLE_WORKFLOW_INSTANCES) + 1:03d}"
        
        # Create new workflow instance
        new_instance = WorkflowInstance(
            id=instance_id,
            definition_id=definition_id,
            patient_id=patient_id,
            status="active",
            start_time=datetime.now().isoformat(),
            variables=f'{{"trigger": "{trigger or "manual"}"}}',
            current_step="initial"
        )
        
        SAMPLE_WORKFLOW_INSTANCES.append(new_instance)
        
        return f"Started {workflow.name} for patient {patient.name} (Instance: {instance_id})"
    
    @strawberry.field
    def complete_task(self, task_id: str, result: Optional[str] = None) -> str:
        """Complete a task."""
        task = next((t for t in SAMPLE_TASKS if t.id == task_id), None)
        if not task:
            return "Error: Task not found"
        
        task.status = "completed"
        return f"Task '{task.description}' completed with result: {result or 'No result provided'}"

# Create schema
schema = strawberry.Schema(query=Query, mutation=Mutation)

# Create FastAPI app
app = FastAPI(
    title="Enhanced Workflow Service",
    description="Workflow service with patient, vitals, and encounter integration",
    version="2.0.0"
)

# Add GraphQL router
graphql_router = GraphQLRouter(schema)
app.include_router(graphql_router, prefix="/api/federation")

@app.get("/")
async def root():
    return {
        "message": "Enhanced Workflow Service with Patient Integration",
        "features": [
            "Patient data integration",
            "Vitals monitoring workflows", 
            "Encounter-based task creation",
            "Comprehensive patient summaries"
        ],
        "endpoints": {
            "graphql": "/api/federation",
            "health": "/health"
        }
    }

@app.get("/health")
async def health():
    return {
        "status": "healthy",
        "service": "enhanced-workflow-service",
        "version": "2.0.0",
        "features": {
            "patients": len(SAMPLE_PATIENTS),
            "vitals": len(SAMPLE_VITALS),
            "encounters": len(SAMPLE_ENCOUNTERS),
            "workflows": len(SAMPLE_WORKFLOWS),
            "active_tasks": len([t for t in SAMPLE_TASKS if t.status == "ready"])
        }
    }

if __name__ == "__main__":
    import uvicorn
    print("🚀 Starting Enhanced Workflow Service...")
    print("📍 GraphQL endpoint: http://localhost:8015/api/federation")
    print("🏥 Health check: http://localhost:8015/health")
    print("🧬 Features: Patient data, Vitals, Encounters, Workflows")
    
    uvicorn.run(
        "enhanced_workflow_service:app",
        host="0.0.0.0",
        port=8015,
        reload=True
    )
