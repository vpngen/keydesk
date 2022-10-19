package user

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/netip"
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

// AddUser - create user.
func AddUser(params operations.PostUserParams, principal interface{}) middleware.Responder {
	fullname, person, err := namesgenerator.PhysicsAwardee()
	if err != nil {
		return operations.NewPostUserDefault(500)
	}

	user, wgPriv, err := addUser(fullname, person, false)
	if err != nil {
		return operations.NewPostUserDefault(500)
	}

	wgconf := genWgConf(user, wgPriv)

	rc := io.NopCloser(strings.NewReader(wgconf))

	return operations.NewPostUserCreated().WithContentDisposition(constructContentDisposition(user.Name, user.ID)).WithPayload(rc)
}

// AddBrigadier - create brigadier user.
func AddBrigadier(fullname string, person namesgenerator.Person) (string, error) {
	user, wgPriv, err := addUser(fullname, person, false)
	if err != nil {
		return "", fmt.Errorf("addUser: %w", err)
	}

	wgconf := genWgConf(user, wgPriv)

	return wgconf, nil
}

func addUser(fullname string, person namesgenerator.Person, boss bool) (*UserConfig, []byte, error) {
	user := &UserConfig{
		Name:   fullname,
		Person: person,
		Boss:   boss,
	}

	wgPub, wgPriv, wgRouterPriv, wgShufflerPriv, err := genwgKey(&env.Env.RouterPublicKey, &env.Env.ShufflerPublicKey)
	if err != nil {
		fmt.Printf("wggen: %s", err)

		return nil, nil, fmt.Errorf("wggen: %w", err)
	}

	user.WgPublicKey = wgPub
	user.WgRouterPriv = wgRouterPriv
	user.WgShufflerPriv = wgShufflerPriv

	if err := storage.put(user); err != nil {
		fmt.Printf("put: %s", err)

		return nil, nil, fmt.Errorf("put: %w", err)
	}

	return user, wgPriv, nil
}

func genWgConf(u *UserConfig, wgPriv []byte) string {

	tmpl := `[Interface]
Address = %s
PrivateKey = %s
DNS = %s

[Peer]
Endpoint = %s:51820
PublicKey = %s
AllowedIPs = 0.0.0.0/0
`

	wgconf := fmt.Sprintf(tmpl,
		netip.PrefixFrom(u.IPv4, 32).String()+","+netip.PrefixFrom(u.IPv6, 64).String(),
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPriv),
		u.DNSv4.String()+","+u.DNSv6.String(),
		u.EndpointIPv4.String(),
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(u.EndpointWgPublic),
	)

	return wgconf
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

func genwgKey(ruouterPubkey, shufflerPubkey *[naclkey.NaclBoxKeyLength]byte) ([]byte, []byte, []byte, []byte, error) {
	key, err := wgtypes.GenerateKey()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("gen wg key: %w", err)
	}

	routerKey, err := box.SealAnonymous(nil, key[:], ruouterPubkey, rand.Reader)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("router seal: %w", err)
	}

	shufflerKey, err := box.SealAnonymous(nil, key[:], shufflerPubkey, rand.Reader)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("shuffler seal: %w", err)
	}

	pub := key.PublicKey()

	return pub[:], key[:], routerKey, shufflerKey, nil
}
