package vpn

const (
	Outline   = "outline"
	Amnezia   = "amnezia"
	Wireguard = "wireguard"
	IPSec     = "ipsec"
	Universal = "universal"
)

type FileConfig struct {
	Content    string
	FileName   string
	ConfigName string
}
