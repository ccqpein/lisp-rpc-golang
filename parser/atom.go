package parser

// Atom represents an atomic value token in a Lisp S-expression.
type Atom struct {
	Value TypeValue
}


// NewAtomSymbol creates an Atom containing a symbol value.
func NewAtomSymbol(s string) Atom {
	return Atom{Value: NewSymbol(s)}
}

// NewAtomString creates an Atom containing a string literal value.
func NewAtomString(s string) Atom {
	return Atom{Value: NewString(s)}
}

// NewAtomKeyword creates an Atom containing a keyword value.
func NewAtomKeyword(s string) Atom {
	return Atom{Value: NewKeyword(s)}
}

// NewAtomNumber creates an Atom containing a numeric value.
func NewAtomNumber(n TypeValueNumber) Atom {
	return Atom{Value: NewNumber(n)}
}

// IsString returns true if the atom contains a string value.
func (a Atom) IsString() bool {
	return a.Value.Kind == TypeValueString
}

// String formats the atom as an S-expression token string.
func (a Atom) String() string {
	return a.Value.String()
}

// Equal compares two Atoms for equality.
func (a Atom) Equal(other Atom) bool {
	return a.Value.Equal(other.Value)
}
