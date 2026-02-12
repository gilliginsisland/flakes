package marshaler

import (
	"io"
	"reflect"
)

type structMarshaler struct {
	Fields []*fieldMarshaler
}

func (sm *structMarshaler) Marshal(v reflect.Value, p string, w io.Writer) error {
	if p != "" {
		p += "."
	}
	for _, fm := range sm.Fields {
		if err := fm.Marshal(v.Field(fm.Index), p, w); err != nil {
			return err
		}
	}
	return nil
}

type fieldMarshaler struct {
	Index     int
	Tag       string
	Marshaler marshaler
}

func (fm *fieldMarshaler) Marshal(v reflect.Value, p string, w io.Writer) error {
	return fm.Marshaler.Marshal(v, p+fm.Tag, w)
}

func newStructMarshaler(t reflect.Type) marshaler {
	var fields []*fieldMarshaler
	for i, numField := 0, t.NumField(); i < numField; i++ {
		sf := t.Field(i)
		fm := newFieldMarshaler(sf)
		if fm == nil {
			// TODO: check if there was a csd tag and we still skipped
			continue
		}
		fm.Index = i
		fields = append(fields, fm)
	}
	return &structMarshaler{
		Fields: fields,
	}
}

func newFieldMarshaler(sf reflect.StructField) *fieldMarshaler {
	t := sf.Type
	tag, found := sf.Tag.Lookup("csd")
	if !found {
		return nil
	}

	m := newMarshaler(t)
	if m == nil {
		return nil
	}

	return &fieldMarshaler{
		Tag:       tag,
		Marshaler: m,
	}
}
