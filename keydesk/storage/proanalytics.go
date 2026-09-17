package storage

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

// Analytics of a PRO brigade, folded from the PRO ledger and the current
// state of the keys. Definitions follow the product spec «PRO-Ключница -
// обновление аналитики» (Sep 2026):
//
//   - the main KPIs are computed for the brigade's current billing period
//     (the sliding cycle of procycle.go); the chart uses calendar months;
//   - an active paid user is a live, unblocked key on a paid tier; the sale
//     price is a separate attribute (keys without one still count as paid);
//   - «expected revenue» is the sum of the sale prices the brigadier entered -
//     Generator never sees whether the end user actually paid;
//   - «new» is the first time a key became paid; a Basic↔Unlim switch is not
//     a new user; «stopped paying» left the paid set since the period start;
//   - a renewal is a key that was paid at the end of the previous period and
//     is still paid in the current one: there is no per-key payment to
//     observe, the brigadier either keeps the key on a paid tier or not;
//   - history is folded from the events as of then, so a price change today
//     never rewrites an earlier month.

// ProAnalyticsConfig - thresholds of the recommendation blocks. They live in
// the backend config (env of the keydesk unit) so the frontend never
// hardcodes them.
type ProAnalyticsConfig struct {
	BasicHighUsagePct int // Basic keys that used at least this share of the monthly quota
	InactiveDays      int // paid keys without a connection for this many days
	GoodRenewalPct    int // renewal rate that counts as a positive signal
}

// DefaultProAnalyticsConfig - thresholds used when the env sets nothing.
func DefaultProAnalyticsConfig() ProAnalyticsConfig {
	return ProAnalyticsConfig{BasicHighUsagePct: 80, InactiveDays: 14, GoodRenewalPct: 75}
}

// Env variables of the keydesk unit that override the defaults.
const (
	ProAnalyticsEnvHighUsage   = "PRO_ANALYTICS_BASIC_HIGH_USAGE_PCT"
	ProAnalyticsEnvInactive    = "PRO_ANALYTICS_INACTIVE_DAYS"
	ProAnalyticsEnvGoodRenewal = "PRO_ANALYTICS_GOOD_RENEWAL_PCT"
)

// ProAnalyticsConfigFromEnv - defaults overridden by the env (getenv is
// os.Getenv in production, injectable for tests). Invalid values are ignored.
func ProAnalyticsConfigFromEnv(getenv func(string) string) ProAnalyticsConfig {
	cfg := DefaultProAnalyticsConfig()

	read := func(name string, dst *int, min, max int) {
		if v, err := strconv.Atoi(getenv(name)); err == nil && v >= min && v <= max {
			*dst = v
		}
	}

	read(ProAnalyticsEnvHighUsage, &cfg.BasicHighUsagePct, 1, 100)
	read(ProAnalyticsEnvInactive, &cfg.InactiveDays, 1, 365)
	read(ProAnalyticsEnvGoodRenewal, &cfg.GoodRenewalPct, 1, 100)

	return cfg
}

// ProAnalyticsConfig - the thresholds of this storage (defaults if unset).
func (db *BrigadeStorage) ProAnalyticsConfig() ProAnalyticsConfig {
	cfg := db.ProAnalyticsThresholds
	if cfg.BasicHighUsagePct == 0 || cfg.InactiveDays == 0 || cfg.GoodRenewalPct == 0 {
		return DefaultProAnalyticsConfig()
	}

	return cfg
}

// Renewal block states.
const (
	ProRenewalsNoData      = "no_data"     // first period, or nobody to renew
	ProRenewalsCalculating = "calculating" // the invoice is still payable: the result may change
	ProRenewalsFinal       = "final"
)

const proAnalyticsMonths = 6

// ProPeriod - the billing period the KPIs are computed for.
type ProPeriod struct {
	Start time.Time
	End   time.Time
	Index int
}

// ProEconomics - the money block, euro cents.
type ProEconomics struct {
	ExpectedRevenueCents int64 // sum of the sale prices of active paid keys
	ForecastKeyCostCents int64 // Basic/Unlim cost for the period at the current state
	ForecastProfitCents  int64 // revenue minus cost (PRO subscription excluded)
}

// ProTierGroup - active paid keys of one tier.
type ProTierGroup struct {
	Count               int
	Priced              int // keys with a sale price
	SharePct            int // of all active paid keys
	AverageSellingCents int64
}

