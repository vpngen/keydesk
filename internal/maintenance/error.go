package maintenance

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// IsMaintenance Reads .maintenance file. If file contains timestamp, till is timestamp.
// If file is empty or timestamp is invalid or zero, till is file modtime + 10800 sec.
// Returns (true, till) if till is later than now.
func IsMaintenance(path string) (bool, time.Time, string) {
	file, err := os.Open(path)
	if err != nil {
		// can't open file, it's not maintenance
		_, _ = fmt.Fprintln(os.Stderr, "read maintenance file error:", err)
		return false, time.Time{}, ""
	}
	defer file.Close()

	till, msg, err := readMaintenance(bufio.NewScanner(file))
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "read maintenance error:", err)
		return false, time.Time{}, ""
	}

	if till.IsZero() {
		stat, err := file.Stat()
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "read maintenance file stat error:", err)
			return false, time.Time{}, ""
		}
		till = stat.ModTime().Add(10800 * time.Second)
	}

	return till.After(time.Now()), till, msg
}

// CheckInPaths checks if any of the paths is in maintenance and returns the max time if multiple found
func CheckInPaths(paths ...string) (isMaintenance bool, till time.Time, msg string) {
	for _, path := range paths {
		if ok, t, m := IsMaintenance(path); ok && t.After(till) {
			isMaintenance = true
			till = t
			msg = m
		}
	}
	return
}

type Error struct {
	now, till  time.Time
	retryAfter time.Duration
	msg        string
}

func NewError(till time.Time, msg string) Error {
	now := time.Now()
	return Error{now: now, till: till, retryAfter: till.Sub(now), msg: msg}
}

func (e Error) MarshalJSON() ([]byte, error) {
	data := map[string]any{
		"till":        e.till,
		"retry_after": e.RetryAfter().String(),
	}
	if e.msg != "" {
		data["msg"] = e.msg
	}
	return json.Marshal(data)
}

func (e Error) RetryAfter() time.Duration {
	return e.retryAfter
}

func (e Error) Error() string {
	return fmt.Sprintf("on maintenance till %s, retry after %s", e.till, e.till.Sub(e.now))
}
