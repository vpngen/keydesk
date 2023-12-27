package maintenance

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// IsMaintenance Reads .maintenance file. If file contains timestamp, till is timestamp.
// If file is empty or timestamp is invalid or zero, till is file modtime + 10800 sec.
// Returns (true, till) if till is later than now.
func IsMaintenance(path string) (bool, time.Time) {
	file, err := os.Open(path)
	if err != nil {
		// can't open file, it's not maintenance
		_, _ = fmt.Fprintln(os.Stderr, "read maintenance file error:", err)
		return false, time.Time{}
	}
	defer file.Close()

	// read timestamp
	data, err := io.ReadAll(file)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "read maintenance timestamp error:", err)
		return false, time.Time{}
	}

	var till time.Time // time till which we're in maintenance

	// read timestamp
	timestamp, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || timestamp == 0 {
		// timestamp is 0 or invalid, maintenance is till modtime + 10800 sec
		stat, err := file.Stat()
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "read maintenance file stat error:", err)
			return false, time.Time{}
		}
		till = stat.ModTime().Add(10800 * time.Second)
	} else {
		// otherwise, till is timestamp
		till = time.Unix(int64(timestamp), 0)
	}

	// check if we're in maintenance now
	return time.Now().Before(till), till
}

// CheckInPaths checks if any of the paths is in maintenance and returns the max time if multiple found
func CheckInPaths(paths ...string) (isMaintenance bool, till time.Time) {
	for _, path := range paths {
		if ok, t := IsMaintenance(path); ok && t.After(till) {
			isMaintenance = true
			till = t
		}
	}
	return
}

type Error struct {
	now, till  time.Time
	retryAfter time.Duration
}

func NewError(till time.Time) Error {
	now := time.Now()
	return Error{now: now, till: till, retryAfter: till.Sub(now)}
}

func (e Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"message":     e.Error(),
		"till":        e.till,
		"retry_after": e.RetryAfter().String(),
	})
}

func (e Error) RetryAfter() time.Duration {
	return e.retryAfter
}

func (e Error) Error() string {
	return fmt.Sprintf("on maintenance till %s, retry after %s", e.till, e.till.Sub(e.now))
}
