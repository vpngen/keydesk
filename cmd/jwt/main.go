package main

import (
	"flag"
	"fmt"
	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/vpngen/keydesk/pkg/jwt"
	"github.com/vpngen/keydesk/utils"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	keyFile := flag.String("key", "key.pem", "private key file")
	iss := flag.String("iss", "", "issuer")
	sub := flag.String("sub", "", "subject")
	aud := flag.String("aud", "", "audience, comma separated")
	ttl := flag.String("ttl", "", "token ttl, duration string")
	scopes := flag.String("scopes", "", "scopes, comma separated")
	flag.Parse()

	log.Default().SetFlags(log.Lshortfile)
	log.Default().SetOutput(os.Stderr)
	log.Default().SetPrefix("[jwt]\t")

	file, err := os.Open(*keyFile)
	if err != nil {
		log.Fatal("read key file:", err)
	}

	key, err := utils.ReadECPrivateKey(file)
	if err != nil {
		log.Fatal("read private key:", err)
	}

	issuer := jwt.NewIssuer(key, jwt.Options{
		Issuer:        *iss,
		Audience:      strings.Split(*aud, ","),
		Subject:       *sub,
		SigningMethod: gojwt.SigningMethodES256,
	})

	ttlD, err := time.ParseDuration(*ttl)
	if err != nil {
		log.Fatal("parse ttl:", err)
	}

	claims := issuer.CreateToken(ttlD, strings.Split(*scopes, ",")...)
	token, err := issuer.Sign(claims)
	if err != nil {
		log.Fatal("sign token:", err)
	}

	fmt.Println(token)
}
