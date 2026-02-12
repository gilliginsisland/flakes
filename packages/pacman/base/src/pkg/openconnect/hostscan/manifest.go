package hostscan

import (
	"encoding/xml"
	"io"
	"regexp"
	"strings"
)

var re = regexp.MustCompile(`^'([^']+)','([^']+)','([^']+)'$`)

type Manifest struct {
	XMLName xml.Name `xml:"data"`
	Fields  []Field  `xml:"hostscan>field"`
}

type Field struct {
	Type  string
	Label string
	Value string
}

func (f *Field) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	v := struct {
		Name  string `xml:"name,attr"`
		Value string `xml:"value,attr"`
	}{}
	if err := d.DecodeElement(&v, &start); err != nil {
		return err
	}

	switch {
	case strings.HasPrefix(v.Name, "lHostScanList"):
		if m := re.FindStringSubmatch(v.Value); m != nil {
			f.Type = m[1]
			f.Label = m[2]
			f.Value = m[3]
		}
	case strings.HasPrefix(v.Name, "cInspectorExtension"):
		f.Type = "Inspector"
		f.Label = v.Value
		f.Value = v.Value
	}

	return nil
}

func ReadManifest(r io.Reader) (*Manifest, error) {
	var m Manifest
	err := xml.NewDecoder(r).Decode(&m)
	return &m, err
}
