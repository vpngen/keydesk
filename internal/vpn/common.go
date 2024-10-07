package vpn

const (
	Outline   = "outline"
	Amnezia   = "amnezia"
	Wireguard = "wireguard"
	IPSec     = "ipsec"
	Universal = "universal"
	Proto0    = "proto0"
)

type FileConfig struct {
	Content    string
	FileName   string
	ConfigName string
}
