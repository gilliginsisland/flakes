package marshaler

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

type Encoder struct {
	w io.Writer
}

func (enc *Encoder) Encode(i any, p string) error {
	v := reflect.ValueOf(i)
	t := v.Type()
	m := newMarshaler(t)
	if m == nil {
		return fmt.Errorf("marshal: no marshaler for type %q", t)
	}
	return m.Marshal(v, p, enc.w)
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

func Marshal(i any, p string) (string, error) {
	var b strings.Builder
	enc := NewEncoder(&b)
	err := enc.Encode(i, p)
	return b.String(), err
}
