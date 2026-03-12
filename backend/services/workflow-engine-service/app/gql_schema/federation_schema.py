"""
Apollo Federation schema for Workflow Engine Service.

Now includes Strategic Orchestration (Calculate > Validate > Commit pattern)
alongside traditional workflow management capabilities.
"""
import strawberry
from app.gql_schema.types import (
    Patient, User, Task, WorkflowDefinition, WorkflowInstance_Summary,
    CodeableConcept, Coding, Reference, Identifier, Period
)
from app.gql_schema.queries import WorkflowQuery
from app.gql_schema.mutations import WorkflowMutation
from app.gql_schema.strategic_orchestration_schema import (
    Query as StrategicQuery, 
    Mutation as StrategicMutation
)


@strawberry.type
class Query(WorkflowQuery, StrategicQuery):
    """
    Root query type combining all workflow queries and strategic orchestration.
    """
    pass


@strawberry.type
class Mutation(WorkflowMutation, StrategicMutation):
    """
    Root mutation type combining all workflow mutations and strategic orchestration.
    """
    pass


# Create the federated schema
schema = strawberry.federation.Schema(
    query=Query,
    mutation=Mutation,
    types=[
        Patient,  # Federation extension
        User,     # Federation extension
        Task,     # Core workflow type with @key directive
        WorkflowDefinition,  # Workflow definition type
        WorkflowInstance_Summary,  # Workflow instance summary type
        CodeableConcept, Coding, Reference, Identifier, Period,  # Shared FHIR types
    ],
    enable_federation_2=True
)
