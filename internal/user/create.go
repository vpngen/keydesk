package user

import (
	"fmt"
	"os"
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
	FreeSlots  int
	TotalSlots uint
}

var (
	ErrNoFreeSlots = fmt.Errorf("no free slots")
	ErrNotAllowed  = fmt.Errorf("not allowed")
)

func (s Service) CreateUser(configs []string, domain string) (res createUserResponse, err error) {
	err = s.db.RunInTransaction(func(brigade *storage.Brigade) error {
		//if brigade.Mode == storage.ModeBrigade {
		//	return ErrNotAllowed
		//} TODO!!!

		if len(brigade.Users) >= int(brigade.MaxUsers) {
			return ErrNoFreeSlots
		}

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
	if domain == "" {
		domain = brigade.EndpointDomain
	}

	dbUser, err := newUser(brigade, domain)
	if err != nil {
		return User{}, fmt.Errorf("new user: %w", err)
	}

	cfgs, name, err := s.generator.GenerateConfigs(brigade, &dbUser, configs)
	if err != nil {
		return User{}, fmt.Errorf("generate configs: %w", err)
	}

	brigade.Users = append(brigade.Users, &dbUser)

	endpointHostString := domain
	if endpointHostString == "" {
		endpointHostString = brigade.EndpointIPv4.String()
	}

	return User{
		UUID:    dbUser.UserID,
		Name:    name,
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

	user := storage.NewUser(uid, name, time.Now(), false, true, ip4, ip6, person)
	if domain != "" {
		user.EndpointDomain = domain
	}

	return user, nil
}

func (s Service) UnblockUser(id uuid.UUID) (free int, err error) {
	if err := s.db.RunInTransaction(func(brigade *storage.Brigade) error {
		//if brigade.Mode == storage.ModeBrigade {
		//	return ErrNotAllowed
		//}

		return nil
	}); err != nil {
		return 0, fmt.Errorf("run in transaction: %w", err)
	}

	if err := s.db.UnblockUser(id.String()); err != nil {
		return 0, fmt.Errorf("unblock user %s: %w", id, err)
	}

	if err := s.db.RunInTransaction(func(brigade *storage.Brigade) error {
		free, _ = s.getSlotsInfo(brigade)

		return nil
	}); err != nil {
		return 0, fmt.Errorf("run in transaction: %w", err)
	}

	fmt.Fprintf(os.Stderr, "User %s unblocked\n", id)

	return
}
