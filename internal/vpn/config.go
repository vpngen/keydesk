package vpn

import (
	"github.com/vpngen/keydesk/internal/vpn/endpoint"
	"github.com/vpngen/keydesk/keydesk/storage"
)

type Config interface {
	// Protocol returns the name of the protocol
	Protocol() string

	// GetEndpointParams returns the parameters to be passed to the endpoint API
	GetEndpointParams() (map[string]string, error)

	// Store saves encrypted config to brigade storage
	Store(user *storage.User) error

	// GetClientConfig returns the config for client connection
	GetClientConfig(data endpoint.APIResponse) (any, error)
}
