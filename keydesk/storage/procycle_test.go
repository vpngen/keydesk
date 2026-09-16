package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func d(y int, m time.Month, day int) time.Time { return time.Date(y, m, day, 12, 0, 0, 0, time.UTC) }

func TestAddMonthsClamped(t *testing.T) {
	if got := addMonthsClamped(d(2026, time.January, 31), 1); got.Month() != time.February || got.Day() != 28 {
		t.Fatalf("Jan 31 + 1 month = %s", got)
	}

	if got := addMonthsClamped(d(2026, time.September, 21), 2); got.Month() != time.November || got.Day() != 21 {
		t.Fatalf("Sep 21 + 2 months = %s", got)
	}
}

func TestProCycleAt(t *testing.T) {
	since := d(2026, time.September, 21)

	cases := []struct {
		now   time.Time
		index int
		end   time.Time
	}{
		{d(2026, time.October, 5), 0, d(2026, time.October, 21)},
		{d(2026, time.October, 21), 1, d(2026, time.November, 21)},
		{d(2026, time.November, 25), 2, d(2026, time.December, 21)},
	}

	for _, c := range cases {
		got := ProCycleAt(since, c.now)
		if got.Index != c.index || !got.End.Equal(c.end) {
			t.Fatalf("cycle at %s: got %+v, want index %d end %s", c.now, got, c.index, c.end)
		}
	}
}

func paidUser(tier string, setAt time.Time) *User {
	return &User{UserID: uuid.New(), Name: tier, ProTier: tier, ProTierSetAt: setAt, ProPaidUntil: setAt.AddDate(0, 1, 0)}
}

func TestBuildProInvoiceProratesByDays(t *testing.T) {
	since := d(2026, time.September, 21)
	cycle := ProCycle{Index: 1, Start: d(2026, time.October, 21), End: d(2026, time.November, 21)} // 31 days

	a := paidUser(TierBasic, d(2026, time.September, 25)) // prepaid to Oct 25 -> 27 billed days
	b := paidUser(TierUnlim, d(2026, time.November, 6))   // created in the cycle, not prepaid -> 15 days
	c := paidUser(TierBasic, d(2026, time.October, 1))
	c.IsBlocked = true                                   // deactivated: not billed
	e := paidUser(TierUnlim, d(2026, time.November, 11)) // basic from Oct 1 (prepaid to Nov 1), unlim from Nov 11
	free := &User{UserID: uuid.New(), Name: "free"}

	aPrepaid := d(2026, time.October, 25)
	ePrepaid := d(2026, time.November, 1)
	events := []ProLedgerEvent{
		{At: d(2026, time.September, 25), Type: ProEvTierChanged, UserID: a.UserID.String(), From: "free", Tier: TierBasic},
		{At: d(2026, time.September, 25), Type: ProEvCharged, UserID: a.UserID.String(), Kind: ProChargePurchase, Cents: 200, PeriodTo: &aPrepaid},
		{At: d(2026, time.November, 6), Type: ProEvTierChanged, UserID: b.UserID.String(), From: "free", Tier: TierUnlim},
		{At: d(2026, time.October, 1), Type: ProEvTierChanged, UserID: e.UserID.String(), From: "free", Tier: TierBasic},
		{At: d(2026, time.October, 1), Type: ProEvCharged, UserID: e.UserID.String(), Kind: ProChargePurchase, Cents: 200, PeriodTo: &ePrepaid},
		{At: d(2026, time.November, 11), Type: ProEvTierChanged, UserID: e.UserID.String(), From: TierBasic, Tier: TierUnlim},
	}

	data := &Brigade{PRO: 1, ProSince: since, Users: []*User{a, b, c, e, free}}
	inv := buildProInvoice(data, events, cycle, cycle.End)

	if inv.ID != "2026-11-21" || inv.KeysCount != 3 {
		t.Fatalf("invoice %+v", inv)
	}

	// a: 200*27/31 = 174, b: 500*15/31 = 242, e: 200*10/31 = 65 + 500*10/31 = 161
	if inv.TotalCents != 174+242+65+161 {
		t.Fatalf("total %d, lines %+v", inv.TotalCents, inv.Lines)
	}

	for _, l := range inv.Lines {
		switch l.Tier {
		case TierBasic:
			if l.Days != 37 || l.AmountCents != 239 || l.Qty != 1 {
				t.Fatalf("basic line %+v", l)
			}
		case TierUnlim:
			if l.Days != 25 || l.AmountCents != 403 || l.Qty != 2 {
				t.Fatalf("unlim line %+v", l)
			}
		}
	}
}

func TestGenerateProInvoiceClosesFinishedCycles(t *testing.T) {
	dir := t.TempDir()
	now := d(2026, time.November, 25)
	since := d(2026, time.September, 21)

	data := &Brigade{PRO: 1, ProSince: since, Users: []*User{paidUser(TierBasic, d(2026, time.October, 25))}}
	raw, _ := json.Marshal(data)
	if err := os.WriteFile(filepath.Join(dir, "brigade.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	db := &BrigadeStorage{BrigadeFilename: filepath.Join(dir, "brigade.json"), BrigadeSpinlock: filepath.Join(dir, "brigade.lock")}

	issued, err := db.GenerateProInvoice(now)
	if err != nil {
		t.Fatal(err)
	}

	if len(issued) != 1 || issued[0].ID != "2026-11-21" || issued[0].Status != ProBillingIssued {
		t.Fatalf("issued %+v", issued)
	}

	// Oct 25 .. Nov 21 = 27 days of 31 -> 174 cents, due Nov 28.
	if issued[0].TotalCents != 174 || !issued[0].DueAt.Equal(d(2026, time.November, 28)) {
		t.Fatalf("invoice %+v", issued[0])
	}

	again, err := db.GenerateProInvoice(now)
	if err != nil || len(again) != 0 {
		t.Fatalf("second pass issued %+v err %v", again, err)
	}

	info, err := db.GetProBilling()
	if err != nil || info.Current == nil || info.State != ProBillingIssued {
		t.Fatalf("billing %+v err %v", info, err)
	}
}
