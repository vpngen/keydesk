package user

import (
	"errors"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/vpngen/keydesk/keydesk/storage"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func (s Service) DeleteUser(id uuid.UUID) (free uint, err error) {
	err = s.db.RunInTransaction(func(brigade *storage.Brigade) error {
		err = s.deleteUser(brigade, id)
		if err != nil {
			return fmt.Errorf("delete user %s: %w", id, err)
		}

		free, _ = s.getSlotsInfo(brigade)

		return nil
	})

	return
}

var ErrNotFound = errors.New("user not found")

func (s Service) deleteUser(brigade *storage.Brigade, id uuid.UUID) error {
	var (
		user *storage.User
		idx  int
	)

	for i, u := range brigade.Users {
		if u.UserID == id {
			idx, user = i, u
			break
		}
	}

	if user == nil {
		return ErrNotFound
	}

	usrPub, err := wgtypes.NewKey(user.WgPublicKey)
	if err != nil {
		return fmt.Errorf("user public key: %w", err)
	}

	epPub, err := wgtypes.NewKey(brigade.WgPublicKey)
	if err != nil {
		return fmt.Errorf("endpoint public key: %w", err)
	}

	if err = s.epClient.PeerDel(usrPub, epPub); err != nil {
		return fmt.Errorf("peer del: %w", err)
	}

	fmt.Fprintf(os.Stderr, "User %s (%s) deleted\n", user.UserID, usrPub)

	brigade.Users = append(brigade.Users[:idx], brigade.Users[idx+1:]...)

	return nil
}
