package user

import "github.com/vpngen/keydesk/keydesk/storage"

func (s Service) GetSlotsInfo() (free int, total uint, err error) {
	err = s.db.RunInTransaction(func(brigade *storage.Brigade) error {
		free, total = s.getSlotsInfo(brigade)
		return nil
	})
	return
}

func (s Service) getSlotsInfo(brigade *storage.Brigade) (free int, total uint) {
	return int(brigade.MaxUsers) - len(brigade.Users), brigade.MaxUsers
}
