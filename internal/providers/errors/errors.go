// Package errors defines error types for the providers package.
package errors

import "errors"

var (
	// ErrUnsupportedProvider is returned when an unsupported provider type is requested.
	ErrUnsupportedProvider = errors.New("unsupported provider type")

	// ErrProviderNotAvailable is returned when a provider is not available in the current environment.
	ErrProviderNotAvailable = errors.New("provider not available in current environment")

	// ErrProviderInitializationFailed is returned when provider initialization fails.
	ErrProviderInitializationFailed = errors.New("provider initialization failed")

	// ErrResourceNotFound is returned when a requested resource is not found.
	ErrResourceNotFound = errors.New("resource not found")

	// ErrInvalidFilter is returned when an invalid filter is provided.
	ErrInvalidFilter = errors.New("invalid filter")

	// ErrScoutUnavailable is returned when a scout is not available.
	ErrScoutUnavailable = errors.New("scout unavailable")

	// ErrAuthenticationFailed is returned when authentication fails.
	ErrAuthenticationFailed = errors.New("authentication failed")
)
