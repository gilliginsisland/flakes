package openconnect

/*
#cgo pkg-config: openconnect
#include <openconnect.h>
*/
import "C"

import (
	"context"
	"io"
)

type CMDPipe struct {
	w io.Writer
}

func (cp *CMDPipe) write(r rune) error {
	_, err := cp.w.Write([]byte{byte(r)})
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

// PropagateContext links ctx cancellation to CmdPipe.Cancel.
// It returns a cancel unregister function.
func (cp *CMDPipe) PropagateContext(ctx context.Context) func() bool {
	return context.AfterFunc(ctx, func() {
		cp.Cancel()
	})
}
