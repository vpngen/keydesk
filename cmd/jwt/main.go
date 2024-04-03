package main

import (
	"flag"
	"fmt"
	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/vpngen/keydesk/pkg/jwt"
	"github.com/vpngen/keydesk/utils"
	"os"
	"strings"
	"time"
)

func errQuit(msg string, err error) {
	_, _ = fmt.Fprintf(os.Stderr, "%s: %s\n", msg, err)
	os.Exit(1)
}

func main() {
	keyFile := flag.String("key", "jwtkey.pem", "key file")
	iss := flag.String("iss", "keydesk", "issuer")
	sub := flag.String("sub", "keydesk", "subject")
	aud := flag.String("aud", "keydesk", "audience, comma separated")
	ttl := flag.String("ttl", "1h", "token ttl, duration string")
	scopes := flag.String("scopes", "", "scopes, comma separated")
	flag.Parse()

	audSlice := strings.Split(*aud, ",")
	scopeSlice := strings.Split(*scopes, ",")
	dur, err := time.ParseDuration(*ttl)
	if err != nil {
		errQuit("parse duration", err)
	}

	file, err := os.Open(*keyFile)
	if err != nil {
		errQuit("open key file", err)
	}

	key, err := utils.ReadECPrivateKey(file)
	if err != nil {
		errQuit("read private key", err)
	}

	issuer := jwt.NewIssuer(key, jwt.Options{
		Issuer:        *iss,
		Subject:       *sub,
		Audience:      audSlice,
		SigningMethod: gojwt.SigningMethodES256,
	})
	token := issuer.CreateToken(dur, scopeSlice...)
	signed, err := issuer.Sign(token)
	if err != nil {
		errQuit("sign token", err)
	}

	fmt.Println(signed)
}
