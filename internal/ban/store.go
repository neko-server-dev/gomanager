package ban

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Store struct {
	path string
	mu   sync.Mutex
	data map[string]time.Time
}

func NewStore(configPath string) *Store {
	dir := filepath.Dir(configPath)
	base := filepath.Base(configPath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	if name == "" {
		name = "goban"
	}
	return &Store{
		path: filepath.Join(dir, name+".expires.json"),
		data: make(map[string]time.Time),
	}
}

func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.data = make(map[string]time.Time)
			return nil
		}
		return fmt.Errorf("read expiration store: %w", err)
	}

	var raw map[string]time.Time
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse expiration store: %w", err)
	}
	if raw == nil {
		raw = make(map[string]time.Time)
	}
	s.data = raw
	return nil
}

func (s *Store) saveLocked() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal expiration store: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("write expiration store: %w", err)
	}
	return nil
}

func (s *Store) Set(ip string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[ip] = expiresAt
	return s.saveLocked()
}

func (s *Store) Delete(ip string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, ip)
	return s.saveLocked()
}

func (s *Store) Get(ip string) (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.data[ip]
	return t, ok
}

func (s *Store) Snapshot() map[string]time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]time.Time, len(s.data))
	for ip, t := range s.data {
		out[ip] = t
	}
	return out
}

func (s *Store) Expired(now time.Time) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var ips []string
	for ip, t := range s.data {
		if !t.After(now) {
			ips = append(ips, ip)
		}
	}
	return ips
}

func (s *Store) DeleteMany(ips []string) error {
	if len(ips) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ip := range ips {
		delete(s.data, ip)
	}
	return s.saveLocked()
}
