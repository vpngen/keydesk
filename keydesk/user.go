package keydesk

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

	"github.com/vpngen/keydesk/gen/models"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	"github.com/vpngen/keydesk/kdlib"
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
func AddUser(db BrigadeStorage, params operations.PostUserParams, principal interface{}, routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) middleware.Responder {
	var (
		user          *UserConfig
		wgPriv, wgPSK []byte
	)

	for {
		fullname, person, err := namesgenerator.PeaceAwardeeShort()
		if err != nil {
			return operations.NewPostUserDefault(500)
		}

		user, wgPriv, wgPSK, err = addUser(db, fullname, person, false, routerPublicKey, shufflerPublicKey)
		if err != nil {
			if errors.Is(err, ErrUserCollision) {
				continue
			}

			fmt.Fprintf(os.Stderr, "Add error: %s\n", err)

			return operations.NewPostUserDefault(500)
		}

		break
	}

	wgconf := genWgConf(user, wgPriv, wgPSK)

	rc := io.NopCloser(strings.NewReader(wgconf))

	return operations.NewPostUserCreated().WithContentDisposition(constructContentDisposition(user.Name, user.ID.String())).WithPayload(rc)
}

// AddBrigadier - create brigadier user.
func AddBrigadier(db BrigadeStorage, fullname string, person namesgenerator.Person, routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) (string, string, error) {
	userconf, wgPriv, wgPSK, err := addUser(db, fullname, person, true, routerPublicKey, shufflerPublicKey)
	if err != nil {
		return "", "", fmt.Errorf("addUser: %w", err)
	}

	wgconf := genWgConf(userconf, wgPriv, wgPSK)

	return wgconf, kdlib.SanitizeFilename(userconf.Name), nil
}

func addUser(db BrigadeStorage, fullname string, person namesgenerator.Person, IsBrigadier bool, routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) (*UserConfig, []byte, []byte, error) {
	wgPub, wgPriv, wgPSK, wgRouterPSK, wgShufflerPSK, err := genwgKey(routerPublicKey, shufflerPublicKey)
	if err != nil {
		fmt.Printf("wg gen: %s", err)

		return nil, nil, nil, fmt.Errorf("wg gen: %w", err)
	}

	userconf, err := db.userPut(fullname, person, IsBrigadier, wgPub, wgRouterPSK, wgShufflerPSK)
	if err != nil {
		fmt.Printf("put: %s", err)

		return nil, nil, nil, fmt.Errorf("put: %w", err)
	}

	return userconf, wgPriv, wgPSK, nil
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
	filename := kdlib.SanitizeFilename(name)

	return fmt.Sprintf("attachment; filename=%s; filename*=%s", "wg-"+id+".conf", "utf-8''"+url.QueryEscape(filename))
}

// DelUserUserID - delete user by UserID.
func DelUserUserID(db BrigadeStorage, params operations.DeleteUserUserIDParams, principal interface{}) middleware.Responder {
	err := db.userRemove(params.UserID, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Delete user: %s :%s\n", params.UserID, err)

		return operations.NewDeleteUserUserIDForbidden()
	}

	return operations.NewDeleteUserUserIDNoContent()
}

// GetUsers - .
func GetUsers(db BrigadeStorage, params operations.GetUserParams, principal interface{}) middleware.Responder {
	storageUsers, err := db.userList()
	if err != nil {
		fmt.Fprintf(os.Stderr, "List error: %s\n", err)

		return operations.NewGetUserDefault(500)
	}

	apiUsers := make([]*models.User, len(storageUsers))
	for i := range storageUsers {
		user := storageUsers[i]
		id := user.UserID.String()
		apiUsers[i] = &models.User{
			UserID:                  &id,
			UserName:                &user.Name,
			MonthlyQuotaRemainingGB: float32(user.Quota.LimitMonthlyRemaining),
			LastVisitHour:           (*strfmt.DateTime)(&user.Quota.LastActivity),
			PersonName:              user.Person.Name,
			PersonDesc:              user.Person.Desc,
			PersonDescLink:          user.Person.URL,
		}
		//copy(apiUsers[i].Problems, user.Problems)

		if !user.Quota.ThrottlingTill.IsZero() {
			apiUsers[i].ThrottlingTill = (*strfmt.DateTime)(&user.Quota.ThrottlingTill)
		}

		if !user.Quota.LastActivity.IsZero() {
			apiUsers[i].LastVisitHour = (*strfmt.DateTime)(&user.Quota.LastActivity)
		}
	}

	return operations.NewGetUserOK().WithPayload(apiUsers)
}

func genwgKey(routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) ([]byte, []byte, []byte, []byte, []byte, error) {
	wgPSK, err := wgtypes.GenerateKey()
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("psk: %w", err)
	}

	routerWgPSK, err := box.SealAnonymous(nil, wgPSK[:], routerPublicKey, rand.Reader)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("psk router seal: %w", err)
	}

	shufflerWgPSK, err := box.SealAnonymous(nil, wgPSK[:], shufflerPublicKey, rand.Reader)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("psk shuffler seal: %w", err)
	}

	wgPrivKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("priv: %w", err)
	}

	wgPubKey := wgPrivKey.PublicKey()

	return wgPubKey[:], wgPrivKey[:], wgPSK[:], routerWgPSK, shufflerWgPSK, nil
}
