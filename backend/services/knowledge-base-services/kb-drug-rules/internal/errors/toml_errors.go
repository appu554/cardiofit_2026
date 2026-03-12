package errors

import (
	"fmt"
	"net/http"
	"time"
)

// TOMLError represents a TOML-specific error with context
type TOMLError struct {
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Line        int                    `json:"line,omitempty"`
	Column      int                    `json:"column,omitempty"`
	HTTPStatus  int                    `json:"-"`
	Suggestions []string               `json:"suggestions,omitempty"`
}

// Error implements the error interface
func (e *TOMLError) Error() string {
	if e.Line > 0 && e.Column > 0 {
		return fmt.Sprintf("%s at line %d, column %d: %s", e.Code, e.Line, e.Column, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// WithDetails adds details to the error
func (e *TOMLError) WithDetails(details map[string]interface{}) *TOMLError {
	e.Details = details
	return e
}

// WithSuggestions adds suggestions to the error
func (e *TOMLError) WithSuggestions(suggestions []string) *TOMLError {
	e.Suggestions = suggestions
	return e
}

// WithLocation adds line and column information
func (e *TOMLError) WithLocation(line, column int) *TOMLError {
	e.Line = line
	e.Column = column
	return e
}

// Predefined TOML errors
var (
	// Syntax Errors
	ErrTOMLSyntax = &TOMLError{
		Code:       "TOML_SYNTAX_ERROR",
		Message:    "Invalid TOML syntax",
		HTTPStatus: http.StatusBadRequest,
		Suggestions: []string{
			"Check for missing quotes around strings",
			"Verify proper table and array syntax",
			"Ensure proper escaping of special characters",
		},
	}

	ErrTOMLInvalidKey = &TOMLError{
		Code:       "TOML_INVALID_KEY",
		Message:    "Invalid key format in TOML",
		HTTPStatus: http.StatusBadRequest,
		Suggestions: []string{
			"Keys must be valid identifiers or quoted strings",
			"Avoid special characters in unquoted keys",
		},
	}

	ErrTOMLInvalidValue = &TOMLError{
		Code:       "TOML_INVALID_VALUE",
		Message:    "Invalid value format in TOML",
		HTTPStatus: http.StatusBadRequest,
		Suggestions: []string{
			"Check date/time format (RFC 3339)",
			"Verify numeric values are properly formatted",
			"Ensure boolean values are 'true' or 'false'",
		},
	}

	// Validation Errors
	ErrTOMLMissingRequired = &TOMLError{
		Code:       "TOML_MISSING_REQUIRED_FIELD",
		Message:    "Required field is missing",
		HTTPStatus: http.StatusBadRequest,
		Suggestions: []string{
			"Add the required field to your TOML",
			"Check the field name spelling",
		},
	}

	ErrTOMLInvalidSchema = &TOMLError{
		Code:       "TOML_SCHEMA_VIOLATION",
		Message:    "TOML content violates expected schema",
		HTTPStatus: http.StatusBadRequest,
		Suggestions: []string{
			"Review the expected schema documentation",
			"Validate field types and structure",
		},
	}

	ErrTOMLClinicalValidation = &TOMLError{
		Code:       "TOML_CLINICAL_VALIDATION_FAILED",
		Message:    "Clinical validation rules failed",
		HTTPStatus: http.StatusBadRequest,
		Suggestions: []string{
			"Review clinical guidelines",
			"Check dose limits and safety parameters",
			"Verify evidence sources and references",
		},
	}

	// Conversion Errors
	ErrTOMLConversionFailed = &TOMLError{
		Code:       "TOML_CONVERSION_FAILED",
		Message:    "Failed to convert TOML to JSON",
		HTTPStatus: http.StatusInternalServerError,
		Suggestions: []string{
			"Check for unsupported TOML features",
			"Verify data types are JSON-compatible",
		},
	}

	ErrJSONConversionFailed = &TOMLError{
		Code:       "JSON_CONVERSION_FAILED",
		Message:    "Failed to convert JSON to TOML",
		HTTPStatus: http.StatusInternalServerError,
		Suggestions: []string{
			"Ensure JSON structure is TOML-compatible",
			"Check for complex nested structures",
		},
	}

	// Version Management Errors
	ErrVersionNotFound = &TOMLError{
		Code:       "VERSION_NOT_FOUND",
		Message:    "Requested version not found",
		HTTPStatus: http.StatusNotFound,
		Suggestions: []string{
			"Check the version number format",
			"Verify the version exists in the database",
		},
	}

	ErrVersionConflict = &TOMLError{
		Code:       "VERSION_CONFLICT",
		Message:    "Version already exists",
		HTTPStatus: http.StatusConflict,
		Suggestions: []string{
			"Use a different version number",
			"Consider updating the existing version",
		},
	}

	ErrInvalidVersionFormat = &TOMLError{
		Code:       "INVALID_VERSION_FORMAT",
		Message:    "Version format is invalid",
		HTTPStatus: http.StatusBadRequest,
		Suggestions: []string{
			"Use semantic versioning (e.g., 1.0.0)",
			"Ensure version contains only numbers and dots",
		},
	}

	// Database Errors
	ErrDatabaseSave = &TOMLError{
		Code:       "DATABASE_SAVE_FAILED",
		Message:    "Failed to save to database",
		HTTPStatus: http.StatusInternalServerError,
		Suggestions: []string{
			"Check database connectivity",
			"Verify data constraints are met",
		},
	}

	ErrDatabaseQuery = &TOMLError{
		Code:       "DATABASE_QUERY_FAILED",
		Message:    "Database query failed",
		HTTPStatus: http.StatusInternalServerError,
		Suggestions: []string{
			"Check database connectivity",
			"Verify query parameters",
		},
	}

	// Authentication/Authorization Errors
	ErrUnauthorized = &TOMLError{
		Code:       "UNAUTHORIZED",
		Message:    "Authentication required",
		HTTPStatus: http.StatusUnauthorized,
		Suggestions: []string{
			"Provide valid authentication credentials",
			"Check API key or token",
		},
	}

	ErrForbidden = &TOMLError{
		Code:       "FORBIDDEN",
		Message:    "Insufficient permissions",
		HTTPStatus: http.StatusForbidden,
		Suggestions: []string{
			"Contact administrator for required permissions",
			"Verify user role and access rights",
		},
	}
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Success   bool                   `json:"success"`
	Error     *TOMLError             `json:"error"`
	RequestID string                 `json:"request_id,omitempty"`
	Timestamp string                 `json:"timestamp"`
	Path      string                 `json:"path,omitempty"`
	Method    string                 `json:"method,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(err *TOMLError, requestID, path, method string) *ErrorResponse {
	return &ErrorResponse{
		Success:   false,
		Error:     err,
		RequestID: requestID,
		Timestamp: fmt.Sprintf("%d", time.Now().Unix()),
		Path:      path,
		Method:    method,
	}
}

// ValidationErrorCollection represents multiple validation errors
type ValidationErrorCollection struct {
	Errors   []*TOMLError `json:"errors"`
	Warnings []*TOMLError `json:"warnings"`
	Count    int          `json:"count"`
}

// AddError adds an error to the collection
func (vec *ValidationErrorCollection) AddError(err *TOMLError) {
	vec.Errors = append(vec.Errors, err)
	vec.Count++
}

// AddWarning adds a warning to the collection
func (vec *ValidationErrorCollection) AddWarning(warning *TOMLError) {
	vec.Warnings = append(vec.Warnings, warning)
}

// HasErrors returns true if there are any errors
func (vec *ValidationErrorCollection) HasErrors() bool {
	return len(vec.Errors) > 0
}

// HasWarnings returns true if there are any warnings
func (vec *ValidationErrorCollection) HasWarnings() bool {
	return len(vec.Warnings) > 0
}

// Error implements the error interface
func (vec *ValidationErrorCollection) Error() string {
	if len(vec.Errors) == 0 {
		return "no errors"
	}
	if len(vec.Errors) == 1 {
		return vec.Errors[0].Error()
	}
	return fmt.Sprintf("%d validation errors occurred", len(vec.Errors))
}

// Helper functions for creating specific errors

// NewTOMLSyntaxError creates a new TOML syntax error with location
func NewTOMLSyntaxError(message string, line, column int) *TOMLError {
	return &TOMLError{
		Code:       "TOML_SYNTAX_ERROR",
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
		Line:       line,
		Column:     column,
		Suggestions: []string{
			"Check TOML syntax at the specified location",
			"Verify proper quoting and escaping",
		},
	}
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *TOMLError {
	return &TOMLError{
		Code:       "VALIDATION_ERROR",
		Message:    fmt.Sprintf("Validation failed for field '%s': %s", field, message),
		HTTPStatus: http.StatusBadRequest,
		Details: map[string]interface{}{
			"field": field,
		},
		Suggestions: []string{
			"Check the field value and format",
			"Review validation requirements",
		},
	}
}

// NewConversionError creates a new conversion error
func NewConversionError(fromFormat, toFormat, message string) *TOMLError {
	return &TOMLError{
		Code:       "CONVERSION_ERROR",
		Message:    fmt.Sprintf("Failed to convert from %s to %s: %s", fromFormat, toFormat, message),
		HTTPStatus: http.StatusBadRequest,
		Details: map[string]interface{}{
			"from_format": fromFormat,
			"to_format":   toFormat,
		},
		Suggestions: []string{
			"Check format compatibility",
			"Verify data structure is convertible",
		},
	}
}

// NewDatabaseError creates a new database error
func NewDatabaseError(operation, message string) *TOMLError {
	return &TOMLError{
		Code:       "DATABASE_ERROR",
		Message:    fmt.Sprintf("Database %s failed: %s", operation, message),
		HTTPStatus: http.StatusInternalServerError,
		Details: map[string]interface{}{
			"operation": operation,
		},
		Suggestions: []string{
			"Check database connectivity",
			"Verify operation parameters",
			"Contact system administrator if issue persists",
		},
	}
}
