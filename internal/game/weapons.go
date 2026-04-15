package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/window"
)

const (
	weaponPickupRadius  = 2.1
	weaponPickupRespawn = 24 * time.Second
	ammoPickupRespawn   = 14 * time.Second
)

type weaponID int

const (
	weaponBurrowBlaster weaponID = iota
	weaponAcornScatterer
	weaponTurnipThumper
	weaponCount
)

type weaponSpec struct {
	id             weaponID
	slot           int
	name           string
	shortName      string
	pickupName     string
	damage         int
	rangeMeters    float32
	cooldown       time.Duration
	reloadDuration time.Duration
	magazineSize   int
	startReserve   int
	maxReserve     int
	pickupAmmo     int
	pellets        int
	spreadRadians  float32
	color          math32.Color
}

type weaponState struct {
	unlocked        bool
	ammoInMagazine  int
	reserveAmmo     int
	reloadRemaining time.Duration
}

type weaponPickupKind int

const (
	weaponPickupUnlock weaponPickupKind = iota
	weaponPickupAmmo
)

type weaponPickup struct {
	kind     weaponPickupKind
	weaponID weaponID
	position math32.Vector3
	root     *core.Node
	respawn  time.Duration
	phase    float32
	homeY    float32
	cooldown time.Duration
}

var weaponSpecs = [weaponCount]weaponSpec{
	weaponBurrowBlaster: {
		id:             weaponBurrowBlaster,
		slot:           1,
		name:           "Burrow Blaster",
		shortName:      "Blaster",
		pickupName:     "spare tunnel batteries",
		damage:         24,
		rangeMeters:    86,
		cooldown:       190 * time.Millisecond,
		reloadDuration: 1050 * time.Millisecond,
		magazineSize:   10,
		startReserve:   40,
		maxReserve:     60,
		pickupAmmo:     20,
		pellets:        1,
		color:          math32.Color{R: 0.18, G: 0.78, B: 1.0},
	},
	weaponAcornScatterer: {
		id:             weaponAcornScatterer,
		slot:           2,
		name:           "Acorn Scatterer",
		shortName:      "Acorns",
		pickupName:     "jar of tactical acorns",
		damage:         8,
		rangeMeters:    34,
		cooldown:       720 * time.Millisecond,
		reloadDuration: 1300 * time.Millisecond,
		magazineSize:   4,
		maxReserve:     20,
		pickupAmmo:     8,
		pellets:        6,
		spreadRadians:  0.065,
		color:          math32.Color{R: 1.0, G: 0.58, B: 0.16},
	},
	weaponTurnipThumper: {
		id:             weaponTurnipThumper,
		slot:           3,
		name:           "Turnip Thumper",
		shortName:      "Turnips",
		pickupName:     "suspicious root vegetable",
		damage:         52,
		rangeMeters:    70,
		cooldown:       900 * time.Millisecond,
		reloadDuration: 1450 * time.Millisecond,
		magazineSize:   3,
		maxReserve:     9,
		pickupAmmo:     3,
		pellets:        1,
		color:          math32.Color{R: 0.72, G: 0.92, B: 0.33},
	},
}

func (g *Game) resetWeaponLoadout() {

	for id := weaponID(0); id < weaponCount; id++ {
		spec := weaponSpecs[id]
		g.weapons[id] = weaponState{
			unlocked:       id == weaponBurrowBlaster,
			ammoInMagazine: spec.magazineSize,
			reserveAmmo:    spec.startReserve,
		}
	}
	g.activeWeapon = weaponBurrowBlaster
	g.updateCarriedWeaponVisibility()
}

func (g *Game) activeWeaponSpec() weaponSpec {

	if g.activeWeapon < 0 || g.activeWeapon >= weaponCount {
		return weaponSpecs[weaponBurrowBlaster]
	}
	return weaponSpecs[g.activeWeapon]
}

