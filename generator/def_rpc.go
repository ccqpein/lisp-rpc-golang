package generator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ccqpein/lisp-rpc-golang/parser"
)

// DefRPC represents a parsed (def-rpc name [doc] '(:key type ...) 'return-type) declaration.
type DefRPC struct {
	RPCName    string
	Doc        string
	Args       []parser.Expr
	ReturnType string
}

// IfDefRPCExpr checks if an expression is a (def-rpc ...) list expression.
func IfDefRPCExpr(expr *parser.Expr) bool {
	if expr == nil || expr.Kind != parser.ExprList {
		return false
	}
	if len(expr.List) == 0 {
		return false
	}
	first := expr.List[0]
	if first.Kind != parser.ExprAtom || first.Atom.Kind != parser.TypeValueSymbol {
		return false
	}
	return first.Atom.Str == "def-rpc"
}

// ParseDefRPCExpr parses a DefRPC from an Expr AST node.
func ParseDefRPCExpr(expr *parser.Expr) (*DefRPC, error) {
	if !IfDefRPCExpr(expr) {
		return nil, errors.New("parsing failed, the first symbol should be def-rpc")
	}

	if len(expr.List) < 3 {
		return nil, errors.New("parsing failed, def-rpc missing arguments")
	}

	rpcNameAtom := expr.List[1]
	if rpcNameAtom.Kind != parser.ExprAtom || rpcNameAtom.Atom.Kind != parser.TypeValueSymbol {
		return nil, errors.New("parsing failed, rpc name should be symbol")
	}
	rpcName := rpcNameAtom.Atom.Str

	var doc string
	idx := 2
	if idx < len(expr.List) {
		first := &expr.List[idx]
		if first.Kind == parser.ExprAtom && first.Atom.Kind == parser.TypeValueString {
			doc = first.Atom.Str
			idx++
		}
	}

	if idx >= len(expr.List) {
		return nil, errors.New("parsing failed, def-rpc missing arguments")
	}

	rawArgs := deQuoted(&expr.List[idx])
	if rawArgs.Kind != parser.ExprList {
		return nil, errors.New("parsing failed, second arguments has to be list of keyword-value pairs")
	}
	idx++

	var returnType string
	if idx < len(expr.List) {
		rtExpr := deQuoted(&expr.List[idx])
		if rtExpr.Kind != parser.ExprAtom || rtExpr.Atom.Kind != parser.TypeValueSymbol {
			return nil, errors.New("parsing failed, return type has to be quoted")
		}
		returnType = rtExpr.Atom.Str
	}

	return &DefRPC{
		RPCName:    rpcName,
		Doc:        doc,
		Args:       rawArgs.List,
		ReturnType: returnType,
	}, nil
}

// ParseDefRPC parses a DefRPC from a raw S-expression string.
func ParseDefRPC(source string) (*DefRPC, error) {
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

	return ParseDefRPCExpr(&p.Exprs[len(p.Exprs)-1])
}

// CreateGenStructs transforms this RPC specification into GeneratedStruct definitions.
func (dr *DefRPC) CreateGenStructs() ([]*GeneratedStruct, error) {
	if len(dr.Args)%2 != 0 {
		return nil, errors.New("create gen structs failed, arguments has to be the keywords-value pair")
	}

	var res []*GeneratedStruct
	var fields []GeneratedField

	for i := 0; i < len(dr.Args); i += 2 {
		k := &dr.Args[i]
		v := &dr.Args[i+1]

		if k.Kind != parser.ExprAtom || k.Atom.Kind != parser.TypeValueKeyword {
			return nil, errors.New("create gen structs failed, arguments has to be the keywords-value pair")
		}
		f := k.Atom.Str

		vUnquoted := deQuoted(v)

		if vUnquoted.Kind == parser.ExprAtom && vUnquoted.Atom.Kind == parser.TypeValueSymbol {
			t := vUnquoted.Atom.Str
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
				if firstElem.Atom.Kind == parser.TypeValueKeyword {
					// Anonymous map type: first element is a keyword
					newMsgName := dr.RPCName + "-" + f
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

				if firstElem.Atom.Kind == parser.TypeValueSymbol {
					sym := firstElem.Atom.Str
					secondElem := deQuoted(&innerExprs[1])
					if secondElem.Kind != parser.ExprAtom || secondElem.Atom.Kind != parser.TypeValueSymbol {
						return nil, errors.New("create gen structs failed, composite element type must be a symbol")
					}
					elemType := secondElem.Atom.Str

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

	mainStruct := NewGeneratedStruct(dr.RPCName, fields, dr.Doc, RPCDataTypeRpc, dr.ReturnType)
	res = append(res, mainStruct)

	return res, nil
}

// GenCode renders Go code for this RPC request structure and any sub-structs.
func (dr *DefRPC) GenCode() (string, error) {
	structs, err := dr.CreateGenStructs()
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

// SymbolName returns the RPC command symbol name.
func (dr *DefRPC) SymbolName() string {
	return dr.RPCName
}

// GenerateStructs transforms this RPC command into generated struct representations.
func (dr *DefRPC) GenerateStructs() ([]*GeneratedStruct, error) {
	return dr.CreateGenStructs()
}
