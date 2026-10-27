#ifndef SHM_RING_H
#define SHM_RING_H

#include <stddef.h>
#include <stdint.h>
#include <sys/types.h>

#define SHM_RING_CLOSED_BIT (UINT64_C(1) << 63)
#define SHM_RING_COUNT_MASK (SHM_RING_CLOSED_BIT - 1)
#define SHM_RING_READ_PENDING UINT64_C(1)
#define SHM_RING_WRITE_PENDING UINT64_C(2)

/* A bounded, single-producer/single-consumer ring of variable-length records.
 * The reader and writer each own one half. Close a half after its operations
 * stop; closing both halves frees the ring and storage. Capacity must be a
 * nonzero power of two; record_size must fit the aligned uint32_t stride.
 * head/tail use aligned, lock-free, sequentially consistent 64-bit atomics in
 * both C and Go. Each counter's high bit marks its owner's half closed; the
 * lower 63 bits wrap independently. Record bytes are published by head and
 * released by tail. */
struct shm_ring {
	uint64_t head; /* Producer counter and writer-closed flag. */
	uint64_t tail; /* Consumer counter and reader-closed flag. */
	uint64_t pending; /* Reader/data and writer/space notification bits. */
	uint32_t closed; /* Last-half release handshake; not used by packet I/O. */
	uint32_t capacity;
	uint32_t record_size;
	uint32_t stride;
	unsigned char *slots;
};

enum {
	SHM_RING_READ_CLOSED = 1,
	SHM_RING_WRITE_CLOSED = 2,
};

/* Capacity must be a nonzero power of two. Its uint32_t range keeps it below
 * the reserved counter bit. record_size must be nonzero and leave room for
 * the length field and alignment within a uint32_t stride.
 * Returns 0 and sets result on success, or a negative error code on failure. */
int shm_ring_new(uint32_t capacity, uint32_t record_size, struct shm_ring **result);
void shm_ring_close_read(struct shm_ring *ring);
void shm_ring_close_write(struct shm_ring *ring);

/* Consumer: copy and remove one record without blocking. Returns its byte count,
 * -EAGAIN when empty, -EPIPE when drained after write closure, or -EMSGSIZE when
 * the record exceeds the destination or ring capacity. Failure leaves the
 * record in the ring. */
ssize_t shm_ring_pop(struct shm_ring *ring, void *data, size_t size);

/* Producer: copy and publish one record without blocking. Returns its byte count,
 * -EAGAIN when full, -EPIPE when the reader has closed, or -EMSGSIZE when size
 * exceeds record_size. Failure leaves the ring unchanged. */
ssize_t shm_ring_commit(struct shm_ring *ring, const void *data, size_t size);

#endif
