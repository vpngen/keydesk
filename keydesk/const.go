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

const (
	DefaultEndpointsTTL = 24 * 7 * time.Hour
)

// User statuses.
const (
	UserStatusOK              = "green"
	UserStatusNeverUsed       = "black"
	UserStatusMonthlyInactive = "grey"
	UserStatusLimited         = "yellow"
	UserStatusBlocked         = "red"
)
