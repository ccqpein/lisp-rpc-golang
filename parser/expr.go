package parser

import (
	"errors"
	"strings"
)

// ExprKind indicates the variant of an S-expression node.
type ExprKind int

const (
	// ExprAtom represents an atomic value node.
	ExprAtom ExprKind = iota
	// ExprList represents an S-expression list node `(...)`.
	ExprList
	// ExprQuote represents a quoted expression node `'...`.
	ExprQuote
	// ExprComment represents a comment node `;...`.
	ExprComment
)

// Expr represents a parsed Lisp-RPC S-expression node.
type Expr struct {
	Kind    ExprKind
	Atom    Atom
	List    []Expr
	Quote   *Expr
	Comment string
}

// NewExprAtom creates a new Expr containing an Atom.
func NewExprAtom(a Atom) Expr {
	return Expr{Kind: ExprAtom, Atom: a}
}

// NewExprList creates a new Expr containing a list of Exprs.
func NewExprList(list []Expr) Expr {
	return Expr{Kind: ExprList, List: list}
}

// NewExprQuote creates a new Expr containing a quoted Expr.
func NewExprQuote(inner Expr) Expr {
	return Expr{Kind: ExprQuote, Quote: &inner}
}

// NewExprComment creates a new Expr containing a comment string.
func NewExprComment(comment string) Expr {
	return Expr{Kind: ExprComment, Comment: comment}
}

// String formats the expression tree as an S-expression string.
func (e Expr) String() string {
	switch e.Kind {
	case ExprAtom:
		return e.Atom.String()
	case ExprList:
		parts := make([]string, len(e.List))
		for i, child := range e.List {
			parts[i] = child.String()
		}
		return "(" + strings.Join(parts, " ") + ")"
	case ExprQuote:
		if e.Quote != nil {
			return "'" + e.Quote.String()
		}
		return "'"
	case ExprComment:
		return "; " + e.Comment
	default:
		return ""
	}
}

// Nth returns a pointer to the child element at index ind if this is an ExprList.
func (e Expr) Nth(ind int) (*Expr, bool) {
	if e.Kind == ExprList && ind >= 0 && ind < len(e.List) {
		return &e.List[ind], true
	}
	return nil, false
}

// IsComment returns true if this expression is an ExprComment.
func (e Expr) IsComment() bool {
	return e.Kind == ExprComment
}

// FilterOutAllComments returns all non-comment child expressions from an ExprList.
func (e Expr) FilterOutAllComments() ([]Expr, error) {
	if e.Kind != ExprList {
		return nil, errors.New("Not the Expr::List")
	}
	res := make([]Expr, 0, len(e.List))
	for _, item := range e.List {
		if !item.IsComment() {
			res = append(res, item)
		}
	}
	return res, nil
}

// Equal compares two Exprs for deep equality.
func (e Expr) Equal(other Expr) bool {
	if e.Kind != other.Kind {
		return false
	}
	switch e.Kind {
	case ExprAtom:
		return e.Atom.Equal(other.Atom)
	case ExprList:
		if len(e.List) != len(other.List) {
			return false
		}
		for i := range e.List {
			if !e.List[i].Equal(other.List[i]) {
				return false
			}
		}
		return true
	case ExprQuote:
		if e.Quote == nil && other.Quote == nil {
			return true
		}
		if e.Quote == nil || other.Quote == nil {
			return false
		}
		return e.Quote.Equal(*other.Quote)
	case ExprComment:
		return e.Comment == other.Comment
	default:
		return false
	}
}
