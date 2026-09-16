package storage

import (
	"fmt"
	"os"
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
	ProBillingPaid    = ""
	ProBillingIssued  = "issued"
	ProBillingOverdue = "overdue" // unpaid past the due date: the brigade is downgraded to free
)

// Billing timeline (sliding monthly cycle anchored at PRO activation, see
// procycle.go): the invoice for a finished cycle is issued on the cycle
// boundary and is due in 7 days.
const (
	ProInvoiceDueDays  = 7
	proInvoiceIDLayout = "2006-01-02"
)

// ProTierPriceCents - monthly price per paid tier, euro cents.
var ProTierPriceCents = map[string]int64{
	TierBasic: 200,
	TierUnlim: 500,
}

// ProInvoiceLine - one aggregated line of a local PRO invoice: keys of one
// tier, billed by the days they were used and not prepaid within the cycle.
type ProInvoiceLine struct {
	Kind        string `json:"kind"` // "days"
	Tier        string `json:"tier"`
	Qty         int    `json:"qty"`  // keys
	Days        int64  `json:"days"` // billed key-days in total
	PriceCents  int64  `json:"price_cents"`
	AmountCents int64  `json:"amount_cents"`
}

// ProInvoiceItem - one key on an invoice (what the ledger records as the
// key's monthly charge once the invoice is paid).
type ProInvoiceItem struct {
	UserID      string `json:"user_id"`
	Tier        string `json:"tier"`
	Days        int64  `json:"days"`
	AmountCents int64  `json:"amount_cents"`
}

// ProInvoice - a locally generated PRO invoice for one finished cycle.
type ProInvoice struct {
	ID         string           `json:"id"` // cycle end date, YYYY-MM-DD
	PeriodFrom time.Time        `json:"period_from,omitempty"`
	PeriodTo   time.Time        `json:"period_to,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
	DueAt      time.Time        `json:"due_at"`
	PaidAt     time.Time        `json:"paid_at,omitempty"`
	Status     string           `json:"status"` // "issued" | "overdue" | "paid"
	KeysCount  int              `json:"keys_count"`
	TotalCents int64            `json:"total_cents"`
	Lines      []ProInvoiceLine `json:"lines,omitempty"`
	Items      []ProInvoiceItem `json:"items,omitempty"`
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
		// (Re)activation starts a new billing cycle anchored at this moment.
		if atomic.LoadInt64(&brigade.PRO) == 0 || brigade.ProSince.IsZero() {
			brigade.ProSince = time.Now().UTC()
			brigade.ProBillingState = ProBillingPaid
		}

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

	if err := db.setProBlockReason(id, reason); err != nil {
		return err
	}

	db.logProLedgerEvent(ProLedgerEvent{Type: ProEvBlocked, UserID: id, Reason: reason})

	return nil
}

// UnblockUserPro - undo a PRO block and clear its reason.
func (db *BrigadeStorage) UnblockUserPro(id string) error {
	if err := db.UnblockUser(id); err != nil {
		return fmt.Errorf("unblock: %w", err)
	}

	if err := db.setProBlockReason(id, ""); err != nil {
		return err
	}

	db.logProLedgerEvent(ProLedgerEvent{Type: ProEvUnblocked, UserID: id})

	return nil
}

// logProLedgerEvent - best-effort ledger write from inside the storage
// (called with no lock held); errors only go to stderr.
func (db *BrigadeStorage) logProLedgerEvent(ev ProLedgerEvent) {
	if err := db.EnsureProLedger(); err != nil {
		fmt.Fprintf(os.Stderr, "pro ledger: ensure: %s\n", err)

		return
	}

	if err := db.AppendProLedger(ev); err != nil {
		fmt.Fprintf(os.Stderr, "pro ledger: append: %s\n", err)
	}
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

	if err := commitBrigade(f, data); err != nil {
		return nil, fmt.Errorf("save: %w", err)
	}

	return nil, nil
}

// ProBillingInfo - billing state, cycle position, the preliminary calculation
// of the next invoice and the invoice history of the brigade.
type ProBillingInfo struct {
	State            string
	Since            time.Time
	Cycle            ProCycle
	ImmediateCharges bool // cycle 0: paid keys are charged at purchase
	NextInvoiceAt    time.Time
	Estimate         ProInvoice // what the next invoice would contain if the keys stay as they are
	Current          *ProInvoice
	Invoices         []ProInvoice
}

// GetProBilling - read the billing state, the cycle and the invoices.
func (db *BrigadeStorage) GetProBilling() (ProBillingInfo, error) {
	now := time.Now().UTC()

	f, data, err := db.openWithReading()
	if err != nil {
		return ProBillingInfo{}, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	info := ProBillingInfo{
		State:    data.ProBillingState,
		Since:    data.ProSince,
		Invoices: append([]ProInvoice(nil), data.ProInvoices...),
	}

	if cur := currentUnpaidProInvoice(data); cur != nil {
		copied := *cur
		info.Current = &copied
	}

	if data.ProSince.IsZero() {
		return info, nil
	}

	info.Cycle = ProCycleAt(data.ProSince, now)
	info.ImmediateCharges = info.Cycle.Index == 0

	// The first invoice closes cycle 1, so during cycle 0 the next invoice
	// date is the end of the following cycle.
	billed := info.Cycle
	if billed.Index == 0 {
		billed = ProCycle{Index: 1, Start: billed.End, End: addMonthsClamped(data.ProSince, 2)}
	}

	info.NextInvoiceAt = billed.End

	ledger, _ := db.ReadProLedger()
	info.Estimate = buildProInvoice(data, ledger, billed, now)

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
