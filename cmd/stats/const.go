package main

import "time"

const (
	DefultWorkingDir = "/var/lib/vgqwd"
	WorkingFilename  = "qwork.json"
)

const (
	DefaultBrigadesListFileCheckDuration = 30 * time.Second
	DefaultStatisticsFetchingDuration    = 60 * time.Second
	DefaultJitterValue                   = 10 // sec
)
