package internal

import (
	"fmt"
	"unsafe"
	"net/netip"
)

// A Binding selects which packets to redirect.
//
// You have to add a Binding to a Dispatcher for it to take effect.
type Binding struct {
	Label    string
	Protocol Protocol
	Prefix   netip.Prefix
	Port     uint16
}

// NewBinding creates a new binding.
//
// prefix may either be in CIDR notation (::1/128) or a plain IP address.
// Specifying ::1 is equivalent to passing ::1/128.
func NewBinding(label string, proto Protocol, prefix string, port uint16) (*Binding, error) {
	cidr, err := ParsePrefix(prefix)
	if err != nil {
		return nil, err
	}

	return &Binding{
		label,
		proto,
		cidr,
		port,
	}, nil
}

// ParsePrefix parses a prefix with an optional mask into an netip.Prefix.
//
// A missing prefix is interpreted as a /128 or /32.
func ParsePrefix(p string) (netip.Prefix, error) {
	addr, err := netip.ParseAddr(p)
	if err == nil {
		return netip.PrefixFrom(addr, addr.BitLen()), nil
	}

	return netip.ParsePrefix(p)
}

// MustParsePrefix calls ParsePrefix and panics on error. It is intended for use in tests with hard-coded strings.
func MustParsePrefix(p string) netip.Prefix {
	prefix, err := ParsePrefix(p)
	if err != nil {
		panic(err)
	}

	return prefix
}


func newBindingFromBPF(label string, key *bindingKey) *Binding {
	ones := uint8(key.PrefixLen) - bindingKeyHeaderBits
	ip := netip.AddrFrom16(key.IP).Unmap()

	var prefix netip.Prefix
	if ip.Is4() {
		prefix = netip.PrefixFrom(ip, int(ones-96))
	} else {
		prefix = netip.PrefixFrom(ip, int(ones))
	}

	return &Binding{
		label,
		key.Protocol,
		prefix,
		key.Port,
	}
}

func (b *Binding) String() string {
	return fmt.Sprintf("%s#%v:[%s]:%d", b.Label, b.Protocol, b.Prefix, b.Port)
}

// bindingKey mirrors struct addr
type bindingKey struct {
	PrefixLen uint32
	Protocol  Protocol
	Port      uint16
	IP        [16]byte
}

const bindingKeyHeaderBits = uint8(unsafe.Sizeof(bindingKey{}.Protocol)+unsafe.Sizeof(bindingKey{}.Port)) * 8

func newBindingKey(bind *Binding) *bindingKey {
	// Get the length of the prefix
	prefixLen := bind.Prefix.Bits()

	// If the prefix is v4, offset it by 96
	if bind.Prefix.Addr().Is4() {
		prefixLen += 96
	}

	key := bindingKey{
		PrefixLen: uint32(bindingKeyHeaderBits + uint8(prefixLen)),
		Protocol:  bind.Protocol,
		Port:      bind.Port,
		IP:        bind.Prefix.Addr().As16(),
	}

	return &key
}

// bindingValue mirrors struct binding.
type bindingValue struct {
	ID        destinationID
	PrefixLen uint32
}

// Bindings is a list of bindings.
//
// They may be sorted using sort.Sort in the order of precedence used by the
// data plane, with the most specific binding at the start of the list.
type Bindings []*Binding

func (sb Bindings) Len() int      { return len(sb) }
func (sb Bindings) Swap(i, j int) { sb[i], sb[j] = sb[j], sb[i] }
func (sb Bindings) Less(i, j int) bool {
	a, b := sb[i], sb[j]

	if a.Protocol != b.Protocol {
		return a.Protocol < b.Protocol
	}

	if a.Prefix.Addr().Is4() != b.Prefix.Addr().Is4() {
		return a.Prefix.Addr().Is4()
	}

	// We only care to sort on overlap if the prefix length is different
	if a.Prefix.Bits() != b.Prefix.Bits() && a.Prefix.Overlaps(b.Prefix) {
		// Both prefixes overlap, like fd::/64 and fd::1. The longer prefix
		// is more specific.
		return a.Prefix.Bits() > b.Prefix.Bits()
	}

	if c := a.Prefix.Addr().Compare(b.Prefix.Addr()); c != 0 {
		// Prefixes don't share a prefix, use lexicographical order.
		return c < 0
	}

	// Prefixes are identical, discern by port.
	if a.Port != b.Port {
		if a.Port == 0 || b.Port == 0 {
			// Wildcard is less specific than a real port.
			return a.Port > b.Port
		}

		// No wildcard, low ports go first.
		return a.Port < b.Port
	}

	return a.Label < b.Label
}

func (bindings Bindings) metrics() map[Destination]uint64 {
	metrics := map[Destination]uint64{}

	for _, b := range bindings {
		label := b.Label
		domain := AF_INET
		if b.Prefix.Addr().Unmap().Is6() {
			domain = AF_INET6
		}
		protocol := b.Protocol

		metrics[Destination{label, domain, protocol}]++
	}
	return metrics
}

func diffBindings(have, want map[bindingKey]string) (added, removed Bindings) {
	for key, label := range want {
		if have[key] != label {
			added = append(added, newBindingFromBPF(label, &key))
		}
	}

	for key, label := range have {
		if want[key] == "" {
			removed = append(removed, newBindingFromBPF(label, &key))
		}
	}

	return
}
