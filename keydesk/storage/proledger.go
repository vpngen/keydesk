package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// PRO ledger - append-only history of PRO events, one JSON object per line in
// /home/<BrigadeID>/pro-ledger.jsonl next to brigade.json. brigade.json keeps
// only the current state of every key; analytics (paying keys, MRR, churn,
// retention by repeat payment) is folded from these events. The file is
// written for PRO brigades only, so free and VIP brigades never get one.
// Secrets, labels and notes never go here - only ids, tiers, cents and dates.

const ProLedgerFilename = "pro-ledger.jsonl"

// Event types.
const (
	ProEvKeyCreated    = "key_created"
	ProEvTierChanged   = "tier_changed"
	ProEvPriceSet      = "price_set"
	ProEvPriceCleared  = "price_cleared"
	ProEvCharged       = "charged"
	ProEvChargeFailed  = "charge_failed"
	ProEvBlocked       = "blocked"
	ProEvUnblocked     = "unblocked"
	ProEvDeleted       = "deleted"
	ProEvInvoiceIssued = "invoice_issued"
	ProEvInvoicePaid   = "invoice_paid"
)

// Kinds of a charge.
const (
	ProChargePurchase = "purchase" // first paid month of a new key
	ProChargeUpgrade  = "upgrade"  // tier change, prorated
	ProChargeMonthly  = "monthly"  // the combined monthly invoice
)

// Block reasons written to the ledger (ProBlockExpired / ProBlockBilling are
// reused for sweep blocks).
const ProBlockManual = "manual"

// ProLedgerEvent - one ledger line. Zero fields are omitted.
type ProLedgerEvent struct {
	At         time.Time  `json:"at"`
	Type       string     `json:"type"`
	UserID     string     `json:"user_id,omitempty"`
	Tier       string     `json:"tier,omitempty"`        // tier after the event ("free" | "basic" | "unlim")
	From       string     `json:"from,omitempty"`        // tier_changed: previous tier
	Cents      int64      `json:"cents,omitempty"`       // price / charge amount, euro cents
	PeriodFrom *time.Time `json:"period_from,omitempty"` // charged: paid period
	PeriodTo   *time.Time `json:"period_to,omitempty"`
	Kind       string     `json:"kind,omitempty"`      // charged: purchase | upgrade | monthly
	Reason     string     `json:"reason,omitempty"`    // blocked: manual | expired | billing
	HadPrice   bool       `json:"had_price,omitempty"` // deleted: the key had a sale price
	Invoice    string     `json:"invoice,omitempty"`   // invoice id (YYYY-MM)
	Keys       int        `json:"keys,omitempty"`      // invoice: keys included
	Backfilled bool       `json:"backfilled,omitempty"`
}

// proTierName - ledger keeps "free" explicitly (storage keeps "").
func proTierName(tier string) string {
	if tier == TierFree {
		return "free"
	}

	return tier
}

func (db *BrigadeStorage) proLedgerPath() string {
	return filepath.Join(filepath.Dir(db.BrigadeFilename), ProLedgerFilename)
}

// AppendProLedger - append events to the ledger. Never takes the brigade lock
// (O_APPEND writes of short lines are atomic), so it is safe to call right
// after a storage operation returned. Callers make sure the brigade is PRO.
func (db *BrigadeStorage) AppendProLedger(events ...ProLedgerEvent) error {
	if len(events) == 0 {
		return nil
	}

	f, err := os.OpenFile(db.proLedgerPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open ledger: %w", err)
	}

	defer f.Close()

	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)

	for i := range events {
		if events[i].At.IsZero() {
			events[i].At = time.Now().UTC()
		}

		if err := enc.Encode(events[i]); err != nil {
			return fmt.Errorf("encode event: %w", err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("write ledger: %w", err)
	}

	return nil
}

// ReadProLedger - all events in chronological order. A missing file is an
// empty ledger; malformed lines are skipped.
func (db *BrigadeStorage) ReadProLedger() ([]ProLedgerEvent, error) {
	f, err := os.Open(db.proLedgerPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("open ledger: %w", err)
	}

	defer f.Close()

	var events []ProLedgerEvent

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var ev ProLedgerEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}

		events = append(events, ev)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read ledger: %w", err)
	}

	sort.SliceStable(events, func(i, j int) bool { return events[i].At.Before(events[j].At) })

	return events, nil
}

