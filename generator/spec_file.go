package generator

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ccqpein/lisp-rpc-golang/parser"
)

// SpecFile represents a parsed .lisprpc specification file containing declarations.
type SpecFile struct {
	Pkg           *DefPkg
	Msgs          []*DefMsg
	RPCs          []*DefRPC
	symTable      map[string]bool
	targetPkgName string
}

// NewSpecFile creates a new empty SpecFile.
func NewSpecFile() *SpecFile {
	return &SpecFile{
		symTable: make(map[string]bool),
	}
}

// AddPkg registers the package declaration.
func (sf *SpecFile) AddPkg(pkg *DefPkg) error {
	if sf.Pkg != nil {
		return fmt.Errorf("package already defined: %s", sf.Pkg.PkgName)
	}
	sf.Pkg = pkg
	sf.targetPkgName = pkg.PkgName
	return nil
}

// AddMsg registers a parsed message specification, checking for duplicate symbol names.
func (sf *SpecFile) AddMsg(msg *DefMsg) error {
	sym := msg.SymbolName()
	if sf.symTable[sym] {
		return fmt.Errorf("sym %s already have", sym)
	}
	sf.symTable[sym] = true
	sf.Msgs = append(sf.Msgs, msg)
	return nil
}

// AddRPC registers a parsed RPC specification, checking for duplicate symbol names.
func (sf *SpecFile) AddRPC(rpc *DefRPC) error {
	sym := rpc.SymbolName()
	if sf.symTable[sym] {
		return fmt.Errorf("sym %s already have", sym)
	}
	sf.symTable[sym] = true
	sf.RPCs = append(sf.RPCs, rpc)
	return nil
}

// SetTargetPkgName sets the target package identifier name.
func (sf *SpecFile) SetTargetPkgName(name string) {
	sf.targetPkgName = name
}

// GetTargetPkgName returns the target package identifier name.
func (sf *SpecFile) GetTargetPkgName() string {
	return sf.targetPkgName
}

// ParseSpecFile reads and parses a .lisprpc specification from an io.Reader.
func ParseSpecFile(r io.Reader) (*SpecFile, error) {
	p := parser.New()
	if err := p.Tokenize(r); err != nil {
		return nil, err
	}
	if err := p.Parse(); err != nil {
		return nil, err
	}

	sf := NewSpecFile()
	for i := range p.Exprs {
		expr := &p.Exprs[i]
		if expr.IsComment() {
			continue
		}

		if IfDefRPCExpr(expr) {
			rpcSpec, err := ParseDefRPCExpr(expr)
			if err != nil {
				return nil, err
			}
			if err := sf.AddRPC(rpcSpec); err != nil {
				return nil, err
			}
		} else if IfDefMsgExpr(expr) {
			msgSpec, err := ParseDefMsgExpr(expr)
			if err != nil {
				return nil, err
			}
			if err := sf.AddMsg(msgSpec); err != nil {
				return nil, err
			}
		} else if IfDefPkgExpr(expr) {
			pkgSpec, err := ParseDefPkgExpr(expr)
			if err != nil {
				return nil, err
			}
			if err := sf.AddPkg(pkgSpec); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("unknown expr: %v", expr)
		}
	}

	return sf, nil
}

// ParseSpecString parses a .lisprpc specification from a raw string.
func ParseSpecString(source string) (*SpecFile, error) {
	return ParseSpecFile(strings.NewReader(source))
}

// GenCodeString generates the file contents for all target files in memory.
func (sf *SpecFile) GenCodeString() (map[string]string, error) {
	files := make(map[string]string)

	pkgName := sf.targetPkgName
	if pkgName == "" {
		pkgName = "rpc"
	}

	header, err := RenderTemplate(DefaultHeaderTemplate(), map[string]any{
		"PackageName": pkgName,
	})
	if err != nil {
		return nil, err
	}

	var libBlocks []string
	libBlocks = append(libBlocks, header)

	for _, msg := range sf.Msgs {
		structs, err := msg.GenerateStructs()
		if err != nil {
			return nil, err
		}
		for _, s := range structs {
			sCode, err := RenderTemplate(DefaultStructTemplate(), s)
			if err != nil {
				return nil, err
			}
			implCode, err := RenderTemplate(DefaultRPCImplTemplate(), s)
			if err != nil {
				return nil, err
			}
			libBlocks = append(libBlocks, sCode+"\n\n"+implCode)
		}
	}

	for _, rpc := range sf.RPCs {
		structs, err := rpc.GenerateStructs()
		if err != nil {
			return nil, err
		}
		for _, s := range structs {
			sCode, err := RenderTemplate(DefaultStructTemplate(), s)
			if err != nil {
				return nil, err
			}
			implCode, err := RenderTemplate(DefaultRPCImplTemplate(), s)
			if err != nil {
				return nil, err
			}
			libBlocks = append(libBlocks, sCode+"\n\n"+implCode)
		}
	}

	files["rpc_libs.go"] = strings.Join(libBlocks, "\n\n") + "\n"

	var modContent string
	if sf.Pkg != nil {
		c, err := sf.Pkg.GenCode()
		if err != nil {
			return nil, err
		}
		modContent = c
	} else {
		var err error
		modContent, err = RenderTemplate(DefaultPackageTemplate(), &DefPkg{PkgName: pkgName})
		if err != nil {
			return nil, err
		}
	}
	files["go.mod"] = modContent + "\n"

	return files, nil
}

// GenCode writes generated code files to disk under outputPath/<target_pkg_name>/.
func (sf *SpecFile) GenCode(outputPath string) error {
	pkgName := sf.targetPkgName
	if pkgName == "" {
		return fmt.Errorf("no target package name specified")
	}

	targetDir := filepath.Join(outputPath, pkgName)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", targetDir, err)
	}

	files, err := sf.GenCodeString()
	if err != nil {
		return err
	}

	for fileName, content := range files {
		filePath := filepath.Join(targetDir, fileName)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", filePath, err)
		}
	}

	return nil
}
