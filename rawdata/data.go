package rawdata

import (
	"fmt"
	"strings"

	"github.com/ccqpein/lisp-rpc-golang/parser"
)

// DataKind represents the variant category of a Data element.
type DataKind int

const (
	// DataExprKind represents a named S-expression data structure (name :key value ...).
	DataExprKind DataKind = iota
	// DataListKind represents a quoted sequence '("a" "b" 1).
	DataListKind
	// DataMapKind represents a quoted key-value map '(:a 1 :b 2).
	DataMapKind
	// DataValueKind represents a primitive atom value (symbol, string, keyword, or number).
	DataValueKind
)

// Data represents dynamic Lisp-RPC data: named expressions, lists, maps, or primitive values.
type Data struct {
	Kind   DataKind
	expr   *ExprData
	list   *ListData
	mapVal *MapData
	value  *parser.TypeValue
}

// NewDataExpr creates a Data wrapping an ExprData.
func NewDataExpr(ed *ExprData) Data {
	return Data{Kind: DataExprKind, expr: ed}
}

// NewDataList creates a Data wrapping a ListData.
func NewDataList(ld *ListData) Data {
	return Data{Kind: DataListKind, list: ld}
}

// NewDataMap creates a Data wrapping a MapData.
func NewDataMap(md *MapData) Data {
	return Data{Kind: DataMapKind, mapVal: md}
}

// NewDataValue creates a Data wrapping a parser.TypeValue.
func NewDataValue(tv parser.TypeValue) Data {
	return Data{Kind: DataValueKind, value: &tv}
}

// NewDataString creates a Data value containing a string literal.
func NewDataString(s string) Data {
	return NewDataValue(parser.NewString(s))
}

// NewDataSymbol creates a Data value containing a symbol.
func NewDataSymbol(s string) Data {
	return NewDataValue(parser.NewSymbol(s))
}

// NewDataKeyword creates a Data value containing a keyword.
func NewDataKeyword(k string) Data {
	return NewDataValue(parser.NewKeyword(k))
}

// NewDataInt creates a Data value containing a 64-bit integer.
func NewDataInt(i int64) Data {
	return NewDataValue(parser.NewNumber(parser.NewInt(i)))
}

// NewDataFloat creates a Data value containing a 64-bit float.
func NewDataFloat(f float64) Data {
	return NewDataValue(parser.NewNumber(parser.NewFloat(f)))
}

// IsExpr returns true if this is a named S-expression data structure.
func (d Data) IsExpr() bool {
	return d.Kind == DataExprKind
}

// IsList returns true if this is a quoted sequence.
func (d Data) IsList() bool {
	return d.Kind == DataListKind
}

// IsMap returns true if this is a quoted key-value map.
func (d Data) IsMap() bool {
	return d.Kind == DataMapKind
}

// IsValue returns true if this is a primitive atom value.
func (d Data) IsValue() bool {
	return d.Kind == DataValueKind
}

// Expr returns the inner ExprData pointer if this is a DataExprKind, or nil otherwise.
func (d Data) Expr() *ExprData {
	return d.expr
}

// List returns the inner ListData pointer if this is a DataListKind, or nil otherwise.
func (d Data) List() *ListData {
	return d.list
}

// Map returns the inner MapData pointer if this is a DataMapKind, or nil otherwise.
func (d Data) Map() *MapData {
	return d.mapVal
}

// Value returns the inner TypeValue pointer if this is a DataValueKind, or nil otherwise.
func (d Data) Value() *parser.TypeValue {
	return d.value
}

// AsValue returns the inner TypeValue and true if this is a DataValueKind.
func (d Data) AsValue() (*parser.TypeValue, bool) {
	if d.Kind == DataValueKind && d.value != nil {
		return d.value, true
	}
	return nil, false
}

// ToInt returns the integer value if this is a DataValueKind containing an integer.
func (d Data) ToInt() (int64, bool) {
	if tv, ok := d.AsValue(); ok {
		return tv.ToInt()
	}
	return 0, false
}

// ToFloat returns the float value if this is a DataValueKind containing a number.
func (d Data) ToFloat() (float64, bool) {
	if tv, ok := d.AsValue(); ok {
		return tv.ToFloat()
	}
	return 0, false
}

// GetString extracts the unquoted string content if this is a DataValueKind containing a string literal.
func (d Data) GetString() (string, error) {
	if tv, ok := d.AsValue(); ok {
		return tv.GetString()
	}
	return "", fmt.Errorf("%v isn't the Data::Value type that can get string", d)
}

