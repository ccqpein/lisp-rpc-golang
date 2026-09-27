package generator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ccqpein/lisp-rpc-golang/parser"
)

// DefMsg represents a parsed (def-msg name :key type ...) specification declaration.
type DefMsg struct {
	MsgName  string
	RestExpr []parser.Expr
	MsgType  RPCDataType
}

// NewDefMsg creates a new DefMsg validating that restExpr forms keyword-value pairs.
func NewDefMsg(msgName string, restExpr []parser.Expr, ty RPCDataType) (*DefMsg, error) {
	if len(restExpr)%2 != 0 {
		return nil, errors.New("parsing failed, msg name arguments should be keyword-value pairs")
	}

	for i := 0; i < len(restExpr); i += 2 {
		k := &restExpr[i]
		if k.Kind != parser.ExprAtom || k.Atom.Value.Kind != parser.TypeValueKeyword {
			return nil, errors.New("parsing failed, msg name arguments should be keyword-value pairs")
		}
	}

	return &DefMsg{
		MsgName:  msgName,
		RestExpr: restExpr,
		MsgType:  ty,
	}, nil
}

// IfDefMsgExpr checks if an expression is a (def-msg ...) list expression.
func IfDefMsgExpr(expr *parser.Expr) bool {
	if expr == nil || expr.Kind != parser.ExprList {
		return false
	}
	if len(expr.List) == 0 {
		return false
	}
	first := expr.List[0]
	if first.Kind != parser.ExprAtom || first.Atom.Value.Kind != parser.TypeValueSymbol {
		return false
	}
	return first.Atom.Value.Str == "def-msg"
}

// ParseDefMsgExpr parses a DefMsg from an Expr AST node.
func ParseDefMsgExpr(expr *parser.Expr) (*DefMsg, error) {
	if !IfDefMsgExpr(expr) {
		return nil, errors.New("parsing failed, the first symbol should be def-msg")
	}

	if len(expr.List) < 2 {
		return nil, errors.New("parsing failed, def-msg missing message name")
	}

	nameAtom := expr.List[1]
	if nameAtom.Kind != parser.ExprAtom || nameAtom.Atom.Value.Kind != parser.TypeValueSymbol {
		return nil, errors.New("parsing failed, msg name should be symbol")
	}

	return NewDefMsg(nameAtom.Atom.Value.Str, expr.List[2:], RPCDataTypeMsg)
}

// ParseDefMsg parses a DefMsg from a raw S-expression string.
func ParseDefMsg(source string) (*DefMsg, error) {
	p := parser.New()
	if err := p.Tokenize(strings.NewReader(source)); err != nil {
		return nil, err
	}
	if err := p.Parse(); err != nil {
		return nil, err
	}

	if len(p.Exprs) == 0 {
		return nil, fmt.Errorf("no expression found in %s", source)
	}

	return ParseDefMsgExpr(&p.Exprs[len(p.Exprs)-1])
}

func deQuoted(e *parser.Expr) *parser.Expr {
	if e != nil && e.Kind == parser.ExprQuote && e.Quote != nil {
		return deQuoted(e.Quote)
	}
	return e
}

// CreateGenStructs transforms this message specification into GeneratedStruct definitions.
func (dm *DefMsg) CreateGenStructs() ([]*GeneratedStruct, error) {
	var res []*GeneratedStruct
	var fields []GeneratedField

	for i := 0; i < len(dm.RestExpr); i += 2 {
		k := &dm.RestExpr[i]
		v := &dm.RestExpr[i+1]

		if k.Kind != parser.ExprAtom || k.Atom.Value.Kind != parser.TypeValueKeyword {
			return nil, errors.New("create gen structs failed, arguments has to be the keywords-value pair")
		}
		f := k.Atom.Value.Str

		vUnquoted := deQuoted(v)

		if vUnquoted.Kind == parser.ExprAtom && vUnquoted.Atom.Value.Kind == parser.TypeValueSymbol {
			t := vUnquoted.Atom.Value.Str
			field, err := NewGeneratedField(f, TypeTranslate(t), "")
			if err != nil {
				return nil, err
			}
			fields = append(fields, field)
		} else if vUnquoted.Kind == parser.ExprList {
			innerExprs := vUnquoted.List
			if len(innerExprs) < 2 {
				return nil, errors.New("create gen structs failed, anonymity type can only be the (map|list|optional 'type)")
			}

			firstElem := &innerExprs[0]
			if firstElem.Kind == parser.ExprAtom {
				if firstElem.Atom.Value.Kind == parser.TypeValueKeyword {
					// Anonymous map type: first element is a keyword
					newMsgName := dm.MsgName + "-" + f
					subMsg, err := NewDefMsg(newMsgName, innerExprs, RPCDataTypeMap)
					if err != nil {
						return nil, err
					}
					subStructs, err := subMsg.CreateGenStructs()
					if err != nil {
						return nil, err
					}
					res = append(res, subStructs...)

					field, err := NewGeneratedField(f, TypeTranslate(newMsgName), "")
					if err != nil {
						return nil, err
					}
					fields = append(fields, field)
					continue
				}

				if firstElem.Atom.Value.Kind == parser.TypeValueSymbol {
					sym := firstElem.Atom.Value.Str
					secondElem := deQuoted(&innerExprs[1])
					if secondElem.Kind != parser.ExprAtom || secondElem.Atom.Value.Kind != parser.TypeValueSymbol {
						return nil, errors.New("create gen structs failed, composite element type must be a symbol")
					}
					elemType := secondElem.Atom.Value.Str

					if sym == "list" {
						newTypeName := "[]" + TypeTranslate(elemType)
						field, err := NewGeneratedField(f, newTypeName, "")
						if err != nil {
							return nil, err
						}
						fields = append(fields, field)
						continue
					} else if sym == "optional" {
						newTypeName := "*" + TypeTranslate(elemType)
						field, err := NewGeneratedField(f, newTypeName, "")
						if err != nil {
							return nil, err
						}
						fields = append(fields, field)
						continue
					}
				}
			}

			return nil, errors.New("create gen structs failed, anonymity type can only be the (map|list|optional 'type)")
		} else {
			return nil, errors.New("create gen structs failed, arguments has to be the keywords-value pair")
		}
	}

	mainStruct := NewGeneratedStruct(dm.MsgName, fields, "", dm.MsgType, "")
	res = append(res, mainStruct)

	return res, nil
}

// GenCode renders Go code for this message and any sub-structs.
func (dm *DefMsg) GenCode() (string, error) {
	structs, err := dm.CreateGenStructs()
	if err != nil {
		return "", err
	}

	var bucket []string
	for _, s := range structs {
		sCode, err := RenderTemplate(DefaultStructTemplate(), s)
		if err != nil {
			return "", err
		}
		implCode, err := RenderTemplate(DefaultRPCImplTemplate(), s)
		if err != nil {
			return "", err
		}
		bucket = append(bucket, sCode+"\n\n"+implCode)
	}

	return strings.Join(bucket, "\n\n"), nil
}

// SymbolName returns the message symbol name.
func (dm *DefMsg) SymbolName() string {
	return dm.MsgName
}

// TargetFile indicates this spec item generates library code.
func (dm *DefMsg) TargetFile() TargetFileType {
	return TargetFileLib
}

// GenerateStructs transforms this message into generated struct representations.
func (dm *DefMsg) GenerateStructs() ([]*GeneratedStruct, error) {
	return dm.CreateGenStructs()
}
