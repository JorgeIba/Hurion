package serviceerr

import "fmt"

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

type InternalError struct {
	*ServiceError
}

type ConflictError struct {
	*ServiceError
}

type NotFoundInfoError struct {
	*ServiceError
}

type UserNotBindedError struct {
	*ServiceError
}

func (e *UserNotBindedError) Unwrap() error {
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

func NewUserNotBindedError(message string, original error) *UserNotBindedError {
	return &UserNotBindedError{
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
