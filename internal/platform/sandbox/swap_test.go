package sandbox

import (
	"strings"
	"testing"
)

func TestSwapIsOffUnlessAllowedAndAlwaysCapped(t *testing.T) {
	const gib = 1 << 30

	spec := Spec{Workspace: "/w", Argv: []string{"true"}}

	off := (&Docker{cfg: Config{Limits: Limits{MemoryBytes: 2 * gib}}}).createOptions(spec, nil)
	if got := off.HostConfig.MemorySwap; got != 2*gib {
		t.Errorf("without a swap allowance memory plus swap equals memory, got %d", got)
	}

	on := (&Docker{cfg: Config{Limits: Limits{MemoryBytes: 3 * gib, SwapBytes: 3 * gib}}}).createOptions(spec, nil)
	if on.HostConfig.Memory != 3*gib || on.HostConfig.MemorySwap != 6*gib {
		t.Errorf("memory %d, memory plus swap %d: want 3 GiB and 6 GiB", on.HostConfig.Memory, on.HostConfig.MemorySwap)
	}
}

func TestContainerNamesAreDeterministicAndDistinct(t *testing.T) {
	d := &Docker{cfg: Config{Owner: "78db9bede047"}}

	first := d.createOptions(Spec{Workspace: "/w", Argv: []string{"true"}}, nil).Name
	second := d.createOptions(Spec{Workspace: "/w", Argv: []string{"true"}}, nil).Name
	site := d.createOptions(Spec{Workspace: "/w", Site: "/s", Argv: []string{"true"}}, nil).Name

	if !strings.HasPrefix(first, "agentic-sandbox-78db9b-") || first == second {
		t.Fatalf("expected distinct agentic-sandbox-78db9b-<seq> names, got %q and %q", first, second)
	}

	if !strings.HasPrefix(site, "agentic-site-78db9b-") {
		t.Fatalf("expected agentic-site-78db9b-<seq> name, got %q", site)
	}
}
