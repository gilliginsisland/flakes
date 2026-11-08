// Package shmring accesses bounded packet rings shared with native code.
// Each ring supports one producer and one consumer. Its storage is freed when
// both sides release it, after their operations have stopped.
package shmring

/*
#cgo pkg-config: shmring
#include <shm_ring.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"syscall"
	"unsafe"
)

var (
	ErrEmpty = errors.New("shared ring is empty")
	ErrFull  = errors.New("shared ring is full")
)

type cring = C.struct_shm_ring

type counter uint32

func (c *counter) Load() uint32 {
	return atomic.LoadUint32((*uint32)(c))
}

func (c *counter) Store(position uint32) {
	atomic.StoreUint32((*uint32)(c), position)
}

type flags uint32

func (f *flags) Load() uint32 {
	return atomic.LoadUint32((*uint32)(f))
}

func (f *flags) And(mask uint32) uint32 {
	return atomic.AndUint32((*uint32)(f), mask)
}

func (f *flags) Or(mask uint32) uint32 {
	return atomic.OrUint32((*uint32)(f), mask)
}

type slot C.struct_shm_ring_slot

func (s *slot) bytes() []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(&s.data)), int(s.length))
}

const (
	slotHeader = uint32(unsafe.Offsetof(slot{}.data))
	// cgo does not preserve the C length field's explicit alignment.
	slotAlign = uint32(C.SHM_RING_SLOT_ALIGN)
)

// Ring holds a native ring and a slice over its slot storage. Its allocation
// must remain alive while using it; a Bridge owns its Go-accessed rings.
type Ring struct {
	*cring
	slots []byte
	head  *counter
	tail  *counter
	state *flags
}

// NewRing allocates a native ring sized near targetSlots maximum-size slots.
// The target must be a power of two; one byte of the allocation remains unused.
// Release each half once, after that half's operations stop, to free storage.
func NewRing(targetSlots uint32, slotCapacity uint16) (*Ring, error) {
	var ring *C.struct_shm_ring
	if ret := C.shm_ring_new(C.uint32_t(targetSlots), C.uint16_t(slotCapacity), &ring); ret < 0 {
		return nil, syscall.Errno(-ret)
	}
	return wrapRing(ring), nil
}

func wrapRing(ptr *cring) *Ring {
	return &Ring{
		cring: ptr,
		slots: unsafe.Slice((*byte)(unsafe.Pointer(&ptr.slots)), int(ptr.capacity)),
		head:  (*counter)(&ptr.head),
		tail:  (*counter)(&ptr.tail),
		state: (*flags)(&ptr.state),
	}
}

// CloseRead stops reads without releasing the reader's half.
func (r *Ring) CloseRead() {
	C.shm_ring_close_read(r.cring)
}

// CloseWrite stops writes without releasing the writer's half.
func (r *Ring) CloseWrite() {
	C.shm_ring_close_write(r.cring)
}

// ReleaseRead closes and releases the reader's half after its operations stop.
// The second half's release frees the ring.
func (r *Ring) ReleaseRead() {
	C.shm_ring_release_read(r.cring)
}

// ReleaseWrite closes and releases the writer's half after its operations stop.
// The second half's release frees the ring.
func (r *Ring) ReleaseWrite() {
	C.shm_ring_release_write(r.cring)
}

func (r *Ring) slot(position uint32) *slot {
	return (*slot)(unsafe.Pointer(&r.slots[position]))
}

// advance returns the bytes to consume, including padding when the next
// maximum-size slot would not fit before the physical end of the ring.
func (r *Ring) advance(position, bytes uint32) uint32 {
	mask := slotAlign - 1
	bytes = (bytes + mask) &^ mask
	next := (position + bytes) & uint32(r.capacity-1)
	remaining := uint32(r.capacity) - next
	if remaining < uint32(r.slot_size) {
		bytes += remaining
	}
	return bytes
}

// WriteOneTo writes one packet directly to w, calling Write once. The slot is
// released only if the entire packet is accepted without an error. A failed or
// short write leaves it queued. Writers must not retain or modify the supplied
// bytes. A successful Write consumes the slot even if CloseRead races with it.
// A closed read half returns ErrClosedPipe; a drained ring whose writer closed
// returns EOF.
func (r *Ring) WriteOneTo(w io.Writer) (int, error) {
	head := r.head.Load()
	state := r.state.Load()
	tail := r.tail.Load()
	if state&C.SHM_RING_READ_CLOSED != 0 {
		return 0, io.ErrClosedPipe
	}
	if head == tail {
		if state&C.SHM_RING_WRITE_CLOSED != 0 {
			return 0, io.EOF
		}
		return 0, ErrEmpty
	}
	data := r.slot(head).bytes()
	size := len(data)
	n, err := w.Write(data)
	if err != nil {
		return n, err
	}
	if n < size {
		return n, io.ErrShortWrite
	}
	r.head.Store((head + r.advance(head, slotHeader+uint32(size))) & uint32(r.capacity-1))
	return n, nil
}

// ReadOneFrom reads one packet directly into an unpublished slot, calling Read
// once. The reader must return one complete packet and must not retain the
// supplied storage. If the buffer cannot hold its next packet, it must return
// ErrShortBuffer without consuming it. Errors, including EOF, leave the slot
// unpublished and the producer position unchanged, even if bytes were written.
// A zero-byte read without an error is a no-op. A successful read publishes its
// slot even if a ring half closes during the read.
func (r *Ring) ReadOneFrom(reader io.Reader) (int, error) {
	state := r.state.Load()
	tail := r.tail.Load()
	head := r.head.Load()
	if state&(C.SHM_RING_READ_CLOSED|C.SHM_RING_WRITE_CLOSED) != 0 {
		return 0, io.ErrClosedPipe
	}
	mask := uint32(r.capacity) - 1
	free := mask - ((tail - head) & mask)
	if r.advance(tail, uint32(r.slot_size)) > free {
		return 0, ErrFull
	}
	slot := r.slot(tail)
	// The slot is unpublished; its length can hold the maximum packet size
	// until a successful read replaces it with the actual length.
	slot.length = r.slot_capacity
	data := slot.bytes()
	n, err := reader.Read(data)
	if n < 0 || n > len(data) {
		return 0, fmt.Errorf("shmring: reader returned %d bytes for a %d-byte slot", n, len(data))
	}
	if err != nil {
		return n, err
	}
	if n == 0 {
		return 0, nil
	}
	slot.length = C.uint16_t(n)
	r.tail.Store((tail + r.advance(tail, slotHeader+uint32(n))) & mask)
	return n, nil
}
