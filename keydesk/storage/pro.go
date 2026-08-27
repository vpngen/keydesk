package storage

import (
	"fmt"
	"sync/atomic"
	"time"
)

// PRO tier values stored in User.ProTier. An empty string means a regular
// free key, so existing free and VIP brigades keep working untouched.
const (
	TierFree  = ""
	TierBasic = "basic"
	TierUnlim = "unlim"
)

// IsValidProTier - check that the tier is one of the known paid tiers.
func IsValidProTier(tier string) bool {
	return tier == TierBasic || tier == TierUnlim
}

// SetPRO - turn on/off the PRO flag for the brigade (mirror of SetVIP).
func (db *BrigadeStorage) SetPRO(pro bool) error {
	f, brigade, err := db.OpenDbToModify()
	if err != nil {
		return fmt.Errorf("open to modify: %w", err)
	}
	defer f.Close()

	switch pro {
	case true:
		atomic.StoreInt64(&brigade.PRO, 1)
	default:
		atomic.StoreInt64(&brigade.PRO, 0)
	}

	if err := f.Commit(brigade); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

// IsPRO - check the PRO flag of the brigade (mirror of IsVIP).
func (db *BrigadeStorage) IsPRO() bool {
	f, brigade, err := db.openWithReading()
	if err != nil {
		return false
	}
	defer f.Close()

	if brigade == nil {
		return false
	}

	pro := atomic.LoadInt64(&brigade.PRO)

	return pro > 0
}

// UserProMeta - partial update of the brigadier's own key annotations.
// Nil fields are left untouched.
type UserProMeta struct {
	Label   *string
	Note    *string
	SoldFor *int64
}

// UpdateUserProMeta - update label/note/sold-for of a key.
func (db *BrigadeStorage) UpdateUserProMeta(id string, meta UserProMeta) error {
	f, data, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	user := findUserByID(data, id)
	if user == nil {
		return fmt.Errorf("%w: %s", ErrUserNotFound, id)
	}

	if meta.Label != nil {
		user.ProLabel = *meta.Label
	}

	if meta.Note != nil {
		user.ProNote = *meta.Note
	}

	if meta.SoldFor != nil {
		user.ProSoldFor = *meta.SoldFor
	}

	if err := commitBrigade(f, data); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}

// SetUserProTerm - set the tier and the paid period end of a key.
// TierFree resets the key back to a regular free key.
func (db *BrigadeStorage) SetUserProTerm(id, tier string, paidUntil time.Time) error {
	f, data, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	user := findUserByID(data, id)
	if user == nil {
		return fmt.Errorf("%w: %s", ErrUserNotFound, id)
	}

	user.ProTier = tier
	user.ProPaidUntil = paidUntil

	if err := commitBrigade(f, data); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}

// GetUserProTerm - current tier and paid period end of a key.
func (db *BrigadeStorage) GetUserProTerm(id string) (string, time.Time, error) {
	f, data, err := db.openWithReading()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	user := findUserByID(data, id)
	if user == nil {
		return "", time.Time{}, fmt.Errorf("%w: %s", ErrUserNotFound, id)
	}

	return user.ProTier, user.ProPaidUntil, nil
}

func findUserByID(data *Brigade, id string) *User {
	for _, u := range data.Users {
		if u.UserID.String() == id {
			return u
		}
	}

	return nil
}
