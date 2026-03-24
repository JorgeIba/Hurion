package validatorerr

import "fmt"

type ValidatorError struct {
	Code         int
	Message      string
	WrappedError error
}

func (e *ValidatorError) Error() string {
	if e.WrappedError != nil {
		return fmt.Sprintf("Validator error: (code %d) %s: %v", e.Code, e.Message, e.WrappedError)
	}
	return fmt.Sprintf("Validator error: (code %d) %s", e.Code, e.Message)
}

func (e *ValidatorError) Unwrap() error {
	return e.WrappedError
}

type InternalError struct {
	*ValidatorError
}

type NotFoundError struct {
	*ValidatorError
}

func NewInternalError(message string, original error) *InternalError {
	return &InternalError{
		ValidatorError: &ValidatorError{
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewNotFoundError(message string, original error) *NotFoundError {
	return &NotFoundError{
		ValidatorError: &ValidatorError{
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}
