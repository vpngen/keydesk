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

// Block reasons for PRO keys (User.ProBlockReason). A blocked user with an
// empty reason was deactivated manually by the brigadier.
const (
	ProBlockExpired = "expired"
	ProBlockBilling = "billing"
)

// ProUnlimMonthlyQuota - effectively unlimited monthly traffic for the
// "unlim" tier (1 PiB).
const ProUnlimMonthlyQuota = uint64(1) << 50

// Billing states of a PRO brigade (Brigade.ProBillingState; "" means paid).
const (
	ProBillingPaid      = ""
	ProBillingIssued    = "issued"
	ProBillingOverdue   = "overdue"
	ProBillingSuspended = "suspended"
)

// Billing timeline: the invoice is issued on the 1st, is due in 2 days and
// after 7 days of grace the paid keys get suspended (per the PRO design).
const (
	ProInvoiceDueDays   = 2
	ProInvoiceGraceDays = 7
	proInvoiceIDLayout  = "2006-01"
)

// ProTierPriceCents - monthly price per paid tier, euro cents.
var ProTierPriceCents = map[string]int64{
	TierBasic: 200,
	TierUnlim: 500,
}

// ProInvoiceLine - one aggregated line of a local PRO invoice.
type ProInvoiceLine struct {
	Kind        string `json:"kind"` // "full" (month ahead) | "prorate" (days of the previous month)
	Tier        string `json:"tier"`
	Qty         int    `json:"qty"`
	PriceCents  int64  `json:"price_cents"`
	AmountCents int64  `json:"amount_cents"`
}

// ProInvoice - a locally generated PRO invoice (stub billing v1).
type ProInvoice struct {
	ID         string           `json:"id"` // YYYY-MM
	CreatedAt  time.Time        `json:"created_at"`
	DueAt      time.Time        `json:"due_at"`
	PaidAt     time.Time        `json:"paid_at,omitempty"`
	Status     string           `json:"status"` // "issued" | "overdue" | "paid"
	KeysCount  int              `json:"keys_count"`
	TotalCents int64            `json:"total_cents"`
	Lines      []ProInvoiceLine `json:"lines,omitempty"`
}

// ProMonthlyQuotaFor - the monthly quota for a user given the tier:
// "unlim" keys get ProUnlimMonthlyQuota, everyone else keeps the brigade
// default, so free and VIP brigades see no change.
func ProMonthlyQuotaFor(user *User, defaultQuota uint64) uint64 {
	if user.ProTier == TierUnlim {
		return ProUnlimMonthlyQuota
	}

	return defaultQuota
}

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
// TierFree resets the key back to a regular free key. The monthly quota is
// re-applied according to the new tier right away.
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
	user.ProTierSetAt = time.Now().UTC()

	switch tier {
	case TierUnlim:
		user.Quotas.LimitMonthlyRemaining = ProUnlimMonthlyQuota
	default:
		if user.Quotas.LimitMonthlyRemaining > uint64(db.MonthlyQuotaRemaining) {
			user.Quotas.LimitMonthlyRemaining = uint64(db.MonthlyQuotaRemaining)
		}
	}

	if err := commitBrigade(f, data); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}

// UserProState - the PRO-related state of a key.
type UserProState struct {
	Tier        string
	PaidUntil   time.Time
	Blocked     bool
	BlockReason string
}

