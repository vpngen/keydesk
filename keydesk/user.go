package keydesk

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vpngen/keydesk/internal/vpn/cloak"
	"github.com/vpngen/keydesk/internal/vpn/proto0"
	ss2 "github.com/vpngen/keydesk/internal/vpn/ss"
	"github.com/vpngen/keydesk/internal/vpn/vgc"
	wg2 "github.com/vpngen/keydesk/internal/vpn/wg"

	"github.com/vpngen/keydesk/internal/maintenance"

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
	"github.com/go-openapi/swag"

	"github.com/btcsuite/btcd/btcutil/base58"
)

// Allowed prefixes.
const (
	CGNATPrefix = "100.64.0.0/10"
	ULAPrefix   = "fd00::/8"
	ChaCha20    = "chacha20-ietf-poly1305"

	vpnConfigSchema = "vgc"
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
	user, vpnCfgs, wgPriv, wgPSK, ovcPriv, cloakBypassUID, ipsecUsername, ipsecPassword, outlineSecret, proto0LongID, proto0ShortID, err := pickUpUser(db, routerPublicKey, shufflerPublicKey)
	if err != nil {
		return operations.NewPostUserInternalServerError()
	}

	_, confJson, err := assembleConfig(user, 0, vpnCfgs, wgPriv, wgPSK, ovcPriv, cloakBypassUID, ipsecUsername, ipsecPassword, outlineSecret, proto0LongID, proto0ShortID)
	if err != nil {
		return operations.NewPostUserInternalServerError()
	}

	return operations.NewPostUserCreated().WithPayload(confJson)
}

// AddBrigadier - create brigadier user.
func AddBrigadier(db *storage.BrigadeStorage, fullname string, person namesgenerator.Person, replaceBrigadier bool, reqVpnCfgs *storage.ConfigsImplemented, routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) (string, string, *models.Newuser, error) {
	if ok, till, msg := maintenance.CheckInPaths("/.maintenance", filepath.Dir(db.BrigadeFilename)+"/.maintenance"); ok {
		return "", "", nil, maintenance.NewError(till, msg)
	}

	if replaceBrigadier {
		if _, err := os.Stat(filepath.Dir(db.BrigadeFilename) + "/.maintenance_till_restore"); err == nil {
			if err := os.Remove(filepath.Dir(db.BrigadeFilename) + "/.maintenance_till_restore"); err != nil {
				fmt.Fprintf(os.Stderr, "remove .maintenance_till_restore: %s\n", err)
			}
		}
	}

	dbVpnCfgs, err := db.GetVpnConfigs(reqVpnCfgs)
	if err != nil {
		return "", "", nil, fmt.Errorf("get vpn configs: %w", err)
	}

	user, wgPriv, wgPSK, ovcPriv, cloakBypassUID, ipsecUsername, ipsecPassword, outlineSecret, proto0LongID, proto0ShortID, err := addUser(db, dbVpnCfgs, fullname, person, true, replaceBrigadier, routerPublicKey, shufflerPublicKey)
	if err != nil {
		return "", "", nil, fmt.Errorf("addUser: %w", err)
	}

	wgconf, confJson, err := assembleConfig(user, 1, dbVpnCfgs, wgPriv, wgPSK, ovcPriv, cloakBypassUID, ipsecUsername, ipsecPassword, outlineSecret, proto0LongID, proto0ShortID)
	if err != nil {
		return "", "", nil, fmt.Errorf("assembleConfig: %w", err)
	}

	return wgconf, kdlib.AssembleWgStyleTunName(user.Name) + ".conf", confJson, nil
}

const OutlinePrefix = "%16%03%01%00%C2%A8%01%01"

