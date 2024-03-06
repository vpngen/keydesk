package stat

import "time"

const (
	DefaultStatisticsFetchingDuration = 3600 * time.Second // 1h
	DefaultJitterValue                = 10                 // sec
)
