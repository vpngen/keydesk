package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Sliding monthly billing cycle of a PRO brigade, anchored at the activation
// date (Brigade.ProSince). Cycle k runs from since+k months to since+k+1
// months, on the same day of month (clamped to shorter months).
//
//   - cycle 0: paid keys are charged at purchase (immediate charges);
//   - from cycle 1: nothing is charged at purchase; every cycle boundary from
//     the second one issues an invoice for the cycle just finished, billing
//     each paid key by the days it was used and not prepaid; the invoice is
//     due in ProInvoiceDueDays, after that the brigade is downgraded to free.

// ProCycle - one billing cycle.
type ProCycle struct {
	Index int
	Start time.Time
	End   time.Time
}

// addMonthsClamped - t + n months keeping the day of month, clamped to the
// last day of the target month (31 Jan + 1 month = 28/29 Feb, not 3 Mar).
func addMonthsClamped(t time.Time, n int) time.Time {
	t = t.UTC()
	first := time.Date(t.Year(), t.Month()+time.Month(n), 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
	last := first.AddDate(0, 1, -1).Day()

	day := t.Day()
	if day > last {
		day = last
	}

	return time.Date(first.Year(), first.Month(), day, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
}

// ProCycleAt - the cycle that contains now.
func ProCycleAt(since, now time.Time) ProCycle {
	since = since.UTC()
	now = now.UTC()

	if now.Before(since) {
		return ProCycle{Index: 0, Start: since, End: addMonthsClamped(since, 1)}
	}

	k := (now.Year()-since.Year())*12 + int(now.Month()-since.Month())
	if k < 0 {
		k = 0
	}

	for addMonthsClamped(since, k+1).Before(now) || addMonthsClamped(since, k+1).Equal(now) {
		k++
	}

	for k > 0 && addMonthsClamped(since, k).After(now) {
		k--
	}

	return ProCycle{Index: k, Start: addMonthsClamped(since, k), End: addMonthsClamped(since, k+1)}
}

// proKeyHistory - what the ledger knows about a key for billing: the tier
// segments and until when the key was prepaid by an immediate charge.
type proKeyHistory struct {
	segments     []proTierSegment
	prepaidUntil time.Time
}

type proTierSegment struct {
	tier string
	from time.Time
}

func proKeyHistories(events []ProLedgerEvent) map[string]*proKeyHistory {
	hist := map[string]*proKeyHistory{}

	get := func(id string) *proKeyHistory {
		h, ok := hist[id]
		if !ok {
			h = &proKeyHistory{}
			hist[id] = h
		}

		return h
	}

	for _, ev := range events {
		if ev.UserID == "" {
			continue
		}

		switch ev.Type {
		case ProEvTierChanged:
			tier := ev.Tier
			if tier == "free" {
				tier = TierFree
			}

			get(ev.UserID).segments = append(get(ev.UserID).segments, proTierSegment{tier: tier, from: ev.At})
		case ProEvCharged:
			if (ev.Kind == ProChargePurchase || ev.Kind == ProChargeUpgrade) && ev.PeriodTo != nil && ev.PeriodTo.After(get(ev.UserID).prepaidUntil) {
				get(ev.UserID).prepaidUntil = *ev.PeriodTo
			}
		}
	}

	return hist
}

// buildProInvoice - the invoice for the given cycle from the current state of
// the keys and the ledger history. Each paid key active now is billed for the
// days of the cycle it spent on a paid tier and was not prepaid for; the
// cycle end is capped at «now» for the preliminary calculation of an
// unfinished cycle (the estimate then grows day by day).
func buildProInvoice(data *Brigade, events []ProLedgerEvent, cycle ProCycle, now time.Time) ProInvoice {
	hist := proKeyHistories(events)
	cycleDays := float64(cycle.End.Sub(cycle.Start).Hours()) / 24
	if cycleDays <= 0 {
		cycleDays = 30
	}

	type acc struct {
		qty   int
		days  int64
		cents int64
	}

	perTier := map[string]*acc{}
	keys := 0

	for _, user := range data.Users {
		if !proActivePaidUser(user) || user.IsBlocked {
			continue
		}

		id := user.UserID.String()
		h := hist[id]

		// Tier segments inside the cycle: what the ledger says, else the
		// current tier from its recorded start.
		var segs []proTierSegment
		if h != nil && len(h.segments) > 0 {
			segs = h.segments
		} else {
			segs = []proTierSegment{{tier: user.ProTier, from: user.ProTierSetAt}}
		}

		var prepaid time.Time
		if h != nil {
			prepaid = h.prepaidUntil
		}

		billedKey := false

		for i, seg := range segs {
			if !IsValidProTier(seg.tier) {
				continue
			}

			from := seg.from
			if from.Before(cycle.Start) {
				from = cycle.Start
			}

			if prepaid.After(from) {
				from = prepaid
			}

			to := cycle.End
			if i+1 < len(segs) && segs[i+1].from.Before(to) {
				to = segs[i+1].from
			}

			if !to.After(from) {
				continue
			}

			days := int64(to.Sub(from).Hours()/24 + 0.5)
			if days <= 0 {
				continue
			}

			a := perTier[seg.tier]
			if a == nil {
				a = &acc{}
				perTier[seg.tier] = a
			}

			a.days += days
			a.cents += int64(float64(ProTierPriceCents[seg.tier])*float64(days)/cycleDays + 0.5)
			billedKey = true
		}

		if billedKey {
			keys++
			for _, seg := range segs {
				if seg.tier == user.ProTier && perTier[seg.tier] != nil {
					break
				}
			}
			if a := perTier[user.ProTier]; a != nil {
				a.qty++
			}
		}
	}

	invoice := ProInvoice{
		ID:         cycle.End.Format(proInvoiceIDLayout),
		PeriodFrom: cycle.Start,
		PeriodTo:   cycle.End,
		CreatedAt:  now,
		DueAt:      cycle.End.AddDate(0, 0, ProInvoiceDueDays),
		Status:     ProBillingIssued,
		KeysCount:  keys,
	}

	for _, tier := range []string{TierBasic, TierUnlim} {
		a := perTier[tier]
		if a == nil || a.cents == 0 {
			continue
		}

		invoice.Lines = append(invoice.Lines, ProInvoiceLine{
			Kind: "days", Tier: tier, Qty: a.qty, Days: a.days,
			PriceCents: ProTierPriceCents[tier], AmountCents: a.cents,
		})
		invoice.TotalCents += a.cents
	}

	return invoice
}

// GenerateProInvoice - issue the invoice for every finished cycle from the
// second boundary on that has no invoice yet. Returns the invoices issued.
func (db *BrigadeStorage) GenerateProInvoice(now time.Time) ([]ProInvoice, error) {
	events, err := db.ReadProLedger()
	if err != nil {
		return nil, fmt.Errorf("ledger: %w", err)
	}

	f, data, err := db.openWithReading()
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	if data.PRO == 0 || data.ProSince.IsZero() {
		return nil, nil
	}

	have := map[string]bool{}
	for i := range data.ProInvoices {
		have[data.ProInvoices[i].ID] = true
	}

	var issued []ProInvoice

	// Boundaries 2, 3, ... up to now: each closes the cycle before it.
	for k := 2; ; k++ {
		end := addMonthsClamped(data.ProSince, k)
		if end.After(now) {
			break
		}

		id := end.Format(proInvoiceIDLayout)
		if have[id] {
			continue
		}

		cycle := ProCycle{Index: k - 1, Start: addMonthsClamped(data.ProSince, k-1), End: end}
		invoice := buildProInvoice(data, events, cycle, now)

		// Nothing to pay: keep a paid zero invoice so the cycle is closed.
		if invoice.TotalCents == 0 {
			invoice.Status = "paid"
			invoice.PaidAt = now
		} else {
			data.ProBillingState = ProBillingIssued
		}

		data.ProInvoices = append(data.ProInvoices, invoice)
		issued = append(issued, invoice)
	}

	if len(issued) == 0 {
		return nil, nil
	}

	if err := commitBrigade(f, data); err != nil {
		return nil, fmt.Errorf("save: %w", err)
	}

	return issued, nil
}

// SweepProBilling - mark the current invoice overdue once its due date has
// passed. Returns true when the brigade has just become overdue (the caller
// downgrades it to free outside of the storage lock).
func (db *BrigadeStorage) SweepProBilling(now time.Time) (bool, error) {
	f, data, err := db.openWithReading()
	if err != nil {
		return false, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	if data.PRO == 0 {
		return false, nil
	}

	invoice := currentUnpaidProInvoice(data)
	if invoice == nil || invoice.Status != ProBillingIssued || !now.After(invoice.DueAt) {
		return false, nil
	}

	invoice.Status = ProBillingOverdue
	data.ProBillingState = ProBillingOverdue

	if err := commitBrigade(f, data); err != nil {
		return false, fmt.Errorf("save: %w", err)
	}

	return true, nil
}

// EnsureProSince - PRO brigades activated before the cycle model got no
// anchor date: take it from the .pro marker (touched at activation), else
// from the earliest paid tier start, else now.
func (db *BrigadeStorage) EnsureProSince(now time.Time) error {
	f, data, err := db.openWithReading()
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	if data.PRO == 0 {
		return nil
	}

	// Invoices of the retired calendar-month model (ids «YYYY-MM») are not
	// payable anymore: cancel the unpaid ones so the state goes back to paid.
	migrated := false

	for i := range data.ProInvoices {
		inv := &data.ProInvoices[i]
		if len(inv.ID) == len("2006-01") && (inv.Status == ProBillingIssued || inv.Status == ProBillingOverdue) {
			inv.Status = "cancelled"
			migrated = true
		}
	}

	if migrated && currentUnpaidProInvoice(data) == nil {
		data.ProBillingState = ProBillingPaid
	}

	if !data.ProSince.IsZero() {
		if migrated {
			if err := commitBrigade(f, data); err != nil {
				return fmt.Errorf("save: %w", err)
			}
		}

		return nil
	}

	since := now
	if st, err := os.Stat(filepath.Join(filepath.Dir(db.BrigadeFilename), ".pro")); err == nil && st.ModTime().Before(since) {
		since = st.ModTime().UTC()
	}

	for _, user := range data.Users {
		if IsValidProTier(user.ProTier) && !user.ProTierSetAt.IsZero() && user.ProTierSetAt.Before(since) {
			since = user.ProTierSetAt.UTC()
		}
	}

	data.ProSince = since

	if err := commitBrigade(f, data); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}
