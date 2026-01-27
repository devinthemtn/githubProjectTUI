package errors

import (
	"fmt"
	"strings"
	"time"
)

// ErrorType represents the category of error
type ErrorType int

const (
	ErrorTypeUnknown ErrorType = iota
	ErrorTypeRetryable
	ErrorTypePermission
	ErrorTypeValidation
	ErrorTypeRateLimit
	ErrorTypeConflict
)

func (t ErrorType) String() string {
	switch t {
	case ErrorTypeRetryable:
		return "retryable"
	case ErrorTypePermission:
		return "permission"
	case ErrorTypeValidation:
		return "validation"
	case ErrorTypeRateLimit:
		return "rate_limit"
	case ErrorTypeConflict:
		return "conflict"
	default:
		return "unknown"
	}
}

// APIError represents a structured error from the GitHub API
type APIError struct {
	Type         ErrorType
	Message      string
	OriginalErr  error
	Retryable    bool
	RetryAfter   time.Duration
	HTTPStatus   int
	GraphQLType  string
	FieldErrors  map[string]string // For validation errors
}

func (e *APIError) Error() string {
	return e.Message
}

func (e *APIError) Unwrap() error {
	return e.OriginalErr
}

// IsRetryable returns true if the error should be retried
func (e *APIError) IsRetryable() bool {
	return e.Retryable
}

// RetryableError creates a retryable error
func RetryableError(message string, err error) *APIError {
	return &APIError{
		Type:        ErrorTypeRetryable,
		Message:     message,
		OriginalErr: err,
		Retryable:   true,
	}
}

// PermissionError creates a permission error
func PermissionError(message string, err error) *APIError {
	return &APIError{
		Type:        ErrorTypePermission,
		Message:     message,
		OriginalErr: err,
		Retryable:   false,
	}
}

// ValidationError creates a validation error
func ValidationError(message string, fieldErrors map[string]string) *APIError {
	return &APIError{
		Type:        ErrorTypeValidation,
		Message:     message,
		OriginalErr: nil,
		Retryable:   false,
		FieldErrors: fieldErrors,
	}
}

// RateLimitError creates a rate limit error with retry delay
func RateLimitError(message string, retryAfter time.Duration) *APIError {
	return &APIError{
		Type:        ErrorTypeRateLimit,
		Message:     message,
		OriginalErr: nil,
		Retryable:   true,
		RetryAfter:  retryAfter,
	}
}

// ConflictError creates a conflict error
func ConflictError(message string, err error) *APIError {
	return &APIError{
		Type:        ErrorTypeConflict,
		Message:     message,
		OriginalErr: err,
		Retryable:   false,
	}
}

// GetUserFriendlyMessage returns a user-friendly error message
func (e *APIError) GetUserFriendlyMessage() string {
	switch e.Type {
	case ErrorTypeRateLimit:
		if e.RetryAfter > 0 {
			return fmt.Sprintf("GitHub rate limit reached. Please wait %v before trying again.", e.RetryAfter.Round(time.Second))
		}
		return "GitHub rate limit reached. Please wait a moment before trying again."
	
	case ErrorTypePermission:
		if strings.Contains(strings.ToLower(e.Message), "token") {
			return "Permission denied. Your GitHub token may not have the required scopes. Check your token permissions."
		}
		return "Permission denied. You may not have access to this resource."
	
	case ErrorTypeValidation:
		if len(e.FieldErrors) > 0 {
			var msgs []string
			for field, msg := range e.FieldErrors {
				msgs = append(msgs, fmt.Sprintf("%s: %s", field, msg))
			}
			return fmt.Sprintf("Validation failed: %s", strings.Join(msgs, "; "))
		}
		return e.Message
	
	case ErrorTypeRetryable:
		return fmt.Sprintf("Network error: %s. This will be retried automatically.", e.Message)
	
	case ErrorTypeConflict:
		return fmt.Sprintf("Conflict: %s. The item may have been modified by someone else.", e.Message)
	
	default:
		return e.Message
	}
}
