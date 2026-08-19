package eyeduxsdk

import (
	"errors"
	"fmt"
)

// API error codes returned by the Eyedux API.
const (
	ErrCodeInvalidAPIKey           = "invalid_api_key"
	ErrCodeEventTypeRequired       = "event_type_required"
	ErrCodeEventPropertiesEmpty    = "event_properties_empty"
	ErrCodeEventExternalIDConflict = "event_external_id_conflict"
	ErrCodeEventExternalIDNotFound = "event_external_id_not_found"
	ErrCodeEventExternalIDRequired = "event_external_id_required"
	ErrCodeRateLimitExceeded       = "RATE_LIMIT_EXCEEDED"
	ErrCodeInternalServerError     = "INTERNAL_SERVER_ERROR"
)

// SDK-level sentinel errors, independent of the API response.
var (
	ErrEmptyAPIKey     = errors.New("eyedux: api key must not be empty")
	ErrEmptyExternalID = errors.New("eyedux: external_id must not be empty")
)

// APIError represents an error response from the Eyedux API.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	RetryAfter *int // seconds suggested before retrying; populated on 429 responses
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf("eyedux: %s (status %d)", e.Code, e.StatusCode)
}

// IsNotFound reports whether err is a 404 API error.
func IsNotFound(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.StatusCode == 404
}

// IsConflict reports whether err is a 409 API error.
func IsConflict(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.StatusCode == 409
}

// IsRateLimited reports whether err is a 429 API error.
func IsRateLimited(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.StatusCode == 429
}

// IsAuthError reports whether err is an authentication failure.
func IsAuthError(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.Code == ErrCodeInvalidAPIKey
}
