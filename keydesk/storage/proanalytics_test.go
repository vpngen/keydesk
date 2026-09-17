package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

// analyticsBrigade - a brigade dir with brigade.json and a ledger.
func analyticsBrigade(t *testing.T, data *Brigade, events []ProLedgerEvent) *BrigadeStorage {
	t.Helper()

	dir := t.TempDir()

	raw, _ := json.Marshal(data)
	if err := os.WriteFile(filepath.Join(dir, "brigade.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	db := &BrigadeStorage{
		BrigadeFilename:    filepath.Join(dir, "brigade.json"),
		BrigadeSpinlock:    filepath.Join(dir, "brigade.lock"),
		BrigadeStorageOpts: BrigadeStorageOpts{MonthlyQuotaRemaining: 100 * 1024 * 1024 * 1024},
	}

	if err := db.AppendProLedger(events...); err != nil {
		t.Fatal(err)
	}

	return db
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestProAnalyticsPeriodNumbers(t *testing.T) {
	since := d(2026, time.August, 4)
	now := d(2026, time.September, 20) // cycle 1: Sep 4 .. Oct 4
	quota := uint64(100 * 1024 * 1024 * 1024)

	a, b, c, dd := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	users := []*User{
		{UserID: uuid.New(), IsBrigadier: true},
		// A: Basic since Aug 10, priced, prepaid to Sep 10, active yesterday, 90% of the quota used.
		{UserID: a, ProTier: TierBasic, ProTierSetAt: d(2026, time.August, 10), ProPaidUntil: d(2026, time.September, 10), ProSoldFor: 300, CreatedAt: d(2026, time.August, 10),
			Quotas: Quota{LimitMonthlyRemaining: quota / 10, LastActivity: LastActivityPoints{Total: d(2026, time.September, 19)}}},
		// B: was Unlim, switched to Free on Sep 10 (price stays on the key).
		{UserID: b, ProSoldFor: 600, CreatedAt: d(2026, time.August, 15), Quotas: Quota{LimitMonthlyRemaining: quota}},
		// C: Basic created Sep 12 (new this period), never connected yet.
		{UserID: c, ProTier: TierBasic, ProTierSetAt: d(2026, time.September, 12), ProSoldFor: 400, CreatedAt: d(2026, time.September, 12), Quotas: Quota{LimitMonthlyRemaining: quota}},
		// D: Basic since Sep 1 without a price, quota exhausted, last seen Sep 1.
		{UserID: dd, ProTier: TierBasic, ProTierSetAt: d(2026, time.September, 1), ProPaidUntil: d(2026, time.October, 1), CreatedAt: d(2026, time.August, 20),
			Quotas: Quota{LimitMonthlyRemaining: 0, LastActivity: LastActivityPoints{Total: d(2026, time.September, 1)}}},
	}

	ev := func(at time.Time, typ, id, tier string, backfilled bool) ProLedgerEvent {
		return ProLedgerEvent{At: at, Type: typ, UserID: id, Tier: tier, Backfilled: backfilled}
	}

	charge := func(at time.Time, id, tier string, cents int64, backfilled bool) ProLedgerEvent {
		return ProLedgerEvent{At: at, Type: ProEvCharged, UserID: id, Tier: tier, Kind: ProChargePurchase, Cents: cents, PeriodFrom: ptrTime(at), PeriodTo: ptrTime(at.AddDate(0, 1, 0)), Backfilled: backfilled}
	}

	events := []ProLedgerEvent{
		ev(d(2026, time.August, 10), ProEvKeyCreated, a.String(), "free", true),
		ev(d(2026, time.August, 10), ProEvTierChanged, a.String(), TierBasic, true),
		charge(d(2026, time.August, 10), a.String(), TierBasic, 200, true),
		{At: d(2026, time.August, 10), Type: ProEvPriceSet, UserID: a.String(), Cents: 300, Backfilled: true},
		ev(d(2026, time.August, 15), ProEvKeyCreated, b.String(), "free", true),
		ev(d(2026, time.August, 15), ProEvTierChanged, b.String(), TierUnlim, true),
		charge(d(2026, time.August, 15), b.String(), TierUnlim, 500, true),
		{At: d(2026, time.August, 15), Type: ProEvPriceSet, UserID: b.String(), Cents: 600, Backfilled: true},
		ev(d(2026, time.August, 20), ProEvKeyCreated, dd.String(), "free", true),
		ev(d(2026, time.September, 1), ProEvTierChanged, dd.String(), TierBasic, false),
		charge(d(2026, time.September, 1), dd.String(), TierBasic, 200, false),
		ev(d(2026, time.September, 10), ProEvTierChanged, b.String(), "free", false),
		ev(d(2026, time.September, 12), ProEvKeyCreated, c.String(), "free", false),
		ev(d(2026, time.September, 12), ProEvTierChanged, c.String(), TierBasic, false),
		{At: d(2026, time.September, 12), Type: ProEvPriceSet, UserID: c.String(), Cents: 400, Backfilled: false},
	}

	db := analyticsBrigade(t, &Brigade{PRO: 1, ProSince: since, Users: users}, events)

	res, err := db.ProAnalytics(now, DefaultProAnalyticsConfig())
	if err != nil {
		t.Fatal(err)
	}

	if res.Period.Index != 1 || !res.Period.Start.Equal(d(2026, time.September, 4)) || !res.Period.End.Equal(d(2026, time.October, 4)) {
		t.Fatalf("period %+v", res.Period)
	}

	p := res.PaidUsers
	if p.Active != 3 || p.Priced != 2 || p.New != 1 || p.StoppedPaying != 1 || p.NetGrowth != 0 {
		t.Fatalf("paid users %+v", p)
	}

	if p.Basic.Count != 3 || p.Basic.Priced != 2 || p.Basic.AverageSellingCents != 350 || p.Basic.SharePct != 100 || p.Unlim.Count != 0 {
		t.Fatalf("groups basic %+v unlim %+v", p.Basic, p.Unlim)
	}

	e := res.Economics
	// Cycle Sep 4 .. Oct 4 (30 days): A from Sep 10 (24 d) = 160, C from Sep 12 (22 d) = 147, D from Oct 1 (3 d) = 20.
	if e.ExpectedRevenueCents != 700 || e.ForecastKeyCostCents != 327 || e.ForecastProfitCents != 373 {
		t.Fatalf("economics %+v", e)
	}

	r := res.Renewals
	if r.Status != ProRenewalsFinal || r.Eligible != 3 || r.Renewed != 2 || r.RatePct != 67 || r.Good || r.ChangePp != nil {
		t.Fatalf("renewals %+v", r)
	}

	rc := res.Recommendations
	if rc.NotRenewed.Count != 1 || rc.NotRenewed.IDs[0] != b.String() {
		t.Fatalf("not renewed %+v", rc.NotRenewed)
	}

	if rc.BasicHighUsage.Count != 1 || rc.BasicHighUsage.IDs[0] != a.String() {
		t.Fatalf("high usage %+v", rc.BasicHighUsage)
	}

	if rc.BasicAtLimit.Count != 1 || rc.BasicAtLimit.IDs[0] != dd.String() {
		t.Fatalf("at limit %+v", rc.BasicAtLimit)
	}

	if rc.InactivePaid.Count != 1 || rc.InactivePaid.IDs[0] != dd.String() {
		t.Fatalf("inactive %+v", rc.InactivePaid)
	}

	if len(res.Months) != proAnalyticsMonths {
		t.Fatalf("months %d", len(res.Months))
	}

	byMonth := map[string]ProMonthPoint{}
	for _, m := range res.Months {
		byMonth[m.Month] = m
	}

	// July: nothing known. August: A 300 + B 600 expected at month end; cost
	// A 21.5/31 × 200 + B 16.5/31 × 500 = 405. September (projected to the
	// month end): expected as of now; cost A 200 + B 9.5/30 × 500 + D 29.5/30
	// × 200 + C 18.5/30 × 200 = 678.
	if byMonth["2026-07"].Available {
		t.Fatalf("july %+v", byMonth["2026-07"])
	}

	if aug := byMonth["2026-08"]; !aug.Available || !aug.Reconstructed || aug.ExpectedCents != 900 || aug.CostCents != 405 || aug.ProfitCents != 495 {
		t.Fatalf("august %+v", aug)
	}

	if sep := byMonth["2026-09"]; !sep.Available || sep.Reconstructed || sep.ExpectedCents != 700 || sep.CostCents != 678 || sep.ProfitCents != 22 {
		t.Fatalf("september %+v", sep)
	}
}

func TestProAnalyticsRenewalStates(t *testing.T) {
	id := uuid.New()
	user := &User{UserID: id, ProTier: TierBasic, ProTierSetAt: d(2026, time.September, 15), ProSoldFor: 300, CreatedAt: d(2026, time.September, 15)}
	events := []ProLedgerEvent{
		{At: d(2026, time.September, 15), Type: ProEvKeyCreated, UserID: id.String(), Tier: "free"},
		{At: d(2026, time.September, 15), Type: ProEvTierChanged, UserID: id.String(), Tier: TierBasic},
	}

	// First period: nothing to renew yet.
	db := analyticsBrigade(t, &Brigade{PRO: 1, ProSince: d(2026, time.September, 15), Users: []*User{user}}, events)

	res, err := db.ProAnalytics(d(2026, time.September, 20), DefaultProAnalyticsConfig())
	if err != nil || res.Renewals.Status != ProRenewalsNoData || res.PaidUsers.New != 1 {
		t.Fatalf("first period: %+v err %v", res.Renewals, err)
	}

	// Second period with the invoice still payable: calculating.
	invoice := ProInvoice{ID: "2026-10-15", Status: ProBillingIssued, DueAt: d(2026, time.October, 22)}
	db = analyticsBrigade(t, &Brigade{PRO: 1, ProSince: d(2026, time.September, 15), ProBillingState: ProBillingIssued, ProInvoices: []ProInvoice{invoice}, Users: []*User{user}}, events)

	res, err = db.ProAnalytics(d(2026, time.October, 20), DefaultProAnalyticsConfig())
	if err != nil || res.Renewals.Status != ProRenewalsCalculating || res.Renewals.Eligible != 1 || res.Renewals.Renewed != 1 {
		t.Fatalf("calculating: %+v err %v", res.Renewals, err)
	}

	// Paid: final and good.
	invoice.Status = "paid"
	db = analyticsBrigade(t, &Brigade{PRO: 1, ProSince: d(2026, time.September, 15), ProInvoices: []ProInvoice{invoice}, Users: []*User{user}}, events)

	res, err = db.ProAnalytics(d(2026, time.October, 20), DefaultProAnalyticsConfig())
	if err != nil || res.Renewals.Status != ProRenewalsFinal || res.Renewals.RatePct != 100 || !res.Renewals.Good {
		t.Fatalf("final: %+v err %v", res.Renewals, err)
	}
}

func TestProAnalyticsConfigFromEnv(t *testing.T) {
	env := map[string]string{ProAnalyticsEnvInactive: "30", ProAnalyticsEnvHighUsage: "abc", ProAnalyticsEnvGoodRenewal: "0"}

	cfg := ProAnalyticsConfigFromEnv(func(k string) string { return env[k] })
	if cfg.InactiveDays != 30 || cfg.BasicHighUsagePct != 80 || cfg.GoodRenewalPct != 75 {
		t.Fatalf("config %+v", cfg)
	}
}
