package main

import (
	_ "embed"
	"flag"
	"fmt"
	"strings"

	"github.com/vpngen/keydesk/kdlib"
)

//go:embed sites.lst
var sitesLst string

//go:embed sites0.lst
var sitesLst0 string

const (
	beginFile = `// Generaged by cmd/fakesitelist/sites.go
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
	beginFile0 = `// Generaged by cmd/fakesitelist/sites.go
package keydesk

import (
	"crypto/rand"
	"math/big"
)

var topSites0= []string{`
	endFile0 = `
}

func GetRandomSites0() []string {
	m := make(map[string]struct{}, len(topSites0))

	n := 5
	if len(topSites0) < n {
		n = len(topSites0)
	}

	for len(m) < n {
		x, err := rand.Int(rand.Reader, big.NewInt(int64(len(topSites0))))
		if err != nil {
			panic(err)
		}

		if _, ok := m[topSites0[x.Int64()]]; !ok {
			m[topSites0[x.Int64()]] = struct{}{}
		}
	}

	list := make([]string, 0, len(m))
	for k := range m {
		list = append(list, k)
	}

	return list
}`
)

func main() {
	var l string

	zero := flag.Bool("0", false, "use sites0.lst instead of sites.lst")
	flag.Parse()

	switch *zero {
	case true:
		l = sitesLst0
		fmt.Println(beginFile0)
	default:
		l = sitesLst
		fmt.Println(beginFile)
	}

	for line := range strings.SplitSeq(l, "\n") {
		line = strings.Trim(line, " \t")
		if line == "" || !kdlib.IsDomainNameValid(line) {
			continue
		}

		fmt.Printf("\t\"%s\",\n", line)
	}

	switch *zero {
	case true:
		fmt.Println(endFile0)
	default:
		fmt.Println(endFile)
	}
}
