package errors

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// GraphQLErrorResponse represents a GraphQL error response
type GraphQLErrorResponse struct {
	Errors []GraphQLError `json:"errors"`
}

// GraphQLError represents a single GraphQL error
type GraphQLError struct {
	Type       string                 `json:"type"`
	Message    string                 `json:"message"`
	Path       []interface{}          `json:"path"`
	Extensions map[string]interface{} `json:"extensions"`
}

// ClassifyError analyzes an error and returns a typed APIError
func ClassifyError(err error, httpStatus int) *APIError {
	if err == nil {
		return nil
	}

	errMsg := err.Error()
	errLower := strings.ToLower(errMsg)

	// Check for rate limit errors
	if httpStatus == http.StatusTooManyRequests || 
	   strings.Contains(errLower, "rate limit") ||
	   strings.Contains(errLower, "rate_limited") {
		retryAfter := extractRetryAfter(errMsg)
		return RateLimitError("GitHub API rate limit exceeded", retryAfter)
	}

	// Check for permission errors
	if httpStatus == http.StatusForbidden ||
	   httpStatus == http.StatusUnauthorized ||
	   strings.Contains(errLower, "not authorized") ||
	   strings.Contains(errLower, "permission denied") ||
	   strings.Contains(errLower, "forbidden") ||
	   strings.Contains(errLower, "does not have access") {
		return PermissionError("Permission denied", err)
	}

	// Check for validation errors
	if httpStatus == http.StatusBadRequest ||
	   strings.Contains(errLower, "invalid") ||
	   strings.Contains(errLower, "validation") ||
	   strings.Contains(errLower, "not found") && strings.Contains(errLower, "user") {
		fieldErrors := extractFieldErrors(errMsg)
		return ValidationError(errMsg, fieldErrors)
	}

	// Check for conflict errors
	if httpStatus == http.StatusConflict ||
	   strings.Contains(errLower, "conflict") ||
	   strings.Contains(errLower, "concurrent") {
		return ConflictError("Resource conflict", err)
	}

	// Check for network/timeout errors (retryable)
	if strings.Contains(errLower, "timeout") ||
	   strings.Contains(errLower, "connection") ||
	   strings.Contains(errLower, "network") ||
	   strings.Contains(errLower, "temporary") ||
	   httpStatus >= 500 { // Server errors are retryable
		return RetryableError(errMsg, err)
	}

	// Default: unknown error, not retryable
	return &APIError{
		Type:        ErrorTypeUnknown,
		Message:     errMsg,
		OriginalErr: err,
		Retryable:   false,
		HTTPStatus:  httpStatus,
	}
}

// ClassifyGraphQLError analyzes a GraphQL-specific error
func ClassifyGraphQLError(gqlErr GraphQLError) *APIError {
	errType := strings.ToUpper(gqlErr.Type)
	errMsg := gqlErr.Message
	errLower := strings.ToLower(errMsg)

	// Check GraphQL error type
	switch errType {
	case "RATE_LIMITED":
		retryAfter := extractRetryAfterFromExtensions(gqlErr.Extensions)
		return RateLimitError(errMsg, retryAfter)
	
	case "FORBIDDEN", "UNAUTHORIZED":
		return PermissionError(errMsg, nil)
	
	case "NOT_FOUND", "INVALID":
		fieldErrors := extractFieldErrorsFromExtensions(gqlErr.Extensions)
		return ValidationError(errMsg, fieldErrors)
	}

	// Fallback to message-based classification
	if strings.Contains(errLower, "rate limit") {
		retryAfter := extractRetryAfterFromExtensions(gqlErr.Extensions)
		return RateLimitError(errMsg, retryAfter)
	}

	if strings.Contains(errLower, "permission") || 
	   strings.Contains(errLower, "authorized") ||
	   strings.Contains(errLower, "access") {
		return PermissionError(errMsg, nil)
	}

	if strings.Contains(errLower, "invalid") || 
	   strings.Contains(errLower, "not found") {
		return ValidationError(errMsg, nil)
	}

	// Unknown error
	return &APIError{
		Type:        ErrorTypeUnknown,
		Message:     errMsg,
		OriginalErr: nil,
		Retryable:   false,
		GraphQLType: errType,
	}
}

// extractRetryAfter attempts to extract retry-after duration from error message
func extractRetryAfter(errMsg string) time.Duration {
	// Try to parse common formats like "retry after 60 seconds"
	// This is a simple implementation - can be enhanced
	if strings.Contains(errMsg, "60") {
		return 60 * time.Second
	}
	// Default retry after 1 minute for rate limits
	return 60 * time.Second
}

// extractRetryAfterFromExtensions extracts retry-after from GraphQL extensions
func extractRetryAfterFromExtensions(extensions map[string]interface{}) time.Duration {
	if extensions == nil {
		return 60 * time.Second // default
	}

	if retryAfter, ok := extensions["retryAfter"].(float64); ok {
		return time.Duration(retryAfter) * time.Second
	}

	if retryAfter, ok := extensions["retryAfter"].(int); ok {
		return time.Duration(retryAfter) * time.Second
	}

	return 60 * time.Second // default
}

// extractFieldErrors attempts to extract field-specific validation errors
func extractFieldErrors(errMsg string) map[string]string {
	fieldErrors := make(map[string]string)
	
	// Simple heuristic: look for "username" or "assignee" in error
	if strings.Contains(strings.ToLower(errMsg), "user") && 
	   strings.Contains(strings.ToLower(errMsg), "not found") {
		fieldErrors["assignee"] = "User not found"
	}

	if strings.Contains(strings.ToLower(errMsg), "title") {
		fieldErrors["title"] = "Invalid title"
	}

	return fieldErrors
}

// extractFieldErrorsFromExtensions extracts field errors from GraphQL extensions
func extractFieldErrorsFromExtensions(extensions map[string]interface{}) map[string]string {
	if extensions == nil {
		return nil
	}

	fieldErrors := make(map[string]string)

	// Try to extract field errors if present
	if fields, ok := extensions["fields"].(map[string]interface{}); ok {
		for field, errVal := range fields {
			if errStr, ok := errVal.(string); ok {
				fieldErrors[field] = errStr
			}
		}
	}

	return fieldErrors
}

// ParseGraphQLErrors parses GraphQL error response and returns the first classified error
func ParseGraphQLErrors(data []byte) *APIError {
	var response GraphQLErrorResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil
	}

	if len(response.Errors) > 0 {
		return ClassifyGraphQLError(response.Errors[0])
	}

	return nil
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.IsRetryable()
	}
	return false
}

// GetRetryAfter returns the retry-after duration if available
func GetRetryAfter(err error) time.Duration {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.RetryAfter
	}
	return 0
}