// ProPaidUsers - the paid audience of the period.
type ProPaidUsers struct {
	Active        int
	New           int
	StoppedPaying int
	NetGrowth     int
	Priced        int
	Basic         ProTierGroup
	Unlim         ProTierGroup
}

// ProRenewals - keys of the previous period that stayed paid in this one.
type ProRenewals struct {
	Status   string
	Eligible int
	Renewed  int
	RatePct  int
	ChangePp *int // vs the previous boundary; nil without one
	Good     bool // rate at or above the configured threshold
}

// ProRecommendation - one actionable list of keys.
type ProRecommendation struct {
	Count int
	IDs   []string
}

// ProRecommendations - the «things to look at» block.
type ProRecommendations struct {
	NotRenewed     ProRecommendation
	BasicHighUsage ProRecommendation
	BasicAtLimit   ProRecommendation
	InactivePaid   ProRecommendation
}

// ProMonthPoint - one calendar month of history.
type ProMonthPoint struct {
	Month         string // YYYY-MM
	Available     bool   // false for months before the ledger knows anything
	Reconstructed bool   // fully before the live ledger: prices are today's
	ExpectedCents int64  // expected revenue at month end (now for the current month)
	CostCents     int64  // key cost accrued by days (projected to month end for the current month)
	ProfitCents   int64
}

// ProAnalytics - the numbers for the analytics page.
type ProAnalytics struct {
	Period          ProPeriod
	Economics       ProEconomics
	PaidUsers       ProPaidUsers
	Renewals        ProRenewals
	Recommendations ProRecommendations
	Config          ProAnalyticsConfig
	Months          []ProMonthPoint
	LedgerSince     time.Time
}

// proKeyState - what the fold knows about a key at a point in time.
type proKeyState struct {
	tier        string
	price       int64
	blocked     bool
	deleted     bool
	firstPaidAt time.Time // first time on a paid tier
	since       time.Time // start of the current state (cost accrual)
}

func (s *proKeyState) activePaid() bool {
	return !s.deleted && !s.blocked && IsValidProTier(s.tier)
}

type proSnapshot struct {
	paid     map[string]bool
	expected int64
}

func proMonthKey(t time.Time) string {
	return t.UTC().Format("2006-01")
}

