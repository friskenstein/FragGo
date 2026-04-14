package game

import "github.com/g3n/engine/math32"

type traceHit struct {
	hit         bool
	distance    float32
	point       math32.Vector3
	target      *combatant
	surfaceName string
}

func (h traceHit) blockerName() string {

	if h.target != nil {
		return h.target.name
	}
	if h.surfaceName != "" {
		return h.surfaceName
	}
	return "geometry"
}

func (g *Game) traceScene(origin math32.Vector3, direction *math32.Vector3, maxDistance float32) traceHit {

	best := traceHit{
		hit:      false,
		distance: maxDistance,
	}

	if g.arenaCollision != nil {
		hit, ok := g.arenaCollision.raycast(origin, *direction, maxDistance, nil)
		if ok {
			best.hit = true
			best.distance = hit.distance
			best.point = hit.point
			best.surfaceName = hit.triangle.name
		}
	}

	for _, combatant := range g.combatants {
		if !combatant.alive {
			continue
		}
		if distance, ok := raySphereHit(origin, direction, combatant.position(), combatant.radius); ok && distance <= best.distance {
			best.hit = true
			best.distance = distance
			best.point = origin
			best.point.Add(direction.Clone().MultiplyScalar(distance))
			best.target = combatant
			best.surfaceName = ""
		}
	}

	return best
}

func clipCameraTraceToMuzzlePlane(cameraOrigin math32.Vector3, cameraDir *math32.Vector3, muzzleOrigin math32.Vector3, maxDistance float32) (math32.Vector3, float32) {

	traceStart := cameraOrigin
	traceDistance := maxDistance

	toMuzzle := muzzleOrigin
	toMuzzle.Sub(&cameraOrigin)
	planeDistance := toMuzzle.Dot(cameraDir)
	if planeDistance <= 0 {
		return traceStart, traceDistance
	}
	if planeDistance >= maxDistance {
		traceStart.Add(cameraDir.Clone().MultiplyScalar(maxDistance))
		return traceStart, 0
	}

	traceStart.Add(cameraDir.Clone().MultiplyScalar(planeDistance))
	traceDistance -= planeDistance
	return traceStart, traceDistance
}
