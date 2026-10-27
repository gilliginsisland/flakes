package shmring

/*
#cgo pkg-config: shmring
#include <shm_bridge.h>
*/
import "C"

import (
	"os"
	"runtime"
	"sync/atomic"
	"syscall"
)

// Notification bits returned by Bridge.Poll.
const (
	ReadPending  = uint64(C.SHM_RING_READ_PENDING)
	WritePending = uint64(C.SHM_RING_WRITE_PENDING)
)

type bridge = C.struct_shm_bridge

// Bridge owns a native endpoint and its Go notification file. Packet operations
// access the rings directly; notification I/O uses the runtime poller. GC
// releases the native storage once all Go users are gone.
type Bridge struct {
	*bridge
	file *os.File
}

// NewBridgePair allocates two native endpoints. Release each with shm_bridge_close
// after native accesses stop, or transfer it to a Go Bridge with NewBridge.
func NewBridgePair(capacity, recordSize uint32) ([2]*C.struct_shm_bridge, error) {
	var endpoints [2]*C.struct_shm_bridge
	// Protect the close-on-exec fallback against concurrent Go process launches.
	syscall.ForkLock.RLock()
	ret := C.shm_bridge_new(C.uint32_t(capacity), C.uint32_t(recordSize), &endpoints[0])
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
		bridge: ptr,
		file:   os.NewFile(uintptr(ptr.fd), "packet-bridge"),
	}
	b.fd = -1
	// The argument contains only a C pointer, so it cannot retain b.
	runtime.AddCleanup(&b, func(ptr *C.struct_shm_bridge) {
		C.shm_bridge_close(ptr)
	}, ptr)
	return &b
}

// Close shuts down notifications. Native storage remains alive until all Go
// users are gone, including packet copies still in progress.
func (b *Bridge) Close() error {
	defer runtime.KeepAlive(b)
	return b.file.Close()
}

// Poll waits on the notification file and returns the acknowledged ReadPending
// and WritePending bits. A stale wakeup can return zero; retry the relevant rings
// before waiting again.
func (b *Bridge) Poll() (uint64, error) {
	defer runtime.KeepAlive(b)
	var notification [128]byte
	if _, err := b.file.Read(notification[:]); err != nil {
		return 0, err
	}
	// Clear only our bits before retrying I/O; the peer owns the other bit.
	read := atomic.AndUint64((*uint64)(&b.rx.pending), ^ReadPending)
	write := atomic.AndUint64((*uint64)(&b.tx.pending), ^WritePending)
	return read&ReadPending | write&WritePending, nil
}

// Pop attempts to copy one packet. ErrEmpty means the caller should wait for a
// notification and retry. Successful pops notify the peer through the file.
func (b *Bridge) Pop(p []byte) (int, error) {
	defer runtime.KeepAlive(b)
	n, err := (*Ring)(b.rx).Pop(p)
	if err != nil {
		return n, err
	}
	return n, b.signal(&b.rx.pending, C.SHM_RING_WRITE_PENDING)
}

// Commit attempts to copy one packet. ErrFull means the caller should wait for a
// notification and retry. Successful commits notify the peer through the file.
func (b *Bridge) Commit(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	defer runtime.KeepAlive(b)
	n, err := (*Ring)(b.tx).Commit(p)
	if err != nil {
		return n, err
	}
	return n, b.signal(&b.tx.pending, C.SHM_RING_READ_PENDING)
}

func (b *Bridge) signal(pending *C.uint64_t, mask uint64) error {
	if atomic.OrUint64((*uint64)(pending), mask)&mask != 0 {
		return nil
	}
	_, err := b.file.Write([]byte{1})
	return err
}
