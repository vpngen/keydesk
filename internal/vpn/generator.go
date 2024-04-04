package vpn

import (
	"fmt"
	"github.com/vpngen/vpngine/naclkey"
)

type Generator interface {
	Generate() (Config2, error)
}

type EncryptionKeys struct {
	routerPub,
	shufflerPub [naclkey.NaclBoxKeyLength]byte
}

func NewGenerator(routerPub, shufflerPub [naclkey.NaclBoxKeyLength]byte) EncryptionKeys {
	return EncryptionKeys{
		routerPub:   routerPub,
		shufflerPub: shufflerPub,
	}
}

func (g EncryptionKeys) Generate(t ProtocolSet) (Config, error) {
	switch t {
	//case WG:
	//	return wg.generateWG(g.routerPub, g.shufflerPub)
	default:
		return nil, fmt.Errorf("unknown type: %s", t)
	}
}
