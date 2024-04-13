package openvpn

import (
	"bytes"
	"fmt"
	"text/template"
)

const openVPNConfigTemplate = `client
dev tun
proto tcp
resolv-retry infinite
nobind
persist-key
persist-tun
cipher AES-256-GCM
auth SHA512
verb 3
tls-client
tls-version-min 1.2
key-direction 1
remote-cert-tls server
redirect-gateway def1 bypass-dhcp

dhcp-option DNS {{ .DNS }}
block-outside-dns

route {{ .IP }} 255.255.255.255 net_gateway
remote 127.0.0.1 1194

<ca>
{{ .CA }}
</ca>
<cert>
{{ .Cert }}
</cert>
<key>
{{ .Key }}
</key>`

var tmpl = template.Must(template.New("openvpn").Parse(openVPNConfigTemplate))

type Config struct {
	DNS, IP, CA, Cert, Key string
}

func (c Config) Render() (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, c); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	return buf, nil
}
