// Package probilling - background maintenance of PRO brigades: the sliding
// monthly billing cycle (see storage/procycle.go). Every pass is a no-op for
// free and VIP brigades (guarded by the PRO flag inside the storage calls).
package probilling

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/vpngen/keydesk/keydesk/storage"
)

const (
	// DefaultSweepInterval - how often the PRO sweep runs.
	DefaultSweepInterval = time.Hour
	// DefaultJitterValue - max start delay, seconds.
	DefaultJitterValue = 60
)

// EnforcementEnabled - while false, invoices are issued and shown but an
// unpaid one has no consequence. Flip to true when real payments arrive:
// then an invoice unpaid past its due date downgrades the brigade to free.
const EnforcementEnabled = false

// RunSweep - periodic PRO maintenance loop (same shape as stat.CollectingData).
func RunSweep(db *storage.BrigadeStorage, kill <-chan struct{}) {
	jit := rand.Int63n(DefaultJitterValue) + 1
	timer := time.NewTimer(time.Duration(jit) * time.Second)

	defer timer.Stop()

	for {
		select {
		case ts := <-timer.C:
			if err := Sweep(db, ts.UTC()); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "PRO sweep: %s\n", err)
			}

			timer.Reset(DefaultSweepInterval)
		case <-kill:
			_, _ = fmt.Fprintln(os.Stderr, "Shutting down PRO sweep...")
			return
		}
	}
}

// Sweep - one PRO maintenance pass: make sure the brigade has its cycle
// anchor, issue invoices for finished cycles, mark overdue ones and (with
// enforcement on) downgrade the brigade.
func Sweep(db *storage.BrigadeStorage, now time.Time) error {
	if err := db.EnsureProSince(now); err != nil {
		return fmt.Errorf("ensure pro since: %w", err)
	}

	issued, err := db.GenerateProInvoice(now)
	if err != nil {
		return fmt.Errorf("generate invoice: %w", err)
	}

	for _, invoice := range issued {
		_, _ = fmt.Fprintf(os.Stderr, "PRO sweep: invoice %s issued (%d keys, %d cents)\n", invoice.ID, invoice.KeysCount, invoice.TotalCents)

		if err := db.EnsureProLedger(); err == nil {
			ev := storage.ProLedgerEvent{Type: storage.ProEvInvoiceIssued, Invoice: invoice.ID, Cents: invoice.TotalCents, Keys: invoice.KeysCount}
			if err := db.AppendProLedger(ev); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "PRO sweep: ledger: %s\n", err)
			}
		}
	}

	if !EnforcementEnabled {
		return nil
	}

	overdue, err := db.SweepProBilling(now)
	if err != nil {
		return fmt.Errorf("billing: %w", err)
	}

	if overdue {
		_, _ = fmt.Fprintln(os.Stderr, "PRO sweep: invoice unpaid past due, brigade downgraded to free")

		if err := db.AppendProLedger(storage.ProLedgerEvent{Type: storage.ProEvDowngraded}); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "PRO sweep: ledger: %s\n", err)
		}

		if err := db.SetPRO(false); err != nil {
			return fmt.Errorf("downgrade: %w", err)
		}
	}

	return nil
}
