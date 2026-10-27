package openconnect

/*
#cgo pkg-config: openconnect
#include <openconnect.h>
*/
import "C"

import "io"

// CMDPipe owns the command pipe's write end; OpenConnect owns the read end.
type CMDPipe struct {
	io.WriteCloser
}

func (cp *CMDPipe) write(r rune) error {
	_, err := cp.Write([]byte{byte(r)})
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
