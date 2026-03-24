package presentationerr

import "fmt"

type PresentationError struct {
	Code         int
	Message      string
	WrappedError error
}

func (e *PresentationError) Error() string {
	if e.WrappedError != nil {
		return fmt.Sprintf("Presentation error: (code %d) %s: %v", e.Code, e.Message, e.WrappedError)
	}
	return fmt.Sprintf("Presentation error: (code %d) %s", e.Code, e.Message)
}

func (e *PresentationError) Unwrap() error {
	return e.WrappedError
}

type RenderingError struct {
	*PresentationError
}

type StyleCreationError struct {
	*PresentationError
}

type DataWritingError struct {
	*PresentationError
}

type FormatError struct {
	*PresentationError
}

func NewRenderingError(message string, original error) *RenderingError {
	return &RenderingError{
		PresentationError: &PresentationError{
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewStyleCreationError(message string, original error) *StyleCreationError {
	return &StyleCreationError{
		PresentationError: &PresentationError{
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewDataWritingError(message string, original error) *DataWritingError {
	return &DataWritingError{
		PresentationError: &PresentationError{
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewFormatError(message string, original error) *FormatError {
	return &FormatError{
		PresentationError: &PresentationError{
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}
