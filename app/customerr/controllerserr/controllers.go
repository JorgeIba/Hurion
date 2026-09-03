package controllerserr

import (
	"errors"
	"fmt"
)

// ControllerErrorInterface defines the behavior of a technical controller-level error.
type ControllerErrorInterface interface {
	error
	IsControllerError() bool
}

type ControllersError struct {
	Code         int
	Message      string
	WrappedError error
}

func (e *ControllersError) Error() string {
	if e.WrappedError != nil {
		return fmt.Sprintf("Controller error: (code %d) %s: %v", e.Code, e.Message, e.WrappedError)
	}
	return fmt.Sprintf("Controller error: (code %d) %s", e.Code, e.Message)
}

func (e *ControllersError) Unwrap() error {
	return e.WrappedError
}

func (e *ControllersError) IsControllerError() bool {
	return true
}

// ============================================================================
// Specialized Errors
// ============================================================================

type ValidationError struct {
	ControllersError
}

type InternalError struct {
	ControllersError
}

type ParsingError struct {
	ControllersError
}

type UnauthorizedError struct {
	ControllersError
}

// ============================================================================
// Helpers
// ============================================================================

func NewValidationError(message string, original error) *ValidationError {
	return &ValidationError{
		Code:         500,
		Message:      message,
		WrappedError: original,
	}
}

func NewInternalError(message string, original error) *InternalError {
	return &InternalError{
		Code:         500,
		Message:      message,
		WrappedError: original,
	}
}

func NewParsingError(message string, original error) *ParsingError {
	return &ParsingError{
		Code:         500,
		Message:      message,
		WrappedError: original,
	}
}

func NewUnauthorizedError(message string, original error) *UnauthorizedError {
	return &UnauthorizedError{
		Code:         403,
		Message:      message,
		WrappedError: original,
	}
}

// IsControllerError checks if an error implements the ControllerErrorInterface.
func IsControllerError(err error) bool {
	_, ok := errors.AsType[ControllerErrorInterface](err)
	return ok
}
