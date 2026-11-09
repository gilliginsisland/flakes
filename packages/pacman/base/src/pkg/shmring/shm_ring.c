#include "shm_ring.h"

#include <errno.h>
#include <stdlib.h>
#include <string.h>

_Static_assert(__atomic_always_lock_free(4, 0), "shared rings require 32-bit atomics");

static uint64_t aligned_slot_bytes(uint64_t bytes)
{
	return (bytes + SHM_RING_SLOT_ALIGN - 1) &
	       ~(uint64_t)(SHM_RING_SLOT_ALIGN - 1);
}

int shm_ring_new(uint32_t target_slots, uint16_t slot_capacity, struct shm_ring **result)
{
	struct shm_ring *ring;
	uint64_t slot_bytes =
		aligned_slot_bytes(SHM_RING_SLOT_HEADER + (uint64_t)slot_capacity);
	uint32_t requested, capacity;

	if (!target_slots || (target_slots & (target_slots - 1)) ||
	    !slot_capacity || slot_bytes > UINT16_MAX ||
	    target_slots > UINT32_MAX / slot_bytes)
		return -EINVAL;
	requested = (uint32_t)(target_slots * slot_bytes);
	/* A power-of-two byte capacity lets physical offsets wrap with a mask. */
	capacity = 1;
	while (capacity <= requested / 2)
		capacity <<= 1;
	if ((uint64_t)capacity < 2 * slot_bytes ||
	    capacity > PTRDIFF_MAX - sizeof(*ring))
		return -EINVAL;
	ring = calloc(1, sizeof(*ring) + capacity);
	if (!ring)
		return -errno;
	*ring = (struct shm_ring){
		.capacity = capacity,
		.slot_capacity = slot_capacity,
		.slot_size = (uint16_t)slot_bytes,
	};
	*result = ring;
	return 0;
}

void shm_ring_close_read(struct shm_ring *ring)
{
	__atomic_fetch_or(&ring->state, SHM_RING_READ_CLOSED, __ATOMIC_SEQ_CST);
}

void shm_ring_close_write(struct shm_ring *ring)
{
	__atomic_fetch_or(&ring->state, SHM_RING_WRITE_CLOSED, __ATOMIC_SEQ_CST);
}

void shm_ring_release_read(struct shm_ring *ring)
{
	shm_ring_close_read(ring);
	if (__atomic_fetch_or(&ring->state, SHM_RING_READ_RELEASED,
			      __ATOMIC_SEQ_CST) & SHM_RING_WRITE_RELEASED)
		free(ring);
}

void shm_ring_release_write(struct shm_ring *ring)
{
	shm_ring_close_write(ring);
	if (__atomic_fetch_or(&ring->state, SHM_RING_WRITE_RELEASED,
			      __ATOMIC_SEQ_CST) & SHM_RING_READ_RELEASED)
		free(ring);
}

static uint32_t advance_bytes(struct shm_ring *ring, uint32_t position,
			      uint32_t bytes)
{
	uint32_t remaining;

	bytes = aligned_slot_bytes(bytes);
	remaining = ring->capacity - ((position + bytes) & (ring->capacity - 1));

	if (remaining < ring->slot_size)
		bytes += remaining;
	return bytes;
}

ssize_t shm_ring_read(struct shm_ring *ring, void *data, size_t size)
{
	uint32_t head = __atomic_load_n(&ring->head, __ATOMIC_SEQ_CST);
	uint32_t state = __atomic_load_n(&ring->state, __ATOMIC_SEQ_CST);
	uint32_t tail = __atomic_load_n(&ring->tail, __ATOMIC_SEQ_CST);
	uint32_t mask = ring->capacity - 1;
	struct shm_ring_slot *slot;
	uint16_t length;

	if (state & SHM_RING_READ_CLOSED)
		return -EPIPE;
	if (head == tail) {
		/* Load state before tail to see the final position after closure. */
		if (state & SHM_RING_WRITE_CLOSED)
			return -EPIPE;
		return -EAGAIN;
	}
	slot = (struct shm_ring_slot *)(ring->slots + head);
	length = slot->length;
	if (length > size)
		return -EMSGSIZE;
	memcpy(data, slot->data, length);
	__atomic_store_n(&ring->head,
			 (head + advance_bytes(ring, head, SHM_RING_SLOT_HEADER + length)) & mask,
			 __ATOMIC_SEQ_CST);
	return length;
}

ssize_t shm_ring_write(struct shm_ring *ring, const void *data, size_t size)
{
	uint32_t state = __atomic_load_n(&ring->state, __ATOMIC_SEQ_CST);
	uint32_t tail = __atomic_load_n(&ring->tail, __ATOMIC_SEQ_CST);
	uint32_t head = __atomic_load_n(&ring->head, __ATOMIC_SEQ_CST);
	uint32_t mask, free_bytes;
	struct shm_ring_slot *slot;

	if (!size)
		return 0;
	if (state & (SHM_RING_READ_CLOSED | SHM_RING_WRITE_CLOSED))
		return -EPIPE;
	if (size > ring->slot_capacity)
		return -EMSGSIZE;
	mask = ring->capacity - 1;
	free_bytes = mask - ((tail - head) & mask);
	if (advance_bytes(ring, tail, ring->slot_size) > free_bytes)
		return -EAGAIN;
	slot = (struct shm_ring_slot *)(ring->slots + tail);
	memcpy(slot->data, data, size);
	slot->length = size;
	__atomic_store_n(&ring->tail,
			 (tail + advance_bytes(ring, tail, SHM_RING_SLOT_HEADER + size)) & mask,
			 __ATOMIC_SEQ_CST);
	return size;
}
