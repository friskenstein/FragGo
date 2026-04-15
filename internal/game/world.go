package game

import (
	"fmt"
	"math"
	"path/filepath"
	"runtime"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/light"
	gltfloader "github.com/g3n/engine/loader/gltf"
	objloader "github.com/g3n/engine/loader/obj"
	"github.com/g3n/engine/math32"
)

const (
	playerModelScale       = 2.1
	playerModelGroundLift  = 1.055
	playerModelCenterShift = 0.024

	arenaModelScale            = 1.0
	arenaSpawnSearchStep       = 8.0
	arenaSpawnSearchLayers     = 6
	arenaSpawnMinSpacing       = 18.0
	arenaSpawnMinEdgeClearance = 6.0
)

func (g *Game) buildWorld() error {

	g.scene.Add(light.NewAmbient(&math32.Color{R: 0.8, G: 0.85, B: 1.0}, 0.45))

	keyLight := light.NewDirectional(&math32.Color{R: 1.0, G: 0.96, B: 0.88}, 1.6)
	keyLight.SetPosition(14, 28, 10)
	keyLight.LookAt(&math32.Vector3{}, &math32.Vector3{Y: 1})
	g.scene.Add(keyLight)

	fillLight := light.NewPoint(&math32.Color{R: 0.35, G: 0.5, B: 1.0}, 30)
	fillLight.SetPosition(-6, 7, -4)
	g.scene.Add(fillLight)

	if err := g.reloadArena(); err != nil {
		return err
	}

	if err := g.buildPlayerModel(); err != nil {
		return err
	}
	return nil
}

func (g *Game) reloadArena() error {

	arena, ok := g.selectedArena()
	if !ok {
		return fmt.Errorf("selected arena %q is unavailable", g.matchConfig.ArenaID)
	}

	arenaRoot, bounds, collision, err := loadArena(arena)
	if err != nil {
		return err
	}

	if g.arenaRoot != nil {
		g.scene.Remove(g.arenaRoot)
	}

	g.scene.Add(arenaRoot)
	g.arenaRoot = arenaRoot
	g.arenaCollision = collision
	g.worldBounds = bounds
	g.configureArenaSpawns()
	g.resetPlayer()
	return nil
}

func loadArena(arena arenaDefinition) (core.INode, math32.Box3, *meshCollision, error) {

	arenaAssetPath, err := arenaModelPath(arena)
	if err != nil {
		return nil, math32.Box3{}, nil, err
	}

	doc, err := gltfloader.ParseBin(arenaAssetPath)
	if err != nil {
		return nil, math32.Box3{}, nil, err
	}

	sceneIdx := 0
	if doc.Scene != nil {
		sceneIdx = *doc.Scene
	}

	arenaRoot, err := doc.LoadScene(sceneIdx)
	if err != nil {
		return nil, math32.Box3{}, nil, err
	}

	arenaRoot.SetName("arena")
	arenaRoot.GetNode().SetScale(arena.Scale*arenaModelScale, arena.Scale*arenaModelScale, arena.Scale*arenaModelScale)
	arenaRoot.UpdateMatrixWorld()

	bounds := arenaRoot.BoundingBox()
	if !boxIsFinite(bounds) {
		return nil, math32.Box3{}, nil, fmt.Errorf("arena bounds are invalid")
	}

	arenaRoot.GetNode().SetPosition(
		-(bounds.Min.X+bounds.Max.X)*0.5,
		-bounds.Min.Y,
		-(bounds.Min.Z+bounds.Max.Z)*0.5,
	)
	arenaRoot.UpdateMatrixWorld()
	bounds = arenaRoot.BoundingBox()

	collision, err := buildCollisionMesh(arenaRoot)
	if err != nil {
		return nil, math32.Box3{}, nil, err
	}
	if collision == nil {
		return nil, math32.Box3{}, nil, fmt.Errorf("arena collision mesh is empty")
	}

	return arenaRoot, bounds, collision, nil
}

func (g *Game) configureArenaSpawns() {

	spawns := make([]math32.Vector3, 0, 7)
	centerX := (g.worldBounds.Min.X + g.worldBounds.Max.X) * 0.5
	centerZ := (g.worldBounds.Min.Z + g.worldBounds.Max.Z) * 0.5
	minX := g.worldBounds.Min.X + arenaSpawnMinEdgeClearance
	maxX := g.worldBounds.Max.X - arenaSpawnMinEdgeClearance
	minZ := g.worldBounds.Min.Z + arenaSpawnMinEdgeClearance
	maxZ := g.worldBounds.Max.Z - arenaSpawnMinEdgeClearance

	candidates := []math32.Vector3{
		{X: minX, Z: minZ},
		{X: maxX, Z: minZ},
		{X: minX, Z: maxZ},
		{X: maxX, Z: maxZ},
		{X: centerX, Z: minZ},
		{X: centerX, Z: maxZ},
		{X: minX, Z: centerZ},
		{X: maxX, Z: centerZ},
		{X: centerX, Z: centerZ},
	}

	for _, candidate := range candidates {
		spawn, ok := g.findArenaSpawn(candidate, spawns)
		if !ok {
			continue
		}
		spawns = append(spawns, spawn)
		if len(spawns) == 7 {
			break
		}
	}

	if len(spawns) == 0 {
		fallback, ok := g.findArenaSpawn(math32.Vector3{X: centerX, Z: centerZ}, nil)
		if ok {
			spawns = append(spawns, fallback)
		}
	}

	if len(spawns) == 0 {
		g.playerSpawn = math32.Vector3{}
		g.combatantSpawns = nil
		return
	}

	g.playerSpawn = spawns[len(spawns)-1]
	g.combatantSpawns = spawns
}

