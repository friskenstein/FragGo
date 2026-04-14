package game

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/light"
	objloader "github.com/g3n/engine/loader/obj"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
)

const (
	playerModelScale       = 2.1
	playerModelGroundLift  = 1.055
	playerModelCenterShift = 0.024
)

type platform struct {
	center math32.Vector3
	size   math32.Vector3
}

func (p platform) top() float32 {

	return p.center.Y + p.size.Y*0.5
}

func (p platform) contains(x, z, radius float32) bool {

	halfX := p.size.X * 0.5
	halfZ := p.size.Z * 0.5
	return x >= p.center.X-halfX-radius &&
		x <= p.center.X+halfX+radius &&
		z >= p.center.Z-halfZ-radius &&
		z <= p.center.Z+halfZ+radius
}

func (g *Game) addWalkPlatform(name string, plat platform, mat *material.Standard) {

	mesh := graphic.NewMesh(geometry.NewBox(plat.size.X, plat.size.Y, plat.size.Z), mat)
	mesh.SetPositionVec(&plat.center)
	g.scene.Add(mesh)

	g.platforms = append(g.platforms, plat)
	g.colliders = append(g.colliders, boxCollider{
		name:     name,
		center:   plat.center,
		size:     plat.size,
		walkable: true,
	})
}

func (g *Game) addBlock(collider boxCollider, mat *material.Standard) {

	mesh := graphic.NewMesh(geometry.NewBox(collider.size.X, collider.size.Y, collider.size.Z), mat)
	mesh.SetPositionVec(&collider.center)
	g.scene.Add(mesh)
	g.colliders = append(g.colliders, collider)
}

func (g *Game) buildWorld() error {

	g.scene.Add(light.NewAmbient(&math32.Color{R: 0.8, G: 0.85, B: 1.0}, 0.45))

	keyLight := light.NewDirectional(&math32.Color{R: 1.0, G: 0.96, B: 0.88}, 1.6)
	keyLight.SetPosition(14, 28, 10)
	keyLight.LookAt(&math32.Vector3{}, &math32.Vector3{Y: 1})
	g.scene.Add(keyLight)

	fillLight := light.NewPoint(&math32.Color{R: 0.35, G: 0.5, B: 1.0}, 30)
	fillLight.SetPosition(-6, 7, -4)
	g.scene.Add(fillLight)

	g.buildArenaGeometry()
	if err := g.buildPlayerModel(); err != nil {
		return err
	}
	return nil
}

