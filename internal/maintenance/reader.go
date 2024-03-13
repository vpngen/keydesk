package maintenance

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func readMaintenance(scanner *bufio.Scanner) (time.Time, string, error) {
	// read timestamp
	if !scanner.Scan() {
		return time.Time{}, "", scanner.Err()
	}

	timestamp, err := strconv.Atoi(scanner.Text())
	var till time.Time
	if err == nil {
		till = time.Unix(int64(timestamp), 0)
	}
	// read message
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if scanner.Err() != nil {
		return time.Time{}, "", fmt.Errorf("scan message: %w", scanner.Err())
	}

	return till, strings.Join(lines, "\n"), nil
}
