package recon

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// TracerouteHop represents a single hop in a traceroute.
type TracerouteHop struct {
	Hop      int    `json:"hop"`
	IP       string `json:"ip,omitempty" example:"192.168.1.1"`
	Hostname string `json:"hostname,omitempty" example:"router.local"`
	RTTMs    float64 `json:"rtt_ms" example:"1.23"`
	Timeout  bool   `json:"timeout"`
}

// TracerouteResult holds the complete traceroute output.
type TracerouteResult struct {
	Target    string          `json:"target" example:"192.168.1.1"`
	Hops      []TracerouteHop `json:"hops"`
	Reached   bool            `json:"reached"`
	TotalHops int             `json:"total_hops" example:"5"`
	DurationMs float64        `json:"duration_ms" example:"42.5"`
}

// TracerouteRequest is the request body for POST /traceroute.
type TracerouteRequest struct {
	Target  string `json:"target" example:"192.168.1.1"`
	MaxHops int    `json:"max_hops,omitempty" example:"30"`
	TimeoutMs int  `json:"timeout_ms,omitempty" example:"1000"`
}

// tracerouteRunning tracks whether a traceroute is currently in progress.
// Used for rate limiting (max 1 concurrent traceroute).
var tracerouteRunning atomic.Bool

// RunTraceroute performs an ICMP traceroute to the target IP.
func RunTraceroute(ctx context.Context, target string, maxHops, hopTimeoutMs int, logger *zap.Logger) (*TracerouteResult, error) {
	// Resolve the target to an IP address.
	targetIP := net.ParseIP(target)
	if targetIP == nil {
		addrs, err := net.DefaultResolver.LookupHost(ctx, target)
		if err != nil {
			return nil, fmt.Errorf("resolve target %q: %w", target, err)
		}
		if len(addrs) == 0 {
			return nil, fmt.Errorf("no addresses for target %q", target)
		}
		targetIP = net.ParseIP(addrs[0])
		if targetIP == nil {
			return nil, fmt.Errorf("invalid resolved address %q", addrs[0])
		}
	}

	// Ensure IPv4.
	targetIP = targetIP.To4()
	if targetIP == nil {
		return nil, fmt.Errorf("only IPv4 targets are supported")
	}

	if maxHops <= 0 {
		maxHops = 30
	}
	if hopTimeoutMs <= 0 {
		hopTimeoutMs = 1000
	}
	hopTimeout := time.Duration(hopTimeoutMs) * time.Millisecond

	start := time.Now()
	result := &TracerouteResult{
		Target: targetIP.String(),
		Hops:   make([]TracerouteHop, 0, maxHops),
	}

	// Open ICMP listener.
	// On Windows, use "ip4:icmp" (privileged raw socket).
	// On Linux/macOS, try unprivileged "udp4" first, fall back to "ip4:icmp".
	conn, network, err := openICMPConn()
	if err != nil {
		return nil, fmt.Errorf("open ICMP connection: %w", err)
	}
	defer conn.Close()

	// Create a unique ICMP identifier from our PID (masked to 16 bits).
	icmpID := os.Getpid() & 0xffff

	for ttl := 1; ttl <= maxHops; ttl++ {
		select {
		case <-ctx.Done():
			result.TotalHops = len(result.Hops)
			result.DurationMs = float64(time.Since(start).Microseconds()) / 1000.0
			return result, ctx.Err()
		default:
		}

		hop, reached := probeHop(ctx, conn, network, targetIP, ttl, icmpID, ttl, hopTimeout, logger)
		result.Hops = append(result.Hops, hop)

		if reached {
			result.Reached = true
			break
		}
	}

	result.TotalHops = len(result.Hops)
	result.DurationMs = float64(time.Since(start).Microseconds()) / 1000.0

	// Resolve hostnames for non-timeout hops (best effort, short timeout).
	resolveHostnames(result.Hops, logger)

	return result, nil
}

// openICMPConn opens an ICMP packet connection suitable for the current platform.
func openICMPConn() (*icmp.PacketConn, string, error) {
	if runtime.GOOS == "windows" {
		conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
		return conn, "ip4:icmp", err
	}

	// Try unprivileged ICMP first (Linux with sysctl net.ipv4.ping_group_range).
	conn, err := icmp.ListenPacket("udp4", "")
	if err == nil {
		return conn, "udp4", nil
	}

	// Fall back to privileged raw socket.
	conn, err = icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	return conn, "ip4:icmp", err
}

