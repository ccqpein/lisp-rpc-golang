package rawdata

import (
	"strings"

	"github.com/ccqpein/lisp-rpc-golang/parser"
)

// MapData represents a quoted key-value map data structure '(:k1 v1 :k2 v2).
type MapData struct {
	kwrds   []string
	dataMap map[string]Data
}

// NewMapData creates a new MapData with the specified keyword order and map.
func NewMapData(kwrds []string, m map[string]Data) *MapData {
	table := make(map[string]Data, len(m))
	for k, v := range m {
		table[k] = v
	}
	return &MapData{
		kwrds:   append([]string(nil), kwrds...),
		dataMap: table,
	}
}

// MapDataFromExpr parses a MapData from a quoted map S-expression.
func MapDataFromExpr(expr *parser.Expr) (*MapData, error) {
	if expr == nil || expr.Kind != parser.ExprQuote || expr.Quote == nil {
		return nil, NewDataError("MapData has to be quoted like '(:a 1 :b 2)", ErrInvalidInput)
	}

	if expr.Quote.Kind != parser.ExprList {
		return nil, NewDataError("MapData has to be quoted like '(:a 1 :b 2)", ErrInvalidInput)
	}

	rawItems, err := expr.Quote.FilterOutAllComments()
	if err != nil {
		return nil, err
	}

	if len(rawItems)%2 != 0 {
		return nil, NewDataError("MapData has to be keyword pairs like '(:a 1 :b 2)", ErrInvalidInput)
	}

	var kwrds []string
	table := make(map[string]Data, len(rawItems)/2)

	for i := 0; i < len(rawItems); i += 2 {
		k := rawItems[i]
		v := rawItems[i+1]

		if k.Kind != parser.ExprAtom || k.Atom.Value.Kind != parser.TypeValueKeyword {
			return nil, NewDataError("MapData has to be keyword pairs like '(:a 1 :b 2)", ErrInvalidInput)
		}

		keyStr := k.Atom.Value.Str
		if !IsNilSymbol(&v) {
			valData, err := DataFromExpr(&v)
			if err != nil {
				return nil, err
			}
			kwrds = append(kwrds, keyStr)
			table[keyStr] = valData
		}
	}

	return &MapData{
		kwrds:   kwrds,
		dataMap: table,
	}, nil
}

// MapDataFromStr tokenizes and parses a MapData directly from a string using the provided parser.
func MapDataFromStr(p *parser.Parser, s string) (*MapData, error) {
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
	return MapDataFromExpr(&lastExpr)
}

// Get retrieves a reference to the Data value associated with the specified keyword name.
func (m *MapData) Get(k string) *Data {
	if m == nil || m.dataMap == nil {
		return nil
	}
	if val, ok := m.dataMap[k]; ok {
		return &val
	}
	return nil
}

// Len returns the number of key-value pairs in the map.
func (m *MapData) Len() int {
	if m == nil {
		return 0
	}
	return len(m.kwrds)
}

// Keys returns the list of keyword keys in their original parsed order.
func (m *MapData) Keys() []string {
	if m == nil {
		return nil
	}
	return m.kwrds
}

// String serializes the map into a quoted S-expression string '(:k1 v1 :k2 v2).
func (m *MapData) String() string {
	if m == nil {
		return "'()"
	}
	parts := make([]string, 0, len(m.kwrds)*2)
	for _, k := range m.kwrds {
		parts = append(parts, ":"+k)
		if val, ok := m.dataMap[k]; ok {
			parts = append(parts, val.String())
		} else {
			parts = append(parts, "corrupted data")
		}
	}
	return "'(" + strings.Join(parts, " ") + ")"
}

// Equal compares two MapData structures for equality.
func (m *MapData) Equal(other MapData) bool {
	if m == nil {
		return false
	}
	if len(m.dataMap) != len(other.dataMap) {
		return false
	}
	for k, v := range m.dataMap {
		otherV, ok := other.dataMap[k]
		if !ok || !v.Equal(otherV) {
			return false
		}
	}
	return true
}
