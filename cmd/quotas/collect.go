package main

import (
	"fmt"
	"net/netip"
	"os"
	"sync"
	"time"
)

func collectingData(debug bool, wg *sync.WaitGroup, kill <-chan struct{}, brigadeID, quotaDir string, addr netip.Addr, wgPub []byte) {
	defer wg.Wait()

	timer := time.NewTimer(time.Second)

	for {
		select {
		case ts := <-timer.C:
			fmt.Fprintf(os.Stderr, "%s: Collecting data: %s: %s\n", ts.UTC().Format(time.RFC3339), brigadeID, addr)

			timer.Reset(DefaultStatisticsFetchingDuration)
		case <-kill:
			if !timer.Stop() {
				<-timer.C
			}

			return
		}
	}
}
