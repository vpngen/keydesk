package stat

import "time"

const (
	// DefaultStatisticsFetchingDuration = 3600 * time.Second // 1h
	DefaultStatisticsFetchingDuration = 5 * time.Minute // 5m
	// DefaultJitterValue                = 1800        // sec
	DefaultJitterValue = 30 // sec
)
