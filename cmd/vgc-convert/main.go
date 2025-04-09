package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"net/url"
	"os"
	"strings"

	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/vpngen/keydesk/internal/vpn/vgc"
)

// ErrInvalidArgs - invalid arguments.
var ErrInvalidArgs = errors.New("invalid arguments")

func main() {
	wgc := flag.Bool("wg", false, "converted wireguard config")
	awgc := flag.Bool("awg", false, "converted amnezia wireguard config")

	flag.Parse()

	if *wgc && *awgc {
		log.Fatalf("Only one of -wg or -awg can be specified\n")
	}

	if flag.NArg() != 1 {
		log.Fatalf("Usage: <option> <key>\n")
	}

	arg0 := flag.Arg(0)

	res, err := decode(arg0)
	if err != nil {
		log.Fatalf("Decode: %s\n", err)
	}

	switch {
	case *wgc:
		fmt.Fprintf(os.Stderr, "Generate Wireguard config\n")
		out, err := wgPrint(res)
		if err != nil {
			log.Fatalf("Wireguard print: %s\n", err)
		}

		fmt.Fprintf(os.Stdout, "%s\n", out)
	case *awgc:
		fmt.Fprintf(os.Stderr, "Generate Amnezia Wireguard config\n")
		out, err := awgPrint(res)
		if err != nil {
			log.Fatalf("Amnezia Wireguard print: %s\n", err)
		}

		fmt.Fprintf(os.Stdout, "%s\n", out)
	default:
		fmt.Fprintf(os.Stderr, "Generate Raw config\n")
		out, err := rawPrint(res)
		if err != nil {
			log.Fatalf("Raw print: %s\n", err)
		}

		fmt.Fprintf(os.Stdout, "%s\n", out)
	}
}

func decode(key string) (*vgc.Config, error) {
	if key == "" {
		return nil, ErrInvalidArgs
	}

	for {
		// fmt.Fprintf(os.Stderr, "Key: %s\n", key)

		u, err := url.Parse(key)
		if err != nil {
			return nil, fmt.Errorf("parse key: %w", err)
		}

		// fmt.Fprintf(os.Stderr, "URL: %#v\n", u)

		if u.Path == "" {
			key = u.Host

			break
		}

		if u.Scheme != "" {
			key = strings.TrimLeft(u.Path, "/")

			continue
		}

		return nil, fmt.Errorf("invalid key: %s", key)
	}

	buf := base58.Decode(key)

	gz, err := gzip.NewReader(bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("create gzip reader: %w", err)
	}

	defer gz.Close()

	var conf vgc.Config

	if err := json.NewDecoder(gz).Decode(&conf); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	return &conf, nil
}

func rawPrint(c *vgc.Config) (string, error) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "  ")

	if err := enc.Encode(c); err != nil {
		return "", fmt.Errorf("encode config: %w", err)
	}

	return buf.String(), nil
}

func wgPrint(c *vgc.Config) (string, error) {
	tmpl := `[Interface]
Address = %s
PrivateKey = %s
DNS = %s

[Peer]
Endpoint = %s
PublicKey = %s
PresharedKey = %s
AllowedIPs = 0.0.0.0/0,::/0
`

	w := c.Wireguard
	if w == nil {
		return "", fmt.Errorf("wireguard config is nil")
	}

	return fmt.Sprintf(tmpl,
		w.Interface.Address,
		w.Interface.PrivateKey,
		w.Interface.DNS,
		w.Peer.Endpoint,
		w.Peer.PublicKey,
		w.Peer.PresharedKey,
	), nil
}

func awgPrint(c *vgc.Config) (string, error) {
	tmpl := `[Interface]
Address = %s
PrivateKey = %s
DNS = %s

S1 = 0
S2 = 0
Jc = %d
Jmin = 40
Jmax = 70
H1 = 1
H2 = 2
H3 = 3
H4 = 4

[Peer]
Endpoint = %s
PublicKey = %s
PresharedKey = %s
AllowedIPs = 0.0.0.0/0,::/0
`

	w := c.Wireguard
	if w == nil {
		return "", fmt.Errorf("wireguard config is nil")
	}

	count := rand.N(3) + 3

	return fmt.Sprintf(tmpl,
		w.Interface.Address,
		w.Interface.PrivateKey,
		w.Interface.DNS,
		count,
		w.Peer.Endpoint,
		w.Peer.PublicKey,
		w.Peer.PresharedKey,
	), nil
}
