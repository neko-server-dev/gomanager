package netstats

import (
	"sync"
	"time"
)

type InterfaceStats struct {
	Name           string   `json:"name"`
	RxBytes        uint64   `json:"rx_bytes"`
	TxBytes        uint64   `json:"tx_bytes"`
	RxPackets      uint64   `json:"rx_packets"`
	TxPackets      uint64   `json:"tx_packets"`
	SessionRxBytes uint64   `json:"session_rx_bytes"`
	SessionTxBytes uint64   `json:"session_tx_bytes"`
	SessionRxPackets uint64 `json:"session_rx_packets"`
	SessionTxPackets uint64 `json:"session_tx_packets"`
	RxBytesPerSec  *float64 `json:"rx_bytes_per_sec,omitempty"`
	TxBytesPerSec  *float64 `json:"tx_bytes_per_sec,omitempty"`
}

type Totals struct {
	RxBytes          uint64   `json:"rx_bytes"`
	TxBytes          uint64   `json:"tx_bytes"`
	RxPackets        uint64   `json:"rx_packets"`
	TxPackets        uint64   `json:"tx_packets"`
	SessionRxBytes   uint64   `json:"session_rx_bytes"`
	SessionTxBytes   uint64   `json:"session_tx_bytes"`
	SessionRxPackets uint64   `json:"session_rx_packets"`
	SessionTxPackets uint64   `json:"session_tx_packets"`
	RxBytesPerSec    *float64 `json:"rx_bytes_per_sec,omitempty"`
	TxBytesPerSec    *float64 `json:"tx_bytes_per_sec,omitempty"`
}

type Usage struct {
	Interfaces  []InterfaceStats `json:"interfaces"`
	Total       Totals           `json:"total"`
	CollectedAt time.Time        `json:"collected_at"`
}

type Collector struct {
	nics  []string
	store *Store

	mu     sync.Mutex
	prev   []InterfaceStats
	prevAt time.Time
}

func NewCollector(configPath string, nics []string) (*Collector, error) {
	store := NewStore(configPath)
	if err := store.Load(); err != nil {
		return nil, err
	}
	return &Collector{
		nics:  append([]string(nil), nics...),
		store: store,
	}, nil
}

func (c *Collector) Bandwidth() (Usage, error) {
	kernel, err := readInterfaceStats(c.nics)
	if err != nil {
		return Usage{}, err
	}

	stats, err := c.store.Merge(kernel)
	if err != nil {
		return Usage{}, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if !c.prevAt.IsZero() {
		elapsed := now.Sub(c.prevAt).Seconds()
		if elapsed > 0 {
			prevByName := make(map[string]InterfaceStats, len(c.prev))
			for _, s := range c.prev {
				prevByName[s.Name] = s
			}
			for i := range stats {
				prev, ok := prevByName[stats[i].Name]
				if !ok {
					continue
				}
				rxRate := float64(stats[i].RxBytes-prev.RxBytes) / elapsed
				txRate := float64(stats[i].TxBytes-prev.TxBytes) / elapsed
				if rxRate < 0 {
					rxRate = 0
				}
				if txRate < 0 {
					txRate = 0
				}
				stats[i].RxBytesPerSec = &rxRate
				stats[i].TxBytesPerSec = &txRate
			}
		}
	}

	c.prev = cloneStats(stats)
	c.prevAt = now

	if stats == nil {
		stats = []InterfaceStats{}
	}

	return Usage{
		Interfaces:  stats,
		Total:       sumTotals(stats),
		CollectedAt: now,
	}, nil
}

func sumTotals(stats []InterfaceStats) Totals {
	var total Totals
	for _, s := range stats {
		total.RxBytes += s.RxBytes
		total.TxBytes += s.TxBytes
		total.RxPackets += s.RxPackets
		total.TxPackets += s.TxPackets
		total.SessionRxBytes += s.SessionRxBytes
		total.SessionTxBytes += s.SessionTxBytes
		total.SessionRxPackets += s.SessionRxPackets
		total.SessionTxPackets += s.SessionTxPackets
		if s.RxBytesPerSec != nil {
			if total.RxBytesPerSec == nil {
				v := 0.0
				total.RxBytesPerSec = &v
			}
			*total.RxBytesPerSec += *s.RxBytesPerSec
		}
		if s.TxBytesPerSec != nil {
			if total.TxBytesPerSec == nil {
				v := 0.0
				total.TxBytesPerSec = &v
			}
			*total.TxBytesPerSec += *s.TxBytesPerSec
		}
	}
	return total
}

func cloneStats(stats []InterfaceStats) []InterfaceStats {
	out := make([]InterfaceStats, len(stats))
	copy(out, stats)
	return out
}
