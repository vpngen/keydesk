package stat

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
)

func CollectingData(db *storage.BrigadeStorage, kill <-chan struct{}, rdata bool, statsDir string) {
	statsFilename := filepath.Join(statsDir, storage.StatsFilename)
	statsSpinlock := filepath.Join(statsDir, storage.StatsSpinlockFilename)

	jit := rand.Int63n(DefaultJitterValue) + 1
	timer := time.NewTimer(time.Duration(jit) * time.Second)

	defer timer.Stop()

	for {
		select {
		case ts := <-timer.C:
			_, _ = fmt.Fprintf(os.Stdout, "%s: Collecting data: %s: %s\n", ts.UTC().Format(time.RFC3339), db.BrigadeID, statsFilename)

			if err := db.GetStats(rdata, statsFilename, statsSpinlock, keydesk.DefaultEndpointsTTL); err != nil {
				_, _ = fmt.Fprintf(os.Stdout, "Error collecting stats: %s\n", err)
			}

			timer.Reset(DefaultStatisticsFetchingDuration)
		case <-kill:
			_, _ = fmt.Fprintln(os.Stdout, "Shutting down stats...")
			return
		}
	}
}
