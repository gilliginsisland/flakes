#include "shm_bridge.h"

#include <errno.h>
#include <fcntl.h>
#include <stdlib.h>
#include <sys/socket.h>
#include <unistd.h>

int shm_bridge_new(uint32_t capacity, uint32_t record_size, struct shm_bridge *result[2]) {
	struct shm_bridge *endpoints[2] = {NULL, NULL};
	struct shm_ring *rings[2] = {NULL, NULL};
	int fds[2] = {-1, -1}, ret, type = SOCK_STREAM;

	for (unsigned int i = 0; i < 2; i++) {
		ret = shm_ring_new(capacity, record_size, &rings[i]);
		if (ret < 0)
			goto fail;
		endpoints[i] = calloc(1, sizeof(*endpoints[i]));
		if (!endpoints[i])
			goto fail_errno;
	}
#ifdef SOCK_CLOEXEC
	type |= SOCK_CLOEXEC;
#endif
#ifdef SOCK_NONBLOCK
	type |= SOCK_NONBLOCK;
#endif
	if (socketpair(AF_UNIX, type, 0, fds))
		goto fail_errno;
	for (unsigned int i = 0; i < 2; i++) {
#ifndef SOCK_CLOEXEC
		if (fcntl(fds[i], F_SETFD, FD_CLOEXEC) < 0)
			goto fail_errno;
#endif
#ifndef SOCK_NONBLOCK
		int flags = fcntl(fds[i], F_GETFL);
		if (flags < 0 || fcntl(fds[i], F_SETFL, flags | O_NONBLOCK) < 0)
			goto fail_errno;
#endif
	}
	for (unsigned int i = 0; i < 2; i++) {
		*endpoints[i] = (struct shm_bridge){
			.tx = rings[i],
			.rx = rings[1 - i],
			.fd = fds[i],
		};
		result[i] = endpoints[i];
	}
	return 0;

fail_errno:
	ret = -errno;
fail:
	for (unsigned int i = 0; i < 2; i++) {
		if (fds[i] >= 0)
			close(fds[i]);
		free(rings[i]);
		free(endpoints[i]);
	}
	return ret;
}

int shm_bridge_close(struct shm_bridge *bridge) {
	int ret = 0;

	if (bridge->fd >= 0) {
		if (close(bridge->fd) < 0)
			ret = -errno;
		bridge->fd = -1;
	}
	shm_ring_close_read(bridge->rx);
	shm_ring_close_write(bridge->tx);
	free(bridge);
	return ret;
}

int shm_bridge_poll(struct shm_bridge *bridge) {
	char notifications[128];
	ssize_t ret;

	do {
		ret = read(bridge->fd, notifications, sizeof(notifications));
	} while (ret > 0 || (ret < 0 && errno == EINTR));
	if (!ret)
		return -EPIPE;
	if (errno != EAGAIN && errno != EWOULDBLOCK)
		return -errno;
	/* Clear only our bits; the peer owns the other bit in each word. Clear
	 * before inspecting the rings so racing work is seen or leaves a wakeup. */
	__atomic_fetch_and(&bridge->rx->pending, ~SHM_RING_READ_PENDING, __ATOMIC_SEQ_CST);
	__atomic_fetch_and(&bridge->tx->pending, ~SHM_RING_WRITE_PENDING, __ATOMIC_SEQ_CST);
	return 0;
}

static int shm_bridge_signal(struct shm_bridge *bridge, uint64_t *pending, uint64_t mask) {
	ssize_t ret;

	if (__atomic_fetch_or(pending, mask, __ATOMIC_SEQ_CST) & mask)
		return 0;
	do {
		ret = write(bridge->fd, "x", 1);
	} while (ret < 0 && errno == EINTR);
	if (ret < 0 && errno != EAGAIN && errno != EWOULDBLOCK)
		return -errno;
	return 0;
}

ssize_t shm_bridge_read(struct shm_bridge *bridge, void *data, size_t size) {
	ssize_t ret;
	int err;

	ret = shm_ring_pop(bridge->rx, data, size);
	if (ret < 0)
		return ret;
	err = shm_bridge_signal(bridge, &bridge->rx->pending, SHM_RING_WRITE_PENDING);
	return err < 0 ? err : ret;
}

ssize_t shm_bridge_write(struct shm_bridge *bridge, const void *data, size_t size) {
	ssize_t ret;
	int err;

	if (!size)
		return 0;
	ret = shm_ring_commit(bridge->tx, data, size);
	if (ret < 0)
		return ret;
	err = shm_bridge_signal(bridge, &bridge->tx->pending, SHM_RING_READ_PENDING);
	return err < 0 ? err : ret;
}
