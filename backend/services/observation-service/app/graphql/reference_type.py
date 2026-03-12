from __future__ import annotations

import strawberry
import logging
from typing import Optional, Any, Dict, TYPE_CHECKING

# Configure logging
logger = logging.getLogger(__name__)

# if TYPE_CHECKING:
#     from .identifier_type import Identifier # Replaced with LazyType for schema, direct import in from_fhir remains

LazyIdentifier = strawberry.LazyType("Identifier", "app.graphql.identifier_type")

@strawberry.type(description="A reference from one resource to another.")
class Reference:
    reference: Optional[str] = strawberry.field(default=None, description="Literal reference, Relative, internal or absolute URL.")
    type: Optional[str] = strawberry.field(default=None, description="Type the reference refers to (e.g. 'Patient').")
    identifier: Optional[LazyIdentifier] = strawberry.field(default=None, description="Logical reference, when literal reference is not known.")
    display: Optional[str] = strawberry.field(default=None, description="Text alternative for the resource.")

def _reference_from_fhir_impl(cls, fhir_data: Optional[Dict[str, Any]]) -> Optional[Reference]:
    if not fhir_data:
        return None
        
    # Local import for .from_fhir calls
    from .identifier_type import Identifier
        
    return cls(
        reference=fhir_data.get("reference"),
        type=fhir_data.get("type"),
        identifier=Identifier.from_fhir(fhir_data.get("identifier")) if fhir_data.get("identifier") else None,
        display=fhir_data.get("display")
    )

Reference.from_fhir = classmethod(_reference_from_fhir_impl)
