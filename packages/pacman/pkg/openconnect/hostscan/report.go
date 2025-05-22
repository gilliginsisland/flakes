package hostscan

import (
	"io"

	"github.com/gilliginsisland/pacman/pkg/openconnect/hostscan/internal/marshaler"
)

type Report struct {
	OS               OS                          `csd:"os"`
	Policy           Policy                      `csd:"policy"`
	Device           Device                      `csd:"device"`
	Enforce          string                      `csd:"enforce"`
	PersonalFireWall map[string]PersonalFireWall `csd:"pfw"`
	AntiMalware      map[string]AntiMalware      `csd:"am"`
	Files            map[string]File             `csd:"file"`
	Processes        map[string]Process          `csd:"process"`
}

type OS struct {
	Version      string `csd:"version"`
	ServicePack  string `csd:"servicepack"`
	Architecture string `csd:"architecture"`
}

type Policy struct {
	Location string `csd:"location"`
}

type Device struct {
	Protection          string `csd:"protection"`
	ProtectionVersion   string `csd:"protection_version"`
	ProtectionExtension string `csd:"protection_extension"`
}

type PersonalFireWall struct {
	Exists      bool   `csd:"exists"`
	Description string `csd:"description"`
	Version     string `csd:"version"`
	Enabled     string `csd:"enabled"`
}

type AntiMalware struct {
	Exists      bool   `csd:"exists"`
	Description string `csd:"description"`
	Version     string `csd:"version"`
	Activescan  string `csd:"activescan"`
}

type File struct {
	Name         string `csd:"name"`
	Path         string `csd:"path"`
	Exists       bool   `csd:"exists"`
	LastModified int    `csd:"lastmodified"`
	Timestamp    int    `csd:"timestamp"`
}

type Process struct {
	Name   string `csd:"name"`
	Exists bool   `csd:"exists"`
}

func (r *Report) Encode(w io.Writer) error {
	return marshaler.NewEncoder(w).Encode(*r, "endpoint")
}

func (r *Report) Reader(w io.Writer) io.Reader {
	pr, pw := io.Pipe()
	go func() {
		// close the writer, so the reader knows there's no more data
		defer pw.Close()

		// write csd data to the PipeReader through the PipeWriter
		if err := r.Encode(pw); err != nil {
			// close the writer with an error forwarding the error to the reader
			pw.CloseWithError(err)
			return
		}
	}()
	return pr
}

func (r *Report) String() string {
	s, _ := marshaler.Marshal(*r, "endpoint")
	return s
}
