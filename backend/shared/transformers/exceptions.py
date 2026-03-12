"""
Exceptions for the transformers package.
"""

class TransformationError(Exception):
    """
    Exception raised when a transformation fails.
    
    Attributes:
        message -- explanation of the error
        source_data -- the source data that caused the error
        target_type -- the target type that was being transformed to
        details -- additional details about the error
    """
    
    def __init__(self, message, source_data=None, target_type=None, details=None):
        self.message = message
        self.source_data = source_data
        self.target_type = target_type
        self.details = details
        super().__init__(self.message)
    
    def __str__(self):
        result = self.message
        if self.source_data is not None:
            result += f"\nSource data: {self.source_data}"
        if self.target_type is not None:
            result += f"\nTarget type: {self.target_type}"
        if self.details is not None:
            result += f"\nDetails: {self.details}"
        return result


class ValidationError(TransformationError):
    """
    Exception raised when validation fails during transformation.
    
    Attributes:
        message -- explanation of the error
        source_data -- the source data that caused the error
        target_type -- the target type that was being transformed to
        validation_errors -- list of validation errors
    """
    
    def __init__(self, message, source_data=None, target_type=None, validation_errors=None):
        super().__init__(message, source_data, target_type)
        self.validation_errors = validation_errors or []
    
    def __str__(self):
        result = super().__str__()
        if self.validation_errors:
            result += "\nValidation errors:"
            for error in self.validation_errors:
                result += f"\n  - {error}"
        return result
