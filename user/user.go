package user

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/vpngen/keykeeper/env"
	"github.com/vpngen/keykeeper/gen/models"
	"github.com/vpngen/keykeeper/gen/restapi/operations"
	"github.com/vpngen/vpngine/naclkey"
	"github.com/vpngen/wordsgens/namesgenerator"
	"golang.org/x/crypto/nacl/box"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
)

// MaxUsers - maximem limit.
const MaxUsers = 500

// AddUser - creaste user.
func AddUser(params operations.PostUserParams, principal interface{}) middleware.Responder {
	user, err := newUser(false)
	if err != nil {
		return operations.NewPostUserDefault(500)
	}

	rc := io.NopCloser(strings.NewReader(`[Interface]
	# Name = laptop.example-vpn.dev
	Address = 10.0.44.4/32
	PrivateKey = OPmibSXYAAcMIYKNsWqr77zY06Kl750AEB1nWQi1T2o=
	DNS = 1.1.1.1
	
	[Peer]
	# Name = public-server1.example-vpn.tld
	Endpoint = public-server1.example-vpn.tld:51820
	PublicKey = q/+jwmL5tNuYSB3z+t9Caj00Pc1YQ8zf+uNPu/UE1wE=
	# routes traffic to itself and entire subnet of peers as bounce server
	AllowedIPs = 10.0.44.1/24
	PersistentKeepalive = 25
	`))

	return operations.NewPostUserCreated().WithContentDisposition(constructContentDisposition(user.Name, user.ID)).WithPayload(rc)
}

func addUser(fullname string, person namesgenerator.Person, boss bool) (*UserConfig, error) {
	user := &UserConfig{
		Name:                    fullname,
		Person:                  person,
		MonthlyQuotaRemainingGB: MonthlyQuotaRemainingGB,
		Boss:                    boss,
	}

	wgPub, wgRouterPriv, wgShufflerPriv, err := genwgKey(&env.Env.RouterPublicKey, &env.Env.ShufflerPublicKey)
	if err != nil {
		return nil, fmt.Errorf("wggen: %w", err)
	}

	user.WgPublicKey = wgPub
	user.WgRouterPriv = wgRouterPriv
	user.WgShufflerPriv = wgShufflerPriv

	if err := storage.put(user); err != nil {
		return nil, fmt.Errorf("put: %w", err)
	}

	return user, nil
}

func constructContentDisposition(name, id string) string {
	return fmt.Sprintf("attachment; filename=%q; filename*=%q", "wg-"+id+".conf", "utf-8''"+url.QueryEscape(name+".conf"))
}

// DelUserUserID - creaste user.
func DelUserUserID(params operations.DeleteUserUserIDParams, principal interface{}) middleware.Responder {
	if storage.delete(params.UserID) {
		return operations.NewDeleteUserUserIDNoContent()
	}

	return operations.NewDeleteUserUserIDForbidden()
}

// GetUsers - .
func GetUsers(params operations.GetUserParams, principal interface{}) middleware.Responder {
	_users := storage.list()

	users := make([]*models.User, len(_users))
	for i := range _users {
		u := _users[i]
		users[i] = &models.User{
			UserID:                  &u.ID,
			UserName:                &u.Name,
			ThrottlingTill:          strfmt.DateTime(u.ThrottlingTill),
			MonthlyQuotaRemainingGB: u.MonthlyQuotaRemainingGB,
			LastVisitHour:           strfmt.DateTime(u.LastVisitHour),
			LastVisitSubnet:         u.LastVisitSubnet,
			LastVisitASCountry:      u.LastVisitASCountry,
			LastVisitASName:         u.LastVisitASName,
			PersonName:              u.Person.Name,
			PersonDesc:              u.Person.Desc,
			PersonDescLink:          u.Person.URL,
		}
		copy(users[i].Problems, u.Problems)
	}

	return operations.NewGetUserOK().WithPayload(users)
}

func genwgKey(ruouterPubkey, shufflerPubkey *[naclkey.NaclBoxKeyLength]byte) ([]byte, []byte, []byte, error) {
	key, err := wgtypes.GenerateKey()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gen wg key: %w", err)
	}

	routerKey, err := box.SealAnonymous(nil, key[:], ruouterPubkey, rand.Reader)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("router seal: %w", err)
	}

	shufflerKey, err := box.SealAnonymous(nil, key[:], shufflerPubkey, rand.Reader)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("shuffler seal: %w", err)
	}

	pub := key.PublicKey()

	return pub[:], routerKey, shufflerKey, nil
}