func assembleConfig(
	user *storage.UserConfig,
	isBrigadier int,
	vpnCfgs *storage.ConfigsImplemented,
	wgPriv, wgPSK []byte,
	ovcPriv, cloakBypassUID string,
	ipsecUsername, ipsecPassword,
	outlineSecret string,
	proto0LongID, proto0ShortID string,
) (string, *models.Newuser, error) {
	var (
		wgconf        string
		amneziaConfig *AmneziaConfig
	)

	endpointHostString := user.EndpointDomain
	if endpointHostString == "" {
		endpointHostString = user.EndpointIPv4.String()
	}

	newuser := &models.Newuser{
		UserID:   swag.String(user.ID.String()),
		UserName: &user.Name,
		Domain:   endpointHostString,
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
		amneziaConfig = NewAmneziaConfig(
			// endpointHostString,
			user.EndpointIPv4.String(),
			user.Name, defaultInternalDNS+","+defaultInternalDNS)

		aovcConf, err := GenConfAmneziaOpenVPNoverCloak(user, ovcPriv, cloakBypassUID)
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

	if vpnCfgs.IPSec[storage.ConfigIPSecTypeManual] {
		newuser.IPSecL2TPManualConfig = &models.NewuserIPSecL2TPManualConfig{
			Username: &ipsecUsername,
			Password: &ipsecPassword,
			PSK:      &user.IPSecPSK,
			Server:   &endpointHostString,
		}
	}

	if vpnCfgs.Outline[storage.ConfigOutlineTypeAccesskey] {
		accessKey := "ss://" + base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(
			fmt.Appendf([]byte{}, "chacha20-ietf-poly1305:%s", outlineSecret),
		) +
			"@" + fmt.Sprintf("%s:%d", endpointHostString, user.OutlinePort) +
			"/?outline=1&prefix=" + OutlinePrefix +
			"#" + strings.ReplaceAll(url.QueryEscape(user.Name), "+", "%20")
		newuser.OutlineConfig = &models.NewuserOutlineConfig{
			AccessKey: &accessKey,
		}
	}

	if vpnCfgs.Proto0[storage.ConfigProto0TypeAccesskey] {
		accessKey := "\u0076\u006C\u0065\u0073\u0073\u003A\u002F\u002F" + proto0LongID +
			fmt.Sprintf("@%s:%d?", endpointHostString, user.Proto0Port) +
			"\u0073\u0065\u0063\u0075\u0072\u0069\u0074\u0079\u003D\u0072\u0065\u0061\u006C\u0069\u0074\u0079" +
			"\u0026\u0065\u006E\u0063\u0072\u0079\u0070\u0074\u0069\u006F\u006E\u003D\u006E\u006F\u006E\u0065" + "\u0026\u0070\u0062\u006B\u003D" +
			base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(user.EndpointWgPublic) +
			"\u0026\u0068\u0065\u0061\u0064\u0065\u0072\u0054\u0079\u0070\u0065\u003D\u006E\u006F\u006E\u0065" +
			"\u0026\u0066\u0070\u003D\u0063\u0068\u0072\u006F\u006D\u0065\u0026\u0074\u0079\u0070\u0065\u003D" +
			"\u0074\u0063\u0070\u0026\u0066\u006C\u006F\u0077\u003D\u0078\u0074\u006C\u0073\u002D\u0072\u0070\u0072\u0078\u002D\u0076\u0069\u0073\u0069\u006F\u006E" +
			"\u0026\u0073\u006E\u0069\u003D" + user.Proto0FakeDomain +
			"\u0026\u0073\u0069\u0064\u003D" + proto0ShortID +
			"#" + strings.ReplaceAll(url.QueryEscape(user.Name), "+", "%20")

		newuser.Proto0Config = &models.NewuserProto0Config{
			AccessKey: &accessKey,
		}
	}

	// TODO: check vpnCfgs
	{
		key, err := wgtypes.NewKey(wgPriv)
		if err != nil {
			return "", nil, fmt.Errorf("wgtypes.NewKey: %w", err)
		}

		pub, err := wgtypes.NewKey(user.EndpointWgPublic)
		if err != nil {
			return "", nil, fmt.Errorf("wgtypes.NewKey: %w", err)
		}

		psk, err := wgtypes.NewKey(wgPSK)
		if err != nil {
			return "", nil, fmt.Errorf("wgtypes.NewKey: %w", err)
		}

		wg := wg2.NewWireguardAnyIP(
			key.String(),
			netip.PrefixFrom(user.IPv4, 32).String()+","+netip.PrefixFrom(user.IPv6, 128).String(),
			user.DNSv4.String()+","+user.DNSv6.String(),
			pub.String(),
			psk.String(),
			fmt.Sprintf("%s:%d", endpointHostString, user.EndpointPort),
		)

		var (
			ss *ss2.Config
			ck *cloak.VGC
			p0 *proto0.Config
		)

		if outlineSecret != "" && user.OutlinePort > 0 {
			ss = ss2.NewSS(endpointHostString, ChaCha20, outlineSecret, user.OutlinePort)
		}

		if cloakBypassUID != "" && outlineSecret != "" && user.OutlinePort > 0 && user.CloakFakeDomain != "" {
			ck = cloak.NewCloakDefault(endpointHostString, cloakBypassUID, pub.String(), user.CloakFakeDomain, cloak.ProxyBook{
				Shadowsocks: ss2.NewSSProxyBook(ChaCha20, outlineSecret),
			})
		}

		if proto0LongID != "" && proto0ShortID != "" && user.Proto0FakeDomain != "" {
			p0 = proto0.NewProto0(user.EndpointWgPublic, proto0LongID, proto0ShortID, endpointHostString, user.Proto0FakeDomain, user.Proto0Port)
		}

		cfg := vgc.NewV1(user.Name, user.EndpointDomain, wg, ck, ss, p0, isBrigadier)

		encoded, err := cfg.Encode()
		if err != nil {
			return "", nil, fmt.Errorf("vgc.Encode: %w", err)
		}

		redirectURL := fmt.Sprintf("http://%s/%s://%s", endpointHostString, vpnConfigSchema, encoded)

		newuser.VPNGenConfig = models.VGC(redirectURL)
	}

	return wgconf, newuser, nil
}

func pickUpUser(
	db *storage.BrigadeStorage,
	routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte,
) (*storage.UserConfig, *storage.ConfigsImplemented, []byte, []byte, string, string, string, string, string, string, string, error) {
	for {
		fullname, person, err := namesgenerator.PeaceAwardeeShort()
		if err != nil {
			return nil, nil, nil, nil, "", "", "", "", "", "", "", fmt.Errorf("namesgenerator: %w", err)
		}

		vpnCfgs, err := db.GetVpnConfigs(nil)
		if err != nil {
			return nil, nil, nil, nil, "", "", "", "", "", "", "", fmt.Errorf("get vpn configs: %w", err)
		}

		user, wgPriv, wgPSK, ovcPriv, CloakByPassUID, ippsecUsername, ipsecPassword, outlineSecret, proto0LongID, proto0ShortID, err := addUser(db, vpnCfgs, fullname, person, false, false, routerPublicKey, shufflerPublicKey)
		if err != nil {
			if errors.Is(err, storage.ErrUserCollision) {
				continue
			}

			return nil, nil, nil, nil, "", "", "", "", "", "", "", fmt.Errorf("addUser: %w", err)
		}

		return user, vpnCfgs, wgPriv, wgPSK, ovcPriv, CloakByPassUID, ippsecUsername, ipsecPassword, outlineSecret, proto0LongID, proto0ShortID, nil
	}
}

func addUser(
	db *storage.BrigadeStorage,
	vpnCfgs *storage.ConfigsImplemented,
	fullname string,
	person namesgenerator.Person,
	IsBrigadier,
	replaceBrigadier bool,
	routerPublicKey,
	shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte,
) (*storage.UserConfig, []byte, []byte, string, string, string, string, string, string, string, error) {
	wgPub, wgPriv, wgPSK, wgRouterPSK, wgShufflerPSK, err := genUserWGKeys(routerPublicKey, shufflerPublicKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wg gen: %s\n", err)

		return nil, nil, nil, "", "", "", "", "", "", "", fmt.Errorf("wg gen: %w", err)
	}

	var (
		cloakBypassUID, cloakByPassUIDRouterEnc, CloakByPassUIDShufflerEnc string
		ovcKeyPriv, ovcCsrGzipBase64                                       string
	)
	if len(vpnCfgs.Ovc) > 0 {
		var err error

		ovcKeyPriv, ovcCsrGzipBase64, cloakBypassUID, cloakByPassUIDRouterEnc, CloakByPassUIDShufflerEnc, err = genUserOvcKeys(routerPublicKey, shufflerPublicKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ovc gen: %s\n", err)

			return nil, nil, nil, "", "", "", "", "", "", "", fmt.Errorf("ovc gen: %w", err)
		}
	}

	var (
		ipsecUsername, ipsecPassword                 string
		ipsecUsernameRouter, ipsecPasswordRouter     string
		ipsecUsernameShuffler, ipsecPasswordShuffler string
	)
	if len(vpnCfgs.IPSec) > 0 {
		ipsecUsername, ipsecUsernameRouter, ipsecUsernameShuffler,
			ipsecPassword, ipsecPasswordRouter, ipsecPasswordShuffler,
			err = genUserIPSecUserPass(routerPublicKey, shufflerPublicKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ipsec gen: %s\n", err)

			return nil, nil, nil, "", "", "", "", "", "", "", fmt.Errorf("ipsec gen: %w", err)
		}
	}

	var (
		outlineSecret            string
		outlineSecretRouterEnc   string
		outlineSecretShufflerEnc string
	)
	if len(vpnCfgs.Outline) > 0 {
		outlineSecret, outlineSecretRouterEnc, outlineSecretShufflerEnc, err = genUserOutlineSecret(routerPublicKey, shufflerPublicKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "outline gen: %s\n", err)

			return nil, nil, nil, "", "", "", "", "", "", "", fmt.Errorf("outline gen: %w", err)
		}
	}

	var (
		proto0LongID, proto0ShortID                    string
		proto0SecretRouterEnc, proto0SecretShufflerEnc string
	)
	if len(vpnCfgs.Proto0) > 0 {
		proto0LongID, proto0ShortID, proto0SecretRouterEnc, proto0SecretShufflerEnc, err = genUserProto0Secret(routerPublicKey, shufflerPublicKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "proto0 gen: %s\n", err)

			return nil, nil, nil, "", "", "", "", "", "", "", fmt.Errorf("proto0 gen: %w", err)
		}
	}

	userconf, err := db.CreateUser(
		vpnCfgs, fullname, person,
		IsBrigadier, replaceBrigadier,
		wgPub, wgRouterPSK, wgShufflerPSK,
		ovcCsrGzipBase64, cloakByPassUIDRouterEnc, CloakByPassUIDShufflerEnc,
		ipsecUsernameRouter, ipsecPasswordRouter,
		ipsecUsernameShuffler, ipsecPasswordShuffler,
		outlineSecretRouterEnc, outlineSecretShufflerEnc,
		proto0SecretRouterEnc, proto0SecretShufflerEnc,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "put: %s\n", err)

		return nil, nil, nil, "", "", "", "", "", "", "", fmt.Errorf("put: %w", err)
	}

	return userconf, wgPriv, wgPSK, ovcKeyPriv, cloakBypassUID, ipsecUsername, ipsecPassword, outlineSecret, proto0LongID, proto0ShortID, nil
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

