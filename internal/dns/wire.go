package dns

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strings"
)

const (
	qtypeA     uint16 = 1
	qtypeAAAA  uint16 = 28
	qclassIN   uint16 = 1
	maxUDPSize        = 4096
	maxTCPSize        = 65535
	maxJumps          = 16
)

var (
	errInvalidPacket     = errors.New("invalid dns packet")
	errTransactionID     = errors.New("dns transaction id mismatch")
	errResponseCode      = errors.New("dns non-zero rcode")
	errCompressionLoop   = errors.New("dns compression pointer loop")
	errUnsupportedRecord = errors.New("unsupported dns record type")
)

func recordTypeCode(rt RecordType) (uint16, error) {
	switch rt {
	case TypeA:
		return qtypeA, nil
	case TypeAAAA:
		return qtypeAAAA, nil
	default:
		return 0, errUnsupportedRecord
	}
}

func randomTxID() (uint16, error) {
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b[:]), nil
}

func buildDNSQuery(txID uint16, domain string, qtype uint16) ([]byte, error) {
	trimmed := strings.TrimSuffix(strings.TrimSpace(domain), ".")
	if trimmed == "" || len(trimmed) > 253 {
		return nil, ErrDomainInvalid
	}

	labels := strings.Split(trimmed, ".")
	buf := make([]byte, 12, 12+len(trimmed)+6)
	binary.BigEndian.PutUint16(buf[0:2], txID)
	binary.BigEndian.PutUint16(buf[2:4], 0x0100) // standard query, RD=1
	binary.BigEndian.PutUint16(buf[4:6], 1)      // QDCOUNT=1

	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return nil, ErrDomainInvalid
		}
		buf = append(buf, byte(len(label)))
		buf = append(buf, label...)
	}
	buf = append(buf, 0)
	buf = binary.BigEndian.AppendUint16(buf, qtype)
	buf = binary.BigEndian.AppendUint16(buf, qclassIN)
	return buf, nil
}

func skipDNSName(packet []byte, offset int) (int, error) {
	if offset < 0 || offset >= len(packet) {
		return 0, errInvalidPacket
	}
	cur := offset
	nextAfterFirstPtr := -1
	jumps := 0

	for {
		if cur >= len(packet) {
			return 0, errInvalidPacket
		}
		length := packet[cur]
		if length == 0 {
			if nextAfterFirstPtr != -1 {
				return nextAfterFirstPtr, nil
			}
			return cur + 1, nil
		}

		switch length & 0xC0 {
		case 0x00:
			cur += 1 + int(length)
			if cur > len(packet) {
				return 0, errInvalidPacket
			}
		case 0xC0:
			if cur+1 >= len(packet) {
				return 0, errInvalidPacket
			}
			ptr := int(binary.BigEndian.Uint16(packet[cur:cur+2]) & 0x3FFF)
			if ptr >= len(packet) {
				return 0, errInvalidPacket
			}
			if nextAfterFirstPtr == -1 {
				nextAfterFirstPtr = cur + 2
			}
			jumps++
			if jumps > maxJumps {
				return 0, errCompressionLoop
			}
			cur = ptr
		default:
			return 0, errInvalidPacket
		}
	}
}

func parseDNSResponse(packet []byte, expectedID uint16, qtype uint16) ([]netip.Addr, bool, error) {
	if len(packet) < 12 {
		return nil, false, errInvalidPacket
	}
	if binary.BigEndian.Uint16(packet[0:2]) != expectedID {
		return nil, false, errTransactionID
	}

	flags := binary.BigEndian.Uint16(packet[2:4])
	if (flags & 0x8000) == 0 {
		return nil, false, errInvalidPacket
	}
	if (flags & 0x0200) != 0 {
		return nil, true, nil
	}
	if (flags & 0x000F) != 0 {
		return nil, false, errResponseCode
	}

	qdCount := int(binary.BigEndian.Uint16(packet[4:6]))
	anCount := int(binary.BigEndian.Uint16(packet[6:8]))

	offset := 12
	for range qdCount {
		next, err := skipDNSName(packet, offset)
		if err != nil {
			return nil, false, err
		}
		offset = next + 4
		if offset > len(packet) {
			return nil, false, errInvalidPacket
		}
	}

	addrs := make([]netip.Addr, 0, anCount)
	for range anCount {
		next, err := skipDNSName(packet, offset)
		if err != nil {
			return nil, false, err
		}
		if next+10 > len(packet) {
			return nil, false, errInvalidPacket
		}
		rrType := binary.BigEndian.Uint16(packet[next : next+2])
		rrClass := binary.BigEndian.Uint16(packet[next+2 : next+4])
		rdLen := int(binary.BigEndian.Uint16(packet[next+8 : next+10]))
		rdataStart := next + 10
		rdataEnd := rdataStart + rdLen
		if rdataEnd > len(packet) {
			return nil, false, errInvalidPacket
		}

		if rrClass == qclassIN && rrType == qtype {
			switch {
			case qtype == qtypeA && rdLen == 4:
				var raw [4]byte
				copy(raw[:], packet[rdataStart:rdataEnd])
				addrs = append(addrs, netip.AddrFrom4(raw))
			case qtype == qtypeAAAA && rdLen == 16:
				var raw [16]byte
				copy(raw[:], packet[rdataStart:rdataEnd])
				addrs = append(addrs, netip.AddrFrom16(raw))
			}
		}
		offset = rdataEnd
	}

	return addrs, false, nil
}

func queryWireDNS(ctx context.Context, serverAddr, domain string, qtype uint16) ([]netip.Addr, error) {
	txID, err := randomTxID()
	if err != nil {
		return nil, fmt.Errorf("generate txid: %w", err)
	}
	reqPacket, err := buildDNSQuery(txID, domain, qtype)
	if err != nil {
		return nil, err
	}

	respPacket, err := exchangeUDP(ctx, serverAddr, reqPacket)
	if err != nil {
		return nil, err
	}

	addrs, truncated, err := parseDNSResponse(respPacket, txID, qtype)
	if err != nil {
		return nil, err
	}
	if truncated {
		tcpResp, tcpErr := exchangeTCP(ctx, serverAddr, reqPacket)
		if tcpErr != nil {
			return nil, tcpErr
		}
		addrs, _, err = parseDNSResponse(tcpResp, txID, qtype)
		if err != nil {
			return nil, err
		}
	}
	return addrs, nil
}

func exchangeUDP(ctx context.Context, serverAddr string, req []byte) ([]byte, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "udp", serverAddr)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return nil, err
		}
	}
	if _, err := conn.Write(req); err != nil {
		return nil, err
	}
	buf := make([]byte, maxUDPSize)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func exchangeTCP(ctx context.Context, serverAddr string, req []byte) ([]byte, error) {
	if len(req) > maxTCPSize {
		return nil, errInvalidPacket
	}
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", serverAddr)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return nil, err
		}
	}
	framed := make([]byte, 2+len(req))
	binary.BigEndian.PutUint16(framed[0:2], uint16(len(req)))
	copy(framed[2:], req)
	if _, err := conn.Write(framed); err != nil {
		return nil, err
	}

	var lenBuf [2]byte
	if _, err := io.ReadFull(conn, lenBuf[:]); err != nil {
		return nil, err
	}
	respLen := int(binary.BigEndian.Uint16(lenBuf[:]))
	if respLen == 0 || respLen > maxTCPSize {
		return nil, errInvalidPacket
	}
	resp := make([]byte, respLen)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
