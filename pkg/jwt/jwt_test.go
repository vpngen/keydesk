package jwt

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vpngen/keydesk/utils"
)

func TestJWT(t *testing.T) {
	type testCase struct {
		name   string
		key    crypto.PrivateKey
		pub    crypto.PublicKey
		method jwt.SigningMethod
	}

	ec256key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	hmacKey := make([]byte, 32)
	if _, err := rand.Read(hmacKey); err != nil {
		t.Fatal(err)
	}

	testCases := []testCase{
		{"ES256", ec256key, ec256key.Public(), jwt.SigningMethodES256},
		{"HS256", hmacKey, hmacKey, jwt.SigningMethodHS256},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			options := MessagesJwtOptions{
				Issuer:        "issuer",
				Subject:       "subject",
				Audience:      []string{"audience"},
				SigningMethod: tc.method,
			}

			issuer := NewMessagesJwtIssuer(tc.key, options)
			claims := issuer.CreateToken(time.Hour, "scope1", "scope2", "scope3")
			token, err := issuer.Sign(claims)
			if err != nil {
				t.Fatal(err)
			}

			authorizer := NewMessagesJwtAuthorizer(tc.pub, options)
			parsedClaims, err := authorizer.Validate(token)
			if err != nil {
				t.Fatal(err)
			}

			err = authorizer.Authorize(parsedClaims, claims.Scopes...)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestECPublicKeyEncode(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	encoded := utils.EncodePublicKey(key.PublicKey)
	t.Log(encoded)
	decoded, err := utils.DecodePublicKey(encoded)
	if err != nil {
		t.Fatal(err)
	}

	if !key.PublicKey.Equal(&decoded) {
		t.Fatal("decoded public key is not equal to original")
	}
}
