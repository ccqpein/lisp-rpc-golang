package rawdata

import (
	"strings"

	"github.com/ccqpein/lisp-rpc-golang/parser"
)

// ListData represents a quoted sequence data structure, e.g. '("a" 1 2).
type ListData struct {
	innerData []Data
}

// NewListData creates a new ListData containing the specified items.
func NewListData(items ...Data) *ListData {
	return &ListData{innerData: items}
}

// ListDataFromExpr parses a ListData from a quoted list S-expression.
func ListDataFromExpr(expr *parser.Expr) (*ListData, error) {
	if expr == nil || expr.Kind != parser.ExprQuote || expr.Quote == nil {
		return nil, NewDataError("cannot generate ListData from this expr, need quoted", ErrInvalidInput)
	}

	if expr.Quote.Kind != parser.ExprList {
		return nil, NewDataError("cannot generate ListData from this expr, not list after quote", ErrInvalidInput)
	}

	var res []Data
	for _, e := range expr.Quote.List {
		if e.IsComment() {
			continue
		}
		if !IsNilSymbol(&e) {
			d, err := DataFromExpr(&e)
			if err != nil {
				return nil, err
			}
			res = append(res, d)
		}
	}

	return &ListData{innerData: res}, nil
}

// ListDataFromStr tokenizes and parses a ListData directly from a string using the provided parser.
func ListDataFromStr(p *parser.Parser, s string) (*ListData, error) {
	if p == nil {
		p = parser.New()
	}
	if err := p.Tokenize(strings.NewReader(s)); err != nil {
		return nil, err
	}
	if _, err := p.ParseOne(); err != nil {
		return nil, err
	}
	if len(p.Exprs) == 0 {
		return nil, NewDataError("Cannot get the last expr", ErrCorruptedData)
	}
	lastExpr := p.Exprs[len(p.Exprs)-1]
	return ListDataFromExpr(&lastExpr)
}

// Items returns a slice of all items in the list.
func (l *ListData) Items() []Data {
	if l == nil {
		return nil
	}
	return l.innerData
}

// Len returns the number of elements in the list.
func (l *ListData) Len() int {
	if l == nil {
		return 0
	}
	return len(l.innerData)
}

// Nth returns a pointer to the element at the specified index, or false if out of range.
func (l *ListData) Nth(ind int) (*Data, bool) {
	if l == nil || ind < 0 || ind >= len(l.innerData) {
		return nil, false
	}
	return &l.innerData[ind], true
}

// String serializes the list into a quoted S-expression string '(...).
func (l *ListData) String() string {
	if l == nil {
		return "'()"
	}
	parts := make([]string, len(l.innerData))
	for i, d := range l.innerData {
		parts[i] = d.String()
	}
	return "'(" + strings.Join(parts, " ") + ")"
}

// Equal compares two ListData structures for equality.
func (l *ListData) Equal(other ListData) bool {
	if l == nil {
		return false
	}
	if len(l.innerData) != len(other.innerData) {
		return false
	}
	for i := range l.innerData {
		if !l.innerData[i].Equal(other.innerData[i]) {
			return false
		}
	}
	return true
}
