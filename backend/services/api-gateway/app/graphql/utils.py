import logging
from typing import Dict, Any, Optional, TypeVar, Generic, Union, List, Type, get_type_hints

# Import transformer utilities
from .transformer_utils import transform_fhir_to_graphql as dtl_transform_fhir_to_graphql
from .transformer_utils import transform_graphql_to_fhir as dtl_transform_graphql_to_fhir

T = TypeVar('T')

class GraphQLResult(Generic[T]):
    """
    A generic class to represent the result of a GraphQL operation.
    It can either contain a successful result or an error.
    """

    def __init__(
        self,
        success: bool,
        data: Optional[T] = None,
        errors: Optional[List[Dict[str, Any]]] = None,
        message: Optional[str] = None
    ):
        self.success = success
        self.data = data
        self.errors = errors or []
        self.message = message

    @classmethod
    def success_result(cls, data: T, message: Optional[str] = None) -> 'GraphQLResult[T]':
        """Create a successful result."""
        return cls(success=True, data=data, message=message)

    @classmethod
    def error_result(cls, message: str, errors: Optional[List[Dict[str, Any]]] = None) -> 'GraphQLResult[T]':
        """Create an error result."""
        return cls(success=False, errors=errors or [], message=message)

    def to_dict(self) -> Dict[str, Any]:
        """Convert the result to a dictionary."""
        result = {
            "success": self.success
        }

        if self.data is not None:
            result["data"] = self.data

        if self.errors:
            result["errors"] = self.errors

        if self.message:
            result["message"] = self.message

        return result

def log_error(error: Exception, context: Optional[Dict[str, Any]] = None) -> None:
    """
    Log an error with optional context.

    Args:
        error: The exception to log
        context: Optional context information
    """
    logger = logging.getLogger("graphql")

    error_message = f"Error: {str(error)}"
    if context:
        error_message += f" Context: {context}"

    logger.error(error_message, exc_info=True)

async def handle_request(auth_header: Optional[str], request_func, *args, **kwargs) -> Any:
    """
    Handle a GraphQL request with common error handling.

    Args:
        auth_header: The authorization header
        request_func: The function to execute
        args: Positional arguments for the function
        kwargs: Keyword arguments for the function

    Returns:
        The result of the function or None if an error occurs
    """
    if not auth_header:
        return None

    try:
        return await request_func(*args, **kwargs)
    except Exception as e:
        log_error(e, {"args": args, "kwargs": kwargs})
        return None

def convert_fhir_to_graphql(data: Dict[str, Any], target_class: Type[T]) -> T:
    """
    Convert FHIR resource data to a GraphQL type instance.

    Args:
        data: The FHIR resource data
        target_class: The GraphQL type class

    Returns:
        An instance of the target class
    """
    # Use the DTL transformer if available
    result = dtl_transform_fhir_to_graphql(data, target_class)

    # If transformation failed, log the error
    if result is None and data is not None:
        log_error(Exception(f"Failed to transform FHIR data to {target_class.__name__}"), {"data": data})

    return result

def convert_graphql_to_fhir(data: Any, resource_type: str) -> Optional[Dict[str, Any]]:
    """
    Convert GraphQL input data to FHIR format.

    Args:
        data: The GraphQL input data
        resource_type: The FHIR resource type

    Returns:
        FHIR resource data or None if conversion fails
    """
    # Use the DTL transformer if available
    result = dtl_transform_graphql_to_fhir(data, resource_type)

    # If transformation failed, log the error
    if result is None and data is not None:
        log_error(Exception(f"Failed to transform GraphQL data to {resource_type}"), {"data": data})

    return result
