package netutil

import (
	"encoding/hex"
	"log/slog"
	"strings"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func DumpPacket(data []byte, chain string) {
	if len(data) == 0 {
		slog.Debug("empty packet", slog.String("chain", chain))
		return
	}

	firstLayer := gopacket.LayerTypePayload
	switch chain {
	case "inbound-link":
		firstLayer = layers.LayerTypeEthernet
	case "inbound", "outbound":
		switch data[0] >> 4 {
		case 4:
			firstLayer = layers.LayerTypeIPv4
		case 6:
			firstLayer = layers.LayerTypeIPv6
		}
	default:
		firstLayer = gopacket.LayerTypePayload
	}

	packet := gopacket.NewPacket(data, firstLayer, gopacket.Default)

	fields := []any{
		slog.String("chain", chain),
	}

	var (
		hasLayers      bool
		applicationSet bool
	)

	for _, layer := range packet.Layers() {
		switch l := layer.(type) {
		case *layers.IPv4:
			fields = append(fields, slog.Group("ipv4",
				slog.String("src", l.SrcIP.String()),
				slog.String("dst", l.DstIP.String()),
				slog.String("protocol", l.Protocol.String()),
				slog.Int("ttl", int(l.TTL)),
				slog.Bool("flags_df", (l.Flags&layers.IPv4DontFragment) != 0),
				slog.Bool("flags_mf", (l.Flags&layers.IPv4MoreFragments) != 0),
			))
			hasLayers = true
		case *layers.IPv6:
			fields = append(fields, slog.Group("ipv6",
				slog.String("src", l.SrcIP.String()),
				slog.String("dst", l.DstIP.String()),
				slog.String("next_header", l.NextHeader.String()),
				slog.Int("hop_limit", int(l.HopLimit)),
			))
			hasLayers = true
		case *layers.TCP:
			fields = append(fields, slog.Group("tcp",
				slog.Int("src_port", int(l.SrcPort)),
				slog.Int("dst_port", int(l.DstPort)),
				slog.Int("seq", int(l.Seq)),
				slog.Int("ack", int(l.Ack)),
				slog.Bool("syn", l.SYN),
				slog.Bool("ack_flag", l.ACK),
				slog.Bool("fin", l.FIN),
				slog.Bool("rst", l.RST),
			))
			hasLayers = true
		case *layers.UDP:
			fields = append(fields, slog.Group("udp",
				slog.Int("src_port", int(l.SrcPort)),
				slog.Int("dst_port", int(l.DstPort)),
			))
			hasLayers = true
		case *layers.ICMPv4:
			fields = append(fields, slog.Group("icmpv4",
				slog.Int("type", int(l.TypeCode.Type())),
				slog.Int("code", int(l.TypeCode.Code())),
			))
			hasLayers = true
		case *layers.ICMPv6:
			fields = append(fields, slog.Group("icmpv6",
				slog.Int("type", int(l.TypeCode.Type())),
				slog.Int("code", int(l.TypeCode.Code())),
			))
			hasLayers = true
		case *layers.DNS:
			fields = append(fields, slog.Group("dns",
				slog.Int("questions", len(l.Questions)),
				slog.Int("answers", len(l.Answers)),
				slog.Bool("qr", l.QR),
			))
			applicationSet = true
		}
	}

	// Handle raw application payload (e.g. HTTP, TLS) if present
	if app := packet.ApplicationLayer(); app != nil && !applicationSet {
		payload := app.Payload()
		if len(payload) > 0 {
			text := string(payload)
			if isHTTP(text) {
				lines := strings.SplitN(text, "\n", 2)
				fields = append(fields, slog.Group("http",
					slog.String("request_line", strings.TrimSpace(lines[0])),
				))
			} else if isTLSClientHello(payload) {
				fields = append(fields, slog.Group("tls",
					slog.String("type", "ClientHello (likely)"),
					slog.Int("length", len(payload)),
				))
			} else {
				fields = append(fields, slog.Group("app_data",
					slog.Int("length", len(payload)),
				))
			}
		}
	}

	if err := packet.ErrorLayer(); err != nil || !hasLayers {
		fields = append(fields, slog.String("raw_hex", hex.Dump(data)))
	}

	slog.Debug("packet dump", fields...)
}

// Helpers
func isHTTP(text string) bool {
	return strings.HasPrefix(text, "GET ") ||
		strings.HasPrefix(text, "POST ") ||
		strings.HasPrefix(text, "HTTP/1.") ||
		strings.HasPrefix(text, "PUT ") ||
		strings.HasPrefix(text, "HEAD ") ||
		strings.HasPrefix(text, "DELETE ") ||
		strings.HasPrefix(text, "OPTIONS ")
}

func isTLSClientHello(b []byte) bool {
	// TLS ClientHello starts with:
	// [0] 0x16 (Handshake)
	// [1-2] Version (e.g., 0x0301 for TLS 1.0, 0x0303 for TLS 1.2)
	// [5] Handshake Type (0x01 = ClientHello)
	return len(b) > 5 && b[0] == 0x16 && b[5] == 0x01
}
