package user

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vpngen/wordsgens/namesgenerator"
)

// MonthlyQuotaRemainingGB - .
const MonthlyQuotaRemainingGB = 100

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
	m map[string]*User
}

var storage = &userStorage{
	m: make(map[string]*User),
}

func (us *userStorage) put(u *User) {
	us.Lock()
	defer us.Unlock()

	for {
		id := uuid.New().String()
		_, ok := us.m[id]
		if ok {
			continue
		}

		u.ID = id
		us.m[id] = u

		break
	}

}

func (us *userStorage) delete(id string) bool {
	us.Lock()
	defer us.Unlock()

	if u, ok := us.m[id]; ok {
		if u.Boss {
			return false
		}
	}

	delete(us.m, id)

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
	var (
		fullname string
		person   namesgenerator.Person
		err      error
	)

	if boss {
		fullname, person, err = namesgenerator.PeaceAwardee()
	} else {
		fullname, person, err = namesgenerator.PhysicsAwardee()
	}

	if err != nil {
		return nil, fmt.Errorf("namegen: %w", err)
	}

	user := &User{
		Name:                    fullname,
		Person:                  person,
		MonthlyQuotaRemainingGB: MonthlyQuotaRemainingGB,
		Boss:                    boss,
		Problems:                make([]string, 0),
	}

	storage.put(user)

	return user, nil
}

var brigadier *User

func init() {
	brigadier, _ = newUser(true)
}