// EnsureProLedger - create the ledger on first use by backfilling it from the
// current state of brigade.json. History before that moment is unknown, so
// the backfilled events carry the best dates available (creation date, tier
// start, block date) and are marked. No-op for existing ledgers and non-PRO
// brigades. Takes the brigade lock, so call it outside of storage operations.
func (db *BrigadeStorage) EnsureProLedger() error {
	if _, err := os.Stat(db.proLedgerPath()); err == nil {
		return nil
	}

	f, data, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	if data.PRO == 0 {
		return nil
	}

	var events []ProLedgerEvent

	for _, user := range data.Users {
		if user.IsBrigadier {
			continue
		}

		id := user.UserID.String()
		created := user.CreatedAt
		if created.IsZero() {
			created = time.Now().UTC()
		}

		events = append(events, ProLedgerEvent{At: created, Type: ProEvKeyCreated, UserID: id, Tier: "free", Backfilled: true})

		tierAt := user.ProTierSetAt
		if tierAt.IsZero() || tierAt.Before(created) {
			tierAt = created
		}

		if IsValidProTier(user.ProTier) {
			paidUntil := user.ProPaidUntil
			ev := ProLedgerEvent{At: tierAt, Type: ProEvTierChanged, UserID: id, From: "free", Tier: user.ProTier, Backfilled: true}
			charge := ProLedgerEvent{At: tierAt, Type: ProEvCharged, UserID: id, Tier: user.ProTier, Kind: ProChargePurchase, Cents: ProTierPriceCents[user.ProTier], PeriodFrom: &tierAt, Backfilled: true}

			if !paidUntil.IsZero() {
				ev.PeriodTo = &paidUntil
				charge.PeriodTo = &paidUntil
			}

			events = append(events, ev, charge)
		}

		if user.ProSoldFor > 0 {
			events = append(events, ProLedgerEvent{At: tierAt, Type: ProEvPriceSet, UserID: id, Cents: user.ProSoldFor, Backfilled: true})
		}

		if user.IsBlocked {
			blockedAt := user.BlockedAt
			if blockedAt.IsZero() {
				blockedAt = time.Now().UTC()
			}

			reason := user.ProBlockReason
			if reason == "" {
				reason = ProBlockManual
			}

			events = append(events, ProLedgerEvent{At: blockedAt, Type: ProEvBlocked, UserID: id, Reason: reason, Backfilled: true})
		}
	}

	sort.SliceStable(events, func(i, j int) bool { return events[i].At.Before(events[j].At) })

	if len(events) == 0 {
		// Still create the file so the ledger «starts» now.
		f, err := os.OpenFile(db.proLedgerPath(), os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("create ledger: %w", err)
		}

		return f.Close()
	}

	return db.AppendProLedger(events...)
}

// ProSoldAndTier - current price and tier of a key (for ledger diffs).
func (db *BrigadeStorage) ProSoldAndTier(id string) (int64, string, bool, error) {
	f, data, err := db.openWithReading()
	if err != nil {
		return 0, "", false, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	user := findUserByID(data, id)
	if user == nil {
		return 0, "", false, fmt.Errorf("%w: %s", ErrUserNotFound, id)
	}

	return user.ProSoldFor, user.ProTier, user.IsBlocked, nil
}

// ListProPaidUsers - ids and tiers of the paid keys that take part in billing
// (used to write per-key monthly charges when an invoice is paid).
func (db *BrigadeStorage) ListProPaidUsers(now time.Time) (map[string]string, error) {
	f, data, err := db.openWithReading()
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	paid := map[string]string{}

	for _, user := range data.Users {
		if proActivePaidUser(user) && user.ProPaidUntil.After(now) {
			paid[user.UserID.String()] = user.ProTier
		}
	}

	return paid, nil
}
