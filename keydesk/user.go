package keydesk

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/google/uuid"
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
	/// fmt.Fprintf(os.Stderr, "****************** AddUser(db *storage.BrigadeStorage\n")
	user, vpnCfgs, wgPriv, wgPSK, ovcPriv, err := pickUpUser(db, routerPublicKey, shufflerPublicKey)
	if err != nil {
		return operations.NewPostUserngInternalServerError()
	}

	_, confJson, err := assembleConfig(user, vpnCfgs, wgPriv, wgPSK, ovcPriv)
	if err != nil {
		return operations.NewPostUserngInternalServerError()
	}

	return operations.NewPostUserngCreated().WithPayload(confJson)
}

// AddBrigadier - create brigadier user.
func AddBrigadier(db *storage.BrigadeStorage, fullname string, person namesgenerator.Person, replaceBrigadier bool, reqVpnCfgs *storage.ConfigsImplemented, routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) (string, string, *models.Newuser, error) {
	dbVpnCfgs, err := db.GetVpnConfigs(reqVpnCfgs)
	if err != nil {
		return "", "", nil, fmt.Errorf("get vpn configs: %w", err)
	}

	user, wgPriv, wgPSK, ovcPriv, err := addUser(db, dbVpnCfgs, fullname, person, true, replaceBrigadier, routerPublicKey, shufflerPublicKey)
	if err != nil {
		return "", "", nil, fmt.Errorf("addUser: %w", err)
	}

	wgconf, confJson, err := assembleConfig(user, dbVpnCfgs, wgPriv, wgPSK, ovcPriv)
	if err != nil {
		return "", "", nil, fmt.Errorf("assembleConfig: %w", err)
	}

	return wgconf, kdlib.AssembleWgStyleTunName(user.Name), confJson, nil
}

func assembleConfig(user *storage.UserConfig, vpnCfgs *storage.ConfigsImplemented, wgPriv, wgPSK []byte, ovcPriv string) (string, *models.Newuser, error) {
	var (
		wgconf        string
		amneziaConfig *AmneziaConfig
	)

	newuser := &models.Newuser{
		UserName: &user.Name,
	}

	wgStyleTunName := kdlib.AssembleWgStyleTunName(user.Name)

	if len(vpnCfgs.Wg) > 0 {
		wgconf = GenConfWireguard(user, wgPriv, wgPSK)
		wgConfFilename := wgStyleTunName + ".conf"

		newuser.WireguardConfig = &models.NewuserWireguardConfig{
			FileContent: &wgconf,
			FileName:    &wgConfFilename,
			TonnelName:  &wgStyleTunName,
		}
	}

	if vpnCfgs.Ovc[storage.ConfigOvcTypeAmnezia] {
		endpointHostString := user.EndpointDomain
		if endpointHostString == "" {
			endpointHostString = user.EndpointIPv4.String()
		}

		amneziaConfig = NewAmneziaConfig(endpointHostString, user.Name, user.DNSv4.String())

		aovcConf, err := GenConfAmneziaOpenVPNoverCloak(user, ovcPriv)
		if err != nil {
			return "", nil, fmt.Errorf("ovc gen: %w", err)
		}

		amneziaConfig.AddContainer(aovcConf)
		amneziaConfig.SetDefaultContainer(AmneziaContainerOpenVPNCloak)

		amnzConf, err := amneziaConfig.Marshal()
		if err != nil {
			return "", nil, fmt.Errorf("amnz marshal: %w", err)
		}

		amneziaConfFilename := wgStyleTunName + ".vpn"
		newuser.AmnzOvcConfig = &models.NewuserAmnzOvcConfig{
			FileContent: &amnzConf,
			FileName:    &amneziaConfFilename,
			TonnelName:  &user.Name,
		}
	}

	return wgconf, newuser, nil
}

func pickUpUser(db *storage.BrigadeStorage, routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) (*storage.UserConfig, *storage.ConfigsImplemented, []byte, []byte, string, error) {
	for {
		fullname, person, err := namesgenerator.PeaceAwardeeShort()
		if err != nil {
			return nil, nil, nil, nil, "", fmt.Errorf("namesgenerator: %w", err)
		}

		vpnCfgs, err := db.GetVpnConfigs(nil)
		if err != nil {
			return nil, nil, nil, nil, "", fmt.Errorf("get vpn configs: %w", err)
		}

		user, wgPriv, wgPSK, ovcPriv, err := addUser(db, vpnCfgs, fullname, person, false, false, routerPublicKey, shufflerPublicKey)
		if err != nil {
			if errors.Is(err, storage.ErrUserCollision) {
				continue
			}

			return nil, nil, nil, nil, "", fmt.Errorf("addUser: %w", err)
		}

		return user, vpnCfgs, wgPriv, wgPSK, ovcPriv, nil
	}
}

