package user

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"net/url"
	"os"
	"strings"

	"github.com/vpngen/keydesk/env"
	"github.com/vpngen/keydesk/gen/models"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	"github.com/vpngen/vpngine/naclkey"
	"github.com/vpngen/wordsgens/namesgenerator"
	"golang.org/x/crypto/nacl/box"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
)

// MaxUsers - maximem limit.
const MaxUsers = 250

// AddUser - create user.
func AddUser(params operations.PostUserParams, principal interface{}) middleware.Responder {
	var (
		user          *UserConfig
		wgPriv, wgPSK []byte
	)

	for {
		fullname, person, err := namesgenerator.PeaceAwardeeShort()
		if err != nil {
			return operations.NewPostUserDefault(500)
		}

		user, wgPriv, wgPSK, err = addUser(fullname, person, false)
		if err != nil {
			if errors.Is(err, ErrUserCollision) {
				continue
			}

			return operations.NewPostUserDefault(500)
		}

		break
	}

	wgconf := genWgConf(user, wgPriv, wgPSK)

	rc := io.NopCloser(strings.NewReader(wgconf))

	return operations.NewPostUserCreated().WithContentDisposition(constructContentDisposition(user.Name, user.ID)).WithPayload(rc)
}

// AddBrigadier - create brigadier user.
func AddBrigadier(fullname string, person namesgenerator.Person) (string, string, error) {
	user, wgPriv, wgPSK, err := addUser(fullname, person, true)
	if err != nil {
		return "", "", fmt.Errorf("addUser: %w", err)
	}

	wgconf := genWgConf(user, wgPriv, wgPSK)

	return wgconf, SanitizeFilename(user.Name), nil
}

func addUser(fullname string, person namesgenerator.Person, boss bool) (*UserConfig, []byte, []byte, error) {
	user := &UserConfig{
		Name:   fullname,
		Person: person,
		Boss:   boss,
	}

	wgPub, wgPriv, wgPSK, wgRouterPSK, wgShufflerPSK, err := genwgKey(&env.Env.RouterPublicKey, &env.Env.ShufflerPublicKey)
	if err != nil {
		fmt.Printf("wggen: %s", err)

		return nil, nil, nil, fmt.Errorf("wggen: %w", err)
	}

	user.WgPublicKey = wgPub
	user.WgRouterPSK = wgRouterPSK
	user.WgShufflerPSK = wgShufflerPSK

	if err := storage.put(user); err != nil {
		fmt.Printf("put: %s", err)

		return nil, nil, nil, fmt.Errorf("put: %w", err)
	}

	return user, wgPriv, wgPSK, nil
}

func genWgConf(u *UserConfig, wgPriv, wgPSK []byte) string {

	tmpl := `[Interface]
Address = %s
PrivateKey = %s
DNS = %s

[Peer]
Endpoint = %s:51820
PublicKey = %s
PresharedKey = %s
AllowedIPs = 0.0.0.0/0,::/0
`

	wgconf := fmt.Sprintf(tmpl,
		netip.PrefixFrom(u.IPv4, 32).String()+","+netip.PrefixFrom(u.IPv6, 128).String(),
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPriv),
		u.DNSv4.String()+","+u.DNSv6.String(),
		u.EndpointIPv4.String(),
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(u.EndpointWgPublic),
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(wgPSK),
	)

	return wgconf
}

func constructContentDisposition(name, id string) string {
	filename := SanitizeFilename(name)

	return fmt.Sprintf("attachment; filename=%s; filename*=%s", "wg-"+id+".conf", "utf-8''"+url.QueryEscape(filename))
}

// DelUserUserID - creaste user.
func DelUserUserID(params operations.DeleteUserUserIDParams, principal interface{}) middleware.Responder {
	err := storage.delete(params.UserID, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "delete user: %s :%s\n", params.UserID, err)

		return operations.NewDeleteUserUserIDForbidden()
	}

	return operations.NewDeleteUserUserIDNoContent()
}

// GetUsers - .
func GetUsers(params operations.GetUserParams, principal interface{}) middleware.Responder {
	_users, err := storage.list()
	if err != nil {
		fmt.Printf("list: %s\n", err)

		return operations.NewGetUserDefault(500)
	}

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

func genwgKey(ruouterPubkey, shufflerPubkey *[naclkey.NaclBoxKeyLength]byte) ([]byte, []byte, []byte, []byte, []byte, error) {
	key, err := wgtypes.GenerateKey()
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("gen wg psk: %w", err)
	}

	routerKey, err := box.SealAnonymous(nil, key[:], ruouterPubkey, rand.Reader)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("router seal: %w", err)
	}

	shufflerKey, err := box.SealAnonymous(nil, key[:], shufflerPubkey, rand.Reader)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("shuffler seal: %w", err)
	}

	priv, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("gen wg psk: %w", err)
	}

	pub := priv.PublicKey()

	return pub[:], priv[:], key[:], routerKey, shufflerKey, nil
}
