//go:build !linux

package netstats

func readInterfaceStats(_ []string) ([]InterfaceStats, error) {
	return nil, ErrNotSupported
}
