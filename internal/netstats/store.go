package netstats

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type persistedInterface struct {
	LifetimeRxBytes   uint64 `json:"lifetime_rx_bytes"`
	LifetimeTxBytes   uint64 `json:"lifetime_tx_bytes"`
	LifetimeRxPackets uint64 `json:"lifetime_rx_packets"`
	LifetimeTxPackets uint64 `json:"lifetime_tx_packets"`
	LastKernelRxBytes uint64 `json:"last_kernel_rx_bytes"`
	LastKernelTxBytes uint64 `json:"last_kernel_tx_bytes"`
	LastKernelRxPackets uint64 `json:"last_kernel_rx_packets"`
	LastKernelTxPackets uint64 `json:"last_kernel_tx_packets"`
}

type persistedData struct {
	Interfaces map[string]persistedInterface `json:"interfaces"`
	UpdatedAt  time.Time                     `json:"updated_at,omitempty"`
}

type Store struct {
	path string
	mu   sync.Mutex
	data persistedData
}

func NewStore(configPath string) *Store {
	dir := filepath.Dir(configPath)
	base := filepath.Base(configPath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	if name == "" {
		name = "gomanager"
	}
	return &Store{
		path: filepath.Join(dir, name+".bandwidth.json"),
		data: persistedData{Interfaces: make(map[string]persistedInterface)},
	}
}

func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.data = persistedData{Interfaces: make(map[string]persistedInterface)}
			return nil
		}
		return fmt.Errorf("read bandwidth store: %w", err)
	}

	var stored persistedData
	if err := json.Unmarshal(data, &stored); err != nil {
		return fmt.Errorf("parse bandwidth store: %w", err)
	}
	if stored.Interfaces == nil {
		stored.Interfaces = make(map[string]persistedInterface)
	}
	s.data = stored
	return nil
}

func (s *Store) Merge(kernel []InterfaceStats) ([]InterfaceStats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]InterfaceStats, 0, len(kernel))
	for _, k := range kernel {
		p, seen := s.data.Interfaces[k.Name]
		deltaRx, deltaTx, deltaRxPkt, deltaTxPkt := kernelDelta(k, p, seen)

		p.LifetimeRxBytes += deltaRx
		p.LifetimeTxBytes += deltaTx
		p.LifetimeRxPackets += deltaRxPkt
		p.LifetimeTxPackets += deltaTxPkt
		p.LastKernelRxBytes = k.RxBytes
		p.LastKernelTxBytes = k.TxBytes
		p.LastKernelRxPackets = k.RxPackets
		p.LastKernelTxPackets = k.TxPackets
		s.data.Interfaces[k.Name] = p

		out = append(out, InterfaceStats{
			Name:           k.Name,
			RxBytes:        p.LifetimeRxBytes,
			TxBytes:        p.LifetimeTxBytes,
			RxPackets:      p.LifetimeRxPackets,
			TxPackets:      p.LifetimeTxPackets,
			SessionRxBytes: k.RxBytes,
			SessionTxBytes: k.TxBytes,
			SessionRxPackets: k.RxPackets,
			SessionTxPackets: k.TxPackets,
		})
	}

	s.data.UpdatedAt = time.Now()
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return out, nil
}

func kernelDelta(k InterfaceStats, p persistedInterface, seen bool) (rx, tx, rxPkt, txPkt uint64) {
	if !seen {
		return 0, 0, 0, 0
	}
	if k.RxBytes < p.LastKernelRxBytes {
		return k.RxBytes, k.TxBytes, k.RxPackets, k.TxPackets
	}
	return k.RxBytes - p.LastKernelRxBytes,
		k.TxBytes - p.LastKernelTxBytes,
		k.RxPackets - p.LastKernelRxPackets,
		k.TxPackets - p.LastKernelTxPackets
}

func (s *Store) saveLocked() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal bandwidth store: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("write bandwidth store: %w", err)
	}
	return nil
}
