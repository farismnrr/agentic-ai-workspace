package sandbox

import "testing"

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
