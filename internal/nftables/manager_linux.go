//go:build linux

package nftables

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"
	"syscall"

	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
)

type Config struct {
	TableName        string
	SetName          string
	ChainName        string
	ForwardChainName string
	NICs             []string
}

type Manager struct {
	cfg   Config
	conn  *nftables.Conn
	table *nftables.Table
	set4  *nftables.Set
	set6  *nftables.Set
	setIF *nftables.Set
	mu    sync.Mutex
}

func New(cfg Config) (*Manager, error) {
	conn, err := nftables.New()
	if err != nil {
		return nil, fmt.Errorf("connect to nftables: %w", err)
	}

	m := &Manager{cfg: cfg, conn: conn}
	if err := m.ensureRuleset(); err != nil {
		_ = conn.CloseLasting()
		return nil, err
	}
	return m, nil
}

func (m *Manager) Close() error {
	if m.conn != nil {
		return m.conn.CloseLasting()
	}
	return nil
}

func (m *Manager) ensureRuleset() error {
	tables, err := m.conn.ListTables()
	if err != nil {
		return fmt.Errorf("list tables: %w", wrapNetlinkErr(err))
	}

	tableIsNew := false
	for _, t := range tables {
		if t.Family == nftables.TableFamilyINet && t.Name == m.cfg.TableName {
			m.table = t
			break
		}
	}

	if m.table == nil {
		m.table = m.conn.AddTable(&nftables.Table{
			Family: nftables.TableFamilyINet,
			Name:   m.cfg.TableName,
		})
		tableIsNew = true
	}

	set4Name := m.cfg.SetName + "_v4"
	set6Name := m.cfg.SetName + "_v6"
	setIFName := m.cfg.SetName + "_ifnames"

	if !tableIsNew {
		sets, err := m.conn.GetSets(m.table)
		if err != nil {
			return fmt.Errorf("list sets: %w", wrapNetlinkErr(err))
		}
		for _, s := range sets {
			switch s.Name {
			case set4Name:
				m.set4 = s
			case set6Name:
				m.set6 = s
			case setIFName:
				m.setIF = s
			}
		}
	}

	if m.set4 == nil {
		m.set4 = &nftables.Set{
			Table:   m.table,
			Name:    set4Name,
			KeyType: nftables.TypeIPAddr,
		}
		if err := m.conn.AddSet(m.set4, nil); err != nil {
			return fmt.Errorf("add v4 set: %w", err)
		}
	}

	if m.set6 == nil {
		m.set6 = &nftables.Set{
			Table:   m.table,
			Name:    set6Name,
			KeyType: nftables.TypeIP6Addr,
		}
		if err := m.conn.AddSet(m.set6, nil); err != nil {
			return fmt.Errorf("add v6 set: %w", err)
		}
	}

	if len(m.cfg.NICs) > 0 && m.setIF == nil {
		elements := make([]nftables.SetElement, 0, len(m.cfg.NICs))
		for _, nic := range m.cfg.NICs {
			elements = append(elements, nftables.SetElement{Key: ifNameKey(nic)})
		}
		m.setIF = &nftables.Set{
			Table:        m.table,
			Name:         setIFName,
			KeyType:      nftables.TypeIFName,
			KeyByteOrder: binaryutil.NativeEndian,
		}
		if err := m.conn.AddSet(m.setIF, elements); err != nil {
			return fmt.Errorf("add ifname set: %w", err)
		}
	}

	var chains []*nftables.Chain
	if !tableIsNew {
		chains, err = m.conn.ListChainsOfTableFamily(m.table.Family)
		if err != nil {
			return fmt.Errorf("list chains: %w", wrapNetlinkErr(err))
		}
	}

	if err := m.ensureHookChain(chains, nftables.ChainHookInput, m.cfg.ChainName); err != nil {
		return err
	}
	if err := m.ensureHookChain(chains, nftables.ChainHookForward, m.cfg.ForwardChainName); err != nil {
		return err
	}

	if err := m.conn.Flush(); err != nil {
		return fmt.Errorf("apply nftables ruleset: %w", wrapNetlinkErr(err))
	}

	return nil
}

func (m *Manager) ensureHookChain(chains []*nftables.Chain, hook *nftables.ChainHook, name string) error {
	for _, c := range chains {
		if c.Name == name && c.Table != nil && c.Table.Name == m.cfg.TableName {
			return nil
		}
	}

	chain := m.conn.AddChain(&nftables.Chain{
		Name:     name,
		Table:    m.table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  hook,
		Priority: nftables.ChainPriorityFilter,
	})

	m.conn.AddRule(m.buildRejectRule(chain, m.set4, true))
	m.conn.AddRule(m.buildRejectRule(chain, m.set6, false))
	return nil
}

