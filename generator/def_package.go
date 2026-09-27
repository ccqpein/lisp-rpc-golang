package generator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ccqpein/lisp-rpc-golang/parser"
)

// DefPkg represents a parsed (def-rpc-package name) declaration.
type DefPkg struct {
	PkgName string
}

// IfDefPkgExpr checks if an expression is a (def-rpc-package ...) declaration.
func IfDefPkgExpr(expr *parser.Expr) bool {
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
	return first.Atom.Value.Str == "def-rpc-package"
}

// ParseDefPkgExpr parses a DefPkg from an Expr AST node.
func ParseDefPkgExpr(expr *parser.Expr) (*DefPkg, error) {
	if !IfDefPkgExpr(expr) {
		return nil, errors.New("parsing failed, the first symbol should be def-rpc-package")
	}

	if len(expr.List) < 2 {
		return nil, errors.New("parsing failed, def-rpc-package missing package name")
	}

	nameAtom := expr.List[1]
	if nameAtom.Kind != parser.ExprAtom || nameAtom.Atom.Value.Kind != parser.TypeValueSymbol {
		return nil, errors.New("parsing failed, pkg name should be symbol")
	}

	return &DefPkg{
		PkgName: nameAtom.Atom.Value.Str,
	}, nil
}

// ParseDefPkg parses a DefPkg from a raw S-expression string.
func ParseDefPkg(source string) (*DefPkg, error) {
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

	return ParseDefPkgExpr(&p.Exprs[len(p.Exprs)-1])
}

// GenCode generates package configuration code (e.g., go.mod).
func (dp *DefPkg) GenCode() (string, error) {
	return RenderTemplate(DefaultPackageTemplate(), dp)
}

// SymbolName returns the declared package name.
func (dp *DefPkg) SymbolName() string {
	return dp.PkgName
}
