package stackutil

import (
	"context"
	"errors"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

// WrapChannel wraps the provided netstack channel-based Endpoint and returns a wrapper
// that implements io.Reader and io.Writer on the channel. This allows callers to read
// and write packets as raw []byte directly to the channel.
func WrapChannel(channel *channel.Endpoint) *Endpoint {
	return &Endpoint{
		Endpoint: channel,
	}
}

// Endpoint is a wrapper around a channel.Endpoint that implements
// the io.Reader and io.Writer interfaces.
type Endpoint struct {
	*channel.Endpoint
}

func (e *Endpoint) Read(p []byte) (n int, err error) {
	pkt := e.ReadContext(context.Background())
	if pkt == nil {
		return 0, errors.New("nil packet")
	}
	defer pkt.DecRef()
	b := pkt.ToBuffer()
	n = copy(p, b.Flatten())
	return n, nil
}

func (e *Endpoint) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	// NewPacketBuffer takes ownership of the data, so making a copy is necessary
	data := make([]byte, len(p))
	copy(data, p)
	pb := stack.NewPacketBuffer(stack.PacketBufferOptions{
		Payload: buffer.MakeWithData(data),
	})

	var ipv tcpip.NetworkProtocolNumber
	switch header.IPVersion(p) {
	case header.IPv4Version:
		ipv = ipv4.ProtocolNumber
	case header.IPv6Version:
		ipv = ipv6.ProtocolNumber
	default:
		// todo: log this
		return
	}
	e.InjectInbound(ipv, pb)
	return len(p), nil
}

func (e *Endpoint) Close() error {
	e.Endpoint.Close()
	return nil
}
