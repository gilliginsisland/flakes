package marshaler

import (
	"fmt"
	"io"
	"reflect"
)

type mapMarshaler struct {
	ElemMarshaler marshaler
}

func (m *mapMarshaler) Marshal(v reflect.Value, p string, w io.Writer) error {
	vlen := v.Len()
	if vlen == 0 {
		return nil
	}

	for _, key := range v.MapKeys() {
		kp := fmt.Sprintf(`%s["%s"]`, p, key.String())
		if _, err := io.WriteString(w, kp+"={};\n"); err != nil {
			return err
		}

		mv := v.MapIndex(key)
		if err := m.ElemMarshaler.Marshal(mv, kp, w); err != nil {
			return err
		}
	}

	return nil
}

func newMapMarshaler(t reflect.Type) marshaler {
	if t.Key().Kind() != reflect.String {
		return nil
	}

	et := t.Elem()
	m := newMarshaler(et)
	if m == nil {
		return nil
	}

	return &mapMarshaler{
		ElemMarshaler: m,
	}
}
