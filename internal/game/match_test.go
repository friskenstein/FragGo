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
		matchTime: 2 * time.Minute,
	}

	if got := g.currentSpeedMultiplier(); got != 1.5 {
		t.Fatalf("expected midpoint speed multiplier 1.5, got %.2f", got)
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
