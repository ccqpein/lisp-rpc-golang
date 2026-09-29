package parser

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// NumberKind indicates whether a numeric value is an integer or float.
type NumberKind int

const (
	// NumberInt indicates a 64-bit signed integer.
	NumberInt NumberKind = iota
	// NumberFloat indicates a 64-bit floating-point number.
	NumberFloat
)

// TypeValueNumber represents a numeric value (int64 or float64).
type TypeValueNumber struct {
	Kind  NumberKind
	Int   int64
	Float float64
}

// NewInt creates a new integer TypeValueNumber.
func NewInt(i int64) TypeValueNumber {
	return TypeValueNumber{Kind: NumberInt, Int: i}
}

// NewFloat creates a new floating-point TypeValueNumber.
func NewFloat(f float64) TypeValueNumber {
	return TypeValueNumber{Kind: NumberFloat, Float: f}
}

// ToInt returns the integer value if this is an integer.
func (n TypeValueNumber) ToInt() (int64, bool) {
	if n.Kind == NumberInt {
		return n.Int, true
	}
	return 0, false
}

// ToFloat returns the value as a float64.
// For integers, it converts the int64 to float64, matching Rust TypeValueNumber::to_float.
func (n TypeValueNumber) ToFloat() (float64, bool) {
	if n.Kind == NumberFloat {
		return n.Float, true
	}
	return float64(n.Int), true
}

// Equal compares two TypeValueNumbers for equality, matching Rust PartialEq.
// NaN compares equal to NaN, and 0.0 == -0.0.
func (n TypeValueNumber) Equal(other TypeValueNumber) bool {
	if n.Kind != other.Kind {
		return false
	}
	if n.Kind == NumberInt {
		return n.Int == other.Int
	}
	if math.IsNaN(n.Float) && math.IsNaN(other.Float) {
		return true
	}
	return n.Float == other.Float
}

// String formats the number as a string.
func (n TypeValueNumber) String() string {
	if n.Kind == NumberInt {
		return strconv.FormatInt(n.Int, 10)
	}
	return strconv.FormatFloat(n.Float, 'g', -1, 64)
}

// TypeValueKind represents the kind of primitive atom value.
type TypeValueKind int

const (
	// TypeValueSymbol represents a Lisp symbol identifier.
	TypeValueSymbol TypeValueKind = iota
	// TypeValueString represents a string literal.
	TypeValueString
	// TypeValueKeyword represents a keyword identifier prefixed with a colon.
	TypeValueKeyword
	// TypeValueNumberKind represents a numeric literal.
	TypeValueNumberKind
)

// TypeValue represents a primitive atom value in Lisp-RPC expressions.
type TypeValue struct {
	Kind   TypeValueKind
	Str    string
	Number TypeValueNumber
}

// NewSymbol creates a new symbol TypeValue.
func NewSymbol(s string) TypeValue {
	return TypeValue{Kind: TypeValueSymbol, Str: s}
}

// NewString creates a new string literal TypeValue.
func NewString(s string) TypeValue {
	return TypeValue{Kind: TypeValueString, Str: s}
}

// NewKeyword creates a new keyword TypeValue.
func NewKeyword(s string) TypeValue {
	return TypeValue{Kind: TypeValueKeyword, Str: s}
}

// NewNumber creates a new number TypeValue.
func NewNumber(n TypeValueNumber) TypeValue {
	return TypeValue{Kind: TypeValueNumberKind, Number: n}
}

// String formats the TypeValue as an S-expression token string.
func (tv TypeValue) String() string {
	switch tv.Kind {
	case TypeValueSymbol:
		return tv.Str
	case TypeValueString:
		return "\"" + tv.Str + "\""
	case TypeValueKeyword:
		return ":" + tv.Str
	case TypeValueNumberKind:
		return tv.Number.String()
	default:
		return ""
	}
}

// GetString extracts the unquoted string content if this is a TypeValueString.
func (tv TypeValue) GetString() (string, error) {
	if tv.Kind == TypeValueString {
		return tv.Str, nil
	}
	return "", fmt.Errorf("%v isn't the String type that can get string", tv)
}

// IsString returns true if the type value is a string literal.
func (tv TypeValue) IsString() bool {
	return tv.Kind == TypeValueString
}

// IsSymbol returns true if the type value is a symbol identifier.
func (tv TypeValue) IsSymbol() bool {
	return tv.Kind == TypeValueSymbol
}

// IsKeyword returns true if the type value is a keyword identifier.
func (tv TypeValue) IsKeyword() bool {
	return tv.Kind == TypeValueKeyword
}

// IsNumber returns true if the type value is a numeric literal.
func (tv TypeValue) IsNumber() bool {
	return tv.Kind == TypeValueNumberKind
}
// MakeSymbol creates a TypeValueSymbol if the string contains no whitespace.
func MakeSymbol(s string) (TypeValue, error) {
	if strings.Contains(s, " ") {
		return TypeValue{}, NewErrCorruptData("cannot make symbol with this str")
	}
	return NewSymbol(s), nil
}

// ToInt returns the integer value if this is a TypeValueNumberKind containing an integer.
func (tv TypeValue) ToInt() (int64, bool) {
	if tv.Kind == TypeValueNumberKind {
		return tv.Number.ToInt()
	}
	return 0, false
}

// ToFloat returns the floating-point value if this is a TypeValueNumberKind.
func (tv TypeValue) ToFloat() (float64, bool) {
	if tv.Kind == TypeValueNumberKind {
		return tv.Number.ToFloat()
	}
	return 0, false
}

// Equal compares two TypeValues for equality.
func (tv TypeValue) Equal(other TypeValue) bool {
	if tv.Kind != other.Kind {
		return false
	}
	switch tv.Kind {
	case TypeValueSymbol, TypeValueString, TypeValueKeyword:
		return tv.Str == other.Str
	case TypeValueNumberKind:
		return tv.Number.Equal(other.Number)
	default:
		return false
	}
}
