package dataerr

import (
	"errors"
	"fmt"
)

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
	DataError
}

type QueryError struct {
	DataError
}

type TransactionError struct {
	DataError
}

type RowConflictError struct {
	DataError
}

type NotFoundError struct {
	DataError
}

type UserMissingFromDatabaseError struct {
	DataError
}

type ForeignKeyViolationError struct {
	DataError
}

func NewInitializationError(message string, original error) *InitializationError {
	return &InitializationError{
		Code:         500,
		Message:      message,
		WrappedError: original,
	}
}

func NewQueryError(message string, original error) *QueryError {
	return &QueryError{
		Code:         500,
		Message:      message,
		WrappedError: original,
	}
}

func NewTransactionError(message string, original error) *TransactionError {
	return &TransactionError{
		Code:         500,
		Message:      message,
		WrappedError: original,
	}
}

func NewRowConflictError(message string, original error) *RowConflictError {
	return &RowConflictError{
		Code:         500,
		Message:      message,
		WrappedError: original,
	}
}

func NewNotFoundError(message string, original error) *NotFoundError {
	return &NotFoundError{
		Code:         500,
		Message:      message,
		WrappedError: original,
	}
}

func NewUserMissingFromDatabaseError(message string, original error) *UserMissingFromDatabaseError {
	return &UserMissingFromDatabaseError{
		Code:         500,
		Message:      message,
		WrappedError: original,
	}
}

func NewForeignKeyViolationError(message string, original error) *ForeignKeyViolationError {
	return &ForeignKeyViolationError{
		Code:         500,
		Message:      message,
		WrappedError: original,
	}
}

func IsNotFound(err error) bool {
	_, ok := errors.AsType[*NotFoundError](err)
	return ok
}

func IsRowConflict(err error) bool {
	_, ok := errors.AsType[*RowConflictError](err)
	return ok
}
