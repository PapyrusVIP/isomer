package testutil

import (
	"net/netip"

	"github.com/google/go-cmp/cmp"
)

func IPPrefixComparer() cmp.Option {
	return cmp.Comparer(func(x, y netip.Prefix) bool {
		return x == y
	})
}
