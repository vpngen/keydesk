package vpn

import (
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/vpngen/keydesk/internal/vpn/amnezia"
	"github.com/vpngen/keydesk/internal/vpn/cloak"
	"github.com/vpngen/keydesk/internal/vpn/endpoint"
	"github.com/vpngen/keydesk/internal/vpn/ipsec"
	"github.com/vpngen/keydesk/internal/vpn/openvpn"
	"github.com/vpngen/keydesk/internal/vpn/outline"
	"github.com/vpngen/keydesk/internal/vpn/ss"
	"github.com/vpngen/keydesk/internal/vpn/vgc"
	"github.com/vpngen/keydesk/internal/vpn/wg"
	"github.com/vpngen/keydesk/kdlib"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/utils"
)

const (
	ProtocolWireguard   = Wireguard
	ProtocolShadowsocks = "shadowsocks"
	ProtocolCloak       = "cloak"
	ProtocolOpenVPN     = "openvpn"
	ProtocolL2TP        = "l2tp"
)

var cfgProtocols = map[string][]string{
	Wireguard: {}, // wireguard is always required
	Universal: {ProtocolCloak, ProtocolShadowsocks},
	Outline:   {ProtocolShadowsocks},
	Amnezia:   {ProtocolCloak, ProtocolOpenVPN},
	IPSec:     {ProtocolL2TP},
}

type Generator struct {
	NaCl   utils.NaCl
	Client endpoint.RealClient
}

type Configs struct {
	WireGuard *FileConfig
	Universal *string
	Outline   *string
	Amnezia   *FileConfig
	IPSec     *ipsec.ClientConfig
}

type Protocols struct {
	WireGuard   wg.RawConfig
	Shadowsocks *ss.Config
	Cloak       *cloak.Config
	OpenVPN     *openvpn.Config
	L2TP        *ipsec.ClientConfig
}

const defaultInternalDNS = "100.126.0.1"

func (g Generator) GenerateConfigs(brigade *storage.Brigade, user *storage.User, configs []string) (Configs, error) {
	log.Println("generating configs:", configs)
	protos2gen := make(utils.StringSet)
	protos2gen.Add(ProtocolWireguard)
	for _, c := range configs {
		if protos, ok := cfgProtocols[c]; !ok {
			return Configs{}, fmt.Errorf("unsupported config %q", c)
		} else {
			protos2gen.Add(protos...)
		}
	}

	log.Println("protocols to generate:", protos2gen.Slice())

	supported := brigade.GetSupportedVPNProtocols()
	log.Println("supported protocols:", supported)

	for _, p := range protos2gen.Slice() {
		if !slices.Contains(supported, p) {
			return Configs{}, fmt.Errorf("unsupported VPN protocol %q", p)
		}
	}

	epData := make(map[string]string)

	protocolsObj := Protocols{}
	for p := range protos2gen {
		switch p {
		case ProtocolWireguard:
			cfg, err := wg.Generate(brigade, user, g.NaCl, epData)
			if err != nil {
				return Configs{}, fmt.Errorf("generate %q: %w", p, err)
			}
			protocolsObj.WireGuard = cfg

		case ProtocolShadowsocks:
			cfg, err := ss.Generate(brigade, user, g.NaCl, epData)
			if err != nil {
				return Configs{}, fmt.Errorf("generate %q: %w", p, err)
			}
			protocolsObj.Shadowsocks = &cfg

		case ProtocolCloak:
			cfg, err := cloak.Generate(brigade, user, g.NaCl, epData)
			if err != nil {
				return Configs{}, fmt.Errorf("generate %q: %w", p, err)
			}
			protocolsObj.Cloak = &cfg

		case ProtocolOpenVPN:
			cfg, err := openvpn.Generate(brigade, user, g.NaCl, epData)
			if err != nil {
				return Configs{}, fmt.Errorf("generate %q: %w", p, err)
			}
			protocolsObj.OpenVPN = &cfg

		case ProtocolL2TP:
			cfg, err := ipsec.Generate(brigade, user, g.NaCl, epData)
			if err != nil {
				return Configs{}, fmt.Errorf("generate %q: %w", p, err)
			}
			protocolsObj.L2TP = &cfg

		default:
			return Configs{}, fmt.Errorf("unsupported protocol %q", p)
		}
	}

	log.Println("protocols generated:", protocolsObj)

	//epPub, err := wgtypes.NewKey(brigade.WgPublicKey)
	//if err != nil {
	//	return Configs{}, fmt.Errorf("endpoint pub: %w", err)
	//}

	resp, err := g.Client.PeerAdd(protocolsObj.WireGuard.Key.PublicKey(), epData)
	if err != nil {
		return Configs{}, fmt.Errorf("peer add: %w", err)
	}

	fmt.Fprintf(os.Stderr, "User %s (%s) added\n", user.UserID, protocolsObj.WireGuard.Key.PublicKey())

	if protocolsObj.OpenVPN != nil {
		protocolsObj.OpenVPN.Cert = resp.OpenvpnClientCertificate
	}

	ret := Configs{}

	for _, config := range configs {
		switch config {
		case Wireguard:
			raw := protocolsObj.WireGuard.GetVGC()
			native, err := raw.GetNative()
			if err != nil {
				return Configs{}, fmt.Errorf("wireguard native config: %w", err)
			}

			name := kdlib.AssembleWgStyleTunName(user.Name)
			ret.WireGuard = &FileConfig{
				Content:    string(native),
				FileName:   name + ".conf",
				ConfigName: name,
			}

		case Universal:
			sscfg := protocolsObj.Shadowsocks
			ck, err := protocolsObj.Cloak.GetVGC(cloak.ProxyBook{
				Shadowsocks: ss.NewSSProxyBook(sscfg.Cipher, sscfg.Password),
			})
			if err != nil {
				return Configs{}, fmt.Errorf("get cloak config: %w", err)
			}
			cfg := vgc.NewV1(user.Name, user.EndpointDomain, protocolsObj.WireGuard.GetVGC(), ck, *sscfg, 0)
			enc, err := cfg.Encode()
			if err != nil {
				return Configs{}, fmt.Errorf("encode: %w", err)
			}
			ret.Universal = &enc

		case Outline:
			cfg, err := outline.NewFromSS(user.Name, *protocolsObj.Shadowsocks)
			if err != nil {
				return Configs{}, fmt.Errorf("outline: %w", err)
			}
			ret.Outline = &cfg

		case Amnezia:
			amnz := amnezia.NewConfig(storage.GetEndpointHost(brigade, user), user.Name, defaultInternalDNS, defaultInternalDNS)
			container, err := amnezia.NewOVCContainer(*protocolsObj.Cloak, *protocolsObj.OpenVPN)
			if err != nil {
				return Configs{}, fmt.Errorf("amnezia new container: %w", err)
			}

			amnz.AddContainer(container)
			amnz.SetDefaultContainer(amnezia.ContainerOpenVPNCloak)

			amnzConf, err := amnz.Marshal()
			if err != nil {
				return Configs{}, fmt.Errorf("amnezia marshal: %w", err)
			}

			name := kdlib.AssembleWgStyleTunName(user.Name)
			ret.Amnezia = &FileConfig{
				Content:    amnzConf,
				FileName:   name + ".ovc",
				ConfigName: name,
			}

		case IPSec:
			ret.IPSec = protocolsObj.L2TP

		default:
			return Configs{}, fmt.Errorf("unsupported config %q", config)
		}
	}

	return ret, nil
}
