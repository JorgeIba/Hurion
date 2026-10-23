package serviceerr

import (
	"errors"
	"fmt"

	"github.com/JorgeIba/Hurion/app/authorization"
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
	ServiceError
}

// ValidationError contains input-validation reasons safe to show to the caller.
type ValidationError struct {
	ServiceError
}

type ConflictError struct {
	ServiceError
}

// ACAlreadyExistsError identifies a solve that is already recorded.
type ACAlreadyExistsError struct {
	ServiceError
}

type NotFoundInfoError struct {
	ServiceError
}

type ClassCodeNotFoundError struct {
	ServiceError
}

type ProblemNotFoundError struct {
	ServiceError
}

type DuplicateProblemError struct {
	ServiceError
}

type UserNotEnrolledError struct {
	ServiceError
}

type UserNotFoundError struct {
	ServiceError
}

// ============================================================================
// Helpers
// ============================================================================

func NewValidationError(message string, original error) *ValidationError {
	return &ValidationError{
		Code:         400,
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

func NewConflictError(message string, original error) *ConflictError {
	return &ConflictError{
		Code:         409,
		Message:      message,
		WrappedError: original,
	}
}

func NewACAlreadyExistsError(message string, original error) *ACAlreadyExistsError {
	return &ACAlreadyExistsError{
		Code:         409,
		Message:      message,
		WrappedError: original,
	}
}

func NewNotFoundInfoError(message string, original error) *NotFoundInfoError {
	return &NotFoundInfoError{
		Code:         404,
		Message:      message,
		WrappedError: original,
	}
}

func NewUserNotFoundError(message string, original error) *UserNotFoundError {
	return &UserNotFoundError{
		Code:         404,
		Message:      message,
		WrappedError: original,
	}
}

func NewUserNotEnrolledError(message string, original error) *UserNotEnrolledError {
	return &UserNotEnrolledError{
		Code:         404,
		Message:      message,
		WrappedError: original,
	}
}

func NewClassCodeNotFoundError(message string, original error) *ClassCodeNotFoundError {
	return &ClassCodeNotFoundError{
		Code:         404,
		Message:      message,
		WrappedError: original,
	}
}

func NewProblemNotFoundError(message string, original error) *ProblemNotFoundError {
	return &ProblemNotFoundError{
		Code:         404,
		Message:      message,
		WrappedError: original,
	}
}

func NewDuplicateProblemError(message string, original error) *DuplicateProblemError {
	return &DuplicateProblemError{
		Code:         409,
		Message:      message,
		WrappedError: original,
	}
}

func MakeGeneralError(err error) *InternalError {
	return NewInternalError("", err)
}

// IsServiceError checks if an error implements the ServiceErrorInterface.
func IsServiceError(err error) bool {
	_, ok := errors.AsType[ServiceErrorInterface](err)
	return ok
}

// BatchPreparationError indicates that execution stopped before any write.
type BatchPreparationError struct{ ServiceError }

func NewBatchPreparationError(message string, err error) *BatchPreparationError {
	return &BatchPreparationError{
		Code:         500,
		Message:      message,
		WrappedError: err,
	}
}

// ForbiddenError is a domain authorization denial safe for presentation.
type ForbiddenError struct{ ServiceError }

func NewForbiddenError(permission authorization.Permission, classCode string) *ForbiddenError {
	message := "The mentor or admin role is required."
	switch permission {
	case authorization.Admin:
		message = "The admin role is required (is not admin)."
	case authorization.AccessClass:
		message = fmt.Sprintf("You do not have access to class `%s`.", classCode)
	default:
		if classCode != "" {
			message = fmt.Sprintf("The mentor or admin role for class `%s` is required. No changes were applied.", classCode)
		}
	}

	return &ForbiddenError{ServiceError{Code: 403, Message: message}}
}

// AuthorizationUnavailableError identifies a failed permission lookup rather than a denial.
type AuthorizationUnavailableError struct{ ServiceError }

func NewAuthorizationUnavailableError(cause error) *AuthorizationUnavailableError {
	return &AuthorizationUnavailableError{ServiceError{Code: 503, Message: "There was a problem verifying permissions.", WrappedError: cause}}
}

// AuthorizationTargetChangedError asks the caller to retry against the current resource.
type AuthorizationTargetChangedError struct{ ServiceError }

func NewAuthorizationTargetChangedError(cause error) *AuthorizationTargetChangedError {
	return &AuthorizationTargetChangedError{ServiceError{Code: 409, Message: "The problem's class changed while checking permissions. No changes were applied; please try again.", WrappedError: cause}}
}
