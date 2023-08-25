package main

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/vpngen/keydesk/kdlib"
)

//go:embed sites.lst
var sitesLst string

const (
	beginFile = `// Generaged by cmd/fakesitelist/main.go
package keydesk

import (
	"crypto/rand"
	"math/big"
)

var topSites= []string{`
	endFile = `
}

func GetRandomSite() string {
	x, err := rand.Int(rand.Reader, big.NewInt(int64(len(topSites))))
	if err != nil {
		panic(err)
	}

	return topSites[x.Int64()]
}`
)

func main() {
	fmt.Println(beginFile)

	for _, line := range strings.Split(sitesLst, "\n") {
		line = strings.Trim(line, " \t")
		if line == "" || !kdlib.IsDomainNameValid(line) {
			continue
		}

		fmt.Printf("\t\"%s\",\n", line)
	}

	fmt.Println(endFile)
}
