package stackutil

import (
	"errors"
	"io"
	"sync"
	"syscall"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

var _ stack.LinkEndpoint = (*Endpoint)(nil)

// PacketBridge transfers packets and closes the transport owned by Endpoint.
// Close must tolerate repeated calls and concurrent packet operations.
type PacketBridge interface {
	io.Closer
	io.WriterTo
	io.ReaderFrom
}

// Endpoint connects netstack to a packet bridge. The bridge remains
// single-producer/single-consumer: tx serializes outbound callers, and one
// receive worker consumes inbound packets.
type Endpoint struct {
	// Bridge is owned by Endpoint and must be set before attachment.
	Bridge   PacketBridge
	tx       sync.Mutex
	mu       sync.RWMutex
	mtu      uint32
	addr     tcpip.LinkAddress
	attached bool
}

func (e *Endpoint) Attach(d stack.NetworkDispatcher) {
	e.mu.Lock()
	if d != nil && !e.attached {
		go func() {
			defer e.Bridge.Close()
			e.Bridge.WriteTo(dispatcherWriter{
				NetworkDispatcher: d,
			})
		}()
	}
	e.attached = d != nil
	e.mu.Unlock()
	if d == nil {
		e.Bridge.Close()
	}
}

// WritePackets copies packets into the bridge. A full bridge returns
// the accepted packet count and ErrNoBufferSpace; it never waits for capacity.
func (e *Endpoint) WritePackets(pkts stack.PacketBufferList) (int, tcpip.Error) {
	e.tx.Lock()
	defer e.tx.Unlock()
	reader := packetListReader(pkts.AsSlice())
	_, err := e.Bridge.ReadFrom(&reader)
	accepted := pkts.Len() - len(reader)
	switch {
	case err == nil:
		return accepted, nil
	case errors.Is(err, syscall.ENOBUFS):
		// The last packet may have filled the last slot. No further slot is
		// needed when the complete list has already been accepted.
		if len(reader) == 0 {
			return accepted, nil
		}
		return accepted, &tcpip.ErrNoBufferSpace{}
	case errors.Is(err, io.ErrShortBuffer):
		return accepted, &tcpip.ErrMessageTooLong{}
	default:
		e.Bridge.Close()
		return accepted, &tcpip.ErrClosedForSend{}
	}
}

func (e *Endpoint) IsAttached() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.attached
}

func (e *Endpoint) MTU() uint32 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.mtu
}

func (e *Endpoint) SetMTU(mtu uint32) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.mtu = mtu
}

func (e *Endpoint) LinkAddress() tcpip.LinkAddress {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.addr
}

func (e *Endpoint) SetLinkAddress(addr tcpip.LinkAddress) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.addr = addr
}

func (*Endpoint) Capabilities() stack.LinkEndpointCapabilities {
	return 0
}

func (*Endpoint) MaxHeaderLength() uint16 {
	return 0
}

func (*Endpoint) ARPHardwareType() header.ARPHardwareType {
	return header.ARPHardwareNone
}

func (*Endpoint) AddHeader(*stack.PacketBuffer) {
}

func (*Endpoint) ParseHeader(*stack.PacketBuffer) bool {
	return true
}

// SetOnCloseAction is unused: connection completion removes the NIC.
func (*Endpoint) SetOnCloseAction(func()) {}

// Close follows Attach(nil), which closes the bridge.
func (*Endpoint) Close() {}

// Wait is non-blocking. Bridge closure signals packet I/O shutdown.
func (*Endpoint) Wait() {}
