package main

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/vpngen/wordsgens/namesgenerator"
)

func usage() string {
	return "Usage: " + os.Args[0] + " [id] [name] [person] [desc] [url] [ep4] [int4] [dns4] [int6] [dns6] [kd6]"
}

func main() {
	id := uuid.New()
	brigadierID := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(id[:])

	fullname, person, err := namesgenerator.PhysicsAwardeeShort()
	if err != nil {
		log.Fatalf("Can't generate: %s\n", err)
	}

	brigadierName := base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString([]byte(fullname))
	personName := base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString([]byte(person.Name))
	personDesc := base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString([]byte(person.Desc))
	personURL := base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString([]byte(person.URL))

	ep4 := getRandomIPv4Network(32).IP

	int4 := net.IPNet{
		IP:   net.IPv4(100, 64, 0, 0),
		Mask: net.CIDRMask(24, 32),
	}

	dns4 := int4.IP.To4()
	dns4[net.IPv4len-1] = 53

	int6 := getRandomIPv6Network(64)
	int6.IP[0] = 0xfd

	dns6 := int6.IP.Mask(int6.Mask)
	dns6[net.IPv6len-1] = 53

	kd6 := getRandomIPv6Network(128).IP
	kd6[0] = 0xfd

	data := map[string]string{
		"id":     brigadierID,
		"name":   brigadierName,
		"person": personName,
		"desc":   personDesc,
		"url":    personURL,
		"ep4":    ep4.String(),
		"int4":   int4.String(),
		"dns4":   dns4.String(),
		"int6":   int6.String(),
		"dns6":   dns6.String(),
		"kd6":    kd6.String(),
	}

	var flags []string

	if len(os.Args) == 1 {
		for k, v := range data {
			flags = append(flags, makeFlag(k, v))
		}
	} else {
		for _, arg := range os.Args[1:] {
			v, ok := data[arg]
			if !ok {
				log.Fatal("unknown flag:", arg, "\n", usage())
			}
			flags = append(flags, makeFlag(arg, v))
		}
	}

	if _, err := os.Stdout.WriteString(strings.Join(flags, " ")); err != nil {
		log.Fatal(err)
	}
}

func makeFlag(flag, value string) string {
	return fmt.Sprintf("-%s %s", flag, value)
}

func getRandomIPNet(bytes, bits int) net.IPNet {
	b := make([]byte, bytes)
	_, err := rand.Reader.Read(b)
	if err != nil {
		panic(err)
	}
	mask := net.CIDRMask(bits, bytes*8)
	return net.IPNet{
		IP:   net.IP(b).Mask(mask),
		Mask: mask,
	}
}

func getRandomIPv6Network(mask int) net.IPNet {
	return getRandomIPNet(net.IPv6len, mask)
}

func getRandomIPv4Network(mask int) net.IPNet {
	return getRandomIPNet(net.IPv4len, mask)
}
