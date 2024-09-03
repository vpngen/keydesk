package user

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/internal/vpn"
	"github.com/vpngen/keydesk/keydesk/storage"
)

type User struct {
	UUID    uuid.UUID
	Name    string
	Domain  string
	Configs vpn.Configs
}

type createUserResponse struct {
	User
	FreeSlots, TotalSlots uint
}

func (s Service) CreateUser(configs []string, domain string) (res createUserResponse, err error) {
	err = s.db.RunInTransaction(func(brigade *storage.Brigade) error {
		res.User, err = s.createUserWithConfigs(brigade, configs, domain)
		if err != nil {
			return fmt.Errorf("create user with configs %s: %w", configs, err)
		}

		res.FreeSlots, res.TotalSlots = s.getSlotsInfo(brigade)

		return nil
	})

	return
}

func (s Service) createUserWithConfigs(brigade *storage.Brigade, configs []string, domain string) (User, error) {
	dbUser, err := newUser(brigade, domain)
	if err != nil {
		return User{}, fmt.Errorf("new user: %w", err)
	}

	cfgs, err := s.generator.GenerateConfigs(brigade, &dbUser, configs)
	if err != nil {
		return User{}, fmt.Errorf("generate configs: %w", err)
	}

	brigade.Users = append(brigade.Users, &dbUser)

	endpointHostString := dbUser.EndpointDomain
	if endpointHostString == "" {
		endpointHostString = brigade.EndpointDomain
		if endpointHostString == "" {
			endpointHostString = brigade.EndpointIPv4.String()
		}
	}

	return User{
		UUID:    dbUser.UserID,
		Name:    dbUser.Name,
		Domain:  endpointHostString,
		Configs: cfgs,
	}, nil
}

func newUser(brigade *storage.Brigade, domain string) (storage.User, error) {
	names, uids, ips4, ips6 := getExisting(brigade)

	name, person, err := getUniquePerson(names)
	if err != nil {
		return storage.User{}, fmt.Errorf("get unique person: %w", err)
	}

	uid := getUniqueUUID(uids)
	ip4 := getUniqueAddr4(brigade.IPv4CGNAT, ips4)
	ip6 := getUniqueAddr6(brigade.IPv6ULA, ips6)
	name = blurIP4(name, brigade.BrigadeID, ip4, brigade.IPv4CGNAT)

	user := storage.NewUser(uid, name, time.Now(), false, ip4, ip6, person)
	if domain != "" {
		user.EndpointDomain = domain
	}

	return user, nil
}
