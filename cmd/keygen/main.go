package main

import (
	"flag"
	"fmt"
	"github.com/vpngen/keydesk/utils"
	"os"
)

func main() {
	priv := flag.String("key", "key.pem", "private key file name")
	pub := flag.String("pub", "pub.pem", "public key file name")
	flag.Parse()

	key, err := utils.GenEC256Key()
	if err != nil {
		fmt.Println("gen key:", err)
		os.Exit(1)
	}

	file, err := os.OpenFile(*priv, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		fmt.Println("open private key file:", err)
		os.Exit(1)
	}

	if err = utils.WriteECPrivateKey(key, file); err != nil {
		fmt.Println("write key:", err)
		os.Exit(1)
	}

	if err = file.Close(); err != nil {
		fmt.Println("close private key file:", err)
		os.Exit(1)
	}

	file, err = os.OpenFile(*pub, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("open public key file:", err)
		os.Exit(1)
	}

	if err = utils.WriteECPublicKey(&key.PublicKey, file); err != nil {
		fmt.Println("write public key:", err)
		os.Exit(1)
	}

	if err = file.Close(); err != nil {
		fmt.Println("close public key file:", err)
		os.Exit(1)
	}
}
