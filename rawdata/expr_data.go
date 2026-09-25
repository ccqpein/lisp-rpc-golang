package rawdata

import (
	"fmt"
	"strings"

	"github.com/ccqpein/lisp-rpc-golang/parser"
)

// ExprArg represents a keyword-value argument pair in an ExprData expression.
type ExprArg struct {
	Key parser.Expr
	Val Data
}

// ExprData represents a named S-expression data structure (name :key1 val1 :key2 val2).
type ExprData struct {
	name     string
	RestArgs []ExprArg
	innerMap map[string]*Data
}

// NewExprData creates a new ExprData with the given name and argument pairs.
// Validates that the name is a valid Lisp symbol without whitespace.
func NewExprData(name string, restArgs ...ExprArg) (*ExprData, error) {
	if _, err := parser.MakeSymbol(name); err != nil {
		return nil, NewDataError("invalid symbol name: "+name, ErrInvalidInput)
	}

	ed := &ExprData{
		name:     name,
		RestArgs: restArgs,
	}
	ed.initInnerMap()
	return ed, nil
}

// ExprDataFromExpr parses an ExprData from an S-expression list node.
func ExprDataFromExpr(expr *parser.Expr) (*ExprData, error) {
	if expr == nil || expr.Kind != parser.ExprList {
		return nil, NewDataError("expected ExprList for ExprData", ErrInvalidInput)
	}

	exprs, err := expr.FilterOutAllComments()
	if err != nil {
		return nil, err
	}

	if len(exprs) < 1 {
		return nil, NewDataError("empty data", ErrInvalidInput)
	}

	if len(exprs)%2 != 1 {
		shortStr := expr.String()
		if len(shortStr) > 10 {
			shortStr = shortStr[:10]
		}
		return nil, NewDataError(
			fmt.Sprintf("rest data from %s... has to be odd length elements", shortStr),
			ErrInvalidInput,
		)
	}

	if exprs[0].Kind != parser.ExprAtom || exprs[0].Atom.Value.Kind != parser.TypeValueSymbol {
		return nil, NewDataError("data's first element has to be symbol", ErrInvalidInput)
	}

	name := exprs[0].Atom.Value.Str
	var restArgs []ExprArg

	for i := 1; i < len(exprs); i += 2 {
		k := exprs[i]
		v := exprs[i+1]

		if k.Kind != parser.ExprAtom || k.Atom.Value.Kind != parser.TypeValueKeyword {
			return nil, NewDataError("has to be keyword value pairs", ErrInvalidInput)
		}

		if !IsNilSymbol(&v) {
			valData, err := DataFromExpr(&v)
			if err != nil {
				return nil, err
			}
			restArgs = append(restArgs, ExprArg{
				Key: k,
				Val: valData,
			})
		}
	}

	ed := &ExprData{
		name:     name,
		RestArgs: restArgs,
	}
	ed.initInnerMap()
	return ed, nil
}

// ExprDataFromStr tokenizes and parses an ExprData directly from a string using the provided parser.
func ExprDataFromStr(p *parser.Parser, s string) (*ExprData, error) {
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
	return ExprDataFromExpr(&lastExpr)
}

func (ed *ExprData) initInnerMap() {
	ed.innerMap = make(map[string]*Data, len(ed.RestArgs))
	for i := range ed.RestArgs {
		arg := &ed.RestArgs[i]
		if arg.Key.Kind == parser.ExprAtom && arg.Key.Atom.Value.Kind == parser.TypeValueKeyword {
			ed.innerMap[arg.Key.Atom.Value.Str] = &arg.Val
		}
	}
}

// GetName returns the symbol identifier name of the data expression.
func (ed *ExprData) GetName() string {
	if ed == nil {
		return ""
	}
	return ed.name
}

// Get retrieves a reference to the Data value associated with the specified keyword name.
func (ed *ExprData) Get(k string) *Data {
	if ed == nil {
		return nil
	}
	if ed.innerMap == nil {
		ed.initInnerMap()
	}
	return ed.innerMap[k]
}

// String serializes the expression into an S-expression string (name :k1 v1 :k2 v2).
func (ed *ExprData) String() string {
	if ed == nil {
		return "()"
	}
	if len(ed.RestArgs) == 0 {
		return "(" + ed.name + " )"
	}
	parts := make([]string, len(ed.RestArgs))
	for i, arg := range ed.RestArgs {
		parts[i] = arg.Key.String() + " " + arg.Val.String()
	}
	return "(" + ed.name + " " + strings.Join(parts, " ") + ")"
}

// Equal compares two ExprData structures for equality.
func (ed *ExprData) Equal(other ExprData) bool {
	if ed == nil {
		return false
	}
	if ed.name != other.name || len(ed.RestArgs) != len(other.RestArgs) {
		return false
	}
	for i := range ed.RestArgs {
		if !ed.RestArgs[i].Key.Equal(other.RestArgs[i].Key) {
			return false
		}
		if !ed.RestArgs[i].Val.Equal(other.RestArgs[i].Val) {
			return false
		}
	}
	return true
}
