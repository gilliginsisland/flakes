package stackutil

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"

	"github.com/gilliginsisland/pacman/pkg/shmring"
	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

var (
	_ stack.LinkEndpoint = (*RingEndpoint)(nil)
	_ io.Writer          = dispatcherWriter{}
)

// RingEndpoint connects netstack directly to a native packet bridge. The bridge
// remains single-producer/single-consumer: tx serializes outbound callers, and
// one receive worker consumes inbound packets and polls notifications.
type RingEndpoint struct {
	bridge     *shmring.Bridge
	tx         sync.Mutex
	mu         sync.RWMutex
	dispatcher stack.NetworkDispatcher
	onClose    func()
	mtu        uint32
	addr       tcpip.LinkAddress
	started    bool
	workers    sync.WaitGroup
	closed     atomic.Bool
	closeOnce  sync.Once
}

// NewRingEndpoint takes ownership of bridge. Receiving starts when attached;
// Close releases the bridge and wakes the receive worker.
func NewRingEndpoint(bridge *shmring.Bridge, mtu uint32) *RingEndpoint {
	return &RingEndpoint{bridge: bridge, mtu: mtu}
}

func (e *RingEndpoint) Attach(d stack.NetworkDispatcher) {
	e.mu.Lock()
	e.dispatcher = d
	if d == nil {
		e.onClose = nil
	}
	if d != nil && !e.started && !e.closed.Load() {
		e.started = true
		e.workers.Go(func() { e.receive(dispatcherWriter{NetworkDispatcher: d}) })
	}
	e.mu.Unlock()
	if d == nil {
		e.Close()
	}
}

func (e *RingEndpoint) receive(writer dispatcherWriter) {
	defer e.Close()
	for {
		if e.closed.Load() {
			return
		}
		_, err := e.bridge.WriteTo(writer)
		if errors.Is(err, shmring.ErrEmpty) {
			// Notifications are hints: after acknowledging, retry the ring before
			// waiting again, including after a space-only or stale notification.
			if _, err := e.bridge.Poll(); err != nil {
				return
			}
		} else {
			// WriteTo returns nil at EOF. Both EOF and terminal errors end
			// the receive worker; only an empty, open ring needs a retry.
			return
		}
	}
}

// dispatcherWriter writes ring records to the attached network dispatcher.
type dispatcherWriter struct {
	stack.NetworkDispatcher
}

func (w dispatcherWriter) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	var protocol tcpip.NetworkProtocolNumber
	switch header.IPVersion(data) {
	case header.IPv4Version:
		protocol = ipv4.ProtocolNumber
	case header.IPv6Version:
		protocol = ipv6.ProtocolNumber
	default:
		return len(data), nil
	}
	// Netstack may retain the packet after delivery, so it owns a copy. The
	// borrowed ring bytes are never retained beyond this Write call.
	pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{Payload: buffer.MakeWithData(data)})
	defer pkt.DecRef()
	w.DeliverNetworkPacket(protocol, pkt)
	return len(data), nil
}

// WritePackets copies directly into available ring slots. A full ring returns
// the accepted packet count and ErrNoBufferSpace; it never waits for capacity.
func (e *RingEndpoint) WritePackets(pkts stack.PacketBufferList) (int, tcpip.Error) {
	e.tx.Lock()
	defer e.tx.Unlock()
	if e.closed.Load() {
		return 0, &tcpip.ErrClosedForSend{}
	}
	reader := packetListReader{packets: pkts.AsSlice(), endpoint: e}
	_, err := e.bridge.ReadFrom(&reader)
	switch {
	case err == nil:
		return reader.count, nil
	case errors.Is(err, shmring.ErrFull):
		// The last packet may have filled the last slot. No further slot is
		// needed when the complete list has already been accepted.
		if reader.count == pkts.Len() {
			return reader.count, nil
		}
		return reader.count, &tcpip.ErrNoBufferSpace{}
	case errors.Is(err, io.ErrShortBuffer):
		return reader.count, &tcpip.ErrMessageTooLong{}
	default:
		e.Close()
		return reader.count, &tcpip.ErrClosedForSend{}
	}
}

// packetListReader produces one complete packet per Read, copying its fragments
// into the ring's supplied buffer. EOF is returned only after the final packet,
// so a successful packet is never accompanied by an error.
type packetListReader struct {
	packets  []*stack.PacketBuffer
	endpoint *RingEndpoint
	count    int
}

func (r *packetListReader) Read(dst []byte) (int, error) {
	if r.endpoint.closed.Load() {
		return 0, io.ErrClosedPipe
	}
	if r.count == len(r.packets) {
		return 0, io.EOF
	}
	pkt := r.packets[r.count]
	if len(dst) < pkt.Size() {
		return 0, io.ErrShortBuffer
	}
	views, offset := pkt.AsViewList()
	n := 0
	for v := views.Front(); v != nil; v = v.Next() {
		data := v.AsSlice()
		if offset >= len(data) {
			offset -= len(data)
			continue
		}
		data = data[offset:]
		offset = 0
		n += copy(dst[n:], data)
	}
	r.count++
	return n, nil
}

func (e *RingEndpoint) IsAttached() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.dispatcher != nil
}

func (e *RingEndpoint) MTU() uint32 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.mtu
}

func (e *RingEndpoint) SetMTU(mtu uint32) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.mtu = mtu
}

func (e *RingEndpoint) LinkAddress() tcpip.LinkAddress {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.addr
}

func (e *RingEndpoint) SetLinkAddress(addr tcpip.LinkAddress) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.addr = addr
}

func (*RingEndpoint) Capabilities() stack.LinkEndpointCapabilities {
	return 0
}

func (*RingEndpoint) MaxHeaderLength() uint16 {
	return 0
}

func (*RingEndpoint) ARPHardwareType() header.ARPHardwareType {
	return header.ARPHardwareNone
}

func (*RingEndpoint) AddHeader(*stack.PacketBuffer) {
}

func (*RingEndpoint) ParseHeader(*stack.PacketBuffer) bool {
	return true
}

func (e *RingEndpoint) SetOnCloseAction(action func()) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onClose = action
}

// Close releases the owned bridge and removes its NIC when it closes itself.
func (e *RingEndpoint) Close() {
	e.closeOnce.Do(func() {
		e.mu.Lock()
		action := e.onClose
		e.onClose = nil
		e.mu.Unlock()
		if action != nil {
			e.workers.Add(1)
		}
		e.closed.Store(true)
		e.bridge.Close()
		if action != nil {
			// A packet send may hold netstack locks. Remove the NIC in a
			// separate goroutine to avoid reentering the stack here.
			go func() {
				defer e.workers.Done()
				action()
			}()
		}
	})
}

// Wait joins the receive worker and any active outbound write.
func (e *RingEndpoint) Wait() {
	// Synchronize with Attach publishing its one worker before waiting.
	e.mu.RLock()
	started := e.started
	e.mu.RUnlock()
	if started {
		e.workers.Wait()
	}
	e.tx.Lock()
	e.tx.Unlock()
}