// GetUserProState - current tier, paid period end and block state of a key.
func (db *BrigadeStorage) GetUserProState(id string) (UserProState, error) {
	f, data, err := db.openWithReading()
	if err != nil {
		return UserProState{}, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	user := findUserByID(data, id)
	if user == nil {
		return UserProState{}, fmt.Errorf("%w: %s", ErrUserNotFound, id)
	}

	return UserProState{
		Tier:        user.ProTier,
		PaidUntil:   user.ProPaidUntil,
		Blocked:     user.IsBlocked,
		BlockReason: user.ProBlockReason,
	}, nil
}

// setProBlockReason - persist the reason of a PRO block ("" clears it).
func (db *BrigadeStorage) setProBlockReason(id, reason string) error {
	f, data, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	user := findUserByID(data, id)
	if user == nil {
		return fmt.Errorf("%w: %s", ErrUserNotFound, id)
	}

	user.ProBlockReason = reason

	if err := commitBrigade(f, data); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}

// BlockUserPro - block a key for a PRO reason (expired / billing), reusing
// the regular block machinery.
func (db *BrigadeStorage) BlockUserPro(id, reason string) error {
	if err := db.DeleteUser(id, false, true); err != nil {
		return fmt.Errorf("block: %w", err)
	}

	return db.setProBlockReason(id, reason)
}

// UnblockUserPro - undo a PRO block and clear its reason.
func (db *BrigadeStorage) UnblockUserPro(id string) error {
	if err := db.UnblockUser(id); err != nil {
		return fmt.Errorf("unblock: %w", err)
	}

	return db.setProBlockReason(id, "")
}

// ListProExpired - ids of paid keys whose paid period is over and which are
// not blocked yet. Returns nil for non-PRO brigades.
func (db *BrigadeStorage) ListProExpired(now time.Time) ([]string, error) {
	f, data, err := db.openWithReading()
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	if data.PRO == 0 {
		return nil, nil
	}

	var expired []string

	for _, user := range data.Users {
		if IsValidProTier(user.ProTier) &&
			!user.ProPaidUntil.IsZero() && user.ProPaidUntil.Before(now) &&
			!user.IsBlocked {
			expired = append(expired, user.UserID.String())
		}
	}

	return expired, nil
}

func findUserByID(data *Brigade, id string) *User {
	for _, u := range data.Users {
		if u.UserID.String() == id {
			return u
		}
	}

	return nil
}

// ——— Local invoice lifecycle (stub billing v1: no real payments yet) ———

// proActivePaidUser - a paid key that participates in billing: has a paid
// tier and is not deactivated manually by the brigadier.
func proActivePaidUser(user *User) bool {
	if !IsValidProTier(user.ProTier) {
		return false
	}

	if user.IsBlocked && user.ProBlockReason == "" {
		return false
	}

	return true
}

// GenerateProInvoice - issue the invoice for the current month if it was not
// issued yet: a full-month line per paid key that stays paid into the month,
// plus prorated lines for keys whose paid tier started during the previous
// month. Returns true when a new invoice was actually issued.
func (db *BrigadeStorage) GenerateProInvoice(now time.Time) (bool, error) {
	f, data, err := db.openWithReading()
	if err != nil {
		return false, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	if data.PRO == 0 {
		return false, nil
	}

	id := now.Format(proInvoiceIDLayout)
	for i := range data.ProInvoices {
		if data.ProInvoices[i].ID == id {
			return false, nil
		}
	}

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	prevStart := monthStart.AddDate(0, -1, 0)
	prevDays := int64(monthStart.Sub(prevStart).Hours() / 24)

	full := map[string]int{}
	prorate := map[string]int64{}
	prorateQty := map[string]int{}
	keys := 0

	for _, user := range data.Users {
		if !proActivePaidUser(user) || !user.ProPaidUntil.After(now) {
			continue
		}

		keys++
		full[user.ProTier]++

		// The key went paid during the previous month: charge the days from
		// the tier start to the end of that month.
		if user.ProTierSetAt.After(prevStart) && user.ProTierSetAt.Before(monthStart) {
			days := prevDays - int64(user.ProTierSetAt.Day()) + 1
			prorate[user.ProTier] += ProTierPriceCents[user.ProTier] * days / prevDays
			prorateQty[user.ProTier]++
		}
	}

	invoice := ProInvoice{
		ID:        id,
		CreatedAt: now,
		DueAt:     now.AddDate(0, 0, ProInvoiceDueDays),
		Status:    ProBillingIssued,
		KeysCount: keys,
	}

	for _, tier := range []string{TierBasic, TierUnlim} {
		if full[tier] > 0 {
			amount := ProTierPriceCents[tier] * int64(full[tier])
			invoice.Lines = append(invoice.Lines, ProInvoiceLine{
				Kind: "full", Tier: tier, Qty: full[tier],
				PriceCents: ProTierPriceCents[tier], AmountCents: amount,
			})
			invoice.TotalCents += amount
		}

		if prorate[tier] > 0 {
			invoice.Lines = append(invoice.Lines, ProInvoiceLine{
				Kind: "prorate", Tier: tier, Qty: prorateQty[tier],
				PriceCents: ProTierPriceCents[tier], AmountCents: prorate[tier],
			})
			invoice.TotalCents += prorate[tier]
		}
	}

	if invoice.TotalCents == 0 {
		return false, nil
	}

	data.ProInvoices = append(data.ProInvoices, invoice)
	data.ProBillingState = ProBillingIssued

	if err := commitBrigade(f, data); err != nil {
		return false, fmt.Errorf("save: %w", err)
	}

	return true, nil
}

// SweepProBilling - advance the billing state: issued → overdue after the due
// date, → suspended after the grace period. Returns ids of the paid keys to
// block when the brigade just got suspended (the caller blocks them outside
// of the storage lock).
func (db *BrigadeStorage) SweepProBilling(now time.Time) ([]string, error) {
	f, data, err := db.openWithReading()
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	if data.PRO == 0 {
		return nil, nil
	}

	invoice := currentUnpaidProInvoice(data)
	if invoice == nil {
		return nil, nil
	}

	changed := false

	if invoice.Status == ProBillingIssued && now.After(invoice.DueAt) {
		invoice.Status = ProBillingOverdue
		data.ProBillingState = ProBillingOverdue
		changed = true
	}

	var toBlock []string

	if data.ProBillingState != ProBillingSuspended &&
		now.After(invoice.CreatedAt.AddDate(0, 0, ProInvoiceGraceDays)) {
		data.ProBillingState = ProBillingSuspended
		changed = true

		for _, user := range data.Users {
			if proActivePaidUser(user) && !user.IsBlocked {
				toBlock = append(toBlock, user.UserID.String())
			}
		}
	}

	if changed {
		if err := commitBrigade(f, data); err != nil {
			return nil, fmt.Errorf("save: %w", err)
		}
	}

	return toBlock, nil
}

// PayProInvoice - stub payment: mark the current unpaid invoice as paid and
// return ids of the keys blocked over billing (the caller unblocks them
// outside of the storage lock).
func (db *BrigadeStorage) PayProInvoice(now time.Time) ([]string, error) {
	f, data, err := db.openWithReading()
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	if data.PRO == 0 {
		return nil, ErrUserNotFound
	}

	invoice := currentUnpaidProInvoice(data)
	if invoice == nil {
		return nil, nil
	}

	invoice.Status = "paid"
	invoice.PaidAt = now
	data.ProBillingState = ProBillingPaid

	var toUnblock []string

	for _, user := range data.Users {
		if user.IsBlocked && user.ProBlockReason == ProBlockBilling {
			toUnblock = append(toUnblock, user.UserID.String())
		}
	}

	if err := commitBrigade(f, data); err != nil {
		return nil, fmt.Errorf("save: %w", err)
	}

	return toUnblock, nil
}

// ProBillingInfo - billing state and invoice history of the brigade.
type ProBillingInfo struct {
	State    string
	Current  *ProInvoice
	Invoices []ProInvoice
}

// GetProBilling - read the billing state and invoices.
func (db *BrigadeStorage) GetProBilling() (ProBillingInfo, error) {
	f, data, err := db.openWithReading()
	if err != nil {
		return ProBillingInfo{}, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	info := ProBillingInfo{
		State:    data.ProBillingState,
		Invoices: append([]ProInvoice(nil), data.ProInvoices...),
	}

	if cur := currentUnpaidProInvoice(data); cur != nil {
		copied := *cur
		info.Current = &copied
	}

	return info, nil
}

func currentUnpaidProInvoice(data *Brigade) *ProInvoice {
	for i := len(data.ProInvoices) - 1; i >= 0; i-- {
		if data.ProInvoices[i].Status == ProBillingIssued || data.ProInvoices[i].Status == ProBillingOverdue {
			return &data.ProInvoices[i]
		}
	}

	return nil
}

// SetUserProConfigs - persist the ready-to-use access strings of a key
// (called at creation time, PRO brigades only).
func (db *BrigadeStorage) SetUserProConfigs(id string, configs map[string]string) error {
	if len(configs) == 0 {
		return nil
	}

	f, data, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	user := findUserByID(data, id)
	if user == nil {
		return fmt.Errorf("%w: %s", ErrUserNotFound, id)
	}

	user.ProConfigs = configs

	if err := commitBrigade(f, data); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}
