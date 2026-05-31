package ban

import (
	"fmt"
	"time"

	"github.com/neko-server-dev/gomanager/internal/nftables"
)

type Entry struct {
	IP        string     `json:"ip"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type Service struct {
	manager *nftables.Manager
	store   *Store
}

func NewService(manager *nftables.Manager, store *Store) (*Service, error) {
	if err := store.Load(); err != nil {
		return nil, err
	}
	return &Service{manager: manager, store: store}, nil
}

func (s *Service) Add(ip string, expiresAt *time.Time) error {
	if err := s.manager.Add(ip); err != nil {
		return err
	}
	if expiresAt != nil {
		if err := s.store.Set(ip, *expiresAt); err != nil {
			_ = s.manager.Remove(ip)
			return fmt.Errorf("save expiration: %w", err)
		}
	} else if err := s.store.Delete(ip); err != nil {
		_ = s.manager.Remove(ip)
		return fmt.Errorf("clear expiration: %w", err)
	}
	return nil
}

func (s *Service) Remove(ip string) error {
	if err := s.manager.Remove(ip); err != nil {
		return err
	}
	return s.store.Delete(ip)
}

func (s *Service) List() ([]Entry, error) {
	ips, err := s.manager.List()
	if err != nil {
		return nil, err
	}

	expirations := s.store.Snapshot()
	entries := make([]Entry, 0, len(ips))
	for _, ip := range ips {
		entry := Entry{IP: ip}
		if t, ok := expirations[ip]; ok {
			t := t
			entry.ExpiresAt = &t
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *Service) CleanupExpired() (int, error) {
	now := time.Now()
	expired := s.store.Expired(now)
	if len(expired) == 0 {
		return 0, nil
	}

	var removed []string
	var failures int
	for _, ip := range expired {
		if err := s.manager.Remove(ip); err != nil {
			failures++
			continue
		}
		removed = append(removed, ip)
	}
	if len(removed) > 0 {
		if err := s.store.DeleteMany(removed); err != nil {
			return len(removed), err
		}
	}
	if failures > 0 {
		return len(removed), fmt.Errorf("remove expired %d IP(s) from nftables", failures)
	}
	return len(removed), nil
}

func ParseTTL(ttl string) (time.Time, error) {
	d, err := time.ParseDuration(ttl)
	if err != nil || d <= 0 {
		return time.Time{}, ErrInvalidTTL
	}
	return time.Now().Add(d), nil
}

func ParseExpiresAt(raw string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, ErrInvalidExpiresAt
	}
	if !t.After(time.Now()) {
		return time.Time{}, ErrInvalidExpiresAt
	}
	return t, nil
}

func (s *Service) RunCleanupLoop(interval time.Duration, onError func(err error)) {
	if interval <= 0 {
		interval = time.Minute
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := s.CleanupExpired(); err != nil && onError != nil {
				onError(err)
			}
		}
	}()
}
