package vpn

const (
	Outline = "outline"
	OVC     = "ovc"
	WG      = "wg"
	IPSec   = "ipsec"
)

type FileConfig struct {
	Content    string
	FileName   string
	ConfigName string
}
