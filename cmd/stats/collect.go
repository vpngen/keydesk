package main

import (
	"fmt"
	"math/rand"
	"net/netip"
	"os"
	"sync"
	"time"
)

func collectingData(
	debug, fakeaddr bool,
	wg *sync.WaitGroup,
	kill <-chan struct{},
	done chan<- struct{},
	brigadeID,
	workFilename, brigadeFilename, counterFilename, quotasFilename, statsFilename string,
	addr netip.Addr,
	wgPub []byte,
) {
	close(done)
	defer wg.Done()

	jit := rand.Int63n(DefaultJitterValue) + 1
	timer := time.NewTimer(time.Duration(jit) * time.Second)

	for {
		select {
		case ts := <-timer.C:
			fmt.Fprintf(os.Stderr, "%s: Collecting data: %s: %s\n", ts.UTC().Format(time.RFC3339), brigadeID, addr)

			fmt.Fprintf(os.Stderr, "Working file: %s\n", workFilename)
			fmt.Fprintf(os.Stderr, "Brigade file: %s\n", brigadeFilename)
			fmt.Fprintf(os.Stderr, "Counter file: %s\n", counterFilename)
			fmt.Fprintf(os.Stderr, "Quotas file: %s\n", quotasFilename)
			fmt.Fprintf(os.Stderr, "Statistics file: %s\n", statsFilename)

			timer.Reset(DefaultStatisticsFetchingDuration)
		case <-kill:
			if !timer.Stop() {
				<-timer.C
			}

			return
		}
	}
}
