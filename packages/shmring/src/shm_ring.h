#ifndef SHM_RING_H
#define SHM_RING_H

#include <stddef.h>
#include <stdint.h>
#include <sys/types.h>

struct shm_ring_slot {
	_Alignas(uint32_t) uint16_t length;
	unsigned char data[];
};

/* A bounded, single-producer/single-consumer ring of variable-length slots.
 * The reader and writer each own one half. Release both halves after their
 * operations stop to free the ring and storage. Capacity is a
 * power-of-two byte count; slot_capacity is the largest packet.
 * head, tail, and state use aligned, lock-free, sequentially consistent 32-bit
 * atomics in both C and Go. Positions wrap at capacity. One byte remains
 * unused, so equal head and tail positions mean empty.
 * A slot occupies its header and payload, rounded up to slot alignment.
 * After a slot, each side advances past an end fragment too small for a
 * maximum-size slot. */
struct shm_ring {
	uint32_t tail; /* Next write position. */
	uint32_t head; /* Next read position. */
	_Alignas(uint64_t) uint32_t state; /* Flags; trailing padding keeps slots visible to cgo. */
	uint32_t capacity; /* Byte capacity. */
	uint16_t slot_capacity; /* Maximum packet size. */
	uint16_t slot_size; /* Aligned maximum slot reservation. */
	_Alignas(struct shm_ring_slot) unsigned char slots[];
};

enum {
	SHM_RING_READ_PENDING = 1 << 0,
	SHM_RING_WRITE_PENDING = 1 << 1,
	SHM_RING_READ_CLOSED = 1 << 2,
	SHM_RING_WRITE_CLOSED = 1 << 3,
	SHM_RING_READ_RELEASED = 1 << 4,
	SHM_RING_WRITE_RELEASED = 1 << 5,
	SHM_RING_SLOT_HEADER = offsetof(struct shm_ring_slot, data),
	SHM_RING_SLOT_ALIGN = _Alignof(struct shm_ring_slot),
};

/* target_slots is a power-of-two slot count used to size the allocation.
 * The byte capacity is the largest power of two no greater than target_slots
 * times the aligned maximum slot reservation.
 * It must fit at least twice that reservation; the aligned reservation must
 * fit in uint16_t and slot_capacity must be nonzero.
 * Returns 0 and sets result on success, or a negative error code on failure. */
int shm_ring_new(uint32_t target_slots, uint16_t slot_capacity, struct shm_ring **result);
/* Close a half to stop packet I/O without releasing its ownership. */
void shm_ring_close_read(struct shm_ring *ring);
void shm_ring_close_write(struct shm_ring *ring);
/* Release a half after its operations stop. The second release frees the ring. */
void shm_ring_release_read(struct shm_ring *ring);
void shm_ring_release_write(struct shm_ring *ring);

/* Consumer: copy and remove one packet without blocking. Returns its byte count,
 * -EAGAIN when empty, -EPIPE after read closure or when drained after write
 * closure, or -EMSGSIZE when the destination is too small. Failure leaves the
 * slot in the ring. */
ssize_t shm_ring_read(struct shm_ring *ring, void *data, size_t size);

/* Producer: copy and publish one packet without blocking. Requires room for a
 * maximum-size slot, including any end fragment skipped on wrap. Returns its
 * byte count, -EAGAIN when full, -EPIPE when either half has closed, or
 * -EMSGSIZE when size exceeds slot_capacity. Failure leaves the ring unchanged. */
ssize_t shm_ring_write(struct shm_ring *ring, const void *data, size_t size);

#endif
