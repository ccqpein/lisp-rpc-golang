package parser

import "fmt"

// ParserErrorKind defines the category of parsing error.
type ParserErrorKind int

const (
	// ErrInvalidStart indicates an unexpected starting token was encountered.
	ErrInvalidStart ParserErrorKind = iota
	// ErrInvalidToken indicates an invalid or unexpected token was encountered.
	ErrInvalidToken
	// ErrCorruptData indicates corrupted, malformed, or illegal data was encountered.
	ErrCorruptData
	// ErrUnknownToken indicates an unknown token was encountered.
	ErrUnknownToken
)

// ParserError represents errors that can occur during Lisp S-expression parsing.
type ParserError struct {
	Kind ParserErrorKind
	Msg  string
}

// Error formats the parser error message matching the Rust implementation.
func (e *ParserError) Error() string {
	if e == nil {
		return ""
	}
	switch e.Kind {
	case ErrInvalidStart:
		return "parser error: Invalid start token"
	case ErrInvalidToken:
		return "parser error: Invalid token: " + e.Msg
	case ErrUnknownToken:
		return "parser error: Unknown token"
	case ErrCorruptData:
		return "parser error: illegal data: " + e.Msg
	default:
		return fmt.Sprintf("parser error: unknown error (%d): %s", e.Kind, e.Msg)
	}
}

// Is implements error matching for errors.Is.
func (e *ParserError) Is(target error) bool {
	t, ok := target.(*ParserError)
	if !ok {
		return false
	}
	if t.Msg == "" {
		return e.Kind == t.Kind
	}
	return e.Kind == t.Kind && e.Msg == t.Msg
}

// Equal checks equality between two ParserErrors.
func (e *ParserError) Equal(other *ParserError) bool {
	if e == nil || other == nil {
		return e == other
	}
	return e.Kind == other.Kind && e.Msg == other.Msg
}

// NewErrInvalidStart creates a new ErrInvalidStart error.
func NewErrInvalidStart() *ParserError {
	return &ParserError{Kind: ErrInvalidStart}
}

// NewErrInvalidToken creates a new ErrInvalidToken error with a message.
func NewErrInvalidToken(msg string) *ParserError {
	return &ParserError{Kind: ErrInvalidToken, Msg: msg}
}

// NewErrCorruptData creates a new ErrCorruptData error with a message.
func NewErrCorruptData(msg string) *ParserError {
	return &ParserError{Kind: ErrCorruptData, Msg: msg}
}

// NewErrUnknownToken creates a new ErrUnknownToken error.
func NewErrUnknownToken() *ParserError {
	return &ParserError{Kind: ErrUnknownToken}
}
