package main

import (
	"flag"
	"fmt"
	jwt2 "github.com/golang-jwt/jwt/v5"
	"github.com/vpngen/keydesk/pkg/jwt"
	"github.com/vpngen/keydesk/utils"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	keyFile := flag.String("k", "key.pem", "private key file")
	iss := flag.String("iss", "dc-mgmt", "issuer")
	sub := flag.String("sub", "", "subject")
	ttl := flag.String("ttl", "", "ttl, duration string")
	aud := flag.String("aud", "keydesk", "audience, comma separated")
	scopes := flag.String("s", "", "scopes, comma separated")
	flag.Parse()

	log.Default().SetFlags(log.Lshortfile | log.LstdFlags | log.Lmsgprefix)

	file, err := os.Open(*keyFile)
	if err != nil {
		log.Fatal(err)
	}

	ecPub, err := utils.ReadECPrivateKey(file)
	if err != nil {
		log.Fatal(err)
	}

	issuer := jwt.NewIssuer(ecPub, jwt.Options{
		Issuer:        *iss,
		Audience:      strings.Split(*aud, ","),
		Subject:       *sub,
		SigningMethod: jwt2.SigningMethodES256,
	})

	ttlD, err := time.ParseDuration(*ttl)
	if err != nil {
		log.Fatal(err)
	}

	claims := issuer.CreateToken(ttlD, strings.Split(*scopes, ",")...)
	token, err := issuer.Sign(claims)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(token)
}
