package stat

import (
	"log"
	"math/rand"
	"net/netip"
	"path/filepath"
	"time"

	"github.com/vpngen/keydesk/keydesk"
	"github.com/vpngen/keydesk/keydesk/storage"
)

func CollectingData(kill <-chan struct{}, logger *log.Logger, addr netip.AddrPort, rdata bool, brigadeID, dbDir, statsDir string) {
	db := &storage.BrigadeStorage{
		BrigadeID:       brigadeID,
		BrigadeFilename: filepath.Join(dbDir, storage.BrigadeFilename),
		BrigadeSpinlock: filepath.Join(dbDir, storage.BrigadeSpinlockFilename),
		APIAddrPort:     addr,
		BrigadeStorageOpts: storage.BrigadeStorageOpts{
			MaxUsers:               keydesk.MaxUsers,
			MonthlyQuotaRemaining:  keydesk.MonthlyQuotaRemaining,
			MaxUserInctivityPeriod: keydesk.DefaultMaxUserInactivityPeriod,
		},
	}
	if err := db.SelfCheckAndInit(); err != nil {
		log.Fatalf("Storage initialization: %s\n", err)
	}

	statsFilename := filepath.Join(statsDir, storage.StatsFilename)
	statsSpinlock := filepath.Join(statsDir, storage.StatsSpinlockFilename)

	jit := rand.Int63n(DefaultJitterValue) + 1
	timer := time.NewTimer(time.Duration(jit) * time.Second)

	defer timer.Stop()

	for {
		select {
		case ts := <-timer.C:
			logger.Printf("%s: Collecting data: %s: %s\n", ts.UTC().Format(time.RFC3339), brigadeID, statsFilename)

			if err := db.GetStats(rdata, statsFilename, statsSpinlock, keydesk.DefaultEndpointsTTL); err != nil {
				logger.Printf("Error collecting stats: %s\n", err)
			}

			timer.Reset(DefaultStatisticsFetchingDuration)
		case <-kill:
			logger.Println("Shutting down stats...")
			return
		}
	}
}
