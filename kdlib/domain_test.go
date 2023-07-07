package kdlib

import "testing"

func TestIsValidDomainName(t *testing.T) {
	tests := []struct {
		domain string
		want   bool
	}{
		{"example.com", true},
		{"ex@ample.com", false},
		{"_domainkey.example.com", true},
		{"domain-with-hyphen.com", true},
		{"domain..com", false},
		{"toolonglabeltoolonglabeltoolonglabeltoolonglabeltoolonglabel1234.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			if got := IsDomainNameValid(tt.domain); got != tt.want {
				t.Errorf("IsDomainNameValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