func genUserOvcKeys(routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) (string, string, string, string, string, error) {
	cn := uuid.New().String()
	csr, key, err := kdlib.NewOvClientCertRequest(cn)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("ov new csr: %w", err)
	}

	userKey, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("marshal key: %w", err)
	}

	keyPem := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: userKey})

	csrPemGzBase64, err := kdlib.PemGzipBase64(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr})
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("csr pem encode: %w", err)
	}

	cloakBypassUID := uuid.New()

	cloakBypassUIDRouterEnc, err := box.SealAnonymous(nil, cloakBypassUID[:], routerPublicKey, rand.Reader)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("cloakBypassUID router seal: %w", err)
	}

	CloakByPassUIDShufflerEnc, err := box.SealAnonymous(nil, cloakBypassUID[:], shufflerPublicKey, rand.Reader)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("cloakBypassUID shuffler seal: %w", err)
	}

	return string(keyPem),
		string(csrPemGzBase64),
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(cloakBypassUID[:]),
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(cloakBypassUIDRouterEnc),
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(CloakByPassUIDShufflerEnc),
		nil
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

const (
	IPSecUsernameLen = 16 // 12-16
	IPSecPasswordLen = 32 // 16-64
	IPSecPSKLen      = 32 // 16-64

	Base58UsernameLen = IPSecUsernameLen/1.37 + 1
	Base58PasswordLen = IPSecPasswordLen/1.37 + 1
	Base58PSKLen      = IPSecPSKLen/1.37 + 1
)

