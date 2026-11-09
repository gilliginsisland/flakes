package stackutil

import (
	"io"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

var (
	_ io.Writer = (*dispatcherWriter)(nil)
	_ io.Reader = (*packetListReader)(nil)
)

// dispatcherWriter writes packets to the attached network dispatcher.
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
	pkt := stack.NewPacketBuffer(stack.PacketBufferOptions{
		Payload: buffer.MakeWithData(data),
	})
	defer pkt.DecRef()
	w.DeliverNetworkPacket(protocol, pkt)
	return len(data), nil
}

// packetListReader produces one complete packet per Read, copying its fragments
// into the ring's supplied buffer. EOF is returned only after the final packet,
// so a successful packet is never accompanied by an error.
type packetListReader []*stack.PacketBuffer

func (r *packetListReader) Read(dst []byte) (int, error) {
	if len(*r) == 0 {
		return 0, io.EOF
	}
	pkt := (*r)[0]
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
	*r = (*r)[1:]
	return n, nil
}