func (g *Game) switchWeapon(id weaponID) {

	if id < 0 || id >= weaponCount {
		return
	}

	spec := weaponSpecs[id]
	if !g.weapons[id].unlocked {
		g.setStatus(fmt.Sprintf("Find the %s pickup first", spec.name), 900*time.Millisecond)
		return
	}

	if id == g.activeWeapon {
		return
	}

	current := &g.weapons[g.activeWeapon]
	current.reloadRemaining = 0
	g.activeWeapon = id
	g.fireCooldown = 0
	g.updateCarriedWeaponVisibility()
	g.setStatus(fmt.Sprintf("Switched to %s", spec.name), 700*time.Millisecond)
}

func weaponIDForKey(key window.Key) (weaponID, bool) {

	switch key {
	case window.Key1:
		return weaponBurrowBlaster, true
	case window.Key2:
		return weaponAcornScatterer, true
	case window.Key3:
		return weaponTurnipThumper, true
	}
	return 0, false
}

func (g *Game) startReloadActiveWeapon() {

	spec := g.activeWeaponSpec()
	state := &g.weapons[spec.id]
	if !state.unlocked {
		return
	}
	if state.reloadRemaining > 0 {
		g.setStatus(fmt.Sprintf("Reloading %s", spec.name), 500*time.Millisecond)
		return
	}
	if state.ammoInMagazine >= spec.magazineSize {
		g.setStatus(fmt.Sprintf("%s is already full", spec.name), 650*time.Millisecond)
		return
	}
	if state.reserveAmmo <= 0 {
		g.setStatus(fmt.Sprintf("No spare ammo for %s", spec.name), 850*time.Millisecond)
		return
	}

	state.reloadRemaining = spec.reloadDuration
	g.setStatus(fmt.Sprintf("Reloading %s", spec.name), 650*time.Millisecond)
}

func (g *Game) updateWeaponReload(delta time.Duration) {

	state := &g.weapons[g.activeWeapon]
	if state.reloadRemaining <= 0 {
		return
	}

	state.reloadRemaining -= delta
	if state.reloadRemaining > 0 {
		return
	}

	g.completeWeaponReload(g.activeWeapon)
}

func (g *Game) completeWeaponReload(id weaponID) {

	spec := weaponSpecs[id]
	state := &g.weapons[id]
	needed := spec.magazineSize - state.ammoInMagazine
	if needed <= 0 || state.reserveAmmo <= 0 {
		state.reloadRemaining = 0
		return
	}

	loaded := needed
	if loaded > state.reserveAmmo {
		loaded = state.reserveAmmo
	}

	state.ammoInMagazine += loaded
	state.reserveAmmo -= loaded
	state.reloadRemaining = 0
	if id == g.activeWeapon {
		g.setStatus(fmt.Sprintf("%s reloaded", spec.name), 650*time.Millisecond)
	}
}

func (g *Game) weaponHUDLine() string {

	spec := g.activeWeaponSpec()
	state := g.weapons[spec.id]
	status := "Ready"
	if state.reloadRemaining > 0 {
		status = fmt.Sprintf("Reload %.1fs", state.reloadRemaining.Seconds())
	} else if state.ammoInMagazine == 0 {
		status = "Empty"
	}

	return fmt.Sprintf(
		"Weapon: %d %s  Ammo: %d/%d  %s\nLoadout: %s",
		spec.slot,
		spec.name,
		state.ammoInMagazine,
		state.reserveAmmo,
		status,
		g.loadoutHUDLine(),
	)
}

func (g *Game) loadoutHUDLine() string {

	labels := make([]string, 0, int(weaponCount))
	for id := weaponID(0); id < weaponCount; id++ {
		spec := weaponSpecs[id]
		prefix := fmt.Sprintf("%d", spec.slot)
		if id == g.activeWeapon {
			prefix = fmt.Sprintf(">%d", spec.slot)
		}
		if !g.weapons[id].unlocked {
			labels = append(labels, fmt.Sprintf("%s %s locked", prefix, spec.shortName))
			continue
		}
		labels = append(labels, fmt.Sprintf("%s %s", prefix, spec.shortName))
	}
	return strings.Join(labels, "  ")
}

