package game

import (
	"testing"
	"time"

	"github.com/g3n/engine/core"
)

func TestResetWeaponLoadoutStartsWithDefaultWeaponOnly(t *testing.T) {

	g := Game{}
	g.resetWeaponLoadout()

	if g.activeWeapon != weaponBurrowBlaster {
		t.Fatalf("expected burrow blaster to be active, got %d", g.activeWeapon)
	}
	if !g.weapons[weaponBurrowBlaster].unlocked {
		t.Fatal("expected burrow blaster to be unlocked")
	}
	if g.weapons[weaponAcornScatterer].unlocked || g.weapons[weaponTurnipThumper].unlocked {
		t.Fatal("expected special weapons to require pickups")
	}
}

func TestManualReloadTransfersReserveIntoMagazine(t *testing.T) {

	g := Game{}
	g.resetWeaponLoadout()
	state := &g.weapons[weaponBurrowBlaster]
	state.ammoInMagazine = 3
	state.reserveAmmo = 4

	g.startReloadActiveWeapon()
	if state.reloadRemaining <= 0 {
		t.Fatal("expected reload timer to start")
	}

	g.updateWeaponReload(2 * time.Second)
	if state.ammoInMagazine != 7 {
		t.Fatalf("expected partial reserve to load magazine to 7, got %d", state.ammoInMagazine)
	}
	if state.reserveAmmo != 0 {
		t.Fatalf("expected reserve ammo to be consumed, got %d", state.reserveAmmo)
	}
}

func TestWeaponBurstDamageStaysModerate(t *testing.T) {

	for id := weaponID(0); id < weaponCount; id++ {
		spec := weaponSpecs[id]
		pellets := spec.pellets
		if pellets <= 0 {
			pellets = 1
		}
		burstDamage := spec.damage * pellets
		if burstDamage > 60 {
			t.Fatalf("%s burst damage is too high: %d", spec.name, burstDamage)
		}
		if spec.cooldown < 180*time.Millisecond {
			t.Fatalf("%s cooldown is too low: %s", spec.name, spec.cooldown)
		}
	}
}

func TestCarriedWeaponVisibilityFollowsActiveWeapon(t *testing.T) {

	g := Game{
		activeWeapon: weaponTurnipThumper,
	}
	for id := weaponID(0); id < weaponCount; id++ {
		g.playerWeapons[id] = core.NewNode()
	}

	g.updateCarriedWeaponVisibility()
	for id, model := range g.playerWeapons {
		wantVisible := weaponID(id) == weaponTurnipThumper
		if model.Visible() != wantVisible {
			t.Fatalf("weapon %d visibility = %v, want %v", id, model.Visible(), wantVisible)
		}
	}
}
