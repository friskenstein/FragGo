package game

import (
	"testing"
	"time"
)

func TestCurrentSpeedMultiplierInterpolatesAcrossRound(t *testing.T) {

	g := Game{
		matchConfig: matchConfig{
			RoundDuration: 4 * time.Minute,
			StartSpeed:    1.0,
			EndSpeed:      2.0,
		},
		roundElapsed: 2 * time.Minute,
	}

	if got := g.currentSpeedMultiplier(); got != 1.5 {
		t.Fatalf("expected midpoint speed multiplier 1.5, got %.2f", got)
	}
}

func TestCurrentSpeedMultiplierIgnoresSimulationElapsed(t *testing.T) {

	g := Game{
		matchConfig: matchConfig{
			RoundDuration: 4 * time.Minute,
			StartSpeed:    1.0,
			EndSpeed:      2.0,
		},
		roundElapsed: 1 * time.Minute,
		matchTime:    3 * time.Minute,
	}

	if got := g.currentSpeedMultiplier(); got != 1.25 {
		t.Fatalf("expected speed multiplier to follow round elapsed time only, got %.2f", got)
	}
}

func TestDesiredBotCountRespectsFillSetting(t *testing.T) {

	g := Game{
		matchConfig: matchConfig{
			PlayerSlots: 6,
			FillBots:    true,
		},
	}

	if got := g.desiredBotCount(); got != 5 {
		t.Fatalf("expected 5 bot seats, got %d", got)
	}

	g.matchConfig.FillBots = false
	if got := g.desiredBotCount(); got != 0 {
		t.Fatalf("expected 0 bot seats when disabled, got %d", got)
	}
}