func (g *Game) clearWeaponPickups() {

	for _, pickup := range g.weaponPickups {
		if pickup.root != nil && g.scene != nil {
			g.scene.Remove(pickup.root)
		}
	}
	g.weaponPickups = nil
}

func (g *Game) spawnWeaponPickups() {

	g.clearWeaponPickups()

	points := g.pickupSpawnPoints(5)
	if len(points) == 0 {
		return
	}

	g.addWeaponPickup(weaponPickupUnlock, weaponAcornScatterer, points[0], 0.0)
	if len(points) > 1 {
		g.addWeaponPickup(weaponPickupUnlock, weaponTurnipThumper, points[1], 1.4)
	}

	ammoIDs := []weaponID{weaponBurrowBlaster, weaponAcornScatterer, weaponTurnipThumper}
	for idx, id := range ammoIDs {
		pointIdx := idx + 2
		if pointIdx >= len(points) {
			pointIdx = idx % len(points)
		}
		g.addWeaponPickup(weaponPickupAmmo, id, points[pointIdx], float32(idx)*0.9+2.2)
	}
}

func (g *Game) addWeaponPickup(kind weaponPickupKind, id weaponID, position math32.Vector3, phase float32) {

	root := newWeaponPickupRoot(kind, weaponSpecs[id])
	homeY := position.Y + 0.9
	root.SetPosition(position.X, homeY, position.Z)
	g.scene.Add(root)

	g.weaponPickups = append(g.weaponPickups, &weaponPickup{
		kind:     kind,
		weaponID: id,
		position: position,
		root:     root,
		phase:    phase,
		homeY:    homeY,
	})
}

func (g *Game) pickupSpawnPoints(count int) []math32.Vector3 {

	points := make([]math32.Vector3, 0, count)
	for _, spawn := range g.combatantSpawns {
		if spawn.DistanceToSquared(&g.playerSpawn) < weaponPickupRadius*weaponPickupRadius*2 {
			continue
		}
		points = append(points, spawn)
		if len(points) == count {
			return points
		}
	}

	if boxIsFinite(g.worldBounds) {
		centerX := (g.worldBounds.Min.X + g.worldBounds.Max.X) * 0.5
		centerZ := (g.worldBounds.Min.Z + g.worldBounds.Max.Z) * 0.5
		radius := math32.Min(g.worldBounds.Max.X-g.worldBounds.Min.X, g.worldBounds.Max.Z-g.worldBounds.Min.Z) * 0.32
		if radius < 8 {
			radius = 8
		}

		for idx := 0; len(points) < count && idx < count*4; idx++ {
			angle := float32(idx) * math32.Pi * 0.47
			candidate := math32.Vector3{
				X: centerX + math32.Cos(angle)*radius,
				Z: centerZ + math32.Sin(angle)*radius,
			}
			if supportY, ok := g.highestSupportAt(candidate.X, candidate.Z); ok {
				candidate.Y = supportY
			} else {
				candidate.Y = g.playerSpawn.Y
			}
			if g.positionBlocked(candidate, true) {
				continue
			}
			points = append(points, candidate)
		}
	}

	if len(points) == 0 {
		points = append(points, g.playerSpawn)
	}

	for len(points) < count {
		base := points[len(points)%len(points)]
		offset := float32(len(points)+1) * 4
		base.X += offset
		points = append(points, base)
	}
	return points
}

func newWeaponPickupRoot(kind weaponPickupKind, spec weaponSpec) *core.Node {

	root := core.NewNode()

	baseMat := material.NewStandard(&math32.Color{R: 0.08, G: 0.1, B: 0.12})
	baseMat.SetEmissiveColor(&math32.Color{R: 0.02, G: 0.03, B: 0.04})
	base := graphic.NewMesh(geometry.NewCylinder(0.72, 0.16, 20, 1, true, true), baseMat)
	base.SetPosition(0, -0.58, 0)
	root.Add(base)

	glowColor := spec.color
	glowMat := material.NewStandard(&glowColor)
	glowMat.SetEmissiveColor(&glowColor)

	if kind == weaponPickupAmmo {
		box := graphic.NewMesh(geometry.NewBox(0.72, 0.52, 0.72), glowMat)
		root.Add(box)
		return root
	}

	body := graphic.NewMesh(geometry.NewCylinder(0.28, 1.3, 18, 1, true, true), glowMat)
	body.SetRotation(0, 0, math32.Pi*0.5)
	root.Add(body)

	muzzle := graphic.NewMesh(geometry.NewSphere(0.34, 16, 10), glowMat)
	muzzle.SetPosition(0.78, 0, 0)
	root.Add(muzzle)
	return root
}

