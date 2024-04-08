package user

import (
	"github.com/vpngen/keydesk/internal/vpn/endpoint"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/vpngine/naclkey"
)

type Service struct {
	db                     *storage.BrigadeStorage
	epClient               endpoint.Client
	routerPub, shufflerPub [naclkey.NaclBoxKeyLength]byte
}

func New(db *storage.BrigadeStorage) Service {
	var epClient endpoint.Client
	if db.GetActualAddrPort().IsValid() {
		epClient = endpoint.NewClient(db.GetActualAddrPort())
	} else {
		epClient = endpoint.MockClient{
			RealClient: endpoint.NewClient(db.GetCalculatedAddrPort()),
		}
	}
	return Service{
		db:       db,
		epClient: epClient,
	}
}
