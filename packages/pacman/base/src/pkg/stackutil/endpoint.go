package stackutil

import (
	"context"
	"io"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

var (
	_ io.ReadWriteCloser = (*Endpoint)(nil)
	_ io.WriterTo        = (*Endpoint)(nil)
	_ io.ReaderFrom      = (*Endpoint)(nil)
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

func (e *Endpoint) Read(p []byte) (int, error) {
	return e.readPacketData(p)
}

func (e *Endpoint) WriteTo(w io.Writer) (n int64, err error) {
	p := make([]byte, e.Endpoint.MTU())
	for {
		offset, err := e.readPacketData(p)
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return n, err
		}
		count, err := w.Write(p[:offset])
		n += int64(count)
		if err != nil {
			return n, err
		}
	}
}

func (e *Endpoint) readPacketData(p []byte) (n int, err error) {
	pkt := e.ReadContext(context.Background())
	if pkt == nil {
		return 0, io.EOF
	}
	defer pkt.DecRef()
	b := pkt.ToBuffer()
	vl := b.AsViewList()
	for v := vl.Front(); v != nil; v = v.Next() {
		s := v.AsSlice()
		if n+len(s) > len(p) {
			return n, io.ErrShortBuffer
		}
		n += copy(p[n:], s)
	}
	return n, nil
}

func (e *Endpoint) Write(p []byte) (n int, err error) {
	return e.writePacketData(p)
}

func (e *Endpoint) ReadFrom(r io.Reader) (n int64, err error) {
	p := make([]byte, e.Endpoint.MTU())
	for {
		count, err := r.Read(p)
		n += int64(count)
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return n, err
		}
		_, err = e.writePacketData(p[:count])
		if err != nil {
			return n, err
		}
	}
}

func (e *Endpoint) writePacketData(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	pb := stack.NewPacketBuffer(stack.PacketBufferOptions{
		Payload: buffer.MakeWithData(p),
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
