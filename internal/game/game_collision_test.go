package game

import (
	"testing"

	"github.com/g3n/engine/math32"
)

func TestResolvePlayerAxisStopsAtWall(t *testing.T) {

	g := Game{
		playerGrounded: true,
		arenaCollision: testCollisionMesh(
			testQuad(
				"wall",
				math32.Vector3{X: 2, Y: 0, Z: -1},
				math32.Vector3{X: 2, Y: 2, Z: -1},
				math32.Vector3{X: 2, Y: 2, Z: 1},
				math32.Vector3{X: 2, Y: 0, Z: 1},
			)...,
		),
	}

	nextX, nextVelocity := g.resolvePlayerAxis(math32.Vector3{}, 2.2, true, 10)
	expectedX := float32(2 - playerRadius - collisionEpsilon)
	if math32.Abs(nextX-expectedX) > 0.05 {
		t.Fatalf("expected collision stop near %.3f, got %.3f", expectedX, nextX)
	}
	if nextVelocity != 0 {
		t.Fatalf("expected horizontal velocity to clear on impact, got %.3f", nextVelocity)
	}
}

func TestResolvePlayerAxisAllowsWalkableStepUp(t *testing.T) {

	triangles := append(
		testQuad(
			"floor",
			math32.Vector3{X: -4, Y: 0, Z: -4},
			math32.Vector3{X: 4, Y: 0, Z: -4},
			math32.Vector3{X: 4, Y: 0, Z: 4},
			math32.Vector3{X: -4, Y: 0, Z: 4},
		),
		testQuad(
			"step-top",
			math32.Vector3{X: 1, Y: 0.5, Z: -1},
			math32.Vector3{X: 3, Y: 0.5, Z: -1},
			math32.Vector3{X: 3, Y: 0.5, Z: 1},
			math32.Vector3{X: 1, Y: 0.5, Z: 1},
		)...,
	)
	triangles = append(
		triangles,
		testQuad(
			"step-face",
			math32.Vector3{X: 1, Y: 0, Z: -1},
			math32.Vector3{X: 1, Y: 0.5, Z: -1},
			math32.Vector3{X: 1, Y: 0.5, Z: 1},
			math32.Vector3{X: 1, Y: 0, Z: 1},
		)...,
	)

	g := Game{
		playerGrounded: true,
		arenaCollision: testCollisionMesh(triangles...),
		worldBounds: math32.Box3{
			Min: math32.Vector3{X: -10, Y: 0, Z: -10},
			Max: math32.Vector3{X: 10, Y: 10, Z: 10},
		},
	}

	targetDelta := float32(2.2)
	nextX, nextVelocity := g.resolvePlayerAxis(math32.Vector3{}, targetDelta, true, 10)
	if math32.Abs(nextX-targetDelta) > 0.05 {
		t.Fatalf("expected walkable step to allow movement to %.3f, got %.3f", targetDelta, nextX)
	}
	if nextVelocity != 10 {
		t.Fatalf("expected horizontal velocity to be preserved, got %.3f", nextVelocity)
	}
}

func TestResolveGroundPrefersReachableSupportBelowOverOverheadWalkway(t *testing.T) {

	g := Game{
		playerPos:      math32.Vector3{X: 0, Y: 0, Z: 0},
		playerVelocity: math32.Vector3{},
		playerGrounded: true,
		arenaCollision: testCollisionMesh(
			append(
				testQuad(
					"floor",
					math32.Vector3{X: -8, Y: 0, Z: -8},
					math32.Vector3{X: 8, Y: 0, Z: -8},
					math32.Vector3{X: 8, Y: 0, Z: 8},
					math32.Vector3{X: -8, Y: 0, Z: 8},
				),
				testQuad(
					"walkway",
					math32.Vector3{X: -4, Y: 3, Z: -2},
					math32.Vector3{X: 4, Y: 3, Z: -2},
					math32.Vector3{X: 4, Y: 3, Z: 2},
					math32.Vector3{X: -4, Y: 3, Z: 2},
				)...,
			)...,
		),
		worldBounds: math32.Box3{
			Min: math32.Vector3{X: -10, Y: 0, Z: -10},
			Max: math32.Vector3{X: 10, Y: 10, Z: 10},
		},
	}

	landed, supportY := g.resolveGround(0)
	if !landed {
		t.Fatal("expected floor support under player")
	}
	if math32.Abs(supportY) > 0.0001 {
		t.Fatalf("expected floor support at 0, got %.3f", supportY)
	}
}

func testCollisionMesh(triangles ...collisionTriangle) *meshCollision {

	mesh := &meshCollision{
		cellSize: collisionCellSize,
		cells:    make(map[collisionCell][]int),
	}
	for _, tri := range triangles {
		mesh.addTriangle(tri.name, tri.a, tri.b, tri.c)
	}
	return mesh
}

func testQuad(name string, a, b, c, d math32.Vector3) []collisionTriangle {

	return []collisionTriangle{
		{name: name, a: a, b: d, c: c},
		{name: name, a: a, b: c, c: b},
	}
}