// Get retrieves a reference to the Data field matching the given key name if this is an ExprData or MapData.
func (d Data) Get(k string) *Data {
	switch d.Kind {
	case DataExprKind:
		if d.expr != nil {
			return d.expr.Get(k)
		}
	case DataMapKind:
		if d.mapVal != nil {
			return d.mapVal.Get(k)
		}
	}
	return nil
}

// String serializes the data into a Lisp-RPC S-expression string.
func (d Data) String() string {
	switch d.Kind {
	case DataExprKind:
		if d.expr != nil {
			return d.expr.String()
		}
	case DataListKind:
		if d.list != nil {
			return d.list.String()
		}
	case DataMapKind:
		if d.mapVal != nil {
			return d.mapVal.String()
		}
	case DataValueKind:
		if d.value != nil {
			return d.value.String()
		}
	}
	return ""
}

// Equal compares two Data elements for deep equality.
func (d Data) Equal(other Data) bool {
	if d.Kind != other.Kind {
		return false
	}
	switch d.Kind {
	case DataExprKind:
		if d.expr == nil && other.expr == nil {
			return true
		}
		if d.expr == nil || other.expr == nil {
			return false
		}
		return d.expr.Equal(*other.expr)
	case DataListKind:
		if d.list == nil && other.list == nil {
			return true
		}
		if d.list == nil || other.list == nil {
			return false
		}
		return d.list.Equal(*other.list)
	case DataMapKind:
		if d.mapVal == nil && other.mapVal == nil {
			return true
		}
		if d.mapVal == nil || other.mapVal == nil {
			return false
		}
		return d.mapVal.Equal(*other.mapVal)
	case DataValueKind:
		if d.value == nil && other.value == nil {
			return true
		}
		if d.value == nil || other.value == nil {
			return false
		}
		return d.value.Equal(*other.value)
	default:
		return false
	}
}

// IsNilSymbol returns true if the expression is a symbol representing 'nil' (case-insensitive).
func IsNilSymbol(e *parser.Expr) bool {
	if e == nil || e.Kind != parser.ExprAtom {
		return false
	}
	if e.Atom.Value.Kind == parser.TypeValueSymbol && strings.EqualFold(e.Atom.Value.Str, "nil") {
		return true
	}
	return false
}

// IsTSymbol returns true if the expression is a symbol representing 't' (case-insensitive).
func IsTSymbol(e *parser.Expr) bool {
	if e == nil || e.Kind != parser.ExprAtom {
		return false
	}
	if e.Atom.Value.Kind == parser.TypeValueSymbol && strings.EqualFold(e.Atom.Value.Str, "t") {
		return true
	}
	return false
}

// DataFromExpr converts a parsed Lisp Expr node into a dynamic Data representation.
func DataFromExpr(e *parser.Expr) (Data, error) {
	if e == nil {
		return Data{}, NewDataError("nil expression", ErrInvalidInput)
	}

	switch e.Kind {
	case parser.ExprList:
		ed, err := ExprDataFromExpr(e)
		if err != nil {
			return Data{}, err
		}
		return NewDataExpr(ed), nil

	case parser.ExprQuote:
		if e.Quote == nil {
			return Data{}, NewDataError("cannot generate Data from the expr", ErrInvalidInput)
		}
		switch e.Quote.Kind {
		case parser.ExprList:
			exprs, err := e.Quote.FilterOutAllComments()
			if err != nil {
				return Data{}, err
			}
			if len(exprs) == 0 {
				ld, err := ListDataFromExpr(e)
				if err != nil {
					return Data{}, err
				}
				return NewDataList(ld), nil
			}
			switch exprs[0].Kind {
			case parser.ExprAtom:
				if exprs[0].Atom.Value.Kind == parser.TypeValueKeyword {
					md, err := MapDataFromExpr(e)
					if err != nil {
						return Data{}, err
					}
					return NewDataMap(md), nil
				}
				ld, err := ListDataFromExpr(e)
				if err != nil {
					return Data{}, err
				}
				return NewDataList(ld), nil
			case parser.ExprList, parser.ExprQuote:
				ld, err := ListDataFromExpr(e)
				if err != nil {
					return Data{}, err
				}
				return NewDataList(ld), nil
			default:
				return Data{}, NewDataError(fmt.Sprintf("cannot generate Data from the expr %v", e), ErrInvalidInput)
			}
		case parser.ExprAtom:
			return NewDataValue(e.Quote.Atom.Value), nil
		default:
			return Data{}, NewDataError(fmt.Sprintf("cannot generate Data from the expr %v", e), ErrInvalidInput)
		}

	case parser.ExprAtom:
		if e.Atom.Value.Kind == parser.TypeValueSymbol {
			if strings.EqualFold(e.Atom.Value.Str, "t") {
				return NewDataValue(e.Atom.Value), nil
			}
			return Data{}, NewDataError(fmt.Sprintf("cannot generate Data from the symbol %v", e.Atom), ErrInvalidInput)
		}
		return NewDataValue(e.Atom.Value), nil

	case parser.ExprComment:
		return Data{}, NewDataError("cannot generate Data from the comment", ErrInvalidInput)

	default:
		return Data{}, NewDataError("unsupported expr kind", ErrInvalidInput)
	}
}

