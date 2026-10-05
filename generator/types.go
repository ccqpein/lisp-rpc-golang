package generator

import (
	"fmt"
	"strings"
	"unicode"
)

// RPCDataType represents the structural classification of a generated RPC entity.
type RPCDataType int

const (
	// RPCDataTypeMap represents an anonymous nested key-value map.
	RPCDataTypeMap RPCDataType = iota
	// RPCDataTypeList represents a list sequence.
	RPCDataTypeList
	// RPCDataTypeMsg represents a named message structure.
	RPCDataTypeMsg
	// RPCDataTypeRpc represents an RPC command structure.
	RPCDataTypeRpc
)

func (t RPCDataType) String() string {
	switch t {
	case RPCDataTypeMap:
		return "map"
	case RPCDataTypeList:
		return "list"
	case RPCDataTypeMsg:
		return "msg"
	case RPCDataTypeRpc:
		return "rpc"
	default:
		return "unknown"
	}
}

// ReservedWords lists Go language keywords that cannot be used as struct field names.
var ReservedWords = map[string]bool{
	"break":       true,
	"case":        true,
	"chan":        true,
	"const":       true,
	"continue":    true,
	"default":     true,
	"defer":       true,
	"else":        true,
	"fallthrough": true,
	"for":         true,
	"func":        true,
	"go":          true,
	"goto":        true,
	"if":          true,
	"import":      true,
	"interface":   true,
	"map":         true,
	"package":     true,
	"range":       true,
	"return":      true,
	"select":      true,
	"struct":      true,
	"switch":      true,
	"type":        true,
	"var":         true,
}

func isReservedWord(name string) bool {
	return ReservedWords[strings.ToLower(name)]
}

// GeneratedField represents a struct field definition to be rendered in Go code.
type GeneratedField struct {
	Name      string
	FieldType string
	Tag       string
	Comment   string
}

// NewGeneratedField creates a new GeneratedField validating against Go reserved words.
func NewGeneratedField(name, fieldType, comment string) (GeneratedField, error) {
	if isReservedWord(name) {
		return GeneratedField{}, fmt.Errorf("field name %s is reserved word", name)
	}

	tag := ToKebabCase(name)
	pascalName := KebabToPascalCase(name)
	return GeneratedField{
		Name:      pascalName,
		FieldType: fieldType,
		Tag:       tag,
		Comment:   comment,
	}, nil
}

// GeneratedStruct represents an intermediate representation of a Go struct to be rendered.
type GeneratedStruct struct {
	Name       string
	Fields     []GeneratedField
	Comment    string
	DataName   string
	RPCType    RPCDataType
	ReturnType string
}

// NewGeneratedStruct constructs a new GeneratedStruct with PascalCase naming conversions.
func NewGeneratedStruct(
	dataName string,
	fields []GeneratedField,
	comment string,
	ty RPCDataType,
	returnType string,
) *GeneratedStruct {
	var rt string
	if returnType != "" {
		rt = KebabToPascalCase(returnType)
	}

	return &GeneratedStruct{
		Name:       KebabToPascalCase(dataName),
		Fields:     fields,
		Comment:    comment,
		DataName:   dataName,
		RPCType:    ty,
		ReturnType: rt,
	}
}

// CommentFormatted formats Comment as Go doc comments (each line prefixed with //).
func (s GeneratedStruct) CommentFormatted() string {
	if s.Comment == "" {
		return ""
	}
	lines := strings.Split(s.Comment, "\n")
	var formatted []string
	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\r")
		if strings.HasPrefix(trimmed, "//") {
			formatted = append(formatted, trimmed)
		} else {
			formatted = append(formatted, "// "+trimmed)
		}
	}
	return strings.Join(formatted, "\n")
}

// KebabToPascalCase converts a kebab-case or snake_case string identifier into PascalCase.
func KebabToPascalCase(s string) string {
	if s == "" {
		return ""
	}

	// Split by hyphens or underscores
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_'
	})

	var b strings.Builder
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		runes := []rune(part)
		b.WriteRune(unicode.ToUpper(runes[0]))
		for i := 1; i < len(runes); i++ {
			b.WriteRune(runes[i])
		}
	}
	return b.String()
}

// ToKebabCase converts CamelCase or snake_case string identifiers into kebab-case.
func ToKebabCase(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if r == '_' {
			b.WriteByte('-')
			continue
		}
		if unicode.IsUpper(r) {
			if i > 0 && (unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1])) {
				b.WriteByte('-')
			} else if i > 0 && i+1 < len(runes) && unicode.IsUpper(runes[i-1]) && unicode.IsLower(runes[i+1]) {
				b.WriteByte('-')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// TypeTranslate maps Lisp-RPC primitive and composite type symbol names to Go types.
func TypeTranslate(sym string) string {
	switch strings.ToLower(sym) {
	case "string":
		return "string"
	case "number", "int":
		return "int64"
	case "float":
		return "float64"
	case "bool":
		return "bool"
	}

	if strings.HasPrefix(sym, "[]") {
		return "[]" + TypeTranslate(sym[2:])
	}
	if strings.HasPrefix(sym, "Vec<") && strings.HasSuffix(sym, ">") {
		inner := sym[4 : len(sym)-1]
		return "[]" + TypeTranslate(inner)
	}

	return KebabToPascalCase(sym)
}

// GenerateArg defines code generation options.
type GenerateArg int

const (
	// GenerateArgDefault only generates structure definitions without server RPC methods.
	GenerateArgDefault GenerateArg = iota
	// GenerateArgWithServer generates structure definitions and implements server RPC methods (ToRPCType, ReturnType).
	GenerateArgWithServer
)

// GenerateArgFromBool converts a withServer boolean into a GenerateArg.
func GenerateArgFromBool(withServer bool) GenerateArg {
	if withServer {
		return GenerateArgWithServer
	}
	return GenerateArgDefault
}