func (g *Game) buildArenaGeometry() {

	g.platforms = nil
	g.colliders = nil

	floorMat := material.NewStandard(&math32.Color{R: 0.11, G: 0.13, B: 0.17})
	floorMat.SetEmissiveColor(&math32.Color{R: 0.02, G: 0.03, B: 0.04})
	floor := platform{
		center: math32.Vector3{X: 0, Y: -0.5, Z: 0},
		size:   math32.Vector3{X: 104, Y: 1, Z: 104},
	}
	g.addWalkPlatform("floor", floor, floorMat)

	borderMat := material.NewStandard(&math32.Color{R: 0.18, G: 0.22, B: 0.28})
	borderMat.SetEmissiveColor(&math32.Color{R: 0.02, G: 0.02, B: 0.03})

	for _, wall := range []boxCollider{
		{name: "north wall", center: math32.Vector3{X: 0, Y: 4, Z: -52}, size: math32.Vector3{X: 104, Y: 8, Z: 2}},
		{name: "south wall", center: math32.Vector3{X: 0, Y: 4, Z: 52}, size: math32.Vector3{X: 104, Y: 8, Z: 2}},
		{name: "west wall", center: math32.Vector3{X: -52, Y: 4, Z: 0}, size: math32.Vector3{X: 2, Y: 8, Z: 104}},
		{name: "east wall", center: math32.Vector3{X: 52, Y: 4, Z: 0}, size: math32.Vector3{X: 2, Y: 8, Z: 104}},
	} {
		g.addBlock(wall, borderMat)
	}

	platformMat := material.NewStandard(&math32.Color{R: 0.33, G: 0.36, B: 0.42})
	platformMat.SetEmissiveColor(&math32.Color{R: 0.03, G: 0.03, B: 0.04})
	rampMat := material.NewStandard(&math32.Color{R: 0.42, G: 0.39, B: 0.33})
	rampMat.SetEmissiveColor(&math32.Color{R: 0.04, G: 0.03, B: 0.02})

	coverMat := material.NewStandard(&math32.Color{R: 0.24, G: 0.28, B: 0.33})
	coverMat.SetEmissiveColor(&math32.Color{R: 0.03, G: 0.03, B: 0.04})

	for _, plat := range []struct {
		name string
		plat platform
		mat  *material.Standard
	}{
		{
			name: "west gallery",
			plat: platform{center: math32.Vector3{X: -30, Y: 1.5, Z: 0}, size: math32.Vector3{X: 14, Y: 1, Z: 48}},
			mat:  platformMat,
		},
		{
			name: "east gallery",
			plat: platform{center: math32.Vector3{X: 30, Y: 2.5, Z: 0}, size: math32.Vector3{X: 14, Y: 1, Z: 48}},
			mat:  platformMat,
		},
		{
			name: "north bridge",
			plat: platform{center: math32.Vector3{X: 0, Y: 4.5, Z: -26}, size: math32.Vector3{X: 24, Y: 1, Z: 12}},
			mat:  platformMat,
		},
		{
			name: "south catwalk",
			plat: platform{center: math32.Vector3{X: 0, Y: 2.5, Z: 30}, size: math32.Vector3{X: 24, Y: 1, Z: 10}},
			mat:  platformMat,
		},
		{
			name: "center bunker",
			plat: platform{center: math32.Vector3{X: 0, Y: 0.75, Z: 0}, size: math32.Vector3{X: 14, Y: 1.5, Z: 18}},
			mat:  coverMat,
		},
	} {
		g.addWalkPlatform(plat.name, plat.plat, plat.mat)
	}

	for idx, top := range []float32{0.5, 1.0, 1.5, 2.0} {
		stepHeight := float32(0.5)
		g.addWalkPlatform(
			fmt.Sprintf("west ramp %d", idx+1),
			platform{
				center: math32.Vector3{X: -30, Y: top - stepHeight*0.5, Z: 36 - float32(idx)*6},
				size:   math32.Vector3{X: 10, Y: stepHeight, Z: 6},
			},
			rampMat,
		)
	}

	for idx, top := range []float32{2.5, 3.0, 3.5, 4.0, 4.5} {
		stepHeight := float32(0.5)
		g.addWalkPlatform(
			fmt.Sprintf("north ramp %d", idx+1),
			platform{
				center: math32.Vector3{X: -13 + float32(idx)*6.5, Y: top - stepHeight*0.5, Z: -20},
				size:   math32.Vector3{X: 6.5, Y: stepHeight, Z: 6},
			},
			rampMat,
		)
	}

	for idx, top := range []float32{0.5, 1.0, 1.5, 2.0, 2.5, 3.0} {
		stepHeight := float32(0.5)
		g.addWalkPlatform(
			fmt.Sprintf("east stair %d", idx+1),
			platform{
				center: math32.Vector3{X: 30, Y: top - stepHeight*0.5, Z: 34 - float32(idx)*4},
				size:   math32.Vector3{X: 8, Y: stepHeight, Z: 4},
			},
			rampMat,
		)
	}

	for idx, top := range []float32{3.0, 3.5, 4.0, 4.5} {
		stepHeight := float32(0.5)
		g.addWalkPlatform(
			fmt.Sprintf("bridge stair %d", idx+1),
			platform{
				center: math32.Vector3{X: 18 - float32(idx)*4, Y: top - stepHeight*0.5, Z: -18},
				size:   math32.Vector3{X: 4, Y: stepHeight, Z: 6},
			},
			rampMat,
		)
	}

	for idx, top := range []float32{0.5, 1.0, 1.5, 2.0, 2.5} {
		stepHeight := float32(0.5)
		g.addWalkPlatform(
			fmt.Sprintf("south ramp %d", idx+1),
			platform{
				center: math32.Vector3{X: -8 + float32(idx)*4, Y: top - stepHeight*0.5, Z: 24},
				size:   math32.Vector3{X: 4, Y: stepHeight, Z: 6},
			},
			rampMat,
		)
	}

	for _, block := range []boxCollider{
		{name: "lane wall", center: math32.Vector3{X: -16, Y: 2, Z: 10}, size: math32.Vector3{X: 3, Y: 4, Z: 30}},
		{name: "lane wall", center: math32.Vector3{X: 16, Y: 2, Z: -10}, size: math32.Vector3{X: 3, Y: 4, Z: 30}},
		{name: "cross wall", center: math32.Vector3{X: 0, Y: 2, Z: -8}, size: math32.Vector3{X: 18, Y: 4, Z: 3}},
		{name: "cross wall", center: math32.Vector3{X: 0, Y: 2, Z: 14}, size: math32.Vector3{X: 18, Y: 4, Z: 3}},
		{name: "cover block", center: math32.Vector3{X: -6, Y: 0.75, Z: 24}, size: math32.Vector3{X: 4, Y: 1.5, Z: 3}},
		{name: "cover block", center: math32.Vector3{X: 6, Y: 0.75, Z: 24}, size: math32.Vector3{X: 4, Y: 1.5, Z: 3}},
		{name: "cover block", center: math32.Vector3{X: -24, Y: 0.75, Z: -16}, size: math32.Vector3{X: 4, Y: 1.5, Z: 4}},
		{name: "cover block", center: math32.Vector3{X: 24, Y: 0.75, Z: 16}, size: math32.Vector3{X: 4, Y: 1.5, Z: 4}},
		{name: "pillar base", center: math32.Vector3{X: -30, Y: 0.75, Z: -22}, size: math32.Vector3{X: 3, Y: 1.5, Z: 3}},
		{name: "pillar base", center: math32.Vector3{X: 30, Y: 0.75, Z: 22}, size: math32.Vector3{X: 3, Y: 1.5, Z: 3}},
	} {
		g.addBlock(block, coverMat)
	}

	columnMat := material.NewStandard(&math32.Color{R: 0.51, G: 0.34, B: 0.2})
	columnMat.SetEmissiveColor(&math32.Color{R: 0.04, G: 0.02, B: 0.01})

	for idx, pos := range []math32.Vector3{
		{X: -38, Y: 2.0, Z: 34},
		{X: 38, Y: 2.0, Z: -34},
		{X: -18, Y: 2.5, Z: -30},
		{X: 18, Y: 2.5, Z: 30},
	} {
		height := float32(3.0 + float32(idx))
		center := math32.Vector3{X: pos.X, Y: pos.Y, Z: pos.Z}
		column := graphic.NewMesh(geometry.NewCylinder(0.8, float64(height), 16, 1, true, true), columnMat)
		column.SetPosition(center.X, center.Y, center.Z)
		g.scene.Add(column)
		g.colliders = append(g.colliders, boxCollider{
			name:   "pillar",
			center: center,
			size:   math32.Vector3{X: 1.6, Y: height, Z: 1.6},
		})
	}
}

func (g *Game) buildPlayerModel() error {

	root, err := newGopherRoot("local-player")
	if err != nil {
		return err
	}
	g.playerRoot = root

	g.scene.Add(g.playerRoot)
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

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve player model path: runtime caller unavailable")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "assets", "gopher", "gopher.obj")), nil
}