// DataFromExprs converts a slice of parsed Lisp Expr nodes into Data instances.
func DataFromExprs(exprs []parser.Expr) ([]Data, error) {
	res := make([]Data, 0, len(exprs))
	for i := range exprs {
		d, err := DataFromExpr(&exprs[i])
		if err != nil {
			return nil, err
		}
		res = append(res, d)
	}
	return res, nil
}

// DataFromStr tokenizes and parses a Data instance directly from a string using the provided parser.
func DataFromStr(p *parser.Parser, s string) (Data, error) {
	if p == nil {
		p = parser.New()
	}
	if err := p.Tokenize(strings.NewReader(s)); err != nil {
		return Data{}, err
	}
	if _, err := p.ParseOne(); err != nil {
		return Data{}, err
	}
	if len(p.Exprs) == 0 {
		return Data{}, NewDataError("Cannot get the last expr", ErrCorruptedData)
	}
	lastExpr := p.Exprs[len(p.Exprs)-1]
	return DataFromExpr(&lastExpr)
}

// DataFromRootStr parses a root named data expression from a string slice.
// Returns an error if the root parsed element is not an ExprData.
func DataFromRootStr(s string, p *parser.Parser) (Data, error) {
	d, err := DataFromStr(p, s)
	if err != nil {
		return Data{}, err
	}
	if !d.IsExpr() {
		return Data{}, NewDataError("root data has to be expr data", ErrInvalidInput)
	}
	return d, nil
}

// Pair represents a keyword name and value for building dynamic ExprData.
type Pair struct {
	Key string
	Val any
}

// NewPair creates a new Pair.
func NewPair(k string, v any) Pair {
	return Pair{Key: k, Val: v}
}

// ToData converts a Go value into a Data representation.
func ToData(v any) (Data, error) {
	if v == nil {
		return Data{}, NewDataError("nil cannot be converted to Data", ErrInvalidInput)
	}
	switch val := v.(type) {
	case Data:
		return val, nil
	case *Data:
		if val == nil {
			return Data{}, NewDataError("nil *Data", ErrInvalidInput)
		}
		return *val, nil
	case ExprData:
		return NewDataExpr(&val), nil
	case *ExprData:
		return NewDataExpr(val), nil
	case ListData:
		return NewDataList(&val), nil
	case *ListData:
		return NewDataList(val), nil
	case MapData:
		return NewDataMap(&val), nil
	case *MapData:
		return NewDataMap(val), nil
	case parser.TypeValue:
		return NewDataValue(val), nil
	case string:
		return NewDataString(val), nil
	case int:
		return NewDataInt(int64(val)), nil
	case int64:
		return NewDataInt(val), nil
	case int32:
		return NewDataInt(int64(val)), nil
	case int16:
		return NewDataInt(int64(val)), nil
	case int8:
		return NewDataInt(int64(val)), nil
	case float64:
		return NewDataFloat(val), nil
	case float32:
		return NewDataFloat(float64(val)), nil
	case bool:
		if val {
			return NewDataSymbol("T"), nil
		}
		return NewDataSymbol("NIL"), nil
	default:
		return Data{}, NewDataError(fmt.Sprintf("cannot convert %T to Data", v), ErrInvalidInput)
	}
}

// NewData constructs a new root named data expression (ExprData) from a name and key-value pairs.
func NewData(name string, pairs ...Pair) (Data, error) {
	if _, err := parser.MakeSymbol(name); err != nil {
		return Data{}, NewDataError("invalid symbol name: "+name, ErrInvalidInput)
	}

	var restArgs []ExprArg
	for _, p := range pairs {
		d, err := ToData(p.Val)
		if err != nil {
			return Data{}, err
		}
		keyExpr := parser.NewExprAtom(parser.NewAtomKeyword(p.Key))
		restArgs = append(restArgs, ExprArg{
			Key: keyExpr,
			Val: d,
		})
	}

	ed, err := NewExprData(name, restArgs...)
	if err != nil {
		return Data{}, err
	}
	return NewDataExpr(ed), nil
}
