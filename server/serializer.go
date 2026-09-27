package server

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/ccqpein/lisp-rpc-golang/rawdata"
)

// toKebabCase converts CamelCase or snake_case string identifiers into kebab-case.

func toKebabCase(s string) string {
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

func fieldKey(f reflect.StructField) string {
	tag := f.Tag.Get("lisp-rpc")
	if tag != "" && tag != "-" {
		return tag
	}
	return toKebabCase(f.Name)
}

func structTypeName(t reflect.Type) string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return toKebabCase(t.Name())
}

// Marshal is an alias for SerializeLisp.
func Marshal(v any) (string, error) {
	return SerializeLisp(v)
}

// Unmarshal is an alias for DeserializeLisp.
func Unmarshal(raw string, target any) error {
	return DeserializeLisp(raw, target)
}

// SerializeLisp serializes a Go value into a Lisp-RPC S-expression string.
func SerializeLisp(v any) (string, error) {
	if v == nil {
		return "nil", nil
	}

	if s, ok := v.(LispSerializer); ok {
		return s.SerializeLisp()
	}

	val := reflect.ValueOf(v)
	typ := val.Type()

	switch val.Kind() {
	case reflect.Pointer, reflect.Interface:
		if val.IsNil() {
			return "nil", nil
		}
		return SerializeLisp(val.Elem().Interface())

	case reflect.Bool:
		if val.Bool() {
			return "t", nil
		}
		return "nil", nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(val.Uint(), 10), nil

	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(val.Float(), 'g', -1, 64), nil

	case reflect.String:
		return "\"" + val.String() + "\"", nil

	case reflect.Struct:
		if typ.NumField() == 0 {
			return "nil", nil
		}

		var structName string
		isMap := false
		if toRPC, ok := v.(ToRPCType); ok {
			rpcType := toRPC.ToRPCType()
			switch rpcType.Kind {
			case RPCTypeRPC, RPCTypeMsg:
				structName = rpcType.Name
			case RPCTypeMap:
				isMap = true
			}
		}
		if structName == "" {
			structName = structTypeName(typ)
		}

		var pairs []string
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if f.PkgPath != "" { // unexported field
				continue
			}
			fVal := val.Field(i)
			fKey := fieldKey(f)
			fSerialized, err := SerializeLisp(fVal.Interface())
			if err != nil {
				return "", err
			}
			pairs = append(pairs, ":"+fKey+" "+fSerialized)
		}

		if isMap {
			if len(pairs) == 0 {
				return "'()", nil
			}
			return "'(" + strings.Join(pairs, " ") + ")", nil
		}

		if len(pairs) == 0 {
			return "(" + structName + " )", nil
		}
		return "(" + structName + " " + strings.Join(pairs, " ") + ")", nil

	case reflect.Slice, reflect.Array:
		var items []string
		for i := 0; i < val.Len(); i++ {
			itemStr, err := SerializeLisp(val.Index(i).Interface())
			if err != nil {
				return "", err
			}
			items = append(items, itemStr)
		}
		return "'(" + strings.Join(items, " ") + ")", nil

	case reflect.Map:
		var pairs []string
		iter := val.MapRange()
		for iter.Next() {
			k := iter.Key()
			v := iter.Value()
			kStr := toKebabCase(fmt.Sprint(k.Interface()))
			vStr, err := SerializeLisp(v.Interface())
			if err != nil {
				return "", err
			}
			pairs = append(pairs, ":"+kStr+" "+vStr)
		}
		return "'(" + strings.Join(pairs, " ") + ")", nil

	default:
		return "", fmt.Errorf("unsupported type for Lisp-RPC serialization: %T", v)
	}
}

// DeserializeLisp parses an S-expression string and populates the target struct pointer.
func DeserializeLisp(raw string, target any) error {
	targetVal := reflect.ValueOf(target)
	if targetVal.Kind() != reflect.Pointer || targetVal.IsNil() {
		return errors.New("target must be a non-nil pointer")
	}

	d, err := rawdata.DataFromRootStr(raw, nil)
	if err != nil {
		d, err = rawdata.DataFromStr(nil, raw)
		if err != nil {
			return err
		}
	}

	elem := targetVal.Elem()
	return deserializeDataIntoValue(d, elem)
}

