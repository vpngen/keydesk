package amnezia

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/vpngen/keydesk/internal/vpn/cloak"
	"github.com/vpngen/keydesk/internal/vpn/openvpn"
)

func NewOVCContainer(cloakCfg cloak.Config, ovpnCfg openvpn.Config) (Container, error) {
	cloakJSON := new(bytes.Buffer)
	if err := json.NewEncoder(cloakJSON).Encode(cloakCfg); err != nil {
		return Container{}, fmt.Errorf("marshal cloak config: %w", err)
	}

	ovpnCfgStr, err := ovpnCfg.Render()
	if err != nil {
		return Container{}, fmt.Errorf("render openvpn config: %w", err)
	}
	ovpnJSON := new(bytes.Buffer)
	enc := json.NewEncoder(ovpnJSON)
	enc.SetEscapeHTML(false)

	if err = enc.Encode(ConfigInnerJson{
		Config: ovpnCfgStr.String(),
	}); err != nil {
		return Container{}, fmt.Errorf("marshal openvpn config: %w", err)
	}

	return Container{
		Container: ContainerOpenVPNCloak,
		Cloak: &CloakConfig{
			LastConfig: cloakJSON.String(),
			Port:       CloakPort,
			Transport:  CloakTransport,
		},
		OpenVPN:            &OpenVPNConfig{LastConfig: ovpnJSON.String()},
		ShadowSocks:        &ShadowSocksConfig{LastConfig: "{}"},
		Wireguard:          nil,
		IsThirdPartyConfig: false,
	}, nil
}