func genUserIPSecUserPass(routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) (string, string, string, string, string, string, error) {
	// Username/Password

	usernameRand := make([]byte, IPSecUsernameLen)
	if _, err := rand.Read(usernameRand); err != nil {
		return "", "", "", "", "", "", fmt.Errorf("username rand: %w", err)
	}

	passwordRand := make([]byte, IPSecPasswordLen)
	if _, err := rand.Read(passwordRand); err != nil {
		return "", "", "", "", "", "", fmt.Errorf("password rand: %w", err)
	}

	username := base58.Encode(usernameRand)
	password := base58.Encode(passwordRand)

	if len(username) < IPSecUsernameLen || len(password) < IPSecPasswordLen {
		return "", "", "", "", "", "", fmt.Errorf("encoded len err")
	}

	username = username[:IPSecUsernameLen]
	password = password[:IPSecPasswordLen]
	usernameRouter, err := box.SealAnonymous(nil, []byte(username), routerPublicKey, rand.Reader)
	if err != nil {
		return "", "", "", "", "", "", fmt.Errorf("username router seal: %w", err)
	}

	passwordRouter, err := box.SealAnonymous(nil, []byte(password), routerPublicKey, rand.Reader)
	if err != nil {
		return "", "", "", "", "", "", fmt.Errorf("password router seal: %w", err)
	}

	usernameShuffler, err := box.SealAnonymous(nil, []byte(username), shufflerPublicKey, rand.Reader)
	if err != nil {
		return "", "", "", "", "", "", fmt.Errorf("username shuffler seal: %w", err)
	}

	passwordShuffler, err := box.SealAnonymous(nil, []byte(password), shufflerPublicKey, rand.Reader)
	if err != nil {
		return "", "", "", "", "", "", fmt.Errorf("password shuffler seal: %w", err)
	}

	return username,
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(usernameRouter),
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(usernameShuffler),
		password,
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(passwordRouter),
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(passwordShuffler),
		nil
}

