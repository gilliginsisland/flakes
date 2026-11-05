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

// WriteOneTo writes one record directly to w, calling Write once. The slot is
// released only if the entire record is accepted without an error. A failed or
// short write leaves the whole record queued. Writers must not retain or modify
// the supplied bytes.
func (r *Ring) WriteOneTo(w io.Writer) (int, error) {
	tail := atomic.LoadUint64((*uint64)(&r.tail))
	head := atomic.LoadUint64((*uint64)(&r.head))
	if tail == head&C.SHM_RING_COUNT_MASK {
		if head&C.SHM_RING_CLOSED_BIT != 0 {
			return 0, io.EOF
		}
		return 0, ErrEmpty
	}
	slot := r.slot(tail)
	size := *(*uint32)(slot)
	n, err := w.Write(unsafe.Slice((*byte)(unsafe.Add(slot, unsafe.Sizeof(size))), int(size)))
	if n < 0 || n > int(size) {
		return 0, io.ErrShortWrite
	}
	if err != nil {
		return n, err
	}
	if n != int(size) {
		return n, io.ErrShortWrite
	}
	atomic.StoreUint64((*uint64)(&r.tail), (tail+1)&C.SHM_RING_COUNT_MASK)
	return n, nil
}

// ReadOneFrom reads one record directly into an unpublished slot, calling read
// once. The reader must return one complete record and must not retain the
// supplied storage. Errors, including EOF, leave the slot unpublished and the
// producer position unchanged, even when bytes were written into the slot.
// A zero-byte read without an error is a no-op.
func (r *Ring) ReadOneFrom(read func([]byte) (int, error)) (int, error) {
	head := atomic.LoadUint64((*uint64)(&r.head))
	tail := atomic.LoadUint64((*uint64)(&r.tail))
	if tail&C.SHM_RING_CLOSED_BIT != 0 {
		return 0, io.ErrClosedPipe
	}
	if (head-tail)&C.SHM_RING_COUNT_MASK == uint64(r.capacity) {
		return 0, ErrFull
	}
	slot := r.slot(head)
	dst := unsafe.Slice((*byte)(unsafe.Add(slot, unsafe.Sizeof(uint32(0)))), int(r.record_size))
	n, err := read(dst)
	if n < 0 || n > len(dst) {
		return 0, io.ErrShortBuffer
	}
	if err != nil {
		return n, err
	}
	if n == 0 {
		return 0, nil
	}
	*(*uint32)(slot) = uint32(n)
	atomic.StoreUint64((*uint64)(&r.head), (head+1)&C.SHM_RING_COUNT_MASK)
	return n, nil
}

// Read copies one complete record. A short destination leaves the record in
// place. ErrEmpty means no record is available. One consumer only.
func (r *Ring) Read(dst []byte) (int, error) {
	return r.WriteOneTo(byteSlice(dst))
}

// Write copies one complete record. Oversized or empty writes leave the ring
// unchanged. ErrFull means no slot is available. One producer only.
func (r *Ring) Write(src []byte) (int, error) {
	if len(src) == 0 {
		return 0, nil
	}
	return r.ReadOneFrom(byteSlice(src).Read)
}

// byteSlice adapts a caller's buffer for a single record transfer. It neither
// allocates storage nor advances a cursor; short buffers leave it unchanged.
type byteSlice []byte

func (b byteSlice) Read(dst []byte) (int, error) {
	if len(dst) < len(b) {
		return 0, io.ErrShortBuffer
	}
	return copy(dst, b), nil
}

func (b byteSlice) Write(src []byte) (int, error) {
	if len(b) < len(src) {
		return 0, io.ErrShortBuffer
	}
	return copy(b, src), nil
}
