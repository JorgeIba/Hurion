package serviceerr

import (
	"errors"
	"fmt"
)

// ServiceErrorInterface defines the behavior of a domain-level error.
type ServiceErrorInterface interface {
	error
	IsServiceError() bool
}

type ServiceError struct {
	Code         int
	Message      string
	WrappedError error
}

func (e *ServiceError) Error() string {
	if e.WrappedError != nil {
		return fmt.Sprintf("Service Error: (code %d) %s: %v", e.Code, e.Message, e.WrappedError)
	}
	return fmt.Sprintf("Service Error: (code %d) %s", e.Code, e.Message)
}

func (e *ServiceError) Unwrap() error {
	return e.WrappedError
}

func (e *ServiceError) IsServiceError() bool {
	return true
}

// ============================================================================
// Specialized Errors
// ============================================================================

type InternalError struct {
	*ServiceError
}

func (e *InternalError) Unwrap() error {
	if e == nil || e.ServiceError == nil {
		return nil
	}
	return e.WrappedError
}

type ConflictError struct {
	*ServiceError
}

func (e *ConflictError) Unwrap() error {
	if e == nil || e.ServiceError == nil {
		return nil
	}
	return e.WrappedError
}

type NotFoundInfoError struct {
	*ServiceError
}

func (e *NotFoundInfoError) Unwrap() error {
	if e == nil || e.ServiceError == nil {
		return nil
	}
	return e.WrappedError
}

type ClassCodeNotFoundError struct {
	*ServiceError
}

func (e *ClassCodeNotFoundError) Unwrap() error {
	if e == nil || e.ServiceError == nil {
		return nil
	}
	return e.WrappedError
}

type ProblemNotFoundError struct {
	*ServiceError
}

func (e *ProblemNotFoundError) Unwrap() error {
	if e == nil || e.ServiceError == nil {
		return nil
	}
	return e.WrappedError
}

type DuplicateProblemError struct {
	*ServiceError
}

func (e *DuplicateProblemError) Unwrap() error {
	if e == nil || e.ServiceError == nil {
		return nil
	}
	return e.WrappedError
}

type UserNotEnrolledError struct {
	*ServiceError
}

func (e *UserNotEnrolledError) Unwrap() error {
	if e == nil || e.ServiceError == nil {
		return nil
	}
	return e.WrappedError
}

type UserNotFoundError struct {
	*ServiceError
}

func (e *UserNotFoundError) Unwrap() error {
	if e == nil || e.ServiceError == nil {
		return nil
	}
	return e.WrappedError
}

// ============================================================================
// Helpers
// ============================================================================

func NewInternalError(message string, original error) *InternalError {
	return &InternalError{
		ServiceError: &ServiceError{
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewConflictError(message string, original error) *ConflictError {
	return &ConflictError{
		ServiceError: &ServiceError{
			Code:         409,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewNotFoundInfoError(message string, original error) *NotFoundInfoError {
	return &NotFoundInfoError{
		ServiceError: &ServiceError{
			Code:         404,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewUserNotFoundError(message string, original error) *UserNotFoundError {
	return &UserNotFoundError{
		ServiceError: &ServiceError{
			Code:         404,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewUserNotEnrolledError(message string, original error) *UserNotEnrolledError {
	return &UserNotEnrolledError{
		ServiceError: &ServiceError{
			Code:         404,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewClassCodeNotFoundError(message string, original error) *ClassCodeNotFoundError {
	return &ClassCodeNotFoundError{
		ServiceError: &ServiceError{
			Code:         404,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewProblemNotFoundError(message string, original error) *ProblemNotFoundError {
	return &ProblemNotFoundError{
		ServiceError: &ServiceError{
			Code:         404,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewDuplicateProblemError(message string, original error) *DuplicateProblemError {
	return &DuplicateProblemError{
		ServiceError: &ServiceError{
			Code:         409,
			Message:      message,
			WrappedError: original,
		},
	}
}

func MakeGeneralError(err error) *InternalError {
	return NewInternalError("", err)
}

// IsServiceError checks if an error implements the ServiceErrorInterface.
func IsServiceError(err error) bool {
	var sErr ServiceErrorInterface
	return errors.As(err, &sErr)
}
