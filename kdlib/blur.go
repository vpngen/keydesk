package kdlib

import (
	"encoding/base32"
	"encoding/binary"
	"net/netip"
)

// ExtractUint32Salt - make pseudo rand salt for number bluring.
func ExtractUint32Salt(s string) uint32 {
	b, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s)
	if err != nil || len(b) < 16 {
		return 0
	}

	buf := []byte{
		b[b[0]&0x0f],
		b[b[0]&0xf0>>4],
		b[b[1]&0x0f],
		b[b[1]&0xf0>>4],
	}

	return binary.BigEndian.Uint32(buf)
}

// BlurIpv4Addr - blur IPv4 address with uit32 salt.
func BlurIpv4Addr(ip netip.Addr, cidr int, b uint32) uint32 {
	y := binary.BigEndian.Uint32(ip.AsSlice())
	m := ^uint32(0) >> cidr

	return (y ^ b) & m
}
