import strawberry
from .queries import Query
from .mutations import Mutation

# FHIR plugin registration removed to bypass FHIR validation
# Direct routing to microservices is now used instead

# Create GraphQL schema
schema = strawberry.Schema(query=Query, mutation=Mutation)