func (g *Game) buildCarriedWeaponModels() {

	if g.playerRoot == nil {
		return
	}

	for id := weaponID(0); id < weaponCount; id++ {
		model := newCarriedWeaponRoot(weaponSpecs[id])
		model.SetVisible(false)
		g.playerRoot.Add(model)
		g.playerWeapons[id] = model
	}
	g.updateCarriedWeaponVisibility()
}

func (g *Game) updateCarriedWeaponVisibility() {

	for id, model := range g.playerWeapons {
		if model == nil {
			continue
		}
		model.SetVisible(weaponID(id) == g.activeWeapon)
	}
}

func newCarriedWeaponRoot(spec weaponSpec) *core.Node {

	root := core.NewNode()
	root.SetName(spec.shortName + "-carried")
	root.SetPosition(0.7, 1.42, 0.46)
	root.SetRotation(0, 0.08, -0.06)

	color := spec.color
	bodyMat := material.NewStandard(&color)
	bodyMat.SetEmissiveColor(&math32.Color{R: color.R * 0.22, G: color.G * 0.22, B: color.B * 0.22})
	darkMat := material.NewStandard(&math32.Color{R: 0.08, G: 0.085, B: 0.09})
	darkMat.SetEmissiveColor(&math32.Color{R: 0.01, G: 0.012, B: 0.014})

	grip := graphic.NewMesh(geometry.NewBox(0.18, 0.56, 0.2), darkMat)
	grip.SetPosition(-0.18, -0.26, 0)
	grip.SetRotation(0, 0, -0.22)
	root.Add(grip)

	switch spec.id {
	case weaponAcornScatterer:
		makeBarrelCluster(root, bodyMat, darkMat)
	case weaponTurnipThumper:
		makeTurnipThumper(root, bodyMat, darkMat)
	default:
		makeBurrowBlaster(root, bodyMat, darkMat)
	}

	return root
}

func makeBurrowBlaster(root *core.Node, bodyMat, darkMat *material.Standard) {

	body := graphic.NewMesh(geometry.NewCylinder(0.14, 0.92, 16, 1, true, true), bodyMat)
	body.SetRotation(0, 0, math32.Pi*0.5)
	body.SetPosition(0.2, 0, 0)
	root.Add(body)

	muzzle := graphic.NewMesh(geometry.NewCylinder(0.17, 0.18, 16, 1, true, true), darkMat)
	muzzle.SetRotation(0, 0, math32.Pi*0.5)
	muzzle.SetPosition(0.72, 0, 0)
	root.Add(muzzle)

	sight := graphic.NewMesh(geometry.NewBox(0.18, 0.1, 0.12), darkMat)
	sight.SetPosition(0.25, 0.18, 0)
	root.Add(sight)
}

func makeBarrelCluster(root *core.Node, bodyMat, darkMat *material.Standard) {

	stock := graphic.NewMesh(geometry.NewBox(0.46, 0.28, 0.34), darkMat)
	stock.SetPosition(-0.05, 0, 0)
	root.Add(stock)

	offsets := []math32.Vector3{
		{Y: 0.09, Z: 0.09},
		{Y: 0.09, Z: -0.09},
		{Y: -0.09, Z: 0.09},
		{Y: -0.09, Z: -0.09},
	}
	for _, offset := range offsets {
		barrel := graphic.NewMesh(geometry.NewCylinder(0.055, 0.72, 10, 1, true, true), bodyMat)
		barrel.SetRotation(0, 0, math32.Pi*0.5)
		barrel.SetPosition(0.35, offset.Y, offset.Z)
		root.Add(barrel)
	}
}

