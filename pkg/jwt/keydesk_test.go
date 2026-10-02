package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestCreateTokenBackdatesNotBefore(t *testing.T) {
	issuer := NewKeydeskTokenIssuer([]byte("secret"), "kid", KeydeskTokenOptions{SigningMethod: jwt.SigningMethodHS256, VipURL: "vip.example"})

	before := time.Now()
	claims := issuer.CreateToken(time.Hour, true)

	if !claims.NotBefore.Time.Before(before.Add(-ClockSkewAllowance + time.Second)) {
		t.Fatalf("nbf not backdated: %s vs now %s", claims.NotBefore.Time, before)
	}

	if !claims.IssuedAt.Time.Equal(claims.NotBefore.Time) {
		t.Fatalf("iat %s differs from nbf %s", claims.IssuedAt.Time, claims.NotBefore.Time)
	}

	if claims.ExpiresAt.Time.Before(before.Add(time.Hour - time.Second)) {
		t.Fatalf("exp shortened: %s", claims.ExpiresAt.Time)
	}

	if issuer.VipURL() != "vip.example" {
		t.Fatalf("vip url %q", issuer.VipURL())
	}
}

func TestCheckTimeLimitsToleratesSkew(t *testing.T) {
	soon := jwt.NewNumericDate(time.Now().Add(ClockSkewAllowance / 2))
	if err := checkTimeLimits(soon, nil); err != nil {
		t.Fatalf("nbf within the allowance rejected: %v", err)
	}

	late := jwt.NewNumericDate(time.Now().Add(ClockSkewAllowance * 2))
	if err := checkTimeLimits(late, nil); err == nil {
		t.Fatal("nbf far in the future accepted")
	}
}