func GenEndpointIPSecCreds(routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) (*storage.BrigadeIPSecConfig, error) {
	pskRand := make([]byte, IPSecPSKLen)
	if _, err := rand.Read(pskRand); err != nil {
		return nil, fmt.Errorf("psk rand: %w", err)
	}

	psk := base58.Encode(pskRand)
	if len(psk) < IPSecPSKLen {
		return nil, fmt.Errorf("encoded len err")
	}

	psk = psk[:IPSecPSKLen]

	pskRouter, err := box.SealAnonymous(nil, []byte(psk), routerPublicKey, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("psk router seal: %w", err)
	}
	pskShuffler, err := box.SealAnonymous(nil, []byte(psk), shufflerPublicKey, rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("psk shuffler seal: %w", err)
	}

	return &storage.BrigadeIPSecConfig{
		IPSecPSK:            psk,
		IPSecPSKRouterEnc:   base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(pskRouter),
		IPSecPSKShufflerEnc: base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(pskShuffler),
	}, nil
}

const (
	OutlineSecretLen       = 96                        // 64-128
	Base58OutlineSecretLen = OutlineSecretLen/1.37 + 1 // need to rewrite
)

func genUserOutlineSecret(routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) (string, string, string, error) {
	secretRand := make([]byte, OutlineSecretLen)
	if _, err := rand.Read(secretRand); err != nil {
		return "", "", "", fmt.Errorf("secret rand: %w", err)
	}

	secret := base58.Encode(secretRand)

	if len(secret) < IPSecPasswordLen {
		return "", "", "", fmt.Errorf("encoded len err")
	}

	secret = secret[:OutlineSecretLen]

	// TODO: why do we encrypt *encoded* secret?
	secretRouter, err := box.SealAnonymous(nil, []byte(secret), routerPublicKey, rand.Reader)
	if err != nil {
		return "", "", "", fmt.Errorf("secret router seal: %w", err)
	}

	secretShuffler, err := box.SealAnonymous(nil, []byte(secret), shufflerPublicKey, rand.Reader)
	if err != nil {
		return "", "", "", fmt.Errorf("secret shuffler seal: %w", err)
	}

	return secret,
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(secretRouter),
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(secretShuffler),
		nil
}

func genUserProto0Secret(routerPublicKey, shufflerPublicKey *[naclkey.NaclBoxKeyLength]byte) (string, string, string, string, error) {
	longID := uuid.New().String()
	shortID := strings.ReplaceAll(uuid.New().String(), "-", "")[:12]

	secret := shortID + "-" + strings.ReplaceAll(longID, "-", "")

	secretRouter, err := box.SealAnonymous(nil, []byte(secret), routerPublicKey, rand.Reader)
	if err != nil {
		return "", "", "", "", fmt.Errorf("secret router seal: %w", err)
	}

	secretShuffler, err := box.SealAnonymous(nil, []byte(secret), shufflerPublicKey, rand.Reader)
	if err != nil {
		return "", "", "", "", fmt.Errorf("secret shuffler seal: %w", err)
	}

	return longID, shortID,
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(secretRouter),
		base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(secretShuffler),
		nil
}
