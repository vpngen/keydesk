package vpn

import "github.com/vpngen/vpngine/naclkey"

type Generator interface {
	Generate(routerPub, shufflerPub [naclkey.NaclBoxKeyLength]byte) (Config, error)
}
