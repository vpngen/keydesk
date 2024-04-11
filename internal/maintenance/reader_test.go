package maintenance

import (
	"bufio"
	"strings"
	"testing"
	"time"
)

var testCases = []struct {
	name string
	data string
	till time.Time
	msg  string
}{
	{
		"empty",
		``,
		time.Time{},
		"",
	},
	{
		"timestamp",
		`1708611601`,
		time.Unix(1708611601, 0),
		"",
	},
	{
		"timestamp and message",
		`1708611601
test message`,
		time.Unix(1708611601, 0),
		"test message",
	},
	{
		"timestamp and multiline message",
		`1708611601
test message
multi line`,
		time.Unix(1708611601, 0),
		`test message
multi line`,
	},
	{
		"empty timestamp and message",
		`
test message`,
		time.Time{},
		"test message",
	},
	{
		"empty timestamp and multiline message",
		`
test message
multi line`,
		time.Time{},
		`test message
multi line`,
	},
	{
		"invalid timestamp and multiline message",
		`lihsdlfkjsbdlfkjsdbhfksdj
test message
multi line`,
		time.Time{},
		`test message
multi line`,
	},
}

func TestReader(t *testing.T) {
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			till, msg, err := readMaintenance(bufio.NewScanner(strings.NewReader(test.data)))
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if till != test.till {
				t.Errorf("unexpected till: %s", till)
			}
			if msg != test.msg {
				t.Errorf("unexpected msg: %s", msg)
			}
		})
	}
}
