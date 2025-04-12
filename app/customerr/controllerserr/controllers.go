package controllerserr

import "fmt"

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

type ValidationError struct {
	*ControllersError
}

type InternalError struct {
	*ControllersError
}

type ParsingError struct {
	*ControllersError
}

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
