//go:build !linux

package nftables

import "errors"

var ErrNotSupported = errors.New("nftables is only supported on Linux")

type Config struct {
	TableName        string
	SetName          string
	ChainName        string
	ForwardChainName string
	NICs             []string
}

type Manager struct{}

func New(_ Config) (*Manager, error) {
	return nil, ErrNotSupported
}

func (m *Manager) Close() error { return nil }

func (m *Manager) Add(_ string) error { return ErrNotSupported }

func (m *Manager) Remove(_ string) error { return ErrNotSupported }

func (m *Manager) List() ([]string, error) { return nil, ErrNotSupported }
