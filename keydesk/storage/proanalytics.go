package storage

import (
	"fmt"
	"sort"
	"time"
)

// Analytics folded from the PRO ledger. Definitions agreed with the product
// (Sep 2026): a paying key is a live, unblocked paid-tier key with a sale
// price entered; «new» / «stopped paying» are the keys that entered / left
// that set since the start of the month; retention is the share of last
// month's paying keys that were charged again this month.

const proAnalyticsMonths = 6

// ProMonthPoint - one month of history.
type ProMonthPoint struct {
	Month         string // YYYY-MM
	Paying        int
	ExpectedCents int64 // sum of sale prices of the paying keys at month end
	ChargedCents  int64 // what the brigadier was charged during the month
	Available     bool  // false for months before the ledger started
}

// ProAnalytics - the numbers for the analytics page.
type ProAnalytics struct {
	Month              string
	PayingKeys         int
	MRRCents           int64
	NewPaying          int
	StoppedPaying      int
	NetGrowth          int
	Renewals           int // keys charged by the monthly invoice this month
	RetentionAvailable bool
	RetentionBase      int
	RetentionKept      int
	Months             []ProMonthPoint
	LedgerSince        time.Time
}

type proKeyState struct {
	tier    string
	price   int64
	blocked bool
	deleted bool
}

func (s proKeyState) paying() bool {
	return !s.deleted && !s.blocked && IsValidProTier(s.tier) && s.price > 0
}

func proMonthKey(t time.Time) string {
	return t.UTC().Format("2006-01")
}

func proMonthStart(t time.Time) time.Time {
	t = t.UTC()

	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func applyProEvent(state map[string]*proKeyState, ev ProLedgerEvent) {
	if ev.UserID == "" {
		return
	}

	s, ok := state[ev.UserID]
	if !ok {
		s = &proKeyState{}
		state[ev.UserID] = s
	}

	switch ev.Type {
	case ProEvKeyCreated:
		if ev.Tier != "" && ev.Tier != "free" {
			s.tier = ev.Tier
		}
	case ProEvTierChanged:
		if ev.Tier == "free" {
			s.tier = TierFree
		} else {
			s.tier = ev.Tier
		}
	case ProEvPriceSet:
		s.price = ev.Cents
	case ProEvPriceCleared:
		s.price = 0
	case ProEvBlocked:
		s.blocked = true
	case ProEvUnblocked:
		s.blocked = false
	case ProEvDeleted:
		s.deleted = true
	}
}

func payingSet(state map[string]*proKeyState) (map[string]bool, int64) {
	set := map[string]bool{}
	var cents int64

	for id, s := range state {
		if s.paying() {
			set[id] = true
			cents += s.price
		}
	}

	return set, cents
}

// ProAnalytics - fold the ledger into the analytics numbers as of now.
func (db *BrigadeStorage) ProAnalytics(now time.Time) (ProAnalytics, error) {
	events, err := db.ReadProLedger()
	if err != nil {
		return ProAnalytics{}, fmt.Errorf("ledger: %w", err)
	}

	now = now.UTC()
	curStart := proMonthStart(now)
	result := ProAnalytics{Month: proMonthKey(now)}

	if len(events) > 0 {
		result.LedgerSince = events[0].At
	}

	// Month boundaries: the start of each of the last N months plus the
	// start of the next one; the state just before boundary i is the state
	// at the end of month i-1.
	starts := make([]time.Time, 0, proAnalyticsMonths+1)
	for i := proAnalyticsMonths - 1; i >= 0; i-- {
		starts = append(starts, curStart.AddDate(0, -i, 0))
	}
	starts = append(starts, curStart.AddDate(0, 1, 0))

	state := map[string]*proKeyState{}
	charged := map[string]int64{}     // month -> cents charged
	monthlyByKey := map[string]bool{} // this month: keys charged by the invoice
	keptSet := map[string]bool{}      // this month: keys with any charge
	endPaying := map[string]map[string]bool{}
	endCents := map[string]int64{}

	snapshot := func(monthEndBefore time.Time) {
		m := proMonthKey(monthEndBefore.AddDate(0, 0, -1))
		set, cents := payingSet(state)
		endPaying[m] = set
		endCents[m] = cents
	}

	next := 0
	for _, ev := range events {
		for next < len(starts) && !ev.At.Before(starts[next]) {
			snapshot(starts[next])
			next++
		}

		applyProEvent(state, ev)

		if ev.Type == ProEvCharged {
			m := proMonthKey(ev.At)
			charged[m] += ev.Cents

			if m == result.Month && ev.UserID != "" {
				keptSet[ev.UserID] = true

				if ev.Kind == ProChargeMonthly {
					monthlyByKey[ev.UserID] = true
				}
			}
		}
	}

	for next < len(starts) {
		snapshot(starts[next])
		next++
	}

	// The current month is «as of now», not month end.
	nowSet, nowCents := payingSet(state)
	endPaying[result.Month] = nowSet
	endCents[result.Month] = nowCents

	result.PayingKeys = len(nowSet)
	result.MRRCents = nowCents
	result.Renewals = len(monthlyByKey)

	prevMonth := proMonthKey(curStart.AddDate(0, -1, 0))
	prevSet := endPaying[prevMonth]

	for id := range nowSet {
		if !prevSet[id] {
			result.NewPaying++
		}
	}

	for id := range prevSet {
		if !nowSet[id] {
			result.StoppedPaying++
		}
	}

	result.NetGrowth = result.NewPaying - result.StoppedPaying

	// Retention needs a previous cohort and at least one monthly charge ever:
	// before the invoice flow exists nobody could have «paid again».
	hasMonthly := false
	for _, ev := range events {
		if ev.Type == ProEvCharged && ev.Kind == ProChargeMonthly {
			hasMonthly = true
			break
		}
	}

	if hasMonthly && len(prevSet) > 0 {
		result.RetentionAvailable = true
		result.RetentionBase = len(prevSet)

		for id := range prevSet {
			if keptSet[id] {
				result.RetentionKept++
			}
		}
	}

	months := make([]string, 0, proAnalyticsMonths)
	for i := 0; i < proAnalyticsMonths; i++ {
		months = append(months, proMonthKey(starts[i]))
	}
	sort.Strings(months)

	for _, m := range months {
		point := ProMonthPoint{Month: m}
		monthEnd := proMonthStart(mustParseMonth(m)).AddDate(0, 1, 0)

		if !result.LedgerSince.IsZero() && !monthEnd.Before(result.LedgerSince) {
			point.Available = true
			point.Paying = len(endPaying[m])
			point.ExpectedCents = endCents[m]
			point.ChargedCents = charged[m]
		}

		result.Months = append(result.Months, point)
	}

	return result, nil
}

func mustParseMonth(m string) time.Time {
	t, err := time.Parse("2006-01", m)
	if err != nil {
		return time.Time{}
	}

	return t
}
