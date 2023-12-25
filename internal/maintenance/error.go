package maintenance

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func IsMaintenance(path string) (bool, time.Time) {
	data, err := os.ReadFile(path)
	if err != nil {
		//_, _ = fmt.Fprintln(os.Stderr, "read maintenance file error:", err)
		return false, time.Time{}
	}
	timestamp, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "read maintenance timestamp error:", err)
		return false, time.Time{}
	}
	till := time.Unix(int64(timestamp), 0)
	return time.Now().Before(till), till
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
