package vpn

const (
	Outline   = "outline"
	Amnezia   = "amnezia"
	Wireguard = "wireguard"
	IPSec     = "ipsec"
	VGC       = "vgc"
)

type FileConfig struct {
	Content    string
	FileName   string
	ConfigName string
}
