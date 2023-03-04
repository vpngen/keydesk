package keydesk

import "time"

// DefaultEtcDir -  default config dir.
const DefaultEtcDir = "/etc"

// Default key names.
const (
	RouterPublicKeyFilename   = "vg-router.json"
	ShufflerPublicKeyFilename = "vg-shuffler.json"
	MaxIdlePeriod             = 10 * time.Minute
)

// Brigades consts.
const (
	DefaultBrigadesListFile = "brigades.lst"
	DefaultBrigadesListDir  = "/var/lib/vgkeydesk"
)
