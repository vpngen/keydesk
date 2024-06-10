package vpn

import "testing"

func TestSupportedTypes(t *testing.T) {
	type testCase struct {
		types, supported, unsupported ProtocolSet
		isUnsupported                 bool
	}
	for _, tc := range []testCase{
		{
			types:         TypeIPSec | TypeWG | TypeOVC | TypeOutline,
			supported:     TypeWG,
			unsupported:   TypeIPSec | TypeOVC | TypeOutline,
			isUnsupported: true,
		},
		{
			types:         TypeIPSec | TypeWG | TypeOVC | TypeOutline,
			supported:     0,
			unsupported:   TypeWG | TypeIPSec | TypeOVC | TypeOutline,
			isUnsupported: true,
		},
		{
			types:         0,
			supported:     TypeIPSec | TypeWG | TypeOVC | TypeOutline,
			unsupported:   0,
			isUnsupported: false,
		},
	} {
		_, unsupported := tc.types.GetSupported(tc.supported)
		isUnsupported := unsupported > ProtocolSet(0)
		if tc.unsupported != unsupported {
			t.Errorf("expected %s, got %s", tc.unsupported, unsupported)
		}
		if tc.isUnsupported != isUnsupported {
			t.Errorf("expected %t, got %t", tc.isUnsupported, isUnsupported)
		}
	}
}
