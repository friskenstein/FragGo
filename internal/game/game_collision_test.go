package game

import (
	"testing"

	"github.com/g3n/engine/math32"
)

func TestResolvePlayerAxisStopsAtWall(t *testing.T) {

	g := Game{
		playerGrounded: true,
		colliders: []boxCollider{
			{
				name:   "wall",
				center: math32.Vector3{X: 2, Y: 1, Z: 0},
				size:   math32.Vector3{X: 2, Y: 2, Z: 2},
			},
		},
	}

	nextX, nextVelocity := g.resolvePlayerAxis(math32.Vector3{}, 2.2, true, 10)
	expectedX := float32(1 - playerRadius - collisionEpsilon)
	if math32.Abs(nextX-expectedX) > 0.0001 {
		t.Fatalf("expected collision stop at %.3f, got %.3f", expectedX, nextX)
	}
	if nextVelocity != 0 {
		t.Fatalf("expected horizontal velocity to clear on impact, got %.3f", nextVelocity)
	}
}

func TestResolvePlayerAxisAllowsWalkableStepUp(t *testing.T) {

	g := Game{
		playerGrounded: true,
		colliders: []boxCollider{
			{
				name:     "step",
				center:   math32.Vector3{X: 2, Y: 0.25, Z: 0},
				size:     math32.Vector3{X: 2, Y: 0.5, Z: 2},
				walkable: true,
			},
		},
	}

	targetDelta := float32(2.2)
	nextX, nextVelocity := g.resolvePlayerAxis(math32.Vector3{}, targetDelta, true, 10)
	if math32.Abs(nextX-targetDelta) > 0.0001 {
		t.Fatalf("expected walkable step to allow movement to %.3f, got %.3f", targetDelta, nextX)
	}
	if nextVelocity != 10 {
		t.Fatalf("expected horizontal velocity to be preserved, got %.3f", nextVelocity)
	}
}

func TestResolveGroundPrefersReachableSupportBelowOverOverheadWalkway(t *testing.T) {

	g := Game{
		playerPos:      math32.Vector3{X: 0, Y: 0, Z: 34},
		playerVelocity: math32.Vector3{},
		playerGrounded: true,
		platforms: []platform{
			{
				center: math32.Vector3{X: 0, Y: -0.5, Z: 0},
				size:   math32.Vector3{X: 104, Y: 1, Z: 104},
			},
			{
				center: math32.Vector3{X: 0, Y: 2.5, Z: 30},
				size:   math32.Vector3{X: 24, Y: 1, Z: 10},
			},
		},
	}

	landed, supportY := g.resolveGround(0)
	if !landed {
		t.Fatal("expected floor support under player")
	}
	if supportY != 0 {
		t.Fatalf("expected floor support at 0, got %.3f", supportY)
	}
}