func makeTurnipThumper(root *core.Node, bodyMat, darkMat *material.Standard) {

	tube := graphic.NewMesh(geometry.NewCylinder(0.18, 0.78, 16, 1, true, true), darkMat)
	tube.SetRotation(0, 0, math32.Pi*0.5)
	tube.SetPosition(0.16, 0, 0)
	root.Add(tube)

	turnip := graphic.NewMesh(geometry.NewSphere(0.25, 16, 12), bodyMat)
	turnip.SetPosition(0.68, 0.02, 0)
	root.Add(turnip)

	stem := graphic.NewMesh(geometry.NewCone(0.1, 0.24, 10, 1, true), bodyMat)
	stem.SetRotation(0, 0, -math32.Pi*0.5)
	stem.SetPosition(0.91, 0.08, 0)
	root.Add(stem)
}

func (g *Game) updateWeaponPickups(delta time.Duration) {

	for _, pickup := range g.weaponPickups {
		if pickup.respawn > 0 {
			pickup.respawn -= delta
			if pickup.respawn <= 0 {
				pickup.respawn = 0
				pickup.root.SetVisible(true)
			}
			continue
		}

		if pickup.cooldown > 0 {
			pickup.cooldown -= delta
			if pickup.cooldown < 0 {
				pickup.cooldown = 0
			}
		}

		spin := float32(g.matchTime.Seconds())*1.65 + pickup.phase
		pickup.root.SetRotation(0, spin, 0)
		pickup.root.SetPosition(
			pickup.position.X,
			pickup.homeY+math32.Sin(spin*1.3)*0.18,
			pickup.position.Z,
		)

		dx := g.playerPos.X - pickup.position.X
		dz := g.playerPos.Z - pickup.position.Z
		if dx*dx+dz*dz > weaponPickupRadius*weaponPickupRadius {
			continue
		}
		g.collectWeaponPickup(pickup)
	}
}

func (g *Game) collectWeaponPickup(pickup *weaponPickup) {

	if pickup.cooldown > 0 {
		return
	}

	spec := weaponSpecs[pickup.weaponID]
	state := &g.weapons[pickup.weaponID]
	switch pickup.kind {
	case weaponPickupUnlock:
		wasLocked := !state.unlocked
		state.unlocked = true
		if state.ammoInMagazine <= 0 {
			state.ammoInMagazine = spec.magazineSize
		}
		state.reserveAmmo = clampAmmo(state.reserveAmmo+spec.pickupAmmo, spec.maxReserve)
		if wasLocked {
			g.switchWeapon(pickup.weaponID)
			g.setStatus(fmt.Sprintf("Picked up %s", spec.name), 1100*time.Millisecond)
		} else {
			g.setStatus(fmt.Sprintf("Refilled %s", spec.name), 900*time.Millisecond)
		}
	case weaponPickupAmmo:
		if !state.unlocked {
			pickup.cooldown = 700 * time.Millisecond
			g.setStatus(fmt.Sprintf("Need %s before using %s", spec.name, spec.pickupName), 900*time.Millisecond)
			return
		}
		if state.reserveAmmo >= spec.maxReserve {
			pickup.cooldown = 700 * time.Millisecond
			g.setStatus(fmt.Sprintf("%s ammo is full", spec.name), 650*time.Millisecond)
			return
		}
		state.reserveAmmo = clampAmmo(state.reserveAmmo+spec.pickupAmmo, spec.maxReserve)
		g.setStatus(fmt.Sprintf("Picked up %s", spec.pickupName), 900*time.Millisecond)
	}

	if pickup.kind == weaponPickupAmmo {
		pickup.respawn = ammoPickupRespawn
	} else {
		pickup.respawn = weaponPickupRespawn
	}
	pickup.root.SetVisible(false)
}

func clampAmmo(value, maxValue int) int {

	if value > maxValue {
		return maxValue
	}
	if value < 0 {
		return 0
	}
	return value
}
