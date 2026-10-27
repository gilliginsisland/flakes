// Package shmring accesses bounded record rings shared with native code.
// Each ring supports one producer and one consumer. Its storage is freed when
// both sides close, after their operations have stopped.
package shmring

/*
#cgo pkg-config: shmring
#include <shm_ring.h>
*/
import "C"

import (
	"errors"
	"io"
	"sync/atomic"
	"syscall"
	"unsafe"
)

var (
	ErrEmpty = errors.New("shared ring is empty")
	ErrFull  = errors.New("shared ring is full")
)

// Ring has the native shm_ring layout. Its allocation must remain alive while
// using it; a Bridge owns the lifetime of its Go-accessed rings.
type Ring C.struct_shm_ring

// NewRing allocates a native ring. Capacity must be a nonzero power of two.
// Close each half once, after that half's operations stop, to release storage.
func NewRing(capacity, recordSize uint32) (*Ring, error) {
	var ring *C.struct_shm_ring
	if ret := C.shm_ring_new(C.uint32_t(capacity), C.uint32_t(recordSize), &ring); ret < 0 {
		return nil, syscall.Errno(-ret)
	}
	return (*Ring)(ring), nil
}

// CloseRead releases the reader's half after its operations stop.
// The ring is freed when the writer's half has also closed.
func (r *Ring) CloseRead() {
	C.shm_ring_close_read((*C.struct_shm_ring)(r))
}

// CloseWrite releases the writer's half after its operations stop.
// The ring is freed when the reader's half has also closed.
func (r *Ring) CloseWrite() {
	C.shm_ring_close_write((*C.struct_shm_ring)(r))
}

func (r *Ring) slot(index uint64) unsafe.Pointer {
	return unsafe.Add(
		unsafe.Pointer(r.slots),
		uintptr(index&uint64(r.capacity-1))*uintptr(r.stride),
	)
}

// Pop copies and removes one record, or returns ErrEmpty without blocking.
// After the writer closes, it drains published records before returning io.EOF.
// A short destination returns io.ErrShortBuffer and leaves the record in place.
func (r *Ring) Pop(b []byte) (int, error) {
	tail := atomic.LoadUint64((*uint64)(&r.tail))
	head := atomic.LoadUint64((*uint64)(&r.head))
	if tail == head&C.SHM_RING_COUNT_MASK {
		// The closed flag and final producer count are one atomic snapshot.
		if head&C.SHM_RING_CLOSED_BIT != 0 {
			return 0, io.EOF
		}
		return 0, ErrEmpty
	}
	slot := r.slot(tail)
	size := *(*uint32)(slot)
	if size == 0 || size > uint32(r.record_size) || uint64(size) > uint64(len(b)) {
		return 0, io.ErrShortBuffer
	}
	n := copy(b, unsafe.Slice((*byte)(unsafe.Add(slot, unsafe.Sizeof(size))), int(size)))
	atomic.StoreUint64((*uint64)(&r.tail), (tail+1)&C.SHM_RING_COUNT_MASK)
	return n, nil
}

// Commit copies and publishes one record, or returns ErrFull without blocking.
// A closed reader returns io.ErrClosedPipe. Oversized records return
// io.ErrShortBuffer. Failure leaves the ring unchanged.
func (r *Ring) Commit(b []byte) (int, error) {
	head := atomic.LoadUint64((*uint64)(&r.head))
	tail := atomic.LoadUint64((*uint64)(&r.tail))
	if tail&C.SHM_RING_CLOSED_BIT != 0 {
		return 0, io.ErrClosedPipe
	}
	if uint64(len(b)) > uint64(r.record_size) {
		return 0, io.ErrShortBuffer
	}
	if (head-tail)&C.SHM_RING_COUNT_MASK == uint64(r.capacity) {
		return 0, ErrFull
	}
	slot := r.slot(head)
	size := uint32(len(b))
	copy(unsafe.Slice((*byte)(unsafe.Add(slot, unsafe.Sizeof(size))), len(b)), b)
	*(*uint32)(slot) = size
	atomic.StoreUint64((*uint64)(&r.head), (head+1)&C.SHM_RING_COUNT_MASK)
	return len(b), nil
}
