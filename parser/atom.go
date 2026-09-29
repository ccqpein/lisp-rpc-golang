package parser

// Atom is a type alias for TypeValue representing an atomic value token in a Lisp S-expression.
type Atom = TypeValue

// NewAtomSymbol creates an Atom containing a symbol value.
func NewAtomSymbol(s string) Atom {
	return NewSymbol(s)
}

// NewAtomString creates an Atom containing a string literal value.
func NewAtomString(s string) Atom {
	return NewString(s)
}

// NewAtomKeyword creates an Atom containing a keyword value.
func NewAtomKeyword(s string) Atom {
	return NewKeyword(s)
}

// NewAtomNumber creates an Atom containing a numeric value.
func NewAtomNumber(n TypeValueNumber) Atom {
	return NewNumber(n)
}
