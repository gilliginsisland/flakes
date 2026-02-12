package env

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Decoder struct {
	env map[string]string
}

func NewDecoder(environ []string) *Decoder {
	env := make(map[string]string, len(environ))
	for _, e := range environ {
		if i := strings.IndexByte(e, '='); i >= 0 {
			key := e[:i]
			val := e[i+1:]
			env[key] = val
		}
	}
	return &Decoder{env: env}
}

func (d *Decoder) Unmarshal(target any) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("env: target must be a non-nil pointer to a struct")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("env: target must point to a struct")
	}

	t := v.Type()
	tName := t.Name()
	for i := range t.NumField() {
		field := v.Field(i)
		if !field.CanSet() {
			continue
		}

		typeField := t.Field(i)
		tag := typeField.Tag.Get("env")
		if tag == "" {
			continue
		}

		parts := strings.Split(tag, ",")
		key := parts[0]

		var required bool
		for _, opt := range parts[1:] {
			switch opt {
			case "required":
				required = true
			case "":
				// trailing comma — ignore
			default:
				return fmt.Errorf("env: unknown option %q for field '%s.%s'", opt, tName, typeField.Name)
			}
		}

		val, found := d.env[key]
		if !found {
			if required {
				return fmt.Errorf("env: required key %q for field '%s.%s' not found", key, tName, typeField.Name)
			}
			continue
		}

		ptr := field.Addr().Interface()
		if err := unmarshal(ptr, val); err != nil {
			return fmt.Errorf("env: error setting field '%s': %w", typeField.Name, err)
		}
	}
	return nil
}

func Unmarshal(target any, environ []string) error {
	return NewDecoder(environ).Unmarshal(target)
}

func unmarshal(ptr any, s string) error {
	if u, ok := ptr.(encoding.TextUnmarshaler); ok {
		return u.UnmarshalText([]byte(s))
	}

	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("env: expected non-nil pointer")
	}

	for v = v.Elem(); v.Kind() == reflect.Ptr; v = v.Elem() {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(s, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(s, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetUint(u)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		v.SetBool(b)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetFloat(f)
	default:
		return fmt.Errorf("env: unsupported kind %s", v.Kind())
	}
	return nil
}
