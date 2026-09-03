// Package probilling - background maintenance of PRO brigades: expiry of
// paid keys now, invoice lifecycle later. Every pass is a no-op for free
// and VIP brigades (guarded by the PRO flag inside the storage calls).
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

// EnforcementEnabled - PRO-lite switch. While false, invoices are still
// issued (informational) but nothing is ever blocked: expired paid keys keep
// working and an unpaid invoice has no consequences. Flip to true when real
// payments arrive.
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

// Sweep - one PRO maintenance pass: block paid keys whose paid period is
// over, issue the monthly invoice and advance the billing lifecycle
// (issued → overdue → suspended).
func Sweep(db *storage.BrigadeStorage, now time.Time) error {
	if EnforcementEnabled {
		expired, err := db.ListProExpired(now)
		if err != nil {
			return fmt.Errorf("list expired: %w", err)
		}

		for _, id := range expired {
			_, _ = fmt.Fprintf(os.Stderr, "PRO sweep: paid period is over, blocking %s\n", id)

			if err := db.BlockUserPro(id, storage.ProBlockExpired); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "PRO sweep: block %s: %s\n", id, err)
			}
		}
	}

	issued, err := db.GenerateProInvoice(now)
	if err != nil {
		return fmt.Errorf("generate invoice: %w", err)
	}

	if issued {
		_, _ = fmt.Fprintf(os.Stderr, "PRO sweep: invoice %s issued\n", now.Format("2006-01"))
	}

	if !EnforcementEnabled {
		return nil
	}

	toBlock, err := db.SweepProBilling(now)
	if err != nil {
		return fmt.Errorf("billing: %w", err)
	}

	for _, id := range toBlock {
		_, _ = fmt.Fprintf(os.Stderr, "PRO sweep: billing suspended, blocking %s\n", id)

		if err := db.BlockUserPro(id, storage.ProBlockBilling); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "PRO sweep: block %s: %s\n", id, err)
		}
	}

	return nil
}
