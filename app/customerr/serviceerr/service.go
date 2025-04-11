package serviceerr

import "fmt"

type ServiceError struct {
	Code         int
	Message      string
	WrappedError error
}

func (e *ServiceError) Error() string {
	if e.WrappedError != nil {
		return fmt.Sprintf("(code %d) %s: %v", e.Code, e.Message, e.WrappedError)
	}
	return fmt.Sprintf("(code %d) %s", e.Code, e.Message)
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
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewNotFoundInfoError(message string, original error) *NotFoundInfoError {
	return &NotFoundInfoError{
		ServiceError: &ServiceError{
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}

func MakeGeneralError(err error) *InternalError {
	return NewInternalError("", err)
}
