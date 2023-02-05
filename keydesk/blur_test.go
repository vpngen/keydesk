package keydesk

import (
	"encoding/base32"
	"net/netip"
	"testing"
)

var uuidSalt = [...]struct {
	id [16]byte
	x  uint32
}{
	{
		[16]byte{0xa1, 0x2c, 0x9e, 0x8e, 0x22, 0x53, 0x47, 0xc8, 0x30, 0x50, 0x81, 0x31, 0x98, 0x08, 0x1f, 0xd1},
		(0x2c << 24) + (0x81 << 16) + (0x98 << 8) + 0x9e,
	},
	{
		[16]byte{0x6f, 0x10, 0x79, 0x22, 0x3f, 0x56, 0xba, 0x14, 0x8c, 0xfb, 0xe1, 0x7a, 0xdd, 0xc7, 0xcc, 0xc7},
		(0xc7 << 24) + (0xba << 16) + (0x6f << 8) + 0x10,
	},
}

var ipBlur = [...]struct {
	ip   netip.Addr
	cidr int
	b, x uint32
}{
	{
		netip.MustParseAddr("192.168.128.30"),
		23,
		55, 41,
	},
	{
		netip.MustParseAddr("10.10.3.14"),
		22,
		314, 564,
	},
}

func TestExtractUint32Salt(t *testing.T) {
	for _, pre := range uuidSalt {
		s := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(pre.id[:])
		x := extractUint32Salt(s)
		if x != pre.x {
			t.Errorf("x != pre.x | %d != %d\n", x, pre.x)
		}
	}
}

func TestBlurIpv4Addr(t *testing.T) {
	for _, pre := range ipBlur {
		x := blurIpv4Addr(pre.ip, pre.cidr, pre.b)
		if x != pre.x {
			t.Errorf("x != pre.x | %d != %d\n", x, pre.x)
		}
	}
}
