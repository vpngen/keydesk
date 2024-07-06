package vpn

import "strings"

type ProtocolSet uint8

const (
	TypeOutline ProtocolSet = 1 << iota
	TypeOVC
	TypeWG
	TypeIPSec
)

var (
	type2string = map[ProtocolSet]string{
		TypeOutline: Outline,
		TypeOVC:     Amnezia,
		TypeWG:      Wireguard,
		TypeIPSec:   IPSec,
	}
	string2type = map[string]ProtocolSet{
		Outline:   TypeOutline,
		Amnezia:   TypeOVC,
		Wireguard: TypeWG,
		IPSec:     TypeIPSec,
	}
)

func (s ProtocolSet) String() string {
	return strings.Join(s.Protocols(), ",")
}

func (s ProtocolSet) Protocols() []string {
	protocols := make([]string, 0, len(type2string))
	for k, v := range type2string {
		if s&k != 0 {
			protocols = append(protocols, v)
		}
	}
	return protocols
}

func (s ProtocolSet) GetSupported(available ProtocolSet) (supported ProtocolSet, unsupported ProtocolSet) {
	supported = s & available
	unsupported = s & ^available
	return
}

func NewProtocolSet(protocols []string) ProtocolSet {
	t := ProtocolSet(0)
	for _, v := range protocols {
		t |= string2type[v]
	}
	return t
}

func NewProtocolSetFromString(s string) ProtocolSet {
	return NewProtocolSet(strings.Split(s, ","))
}
