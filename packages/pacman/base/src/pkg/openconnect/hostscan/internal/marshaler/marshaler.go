package marshaler

import (
	"io"
	"reflect"
)

type marshaler interface {
	Marshal(v reflect.Value, p string, w io.Writer) error
}

func newMarshaler(t reflect.Type) marshaler {
	switch t.Kind() {
	case reflect.String:
		return marshalerFunc(marshalString)

	case reflect.Bool:
		return marshalerFunc(marshalBool)

	case reflect.Int,
		reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64:
		return marshalerFunc(marshalInt)

	case reflect.Uint,
		reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64:
		return marshalerFunc(marshalUint)

	case reflect.Float32:
		return marshalerFunc(marshalFloat32)

	case reflect.Float64:
		return marshalerFunc(marshalFloat64)

	case reflect.Struct:
		return newStructMarshaler(t)

	case reflect.Map:
		return newMapMarshaler(t)
	}

	return nil
}
