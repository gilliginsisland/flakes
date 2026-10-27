#include "shm_ring.h"

#include <errno.h>
#include <stdlib.h>
#include <string.h>

_Static_assert(__atomic_always_lock_free(8, 0), "shared rings require 64-bit atomics");

int shm_ring_new(uint32_t capacity, uint32_t record_size, struct shm_ring **result)
{
	struct shm_ring *ring;
	size_t stride;

	/* A power-of-two capacity lets slot indexes wrap with a bitmask. */
	if (!capacity || (capacity & (capacity - 1)) || !record_size ||
	    record_size > UINT32_MAX - 11)
		return -EINVAL;
	stride = (sizeof(uint32_t) + record_size + 7) & ~(size_t)7;
	if (capacity > (SIZE_MAX - sizeof(*ring)) / stride)
		return -EINVAL;
	ring = calloc(1, sizeof(*ring) + capacity * stride);
	if (!ring)
		return -errno;
	*ring = (struct shm_ring){
		.capacity = capacity,
		.record_size = record_size,
		.stride = stride,
		.slots = (unsigned char *)(ring + 1),
	};
	*result = ring;
	return 0;
}

void shm_ring_close_read(struct shm_ring *ring)
{
	__atomic_fetch_or(&ring->tail, SHM_RING_CLOSED_BIT, __ATOMIC_SEQ_CST);
	if (__atomic_fetch_or(&ring->closed, SHM_RING_READ_CLOSED,
			      __ATOMIC_SEQ_CST) == SHM_RING_WRITE_CLOSED)
		free(ring);
}

void shm_ring_close_write(struct shm_ring *ring)
{
	__atomic_fetch_or(&ring->head, SHM_RING_CLOSED_BIT, __ATOMIC_SEQ_CST);
	if (__atomic_fetch_or(&ring->closed, SHM_RING_WRITE_CLOSED,
			      __ATOMIC_SEQ_CST) == SHM_RING_READ_CLOSED)
		free(ring);
}

static unsigned char *record_slot(struct shm_ring *ring, uint64_t index)
{
	return ring->slots + (index & (ring->capacity - 1)) * ring->stride;
}

ssize_t shm_ring_pop(struct shm_ring *ring, void *data, size_t size)
{
	uint64_t tail = __atomic_load_n(&ring->tail, __ATOMIC_SEQ_CST);
	uint64_t head = __atomic_load_n(&ring->head, __ATOMIC_SEQ_CST);
	uint32_t length;
	unsigned char *slot;

	if (tail == (head & SHM_RING_COUNT_MASK)) {
		/* The closed flag and final producer count are one atomic snapshot. */
		if (head & SHM_RING_CLOSED_BIT)
			return -EPIPE;
		return -EAGAIN;
	}
	slot = record_slot(ring, tail);
	memcpy(&length, slot, sizeof(length));
	if (!length || length > ring->record_size || length > size)
		return -EMSGSIZE;
	memcpy(data, slot + sizeof(length), length);
	__atomic_store_n(&ring->tail, (tail + 1) & SHM_RING_COUNT_MASK, __ATOMIC_SEQ_CST);
	return length;
}

ssize_t shm_ring_commit(struct shm_ring *ring, const void *data, size_t size)
{
	uint64_t head = __atomic_load_n(&ring->head, __ATOMIC_SEQ_CST);
	uint64_t tail = __atomic_load_n(&ring->tail, __ATOMIC_SEQ_CST);
	uint32_t length;
	unsigned char *slot;

	if (tail & SHM_RING_CLOSED_BIT)
		return -EPIPE;
	if (size > ring->record_size)
		return -EMSGSIZE;
	if (((head - tail) & SHM_RING_COUNT_MASK) == ring->capacity)
		return -EAGAIN;
	slot = record_slot(ring, head);
	length = size;
	memcpy(slot + sizeof(length), data, size);
	memcpy(slot, &length, sizeof(length));
	__atomic_store_n(&ring->head, (head + 1) & SHM_RING_COUNT_MASK, __ATOMIC_SEQ_CST);
	return size;
}
