package user

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vpngen/wordsgens/namesgenerator"
)

// MonthlyQuotaRemainingGB - .
const MonthlyQuotaRemainingGB = 100

// ErrUserLimit - maximun user num exeeded.
var ErrUserLimit = errors.New("num user limit exeeded")

// User - user structure.
type User struct {
	ID                      string
	Name                    string
	Person                  namesgenerator.Person
	MonthlyQuotaRemainingGB float32
	Problems                []string
	ThrottlingTill          time.Time
	LastVisitHour           time.Time
	LastVisitSubnet         string
	LastVisitASName         string
	LastVisitASCountry      string
	Boss                    bool
}

type userStorage struct {
	sync.Mutex
	m  map[string]*User
	nm map[string]struct{}
}

var storage = &userStorage{
	m:  make(map[string]*User),
	nm: make(map[string]struct{}),
}

func (us *userStorage) put(u *User) error {
	if len(us.m) >= MaxUsers {
		return ErrUserLimit
	}

	us.Lock()
	defer us.Unlock()

	for {
		fullname, person, err := namesgenerator.PhysicsAwardee()
		if err != nil {
			return fmt.Errorf("namegen: %w", err)
		}

		if _, ok := us.nm[fullname]; !ok {
			u.Name = fullname
			u.Person = person

			break
		}
	}

	for {
		id := uuid.New().String()
		if _, ok := us.m[id]; !ok {
			u.ID = id

			break
		}
	}

	us.m[u.ID] = u
	us.nm[u.Name] = struct{}{}

	return nil
}

func (us *userStorage) delete(id string) bool {
	us.Lock()
	defer us.Unlock()

	if u, ok := us.m[id]; ok {
		if u.Boss {
			return false
		}

		if _, ok := us.nm[u.Name]; ok {
			delete(us.nm, u.Name)
		}

		delete(us.m, id)
	}

	return true
}

func (us *userStorage) list() []*User {
	us.Lock()
	defer us.Unlock()

	res := make([]*User, 0, len(us.m))

	for _, v := range us.m {
		res = append(res, v)
	}

	return res
}

func newUser(boss bool) (*User, error) {
	user := &User{
		MonthlyQuotaRemainingGB: MonthlyQuotaRemainingGB,
		Boss:                    boss,
		Problems:                make([]string, 0),
	}

	if err := storage.put(user); err != nil {
		return nil, fmt.Errorf("put: %w", err)
	}

	return user, nil
}

var brigadier *User

func init() {
	brigadier, _ = newUser(true)
}