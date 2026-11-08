#ifndef SHM_BRIDGE_H
#define SHM_BRIDGE_H

#include "shm_ring.h"
#include <sys/types.h>

/* Bidirectional packet endpoint. Writes produce tx; reads consume rx.
 * Owns its nonblocking wakeup socket until an external owner adopts fd
 * and sets it to -1. Native I/O requires a descriptor still owned here. */
struct shm_bridge {
	struct shm_ring *tx;
	struct shm_ring *rx;
	int fd;
};

/* Creates two independently allocated endpoints sized for target_slots and
 * a nonblocking, close-on-exec socket pair. Each endpoint initially owns
 * its descriptor.
 * Returns 0 and sets result on success, or a negative error code on failure. */
int shm_bridge_new(uint32_t target_slots, uint16_t slot_capacity, struct shm_bridge *result[2]);

/* Logically close this endpoint. Rejects later packet I/O on its RX and TX
 * halves and closes fd if still owned. Storage remains available for in-flight
 * operations and must later be released with shm_bridge_release.
 * Returns 0 or a negative error from closing the descriptor. */
int shm_bridge_close(struct shm_bridge *bridge);

/* After all operations on this endpoint stop, release its halves and endpoint.
 * Also closes it if needed. Call once per endpoint. */
int shm_bridge_release(struct shm_bridge *bridge);

/* Acknowledge a readable wakeup before processing the rings. */
int shm_bridge_poll(struct shm_bridge *bridge);

/* Nonblocking, whole-packet I/O. Returns the byte count or a negative error
 * code. Empty/full returns -EAGAIN; oversized packets return -EMSGSIZE.
 * Notification failures are terminal; the slot may already be transferred. */
ssize_t shm_bridge_read(struct shm_bridge *bridge, void *data, size_t size);
ssize_t shm_bridge_write(struct shm_bridge *bridge, const void *data, size_t size);

#endif
