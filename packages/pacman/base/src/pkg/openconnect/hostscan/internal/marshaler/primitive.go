package marshaler

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
)

type marshalerFunc func(v reflect.Value) string

func (f marshalerFunc) Marshal(v reflect.Value, p string, w io.Writer) error {
	_, err := io.WriteString(w, fmt.Sprintf("%s=\"%s\";\n", p, f(v)))
	return err
}

func marshalString(v reflect.Value) string {
	return v.String()
}

func marshalBool(v reflect.Value) string {
	return strconv.FormatBool(v.Bool())
}

func marshalInt(v reflect.Value) string {
	return strconv.FormatInt(v.Int(), 10)
}

func marshalUint(v reflect.Value) string {
	return strconv.FormatUint(v.Uint(), 10)
}

func marshalFloat32(v reflect.Value) string {
	return strconv.FormatFloat(v.Float(), 'f', -1, 32)
}

func marshalFloat64(v reflect.Value) string {
	return strconv.FormatFloat(v.Float(), 'f', -1, 64)
}
