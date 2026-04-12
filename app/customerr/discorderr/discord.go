package discorderr

import "fmt"

type DiscordError struct {
	Code         int
	Message      string
	WrappedError error
}

func (e *DiscordError) Error() string {
	if e.WrappedError != nil {
		return fmt.Sprintf("Discord error: (code %d) %s: %v", e.Code, e.Message, e.WrappedError)
	}
	return fmt.Sprintf("Discord error: (code %d) %s", e.Code, e.Message)
}

func (e *DiscordError) Unwrap() error {
	return e.WrappedError
}

type InternalError struct {
	*DiscordError
}

type NotFoundError struct {
	*DiscordError
}

func NewInternalError(message string, original error) *InternalError {
	return &InternalError{
		DiscordError: &DiscordError{
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewNotFoundError(message string, original error) *NotFoundError {
	return &NotFoundError{
		DiscordError: &DiscordError{
			Code:         404,
			Message:      message,
			WrappedError: original,
		},
	}
}
