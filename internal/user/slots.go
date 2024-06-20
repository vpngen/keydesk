package user

import "github.com/vpngen/keydesk/keydesk/storage"

func (s Service) GetSlotsInfo() (free uint, total uint, err error) {
	err = s.db.RunInTransaction(func(brigade *storage.Brigade) error {
		free, total = s.getSlotsInfo(brigade)
		return nil
	})
	return
}

func (s Service) getSlotsInfo(brigade *storage.Brigade) (free uint, total uint) {
	return brigade.MaxUsers - uint(len(brigade.Users)), brigade.MaxUsers
}
