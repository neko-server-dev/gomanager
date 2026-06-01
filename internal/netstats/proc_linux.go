//go:build linux

package netstats

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func readInterfaceStats(nics []string) ([]InterfaceStats, error) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return nil, fmt.Errorf("read /proc/net/dev: %w", err)
	}
	return parseProcNetDev(data, nics)
}

func parseProcNetDev(data []byte, nics []string) ([]InterfaceStats, error) {
	filter := nicFilter(nics)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0
	var stats []InterfaceStats

	for scanner.Scan() {
		lineNum++
		if lineNum <= 2 {
			continue
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		name, fields, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if !filter(name) {
			continue
		}

		parts := strings.Fields(strings.TrimSpace(fields))
		if len(parts) < 16 {
			return nil, fmt.Errorf("parse /proc/net/dev for %q: expected 16 fields, got %d", name, len(parts))
		}

		rxBytes, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse rx_bytes for %q: %w", name, err)
		}
		rxPackets, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse rx_packets for %q: %w", name, err)
		}
		txBytes, err := strconv.ParseUint(parts[8], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse tx_bytes for %q: %w", name, err)
		}
		txPackets, err := strconv.ParseUint(parts[9], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse tx_packets for %q: %w", name, err)
		}

		stats = append(stats, InterfaceStats{
			Name:      name,
			RxBytes:   rxBytes,
			TxBytes:   txBytes,
			RxPackets: rxPackets,
			TxPackets: txPackets,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan /proc/net/dev: %w", err)
	}

	return stats, nil
}

func nicFilter(nics []string) func(string) bool {
	if len(nics) == 0 {
		return func(name string) bool {
			return name != "lo"
		}
	}

	allowed := make(map[string]struct{}, len(nics))
	for _, nic := range nics {
		allowed[nic] = struct{}{}
	}
	return func(name string) bool {
		_, ok := allowed[name]
		return ok
	}
}
