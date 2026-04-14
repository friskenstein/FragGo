package game

import "github.com/g3n/engine/math32"

type boxCollider struct {
	name   string
	center math32.Vector3
	size   math32.Vector3
}

type traceHit struct {
	hit      bool
	distance float32
	point    math32.Vector3
	target   *targetDummy
	collider *boxCollider
}

func (h traceHit) blockerName() string {

	if h.target != nil {
		return h.target.name
	}
	if h.collider != nil && h.collider.name != "" {
		return h.collider.name
	}
	return "geometry"
}

func (g *Game) traceScene(origin math32.Vector3, direction *math32.Vector3, maxDistance float32) traceHit {

	best := traceHit{
		hit:      false,
		distance: maxDistance,
	}

	for idx := range g.colliders {
		collider := &g.colliders[idx]
		if distance, ok := rayBoxHit(origin, direction, collider.center, collider.size); ok && distance <= best.distance {
			best.hit = true
			best.distance = distance
			best.point = origin
			best.point.Add(direction.Clone().MultiplyScalar(distance))
			best.target = nil
			best.collider = collider
		}
	}

	for _, target := range g.targets {
		if !target.alive {
			continue
		}
		if distance, ok := raySphereHit(origin, direction, target.position(), target.radius); ok && distance <= best.distance {
			best.hit = true
			best.distance = distance
			best.point = origin
			best.point.Add(direction.Clone().MultiplyScalar(distance))
			best.target = target
			best.collider = nil
		}
	}

	return best
}

func rayBoxHit(origin math32.Vector3, direction *math32.Vector3, center, size math32.Vector3) (float32, bool) {

	min := math32.Vector3{
		X: center.X - size.X*0.5,
		Y: center.Y - size.Y*0.5,
		Z: center.Z - size.Z*0.5,
	}
	max := math32.Vector3{
		X: center.X + size.X*0.5,
		Y: center.Y + size.Y*0.5,
		Z: center.Z + size.Z*0.5,
	}

	tMin := float32(0)
	tMax := float32(1e9)

	for _, axis := range []struct {
		origin float32
		dir    float32
		min    float32
		max    float32
	}{
		{origin: origin.X, dir: direction.X, min: min.X, max: max.X},
		{origin: origin.Y, dir: direction.Y, min: min.Y, max: max.Y},
		{origin: origin.Z, dir: direction.Z, min: min.Z, max: max.Z},
	} {
		if math32.Abs(axis.dir) < 0.00001 {
			if axis.origin < axis.min || axis.origin > axis.max {
				return 0, false
			}
			continue
		}

		t1 := (axis.min - axis.origin) / axis.dir
		t2 := (axis.max - axis.origin) / axis.dir
		if t1 > t2 {
			t1, t2 = t2, t1
		}

		tMin = math32.Max(tMin, t1)
		tMax = math32.Min(tMax, t2)
		if tMin > tMax {
			return 0, false
		}
	}

	if tMax < 0 {
		return 0, false
	}
	if tMin >= 0 {
		return tMin, true
	}
	return tMax, true
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
