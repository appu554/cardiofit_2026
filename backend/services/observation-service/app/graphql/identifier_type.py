from __future__ import annotations

import strawberry
import logging
from typing import Optional, Any, Dict, TYPE_CHECKING

# Configure logging
logger = logging.getLogger(__name__)

if TYPE_CHECKING:
    from .types import Period, CodeableConcept
    from .reference_type import Reference

# Define lazy types for forward references
LazyCodeableConcept = strawberry.LazyType("CodeableConcept", "app.graphql.types")
LazyPeriod = strawberry.LazyType("Period", "app.graphql.types")
LazyReference = strawberry.LazyType("Reference", "app.graphql.reference_type")

@strawberry.type(description="An identifier - identifies some entity uniquely and unambiguously.")
class Identifier:
    use: Optional[str] = strawberry.field(default=None, description="usual | official | temp | secondary | old (If known)")
    type: Optional[LazyCodeableConcept] = strawberry.field(default=None, description="Description of identifier.")
    system: Optional[str] = strawberry.field(default=None, description="The namespace for the identifier value.")
    value: Optional[str] = strawberry.field(default=None, description="The value that is unique.")
    period: Optional[LazyPeriod] = strawberry.field(default=None, description="Time period when id is/was valid for use.")
    assigner: Optional[LazyReference] = strawberry.field(default=None, description="Organization that issued id (may be just text).")

def _identifier_from_fhir_impl(cls, fhir_data: Optional[Dict[str, Any]]) -> Optional[Identifier]:
    if not fhir_data:
        return None
    
    # Local imports for .from_fhir calls to avoid circular dependency at module load time
    from .types import Period, CodeableConcept 
    from .reference_type import Reference

    return cls(
        use=fhir_data.get("use"),
        type=CodeableConcept.from_fhir(fhir_data.get("type")) if fhir_data.get("type") else None,
        system=fhir_data.get("system"),
        value=fhir_data.get("value"),
        period=Period.from_fhir(fhir_data.get("period")) if fhir_data.get("period") else None,
        assigner=Reference.from_fhir(fhir_data.get("assigner")) if fhir_data.get("assigner") else None
    )

Identifier.from_fhir = classmethod(_identifier_from_fhir_impl)
