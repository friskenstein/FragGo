package game

import "testing"

func TestArenaScaleForIDUsesOverrides(t *testing.T) {

	if got := arenaScaleForID("dust2"); got != 2.0 {
		t.Fatalf("expected dust2 scale override of 2.0, got %.2f", got)
	}

	if got := arenaScaleForID("arena"); got != 1.0 {
		t.Fatalf("expected default arena scale of 1.0, got %.2f", got)
	}
}

func TestDefaultArenaIDPrefersDust2(t *testing.T) {

	arenas := []arenaDefinition{
		{ID: "arena"},
		{ID: "dust2"},
		{ID: "turbine"},
	}

	if got := defaultArenaID(arenas); got != "dust2" {
		t.Fatalf("expected dust2 default arena, got %q", got)
	}
}

func TestDefaultArenaIDFallsBackToFirstArena(t *testing.T) {

	arenas := []arenaDefinition{
		{ID: "arena"},
		{ID: "blockout"},
	}

	if got := defaultArenaID(arenas); got != "arena" {
		t.Fatalf("expected first arena fallback, got %q", got)
	}
}

func TestFormatArenaLabel(t *testing.T) {

	if got := formatArenaLabel("dust2"); got != "Dust2" {
		t.Fatalf("expected Dust2 label, got %q", got)
	}

	if got := formatArenaLabel("test_map"); got != "Test Map" {
		t.Fatalf("expected Test Map label, got %q", got)
	}
}
