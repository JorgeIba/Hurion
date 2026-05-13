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
	*ControllersError
}

func (e *ValidationError) Unwrap() error {
	if e == nil || e.ControllersError == nil {
		return nil
	}
	return e.WrappedError
}

type InternalError struct {
	*ControllersError
}

func (e *InternalError) Unwrap() error {
	if e == nil || e.ControllersError == nil {
		return nil
	}
	return e.WrappedError
}

type ParsingError struct {
	*ControllersError
}

func (e *ParsingError) Unwrap() error {
	if e == nil || e.ControllersError == nil {
		return nil
	}
	return e.WrappedError
}

type UnauthorizedError struct {
	*ControllersError
}

func (e *UnauthorizedError) Unwrap() error {
	if e == nil || e.ControllersError == nil {
		return nil
	}
	return e.WrappedError
}

// ============================================================================
// Helpers
// ============================================================================

func NewValidationError(message string, original error) *ValidationError {
	return &ValidationError{
		ControllersError: &ControllersError{
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewInternalError(message string, original error) *InternalError {
	return &InternalError{
		ControllersError: &ControllersError{
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewParsingError(message string, original error) *ParsingError {
	return &ParsingError{
		ControllersError: &ControllersError{
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewUnauthorizedError(message string, original error) *UnauthorizedError {
	return &UnauthorizedError{
		ControllersError: &ControllersError{
			Code:         403,
			Message:      message,
			WrappedError: original,
		},
	}
}

// IsControllerError checks if an error implements the ControllerErrorInterface.
func IsControllerError(err error) bool {
	var cErr ControllerErrorInterface
	return errors.As(err, &cErr)
}
