package shmring

/*
#cgo pkg-config: shmring
#include <shm_bridge.h>
*/
import "C"

import (
	"io"
	"os"
	"runtime"
	"syscall"
)

// Notification bits returned by Bridge.Poll.
const (
	ReadPending  = uint32(C.SHM_RING_READ_PENDING)
	WritePending = uint32(C.SHM_RING_WRITE_PENDING)
)

type cbridge = C.struct_shm_bridge

// Bridge owns a native endpoint and its Go notification file. Packet operations
// access the rings directly; notification I/O uses the runtime poller. GC
// releases the native storage once all Go users are gone.
type Bridge struct {
	*cbridge
	file *os.File
	rx   *Ring
	tx   *Ring
}

// NewBridgePair allocates two native endpoints sized near targetSlots
// maximum-size slots.
// Release each with shm_bridge_release after native accesses stop, or transfer it
// to a Go Bridge with NewBridge.
func NewBridgePair(targetSlots uint32, slotCapacity uint16) ([2]*C.struct_shm_bridge, error) {
	var endpoints [2]*C.struct_shm_bridge
	// Protect the close-on-exec fallback against concurrent Go process launches.
	syscall.ForkLock.RLock()
	ret := C.shm_bridge_new(C.uint32_t(targetSlots), C.uint16_t(slotCapacity), &endpoints[0])
	syscall.ForkLock.RUnlock()
	if ret < 0 {
		return endpoints, syscall.Errno(-ret)
	}

	return endpoints, nil
}

// NewBridge takes ownership of a native endpoint and adopts its descriptor into
// an os.File. Call once, before concurrent access, and do not use the native
// endpoint separately afterward.
func NewBridge(ptr *C.struct_shm_bridge) *Bridge {
	b := Bridge{
		cbridge: ptr,
		file:    os.NewFile(uintptr(ptr.fd), "packet-bridge"),
		rx:      wrapRing(ptr.rx),
		tx:      wrapRing(ptr.tx),
	}
	b.fd = -1
	// The argument contains only a C pointer, so it cannot retain b.
	runtime.AddCleanup(&b, func(ptr *C.struct_shm_bridge) {
		C.shm_bridge_release(ptr)
	}, ptr)
	return &b
}

// Close stops packet I/O and notifications. Native storage remains alive until
// all Go users are gone, including packet copies still in progress.
func (b *Bridge) Close() error {
	defer runtime.KeepAlive(b)
	C.shm_bridge_close(b.cbridge)
	return b.file.Close()
}

// Poll waits on the notification file and returns the acknowledged ReadPending
// and WritePending bits. A stale wakeup can return zero; retry the relevant rings
// before waiting again.
func (b *Bridge) Poll() (uint32, error) {
	defer runtime.KeepAlive(b)
	var notification [128]byte
	if _, err := b.file.Read(notification[:]); err != nil {
		return 0, err
	}
	// Clear our notification bits before retrying I/O.
	read := b.rx.state.And(^ReadPending)
	write := b.tx.state.And(^WritePending)
	return read&ReadPending | write&WritePending, nil
}

var (
	_ io.ReaderFrom = (*Bridge)(nil)
	_ io.WriterTo   = (*Bridge)(nil)
)

// ReadFrom reads packets directly into ring slots until EOF or an error,
// including ErrFull. Each Read must supply one whole packet and must not retain
// the buffer. If the supplied buffer is too small, it must return ErrShortBuffer
// without consuming the packet. Failed reads leave the slot unpublished. EOF is success.
// Calls to ReadFrom must not overlap.
func (b *Bridge) ReadFrom(reader io.Reader) (total int64, err error) {
	defer runtime.KeepAlive(b)
	emptyReads := 0
	for {
		n, err := b.tx.ReadOneFrom(reader)
		total += int64(n)
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
		if n == 0 {
			emptyReads++
			if emptyReads >= 100 {
				return total, io.ErrNoProgress
			}
			continue
		}
		emptyReads = 0
		if err := b.signal(b.tx.state, ReadPending); err != nil {
			return total, err
		}
	}
}

// WriteTo writes packets directly from ring slots until EOF or an error,
// including ErrEmpty. Each Write receives one whole packet and must not retain
// or modify the bytes. Failed or short writes leave the slot queued.
// Calls to WriteTo must not overlap.
func (b *Bridge) WriteTo(writer io.Writer) (total int64, err error) {
	defer runtime.KeepAlive(b)
	for {
		n, err := b.rx.WriteOneTo(writer)
		total += int64(n)
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
		// Even an empty packet frees a slot when consumed successfully.
		if err := b.signal(b.rx.state, WritePending); err != nil {
			return total, err
		}
	}
}

func (b *Bridge) signal(state *flags, mask uint32) error {
	if state.Or(mask)&mask != 0 {
		return nil
	}
	_, err := b.file.Write([]byte{1})
	return err
}