func addUser(db *storage.BrigadeStorage, vpnCfgs *storage.ConfigsImplemented, fullname string, person namesgenerator.Person, IsBrigadier, replaceBrigadier bool, routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) (*storage.UserConfig, []byte, []byte, string, error) {
	wgPub, wgPriv, wgPSK, wgRouterPSK, wgShufflerPSK, err := genUserWGKeys(routerPublicKey, shufflerPublicKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wg gen: %s\n", err)

		return nil, nil, nil, "", fmt.Errorf("wg gen: %w", err)
	}

	var ovcKeyPriv, ovcCsrGzipBase64 string
	if len(vpnCfgs.Ovc) > 0 {
		var err error

		ovcKeyPriv, ovcCsrGzipBase64, err = genUserOvcKeys(routerPublicKey, shufflerPublicKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ovc gen: %s\n", err)

			return nil, nil, nil, "", fmt.Errorf("ovc gen: %w", err)
		}
	}

	userconf, err := db.CreateUser(vpnCfgs, fullname, person, IsBrigadier, replaceBrigadier, wgPub, wgRouterPSK, wgShufflerPSK, ovcCsrGzipBase64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "put: %s\n", err)

		return nil, nil, nil, "", fmt.Errorf("put: %w", err)
	}

	return userconf, wgPriv, wgPSK, ovcKeyPriv, nil
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

func GetUsersStats(db *storage.BrigadeStorage, params operations.GetUsersStatsParams, principal interface{}) middleware.Responder {
	storageUsersStats, err := db.GetUsersStats()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Stats error: %s\n", err)

		return operations.NewGetUsersStatsDefault(500)
	}

	stats := &models.Stats{}

	prevMonth := int64(storageUsersStats[len(storageUsersStats)-1].CountersUpdateTime.Month())
	for _, monthStat := range storageUsersStats {
		totalUsers := int64(monthStat.TotalUsersCount)
		activeUsers := int64(monthStat.ActiveUsersCount)
		totalTrafficGB := float32(float64(math.Round((float64((monthStat.TotalTraffic.Rx+monthStat.TotalTraffic.Tx)/1024/1024)/1024)*100)) / 100)

		monthNum := int64(monthStat.CountersUpdateTime.Month())
		if monthStat.CountersUpdateTime.IsZero() {
			monthNum = prevMonth + 1
			if monthNum > 12 {
				monthNum = 1
			}
		}

		stats.TotalUsers = append(stats.TotalUsers, &models.StatsTotalUsersItems0{Month: &monthNum, Value: &totalUsers})
		stats.ActiveUsers = append(stats.ActiveUsers, &models.StatsActiveUsersItems0{Month: &monthNum, Value: &activeUsers})
		stats.TotalTrafficGB = append(stats.TotalTrafficGB, &models.StatsTotalTrafficGBItems0{Month: &monthNum, Value: &totalTrafficGB})

		prevMonth = monthNum
	}

	return operations.NewGetUsersStatsOK().WithPayload(stats)
}

// GetUsers - .
func GetUsers(db *storage.BrigadeStorage, params operations.GetUserParams, principal interface{}) middleware.Responder {
	// fmt.Fprintf(os.Stderr, "****************** GetUsers(db *storage.BrigadeStorage\n")
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

func genUserOvcKeys(routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) (string, string, error) {
	cn := uuid.New().String()
	csr, key, err := kdlib.NewOvClientCertRequest(cn)
	if err != nil {
		return "", "", fmt.Errorf("ov new csr: %w", err)
	}

	userKey, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", "", fmt.Errorf("marshal key: %w", err)
	}

	keyPem := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: userKey})

	csrPemGzBase64, err := kdlib.PemGzipBase64(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr})
	if err != nil {
		return "", "", fmt.Errorf("csr pem encode: %w", err)
	}

	return string(keyPem), string(csrPemGzBase64), nil
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