func deserializeDataIntoValue(d rawdata.Data, target reflect.Value) error {
	switch target.Kind() {
	case reflect.Pointer:
		if target.IsNil() {
			target.Set(reflect.New(target.Type().Elem()))
		}
		return deserializeDataIntoValue(d, target.Elem())

	case reflect.Struct:
		if !d.IsExpr() && !d.IsMap() {
			return fmt.Errorf("cannot deserialize non-expr/non-map data into struct %s", target.Type().Name())
		}

		typ := target.Type()
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if f.PkgPath != "" {
				continue
			}
			k := fieldKey(f)
			fieldVal := d.Get(k)
			if fieldVal == nil {
				continue
			}
			if err := setFieldValue(target.Field(i), *fieldVal); err != nil {
				return fmt.Errorf("field %s: %w", f.Name, err)
			}
		}
		return nil

	case reflect.String:
		if s, err := d.GetString(); err == nil {
			target.SetString(s)
			return nil
		}
		if d.IsValue() {
			target.SetString(d.Value().Str)
			return nil
		}
		return errors.New("cannot deserialize into string")

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if i, ok := d.ToInt(); ok {
			target.SetInt(i)
			return nil
		}
		return errors.New("cannot deserialize into int")

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if i, ok := d.ToInt(); ok {
			target.SetUint(uint64(i))
			return nil
		}
		return errors.New("cannot deserialize into uint")

	case reflect.Float32, reflect.Float64:
		if f, ok := d.ToFloat(); ok {
			target.SetFloat(f)
			return nil
		}
		return errors.New("cannot deserialize into float")

	case reflect.Bool:
		if d.IsValue() {
			s := strings.ToLower(d.Value().Str)
			target.SetBool(s == "t" || s == "true")
			return nil
		}
		return errors.New("cannot deserialize into bool")

	default:
		return fmt.Errorf("unsupported target kind %s", target.Kind())
	}
}

func setFieldValue(field reflect.Value, dataVal rawdata.Data) error {
	switch field.Kind() {
	case reflect.Pointer:
		elemType := field.Type().Elem()
		newElem := reflect.New(elemType)
		if err := setFieldValue(newElem.Elem(), dataVal); err != nil {
			return err
		}
		field.Set(newElem)
		return nil

	case reflect.String:
		if s, err := dataVal.GetString(); err == nil {
			field.SetString(s)
			return nil
		}
		if dataVal.IsValue() {
			field.SetString(dataVal.Value().Str)
			return nil
		}
		return errors.New("expected string value")

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if i, ok := dataVal.ToInt(); ok {
			field.SetInt(i)
			return nil
		}
		return errors.New("expected integer value")

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if i, ok := dataVal.ToInt(); ok {
			field.SetUint(uint64(i))
			return nil
		}
		return errors.New("expected uint value")

	case reflect.Float32, reflect.Float64:
		if f, ok := dataVal.ToFloat(); ok {
			field.SetFloat(f)
			return nil
		}
		return errors.New("expected float value")

	case reflect.Bool:
		if dataVal.IsValue() {
			s := strings.ToLower(dataVal.Value().Str)
			field.SetBool(s == "t" || s == "true")
			return nil
		}
		return errors.New("expected bool value")

	case reflect.Struct:
		return deserializeDataIntoValue(dataVal, field)

	case reflect.Slice:
		if !dataVal.IsList() {
			return errors.New("expected list for slice")
		}
		items := dataVal.List().Items()
		slice := reflect.MakeSlice(field.Type(), len(items), len(items))
		for i, item := range items {
			if err := setFieldValue(slice.Index(i), item); err != nil {
				return err
			}
		}
		field.Set(slice)
		return nil

	default:
		return fmt.Errorf("unsupported field type %s", field.Type())
	}
}
