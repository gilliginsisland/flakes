#ifndef SHM_BRIDGE_H
#define SHM_BRIDGE_H

#include "shm_ring.h"
#include <sys/types.h>

/* Bidirectional record endpoint. Writes produce tx; reads consume rx.
 * Owns its nonblocking wakeup socket until an external owner adopts fd
 * and sets it to -1. Native I/O requires a descriptor still owned here. */
struct shm_bridge {
	struct shm_ring *tx;
	struct shm_ring *rx;
	int fd;
};

/* Creates two independently allocated endpoints and a nonblocking, close-on-exec
 * socket pair. Each endpoint initially owns its descriptor.
 * Returns 0 and sets result on success, or a negative error code on failure. */
int shm_bridge_new(uint32_t capacity, uint32_t record_size, struct shm_bridge *result[2]);

/* After stopping all I/O and notification processing, close this endpoint once.
 * Closes fd if still owned, closes the RX read half and TX write half through
 * the rings' last-half release handshake, and frees the endpoint immediately.
 * Returns 0 or a negative error from closing the descriptor. */
int shm_bridge_close(struct shm_bridge *bridge);

/* Acknowledge a readable wakeup before processing the rings. */
int shm_bridge_poll(struct shm_bridge *bridge);

/* Nonblocking, whole-record I/O. Returns the byte count or a negative error
 * code. Empty/full returns -EAGAIN; oversized records return -EMSGSIZE.
 * Notification failures are terminal; the record may already be transferred. */
ssize_t shm_bridge_read(struct shm_bridge *bridge, void *data, size_t size);
ssize_t shm_bridge_write(struct shm_bridge *bridge, const void *data, size_t size);

#endif
