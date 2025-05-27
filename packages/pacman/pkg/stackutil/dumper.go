package stackutil

import (
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

// DumpFunc logs packet data and direction.
type PacketDumper interface {
	DumpPacket(pkt *stack.PacketBuffer, chain string)
}

var _ PacketDumper = (PacketDumperFunc)(nil)

// PacketDumperFunc is an adapter to allow ordinary functions to be used as PacketDumper.
type PacketDumperFunc func(pkt *stack.PacketBuffer, chain string)

// DumpPacket calls f(pkt, dir).
func (f PacketDumperFunc) DumpPacket(pkt *stack.PacketBuffer, chain string) {
	f(pkt, chain)
}

var _ stack.LinkEndpoint = (*DumpingLinkEndpoint)(nil)

// DumpingLinkEndpoint wraps another LinkEndpoint to dump packets.
type DumpingLinkEndpoint struct {
	stack.LinkEndpoint
	Dumper PacketDumper
}

func (dle *DumpingLinkEndpoint) Attach(dispatcher stack.NetworkDispatcher) {
	if dispatcher != nil && dle.Dumper != nil {
		dispatcher = &dumpingDispatcher{
			NetworkDispatcher: dispatcher,
			dumper:            dle.Dumper,
		}
	}
	dle.LinkEndpoint.Attach(dispatcher)
}

func (dle *DumpingLinkEndpoint) WritePackets(pkts stack.PacketBufferList) (int, tcpip.Error) {
	if dmpr := dle.Dumper; dmpr != nil {
		for _, pkt := range pkts.AsSlice() {
			pkt.IncRef()
			go func() {
				defer pkt.DecRef()
				dmpr.DumpPacket(pkt, "outbound")
			}()
		}
	}
	return dle.LinkEndpoint.WritePackets(pkts)
}

var _ stack.NetworkDispatcher = (*dumpingDispatcher)(nil)

// dumpingDispatcher wraps a NetworkDispatcher to dump inbound packets.
type dumpingDispatcher struct {
	stack.NetworkDispatcher
	dumper PacketDumper
}

func (d *dumpingDispatcher) DeliverNetworkPacket(protocol tcpip.NetworkProtocolNumber, pkt *stack.PacketBuffer) {
	if d.dumper != nil {
		pkt.IncRef()
		go func() {
			defer pkt.DecRef()
			d.dumper.DumpPacket(pkt, "inbound")
		}()
	}
	d.NetworkDispatcher.DeliverNetworkPacket(protocol, pkt)
}

func (d *dumpingDispatcher) DeliverLinkPacket(protocol tcpip.NetworkProtocolNumber, pkt *stack.PacketBuffer) {
	if d.dumper != nil {
		pkt.IncRef()
		go func() {
			defer pkt.DecRef()
			d.dumper.DumpPacket(pkt, "inbound-link")
		}()
	}
	d.NetworkDispatcher.DeliverLinkPacket(protocol, pkt)
}