// probeHop sends a single ICMP Echo Request with the given TTL and waits for a response.
func probeHop(ctx context.Context, conn *icmp.PacketConn, network string, target net.IP, ttl, id, seq int, timeout time.Duration, logger *zap.Logger) (hop TracerouteHop, reached bool) {
	hop.Hop = ttl

	// Set TTL on the connection.
	if err := conn.IPv4PacketConn().SetTTL(ttl); err != nil {
		logger.Debug("failed to set TTL", zap.Int("ttl", ttl), zap.Error(err))
		hop.Timeout = true
		return hop, false
	}

	// Build ICMP Echo Request.
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   id,
			Seq:  seq,
			Data: []byte("SubNetree-Traceroute"),
		},
	}
	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		logger.Debug("failed to marshal ICMP message", zap.Error(err))
		hop.Timeout = true
		return hop, false
	}

	// Determine destination address format based on network type.
	var dst net.Addr
	if network == "udp4" {
		dst = &net.UDPAddr{IP: target, Port: 0}
	} else {
		dst = &net.IPAddr{IP: target}
	}

	// Send the probe.
	sendTime := time.Now()
	if _, err := conn.WriteTo(msgBytes, dst); err != nil {
		logger.Debug("failed to send ICMP probe", zap.Int("ttl", ttl), zap.Error(err))
		hop.Timeout = true
		return hop, false
	}

	// Set read deadline.
	deadline := sendTime.Add(timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	if err := conn.SetReadDeadline(deadline); err != nil {
		logger.Debug("failed to set read deadline", zap.Error(err))
		hop.Timeout = true
		return hop, false
	}

	// Read responses until we get one matching our probe or timeout.
	buf := make([]byte, 1500)
	for {
		n, peer, err := conn.ReadFrom(buf)
		if err != nil {
			// Timeout or context cancelled.
			hop.Timeout = true
			return hop, false
		}

		rtt := time.Since(sendTime)

		// Extract the IP from the peer address.
		var peerIP string
		switch p := peer.(type) {
		case *net.UDPAddr:
			peerIP = p.IP.String()
		case *net.IPAddr:
			peerIP = p.IP.String()
		default:
			peerIP = peer.String()
		}

		// Parse the ICMP message.
		proto := 1 // ICMPv4 protocol number for parsing
		reply, err := icmp.ParseMessage(proto, buf[:n])
		if err != nil {
			continue
		}

		switch reply.Type {
		case ipv4.ICMPTypeEchoReply:
			// Check if this is our echo reply.
			if echoReply, ok := reply.Body.(*icmp.Echo); ok {
				if echoReply.ID == id && echoReply.Seq == seq {
					hop.IP = peerIP
					hop.RTTMs = float64(rtt.Microseconds()) / 1000.0
					return hop, true
				}
			}
		case ipv4.ICMPTypeTimeExceeded:
			// Time exceeded -- this is an intermediate router.
			// Verify the inner payload matches our probe by checking the
			// encapsulated ICMP Echo Request's ID and Seq.
			if matchesProbe(reply, id, seq) {
				hop.IP = peerIP
				hop.RTTMs = float64(rtt.Microseconds()) / 1000.0
				return hop, false
			}
		case ipv4.ICMPTypeDestinationUnreachable:
			// Destination unreachable can also indicate we hit the target
			// (e.g., port unreachable on some systems).
			if matchesProbe(reply, id, seq) {
				hop.IP = peerIP
				hop.RTTMs = float64(rtt.Microseconds()) / 1000.0
				return hop, true
			}
		}

		// Not our packet, continue reading if still within deadline.
		if time.Now().After(deadline) {
			hop.Timeout = true
			return hop, false
		}
	}
}

// matchesProbe checks whether an ICMP error message (Time Exceeded or
// Destination Unreachable) contains our original Echo Request in the payload.
// ICMP error messages include the IP header + first 8 bytes of the original
// packet that triggered the error.
func matchesProbe(reply *icmp.Message, expectedID, expectedSeq int) bool {
	body, ok := reply.Body.(*icmp.TimeExceeded)
	if !ok {
		// Try DstUnreach.
		bodyDU, ok2 := reply.Body.(*icmp.DstUnreach)
		if !ok2 {
			return false
		}
		return matchesPayload(bodyDU.Data, expectedID, expectedSeq)
	}
	return matchesPayload(body.Data, expectedID, expectedSeq)
}

// matchesPayload extracts the ICMP Echo ID and Seq from the raw payload
// of an ICMP error message. The payload contains the original IP header
// (typically 20 bytes) followed by at least 8 bytes of the ICMP Echo Request.
func matchesPayload(data []byte, expectedID, expectedSeq int) bool {
	if len(data) < 28 {
		// Need at least 20 (IP header) + 8 (ICMP header with ID + Seq)
		return false
	}

	// The first byte contains the IP header length in the lower 4 bits.
	ihl := int(data[0]&0x0f) * 4
	if ihl < 20 || len(data) < ihl+8 {
		return false
	}

	// After IP header: ICMP type (1), code (1), checksum (2), ID (2), Seq (2).
	icmpData := data[ihl:]
	// Check ICMP type is Echo Request (8).
	if icmpData[0] != 8 {
		return false
	}
	// Extract ID and Seq (big-endian).
	id := int(binary.BigEndian.Uint16(icmpData[4:6]))
	seq := int(binary.BigEndian.Uint16(icmpData[6:8]))

	return id == expectedID && seq == expectedSeq
}

// resolveHostnames performs reverse DNS lookups on hop IPs (best effort).
func resolveHostnames(hops []TracerouteHop, logger *zap.Logger) {
	for i := range hops {
		if hops[i].IP == "" || hops[i].Timeout {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		names, err := net.DefaultResolver.LookupAddr(ctx, hops[i].IP)
		cancel()
		if err != nil || len(names) == 0 {
			continue
		}
		// Remove trailing dot from FQDN.
		hostname := names[0]
		if hostname != "" && hostname[len(hostname)-1] == '.' {
			hostname = hostname[:len(hostname)-1]
		}
		hops[i].Hostname = hostname
	}
}
