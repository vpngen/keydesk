package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/vpngen/keydesk/pkg/jwt"
	"github.com/vpngen/keydesk/utils"
	"golang.org/x/crypto/ssh"
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

	var key crypto.PrivateKey

	key, err = utils.ReadECPrivateKey(file)
	if err != nil {
		buf, err := os.ReadFile(*keyFile)
		if err != nil {
			log.Fatal("read key file:", err)
		}

		key, err = ssh.ParseRawPrivateKey(buf)
		if err != nil {
			log.Fatal("read private key:", err)
		}
	}

	var jwtMethod gojwt.SigningMethod

	switch key.(type) {
	case *ecdsa.PrivateKey:
		jwtMethod = gojwt.SigningMethodES256
	case *rsa.PrivateKey:
		jwtMethod = gojwt.SigningMethodRS256
	case *ed25519.PrivateKey:
		jwtMethod = gojwt.SigningMethodEdDSA
	default:
		log.Fatal("unsupported key type")
	}

	issuer := jwt.NewMessagesJwtIssuer(key, jwt.MessagesJwtOptions{
		Issuer:        *iss,
		Audience:      strings.Split(*aud, ","),
		Subject:       *sub,
		SigningMethod: jwtMethod,
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
