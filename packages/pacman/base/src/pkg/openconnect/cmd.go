package openconnect

/*
#cgo pkg-config: openconnect
#include <openconnect.h>
*/
import "C"

import (
	"io"
	"log/slog"
)

type CMDPipe struct {
	w  io.Writer
	fd int
}

func (cp *CMDPipe) write(r rune) error {
	_, err := cp.w.Write([]byte{byte(r)})
	slog.Debug("openconnect command pipe write",
		slog.Int("fd", cp.fd),
		slog.String("cmd", string(r)),
		slog.Any("error", err),
	)
	return err
}

func (cp *CMDPipe) Cancel() error {
	return cp.write(C.OC_CMD_CANCEL)
}

func (cp *CMDPipe) Pause() error {
	return cp.write(C.OC_CMD_PAUSE)
}

func (cp *CMDPipe) Detach() error {
	return cp.write(C.OC_CMD_DETACH)
}

func (cp *CMDPipe) Stats() error {
	return cp.write(C.OC_CMD_STATS)
}
