package keydesk

import "github.com/vpngen/keydesk/gen/models"

type ConfigsImplemented struct {
	Wg    []string
	Ovc   []string
	IPSec []string
}

type Answer struct {
	Code    int            `json:"code"`
	Status  string         `json:"status"`
	Desc    string         `json:"desc"`
	Message string         `json:"message"`
	Configs models.Newuser `json:"configs"`
}
