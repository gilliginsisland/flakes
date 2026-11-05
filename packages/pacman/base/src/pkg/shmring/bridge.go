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

var (
	_ io.ReadWriteCloser = (*Bridge)(nil)
	_ io.ReaderFrom      = (*Bridge)(nil)
	_ io.WriterTo        = (*Bridge)(nil)
)

// Read copies one complete record. A short destination leaves it in the ring.
// ErrEmpty means the caller should Poll and retry. One consumer only.
func (b *Bridge) Read(dst []byte) (int, error) {
	defer runtime.KeepAlive(b)
	n, err := (*Ring)(b.rx).Read(dst)
	if err != nil {
		return n, err
	}
	return n, b.signal(&b.rx.pending, C.SHM_RING_WRITE_PENDING)
}

// Write copies one complete record. ErrFull does not wait for space. Empty
// writes are a no-op. One producer only.
func (b *Bridge) Write(src []byte) (int, error) {
	defer runtime.KeepAlive(b)
	n, err := (*Ring)(b.tx).Write(src)
	if err != nil || n == 0 {
		return n, err
	}
	return n, b.signal(&b.tx.pending, C.SHM_RING_READ_PENDING)
}

// ReadFrom reads records directly into ring slots until EOF or an error,
// including ErrFull. Each Read must supply one whole record and must not retain
// the buffer. Failed reads leave the slot unpublished. EOF is success.
// Calls to Write and ReadFrom must not overlap.
func (b *Bridge) ReadFrom(reader io.Reader) (total int64, err error) {
	defer runtime.KeepAlive(b)
	emptyReads := 0
	for {
		n, err := (*Ring)(b.tx).ReadOneFrom(reader.Read)
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
		if err := b.signal(&b.tx.pending, C.SHM_RING_READ_PENDING); err != nil {
			return total, err
		}
	}
}

// WriteTo writes records directly from ring slots until EOF or an error,
// including ErrEmpty. Each Write receives one whole record and must not retain
// or modify the bytes. Failed or short writes leave the record queued.
// Calls to Read and WriteTo must not overlap.
func (b *Bridge) WriteTo(writer io.Writer) (total int64, err error) {
	defer runtime.KeepAlive(b)
	for {
		n, err := (*Ring)(b.rx).WriteOneTo(writer)
		total += int64(n)
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
		// Even an empty record frees a slot when consumed successfully.
		if err := b.signal(&b.rx.pending, C.SHM_RING_WRITE_PENDING); err != nil {
			return total, err
		}
	}
}

func (b *Bridge) signal(pending *C.uint64_t, mask uint64) error {
	if atomic.OrUint64((*uint64)(pending), mask)&mask != 0 {
		return nil
	}
	_, err := b.file.Write([]byte{1})
	return err
}