func (g *Game) findArenaSpawn(origin math32.Vector3, existing []math32.Vector3) (math32.Vector3, bool) {

	offsets := []math32.Vector3{
		{},
		{X: 1},
		{X: -1},
		{Z: 1},
		{Z: -1},
		{X: 1, Z: 1},
		{X: 1, Z: -1},
		{X: -1, Z: 1},
		{X: -1, Z: -1},
	}

	for layer := 0; layer <= arenaSpawnSearchLayers; layer++ {
		for _, offset := range offsets {
			candidate := origin
			candidate.X += offset.X * arenaSpawnSearchStep * float32(layer)
			candidate.Z += offset.Z * arenaSpawnSearchStep * float32(layer)

			supportY, ok := g.highestSupportAt(candidate.X, candidate.Z)
			if !ok {
				continue
			}

			candidate.Y = supportY
			if g.positionBlocked(candidate, true) || spawnTooClose(candidate, existing) {
				continue
			}

			return candidate, true
		}
	}

	return math32.Vector3{}, false
}

func (g *Game) highestSupportAt(x, z float32) (float32, bool) {

	if g.arenaCollision == nil || !boxIsFinite(g.worldBounds) {
		return 0, false
	}

	return g.arenaCollision.supportAt(
		x,
		z,
		g.worldBounds.Max.Y+playerHeight+2,
		g.worldBounds.Min.Y-playerHeight-2,
	)
}

func spawnTooClose(candidate math32.Vector3, existing []math32.Vector3) bool {

	minDistanceSq := float32(arenaSpawnMinSpacing * arenaSpawnMinSpacing)
	for _, current := range existing {
		deltaX := candidate.X - current.X
		deltaZ := candidate.Z - current.Z
		if deltaX*deltaX+deltaZ*deltaZ < minDistanceSq {
			return true
		}
	}
	return false
}

func (g *Game) buildPlayerModel() error {

	root, err := newGopherRoot("local-player")
	if err != nil {
		return err
	}
	g.playerRoot = root

	g.scene.Add(g.playerRoot)
	g.buildCarriedWeaponModels()
	g.syncPlayerModel()
	return nil
}

func newGopherRoot(name string) (*core.Node, error) {

	root := core.NewNode()
	root.SetName(name)

	playerAssetPath, err := playerModelPath()
	if err != nil {
		return nil, err
	}

	decoder, err := objloader.Decode(playerAssetPath, "")
	if err != nil {
		return nil, err
	}

	playerModel, err := decoder.NewGroup()
	if err != nil {
		return nil, err
	}

	playerModel.SetName(name + "-model")
	playerModel.SetScale(playerModelScale, playerModelScale, playerModelScale)
	playerModel.SetPosition(playerModelCenterShift, playerModelGroundLift, 0)
	root.Add(playerModel)
	return root, nil
}

func (g *Game) syncPlayerModel() {

	g.playerRoot.SetPosition(g.playerPos.X, g.playerPos.Y, g.playerPos.Z)
	// The gopher OBJ is authored facing +X, while camera forward at yaw 0 points toward -Z.
	g.playerRoot.SetRotation(0, math32.Pi/2-g.yaw, 0)
}

func playerModelPath() (string, error) {

	return assetPath("gopher", "gopher.obj")
}

func arenaModelPath(arena arenaDefinition) (string, error) {

	return assetPath("levels", arena.AssetName)
}

func fragGoLogoPath() (string, error) {

	return assetPath("fraggo_logo.png")
}

func assetPath(parts ...string) (string, error) {

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve asset path: runtime caller unavailable")
	}

	segments := append([]string{filepath.Dir(currentFile), "..", "..", "assets"}, parts...)
	return filepath.Clean(filepath.Join(segments...)), nil
}

func boxIsFinite(box math32.Box3) bool {

	return componentIsFinite(box.Min.X) &&
		componentIsFinite(box.Min.Y) &&
		componentIsFinite(box.Min.Z) &&
		componentIsFinite(box.Max.X) &&
		componentIsFinite(box.Max.Y) &&
		componentIsFinite(box.Max.Z) &&
		box.Min.X <= box.Max.X &&
		box.Min.Y <= box.Max.Y &&
		box.Min.Z <= box.Max.Z
}

func componentIsFinite(value float32) bool {

	f64 := float64(value)
	return !math.IsNaN(f64) && !math.IsInf(f64, 0) && math.Abs(f64) < 1e6
}
