package keydesk

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/vpngen/keydesk/gen/models"
	"github.com/vpngen/keydesk/gen/restapi/operations"
	"github.com/vpngen/keydesk/kdlib"
	"github.com/vpngen/keydesk/keydesk/storage"
	"github.com/vpngen/vpngine/naclkey"
	"github.com/vpngen/wordsgens/namesgenerator"
	"golang.org/x/crypto/nacl/box"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
)

// Allowed prefixes.
const (
	CGNATPrefix = "100.64.0.0/10"
	ULAPrefix   = "fd00::/8"
)

// Users defaults
const (
	// MaxUsers - maximem limit.
	MaxUsers = 250
	// MonthlyQuotaRemaining - .
	MonthlyQuotaRemaining = 100 * 1024 * 1024 * 1024
	// DefaultMaxUserInactivityPeriod
	DefaultMaxUserInactivityPeriod = 24 * 30 * time.Hour // month
)

// AddUser - create user.
func AddUser(db *storage.BrigadeStorage, params operations.PostUserParams, principal interface{}, routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) middleware.Responder {
	var (
		user          *storage.UserConfig
		wgPriv, wgPSK []byte
	)

	for {
		fullname, person, err := namesgenerator.PeaceAwardeeShort()
		if err != nil {
			return operations.NewPostUserInternalServerError()
		}

		user, wgPriv, wgPSK, err = addUser(db, fullname, person, false, false, routerPublicKey, shufflerPublicKey)
		if err != nil {
			if errors.Is(err, storage.ErrUserCollision) {
				continue
			}

			fmt.Fprintf(os.Stderr, "Add error: %s\n", err)

			return operations.NewPostUserInternalServerError()
		}

		break
	}

	wgconf := genWgConf(user, wgPriv, wgPSK)

	rc := io.NopCloser(strings.NewReader(wgconf))

	return operations.NewPostUserCreated().WithContentDisposition(constructContentDisposition(user.Name, user.ID.String())).WithPayload(rc)
}

// AddBrigadier - create brigadier user.
func AddBrigadier(db *storage.BrigadeStorage, fullname string, person namesgenerator.Person, replaceBrigadier bool, routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) (string, string, error) {
	userconf, wgPriv, wgPSK, err := addUser(db, fullname, person, true, replaceBrigadier, routerPublicKey, shufflerPublicKey)
	if err != nil {
		return "", "", fmt.Errorf("addUser: %w", err)
	}

	wgconf := genWgConf(userconf, wgPriv, wgPSK)

	return wgconf, kdlib.SanitizeFilename(userconf.Name), nil
}

func addUser(db *storage.BrigadeStorage, fullname string, person namesgenerator.Person, IsBrigadier, replaceBrigadier bool, routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) (*storage.UserConfig, []byte, []byte, error) {
	wgPub, wgPriv, wgPSK, wgRouterPSK, wgShufflerPSK, err := genUserWGKeys(routerPublicKey, shufflerPublicKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wg gen: %s\n", err)

		return nil, nil, nil, fmt.Errorf("wg gen: %w", err)
	}

	userconf, err := db.CreateUser(fullname, person, IsBrigadier, replaceBrigadier, wgPub, wgRouterPSK, wgShufflerPSK)
	if err != nil {
		fmt.Fprintf(os.Stderr, "put: %s\n", err)

		return nil, nil, nil, fmt.Errorf("put: %w", err)
	}

	return userconf, wgPriv, wgPSK, nil
}

func genWgConf(u *storage.UserConfig, wgPriv, wgPSK []byte) string {
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
func DelUserUserID(db *storage.BrigadeStorage, params operations.DeleteUserUserIDParams, principal interface{}) middleware.Responder {
	err := db.DeleteUser(params.UserID, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Delete user: %s :%s\n", params.UserID, err)

		return operations.NewDeleteUserUserIDForbidden()
	}

	return operations.NewDeleteUserUserIDNoContent()
}

// GetUsers - .
func GetUsers(db *storage.BrigadeStorage, params operations.GetUserParams, principal interface{}) middleware.Responder {
	storageUsers, err := db.ListUsers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "List error: %s\n", err)

		return operations.NewGetUserDefault(500)
	}

	apiUsers := make([]*models.User, len(storageUsers))
	for i := range storageUsers {
		user := storageUsers[i]
		id := user.UserID.String()
		apiUsers[i] = &models.User{
			UserID:         &id,
			UserName:       &user.Name,
			PersonName:     user.Person.Name,
			PersonDesc:     user.Person.Desc,
			PersonDescLink: user.Person.URL,
			CreatedAt:      (*strfmt.DateTime)(&user.CreatedAt),
		}

		if !user.Quotas.ThrottlingTill.IsZero() {
			apiUsers[i].ThrottlingTill = (*strfmt.DateTime)(&user.Quotas.ThrottlingTill)
		}

		if !user.Quotas.LastActivity.Total.IsZero() {
			lastActivity := user.Quotas.LastActivity.Total.UTC().Truncate(time.Hour)
			apiUsers[i].LastVisitHour = (*strfmt.DateTime)(&lastActivity)
		}

		x := float32(float64(math.Round((float64(user.Quotas.LimitMonthlyRemaining/1024/1024)/1024)*100)) / 100)
		apiUsers[i].MonthlyQuotaRemainingGB = &x

		status := UserStatusOK

		switch {
		case user.Quotas.LastActivity.Total.IsZero():
			status = UserStatusNeverUsed
		case user.Quotas.LastActivity.Monthly.IsZero():
			status = UserStatusInactive
		case !user.Quotas.ThrottlingTill.IsZero():
			status = UserStatusLimited
		}

		apiUsers[i].Status = &status
	}

	return operations.NewGetUserOK().WithPayload(apiUsers)
}

func genUserWGKeys(routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) ([]byte, []byte, []byte, []byte, []byte, error) {
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
