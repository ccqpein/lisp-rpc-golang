package rawdata

import "fmt"

// DataErrorType represents categories of errors that can occur during dynamic data operations.
type DataErrorType int

const (
	// ErrInvalidInput indicates that the S-expression input does not match expected data structure format.
	ErrInvalidInput DataErrorType = iota
	// ErrCorruptedData indicates corrupted data state.
	ErrCorruptedData
)

// DataError represents an error encountered during dynamic data operations and conversions.
type DataError struct {
	Msg     string
	ErrType DataErrorType
}

// Error implements the standard error interface.
func (e *DataError) Error() string {
	return fmt.Sprintf("data operation error %s", e.Msg)
}

// NewDataError creates a new DataError with the specified message and error type.
func NewDataError(msg string, errType DataErrorType) *DataError {
	return &DataError{
		Msg:     msg,
		ErrType: errType,
	}
}