func proMonthStart(t time.Time) time.Time {
	t = t.UTC()

	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// accrueCost - spread the monthly price of a tier over the calendar months
// covered by [from, to): each month gets price × the share of it covered.
func accrueCost(acc map[string]float64, tier string, from, to time.Time) {
	price := float64(ProTierPriceCents[tier])
	if price == 0 || !to.After(from) {
		return
	}

	for cur := from; cur.Before(to); {
		monthStart := proMonthStart(cur)
		monthEnd := monthStart.AddDate(0, 1, 0)

		segEnd := to
		if monthEnd.Before(segEnd) {
			segEnd = monthEnd
		}

		acc[proMonthKey(monthStart)] += price * segEnd.Sub(cur).Hours() / monthEnd.Sub(monthStart).Hours()
		cur = segEnd
	}
}

func snapshotOf(state map[string]*proKeyState) proSnapshot {
	snap := proSnapshot{paid: map[string]bool{}}

	for id, s := range state {
		if s.activePaid() {
			snap.paid[id] = true
			snap.expected += s.price
		}
	}

	return snap
}

func applyProEvent(state map[string]*proKeyState, acc map[string]float64, ev ProLedgerEvent) {
	if ev.UserID == "" {
		return
	}

	s, ok := state[ev.UserID]
	if !ok {
		s = &proKeyState{}
		state[ev.UserID] = s
	}

	// Close the running cost segment before the state changes.
	if s.activePaid() {
		accrueCost(acc, s.tier, s.since, ev.At)
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

	if IsValidProTier(s.tier) && s.firstPaidAt.IsZero() {
		s.firstPaidAt = ev.At
	}

	s.since = ev.At
}

func roundPct(part, total int) int {
	if total <= 0 {
		return 0
	}

	return int(math.Round(float64(part) * 100 / float64(total)))
}

// proAnalyticsPeriod - the billing period containing now. Brigades without
// an anchor (should not happen after EnsureProSince) fall back to the
// calendar month.
func proAnalyticsPeriod(since, now time.Time) (ProPeriod, *ProPeriod) {
	if since.IsZero() {
		start := proMonthStart(now)

		return ProPeriod{Start: start, End: start.AddDate(0, 1, 0)}, nil
	}

	cycle := ProCycleAt(since, now)
	period := ProPeriod{Start: cycle.Start, End: cycle.End, Index: cycle.Index}

	if cycle.Index == 0 {
		return period, nil
	}

	prev := ProPeriod{Start: addMonthsClamped(since, cycle.Index-1), End: cycle.Start, Index: cycle.Index - 1}

	return period, &prev
}

// ProAnalytics - fold the ledger and the current keys into the analytics
// numbers as of now.
func (db *BrigadeStorage) ProAnalytics(now time.Time, cfg ProAnalyticsConfig) (ProAnalytics, error) {
	events, err := db.ReadProLedger()
	if err != nil {
		return ProAnalytics{}, fmt.Errorf("ledger: %w", err)
	}

	f, data, err := db.openWithReading()
	if err != nil {
		return ProAnalytics{}, fmt.Errorf("db: %w", err)
	}

	defer f.Close()

	now = now.UTC()
	period, prevPeriod := proAnalyticsPeriod(data.ProSince, now)

	result := ProAnalytics{Period: period, Config: cfg}

	// ----- fold the ledger -----

	realStart := now // first live (non-backfilled) event

	for _, ev := range events {
		if result.LedgerSince.IsZero() || ev.At.Before(result.LedgerSince) {
			result.LedgerSince = ev.At
		}

		if !ev.Backfilled && ev.At.Before(realStart) {
			realStart = ev.At
		}
	}

	// Points in time to snapshot: calendar month starts (the state at the
	// start of a month is the state at the end of the previous one) and the
	// billing period boundaries.
	curMonth := proMonthStart(now)
	marks := map[time.Time]bool{}

	for i := proAnalyticsMonths - 1; i >= 0; i-- {
		marks[curMonth.AddDate(0, -i, 0)] = true
	}

	marks[period.Start] = true

	if prevPeriod != nil {
		marks[prevPeriod.Start] = true
	}

	markList := make([]time.Time, 0, len(marks))

	for m := range marks {
		if !m.After(now) {
			markList = append(markList, m)
		}
	}

	sortTimes(markList)

	state := map[string]*proKeyState{}
	cost := map[string]float64{} // calendar month -> cents accrued
	snaps := map[time.Time]proSnapshot{}
	chargedInPeriod := int64(0)

	next := 0

	for _, ev := range events {
		if ev.At.After(now) {
			continue
		}

		for next < len(markList) && !ev.At.Before(markList[next]) {
			snaps[markList[next]] = snapshotOf(state)
			next++
		}

		applyProEvent(state, cost, ev)

		if ev.Type == ProEvCharged && (ev.Kind == ProChargePurchase || ev.Kind == ProChargeUpgrade) && !ev.At.Before(period.Start) {
			chargedInPeriod += ev.Cents
		}
	}

	for next < len(markList) {
		snaps[markList[next]] = snapshotOf(state)
		next++
	}

	// Open cost segments up to now, plus the projection of the current
	// calendar month at the current state.
	for _, s := range state {
		if s.activePaid() {
			accrueCost(cost, s.tier, s.since, curMonth.AddDate(0, 1, 0))
		}
	}

	// ----- current state from the brigade -----

	quota := uint64(db.MonthlyQuotaRemaining)
	nowSet := map[string]bool{}
	live := map[string]*User{}

	var (
		economics ProEconomics
		paid      ProPaidUsers
		recs      ProRecommendations
	)

	for _, user := range data.Users {
		if user.IsBrigadier {
			continue
		}

		id := user.UserID.String()
		live[id] = user

		if !IsValidProTier(user.ProTier) || user.IsBlocked {
			continue
		}

		nowSet[id] = true
		paid.Active++
		economics.ExpectedRevenueCents += user.ProSoldFor

		group := &paid.Basic
		if user.ProTier == TierUnlim {
			group = &paid.Unlim
		}

		group.Count++

		if user.ProSoldFor > 0 {
			group.Priced++
			group.AverageSellingCents += user.ProSoldFor
			paid.Priced++
		}

		// Recommendations from the live counters.
		if user.ProTier == TierBasic && quota > 0 {
			remaining := user.Quotas.LimitMonthlyRemaining

			switch {
			case remaining == 0:
				recs.BasicAtLimit.IDs = append(recs.BasicAtLimit.IDs, id)
			case remaining < quota && roundPct(int(quota-remaining), int(quota)) >= cfg.BasicHighUsagePct:
				recs.BasicHighUsage.IDs = append(recs.BasicHighUsage.IDs, id)
			}
		}

		last := user.Quotas.LastActivity.Total
		if last.IsZero() {
			last = user.CreatedAt
		}

		if !last.IsZero() && now.Sub(last) >= time.Duration(cfg.InactiveDays)*24*time.Hour {
			recs.InactivePaid.IDs = append(recs.InactivePaid.IDs, id)
		}
	}

	for _, group := range []*ProTierGroup{&paid.Basic, &paid.Unlim} {
		group.SharePct = roundPct(group.Count, paid.Active)

		if group.Priced > 0 {
			group.AverageSellingCents = int64(math.Round(float64(group.AverageSellingCents) / float64(group.Priced)))
		}
	}

	// ----- period dynamics -----

	prevSnap := snaps[period.Start]

	for _, s := range state {
		if !s.firstPaidAt.IsZero() && !s.firstPaidAt.Before(period.Start) {
			paid.New++
		}
	}

	for id := range prevSnap.paid {
		if !nowSet[id] {
			paid.StoppedPaying++

			if _, exists := live[id]; exists {
				recs.NotRenewed.IDs = append(recs.NotRenewed.IDs, id)
			}
		}
	}

	paid.NetGrowth = paid.New - paid.StoppedPaying

	// ----- renewals -----

	renewals := ProRenewals{Status: ProRenewalsNoData}

	if prevPeriod != nil && len(prevSnap.paid) > 0 {
		renewals.Eligible = len(prevSnap.paid)

		for id := range prevSnap.paid {
			if nowSet[id] {
				renewals.Renewed++
			}
		}

		renewals.RatePct = roundPct(renewals.Renewed, renewals.Eligible)
		renewals.Status = ProRenewalsFinal

		// An unpaid invoice still decides the previous period: not paying it
		// means nobody renewed.
		if currentUnpaidProInvoice(data) != nil {
			renewals.Status = ProRenewalsCalculating
		}

		renewals.Good = renewals.Status == ProRenewalsFinal && renewals.RatePct >= cfg.GoodRenewalPct

		if prevPrev, ok := snaps[prevPeriod.Start]; ok && len(prevPrev.paid) > 0 {
			kept := 0

			for id := range prevPrev.paid {
				if prevSnap.paid[id] {
					kept++
				}
			}

			change := renewals.RatePct - roundPct(kept, len(prevPrev.paid))
			renewals.ChangePp = &change
		}
	}

	// ----- economics of the period -----

	cycle := ProCycle{Index: period.Index, Start: period.Start, End: period.End}
	economics.ForecastKeyCostCents = chargedInPeriod + buildProInvoice(data, events, cycle, now).TotalCents
	economics.ForecastProfitCents = economics.ExpectedRevenueCents - economics.ForecastKeyCostCents

	// ----- calendar history -----

	for i := proAnalyticsMonths - 1; i >= 0; i-- {
		monthStart := curMonth.AddDate(0, -i, 0)
		monthEnd := monthStart.AddDate(0, 1, 0)
		point := ProMonthPoint{Month: proMonthKey(monthStart)}

		if !result.LedgerSince.IsZero() && monthEnd.After(result.LedgerSince) {
			point.Available = true
			point.Reconstructed = !monthEnd.After(realStart)

			snap, ok := snaps[monthEnd]
			if i == 0 || !ok {
				snap = proSnapshot{paid: nowSet, expected: economics.ExpectedRevenueCents}
			}

			point.ExpectedCents = snap.expected
			point.CostCents = int64(math.Round(cost[point.Month]))
			point.ProfitCents = point.ExpectedCents - point.CostCents
		}

		result.Months = append(result.Months, point)
	}

	for _, rec := range []*ProRecommendation{&recs.NotRenewed, &recs.BasicHighUsage, &recs.BasicAtLimit, &recs.InactivePaid} {
		rec.Count = len(rec.IDs)
	}

	result.Economics = economics
	result.PaidUsers = paid
	result.Renewals = renewals
	result.Recommendations = recs

	return result, nil
}

func sortTimes(list []time.Time) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j].Before(list[j-1]); j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
}
