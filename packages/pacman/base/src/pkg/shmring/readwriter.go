package shmring

import (
	"io"
	"os"
	"runtime"
	"sync/atomic"
)

const (
	readWriterPolling uint32 = 1 << iota
	readWriterClosed
)

// ReadWriter provides blocking Go I/O for a Bridge. It supports one reader and one
// writer; calls in the same direction must not overlap. Record bytes never
// cross its notification socket. The Bridge owns its file and native storage.
type ReadWriter struct {
	*Bridge
	state atomic.Uint32
	read  chan struct{}
	write chan struct{}
	done  chan struct{}
}

// NewReadWriter provides blocking I/O for bridge. Blocked callers share one
// notification poll. Close the ReadWriter to wake blocked operations.
func NewReadWriter(bridge *Bridge) *ReadWriter {
	var rw ReadWriter
	p := &rw
	*p = ReadWriter{
		Bridge: bridge,
		read:   make(chan struct{}, 1),
		write:  make(chan struct{}, 1),
		done:   make(chan struct{}),
	}
	return p
}

// The caller keeps polling ownership while retrying its ring. It hands ownership
// to the other direction when its operation finishes, even without another event.
func (p *ReadWriter) poll() error {
	pending, err := p.Bridge.Poll()
	if err != nil {
		return err
	}
	if pending&ReadPending != 0 {
		select {
		case p.read <- struct{}{}:
		default:
		}
	}
	if pending&WritePending != 0 {
		select {
		case p.write <- struct{}{}:
		default:
		}
	}
	return nil
}

func (p *ReadWriter) finishPoll(wakeup chan<- struct{}) {
	// Hand ownership to the other direction, even if it has no pending event:
	// it must be able to poll after this caller returns. Clear ownership first.
	p.state.And(^readWriterPolling)
	select {
	case wakeup <- struct{}{}:
	default:
	}
}

func (p *ReadWriter) Read(b []byte) (int, error) {
	return p.transfer(b, p.Bridge.Pop, ErrEmpty, p.read, p.write)
}

func (p *ReadWriter) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	return p.transfer(b, p.Bridge.Commit, ErrFull, p.write, p.read)
}

func (p *ReadWriter) transfer(
	b []byte,
	operation func([]byte) (int, error),
	blocked error,
	wait <-chan struct{},
	handoff chan<- struct{},
) (int, error) {
	defer runtime.KeepAlive(p)
	select {
	case <-p.done:
		return 0, io.ErrClosedPipe
	default:
	}
	var polling bool
	for {
		n, err := operation(b)
		if err != blocked {
			if polling {
				p.finishPoll(handoff)
			}
			return n, err
		}
		if polling {
			if err := p.poll(); err != nil {
				p.finishPoll(handoff)
				p.Close()
				return 0, io.ErrClosedPipe
			}
		} else if p.state.CompareAndSwap(0, readWriterPolling) {
			// Recheck the ring after claiming polling ownership.
			polling = true
		} else {
			select {
			case <-wait:
			case <-p.done:
				return 0, io.ErrClosedPipe
			}
		}
	}
}

// Close shuts down a Go endpoint's socket without waiting for active operations
// to finish unwinding. The Bridge's GC cleanup releases native storage later.
// Subsequent calls return os.ErrClosed without waiting for shutdown to finish.
func (p *ReadWriter) Close() error {
	defer runtime.KeepAlive(p)
	state := p.state.Or(readWriterClosed)
	if state&readWriterClosed != 0 {
		return os.ErrClosed
	}
	defer close(p.done)
	return p.Bridge.Close()
}

// Done closes after explicit Close or a notification poll detects failure.
// Idle endpoints do not poll. It does not wait for active operations, GC cleanup,
// or a native event loop to stop.
func (p *ReadWriter) Done() <-chan struct{} {
	return p.done
}

// Wait waits for notification shutdown. Only the first Close reports its error.
func (p *ReadWriter) Wait() error {
	<-p.Done()
	return nil
}
