package filter

import (
	"testing"
)

func TestOrdered(t *testing.T) {
	testCases := []struct {
		input int
		op    string
		v     int
		want  bool
	}{
		{10, "gt", 10, false},
		{100, "gt", 100, false},
		{11, "gt", 10, true},
		{10, "lt", 10, false},
		{9, "lt", 10, true},
		{10, "eq", 10, true},
		{11, "eq", 10, false},
		{10, "ne", 10, false},
		{11, "ne", 10, true},
		{10, "ge", 10, true},
		{9, "ge", 10, false},
		{11, "ge", 10, true},
		{10, "le", 10, true},
		{9, "le", 10, true},
		{11, "le", 10, false},
	}

	for _, tc := range testCases {
		got := Ordered(tc.op, tc.v)(tc.input)
		if got != tc.want {
			t.Errorf("%d %s %d, expected %t, got %t", tc.input, tc.op, tc.v, tc.want, got)
		}
	}
}
