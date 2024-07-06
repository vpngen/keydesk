package user

import (
	"fmt"
	"github.com/vpngen/keydesk/internal/vpn"
	"github.com/vpngen/keydesk/internal/vpn/endpoint"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/keydesk/utils"
	"github.com/vpngen/vpngine/naclkey"
	"log"
)

type Service struct {
	db        *storage.BrigadeStorage
	epClient  endpoint.RealClient
	generator vpn.Generator
}

func New(db *storage.BrigadeStorage, routerPub, shufflerPub [naclkey.NaclBoxKeyLength]byte, logger *log.Logger) (Service, error) {
	// TODO: mock client
	//var epClient endpoint.Client
	//if db.GetActualAddrPort().IsValid() {
	//	epClient = endpoint.NewClient(db.GetActualAddrPort(), nil)
	//} else {
	//	epClient = endpoint.MockClient{
	//		RealClient: endpoint.NewClient(db.GetCalculatedAddrPort(), nil),
	//	}
	//}
	if !db.GetActualAddrPort().IsValid() {
		return Service{}, fmt.Errorf("invalid address: %s", db.GetActualAddrPort())
	}
	client := endpoint.NewClient(db.GetActualAddrPort(), logger)
	return Service{
		db:       db,
		epClient: client,
		generator: vpn.Generator{
			NaCl: utils.NaCl{
				Router:   routerPub,
				Shuffler: shufflerPub,
			},
			Client: client,
		},
	}, nil
}
