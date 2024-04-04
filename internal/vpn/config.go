package vpn

import (
	"github.com/vpngen/keydesk/internal/vpn/endpoint"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/vpngine/naclkey"
)

type (
	Config interface {
		Protocol() string
		UserConfig(name, host string, port uint16) any
		//AssignToDBUser(user *storage.User, shufflerPub, routerPub [naclkey.NaclBoxKeyLength]byte)
	}

	FileConfig struct {
		Content    string
		FileName   string
		ConfigName string
	}

	Config2 interface {
		Protocol() string
		GetEndpointParams() (map[string]string, error)
		SaveToUser(user *storage.User, router, shuffler [naclkey.NaclBoxKeyLength]byte) error
		GetClientConfig() (any, error)
		ConfigureEndpoint(client endpoint.Client) error
	}
)
