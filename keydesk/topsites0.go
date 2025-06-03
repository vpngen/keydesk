// Generaged by cmd/fakesitelist/sites.go
package keydesk

import (
	"crypto/rand"
	"math/big"
)

var topSites0 = []string{
	"google.com",
	"microsoft.com",
}

func GetRandomSites0() []string {
	n := min(5, len(topSites0))
	m := make(map[string]struct{})

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
}
