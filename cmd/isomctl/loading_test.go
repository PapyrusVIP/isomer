package main

import (
	"strings"
	"testing"

	"github.com/PapyrusVIP/isomer/internal"
	"github.com/PapyrusVIP/isomer/internal/testutil"
)

func TestLoadUnload(t *testing.T) {
	netns := testutil.NewNetNS(t)

	load := isomctlTestCall{
		NetNS:     netns,
		Cmd:       "load",
		Effective: internal.CreateCapabilities,
	}
	load.MustRun(t)

	mustTestIsomctl(t, netns, "unload")
}

func TestUpgrade(t *testing.T) {
	netns := mustReadyNetNS(t)

	upgrade := isomctlTestCall{
		NetNS:     netns,
		Cmd:       "upgrade",
		Effective: internal.CreateCapabilities,
	}

	output := upgrade.MustRun(t)
	if !strings.Contains(output.String(), Version) {
		t.Error("Output doesn't contain version")
	}
}