func (m *Manager) buildRejectRule(chain *nftables.Chain, set *nftables.Set, ipv4 bool) *nftables.Rule {
	proto := byte(unix.NFPROTO_IPV6)
	offset := uint32(8)
	lenBytes := uint32(16)
	if ipv4 {
		proto = byte(unix.NFPROTO_IPV4)
		offset = 12
		lenBytes = 4
	}

	exprs := make([]expr.Any, 0, 7)
	if m.setIF != nil {
		exprs = append(exprs,
			&expr.Meta{
				Key:      expr.MetaKeyIIFNAME,
				Register: 1,
			},
			&expr.Lookup{
				SourceRegister: 1,
				SetName:        m.setIF.Name,
				SetID:          m.setIF.ID,
			},
		)
	}
	exprs = append(exprs,
		&expr.Meta{
			Key:      expr.MetaKeyNFPROTO,
			Register: 1,
		},
		&expr.Cmp{
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     []byte{proto},
		},
		&expr.Payload{
			DestRegister: 1,
			Base:         expr.PayloadBaseNetworkHeader,
			Offset:       offset,
			Len:          lenBytes,
		},
		&expr.Lookup{
			SourceRegister: 1,
			SetName:        set.Name,
			SetID:          set.ID,
		},
		&expr.Reject{
			Type: unix.NFT_REJECT_ICMPX_UNREACH,
			Code: unix.NFT_REJECT_ICMPX_ADMIN_PROHIBITED,
		},
	)

	return &nftables.Rule{
		Table: m.table,
		Chain: chain,
		Exprs: exprs,
	}
}

func ifNameKey(name string) []byte {
	b := make([]byte, 16)
	copy(b, name)
	return b
}

func wrapNetlinkErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, syscall.ENOENT) || strings.Contains(err.Error(), "no such file or directory") {
		return fmt.Errorf("%w: nftables が利用できません (root で実行、modprobe nf_tables、CONFIG_NF_TABLES の有効化を確認してください)", err)
	}
	return err
}

func (m *Manager) Add(ipStr string) error {
	ip, err := parseIP(ipStr)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	set, key, err := m.setForIP(ip)
	if err != nil {
		return err
	}

	if err := m.conn.SetAddElements(set, []nftables.SetElement{{Key: key}}); err != nil {
		return fmt.Errorf("add to blacklist: %w", err)
	}
	if err := m.conn.Flush(); err != nil {
		return fmt.Errorf("apply blacklist add: %w", err)
	}
	return nil
}

func (m *Manager) Remove(ipStr string) error {
	ip, err := parseIP(ipStr)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	set, key, err := m.setForIP(ip)
	if err != nil {
		return err
	}

	elements, err := m.conn.GetSetElements(set)
	if err != nil {
		return fmt.Errorf("list blacklist: %w", err)
	}

	found := false
	for _, el := range elements {
		if net.IP(el.Key).Equal(ip) {
			found = true
			break
		}
	}
	if !found {
		return ErrNotFound
	}

	if err := m.conn.SetDeleteElements(set, []nftables.SetElement{{Key: key}}); err != nil {
		return fmt.Errorf("remove from blacklist: %w", err)
	}
	if err := m.conn.Flush(); err != nil {
		return fmt.Errorf("apply blacklist remove: %w", err)
	}
	return nil
}

func (m *Manager) List() ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var ips []string

	for _, set := range []*nftables.Set{m.set4, m.set6} {
		elements, err := m.conn.GetSetElements(set)
		if err != nil {
			return nil, fmt.Errorf("list blacklist: %w", err)
		}
		for _, el := range elements {
			ips = append(ips, net.IP(el.Key).String())
		}
	}

	return ips, nil
}

func (m *Manager) setForIP(ip net.IP) (*nftables.Set, []byte, error) {
	if ip4 := ip.To4(); ip4 != nil {
		return m.set4, ip4, nil
	}
	if ip.To16() != nil {
		return m.set6, ip, nil
	}
	return nil, nil, ErrInvalidIP
}

func parseIP(s string) (net.IP, error) {
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidIP, s)
	}
	return net.IP(addr.AsSlice()), nil
}
