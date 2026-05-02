package dataerr

import "fmt"

type DataError struct {
	Code         int
	Message      string
	WrappedError error
}

func (e *DataError) Error() string {
	if e.WrappedError != nil {
		return fmt.Sprintf("Data error: (code %d) %s: %v", e.Code, e.Message, e.WrappedError)
	}
	return fmt.Sprintf("Data error: (code %d) %s", e.Code, e.Message)
}

func (e *DataError) Unwrap() error {
	return e.WrappedError
}

type InitializationError struct {
	*DataError
}

type QueryError struct {
	*DataError
}

type TransactionError struct {
	*DataError
}

type RowConflictError struct {
	*DataError
}

type NotFoundError struct {
	*DataError
}

type UserMissingFromDatabaseError struct {
	*DataError
}

func NewInitializationError(message string, original error) *InitializationError {
	return &InitializationError{
		DataError: &DataError{
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewQueryError(message string, original error) *QueryError {
	return &QueryError{
		DataError: &DataError{
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewTransactionError(message string, original error) *TransactionError {
	return &TransactionError{
		DataError: &DataError{
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewRowConflictError(message string, original error) *RowConflictError {
	return &RowConflictError{
		DataError: &DataError{
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewNotFoundError(message string, original error) *NotFoundError {
	return &NotFoundError{
		DataError: &DataError{
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}

func NewUserMissingFromDatabaseError(message string, original error) *UserMissingFromDatabaseError {
	return &UserMissingFromDatabaseError{
		DataError: &DataError{
			Code:         500,
			Message:      message,
			WrappedError: original,
		},
	}
}
