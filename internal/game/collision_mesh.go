package game

import (
	"math"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/math32"
)

const (
	collisionCellSize        = 6.0
	walkableSurfaceMinNormal = 0.55
	collisionHitEpsilon      = 0.001
)

type collisionCell struct {
	x int
	z int
}

type collisionTriangle struct {
	name     string
	a        math32.Vector3
	b        math32.Vector3
	c        math32.Vector3
	normal   math32.Vector3
	bounds   math32.Box3
	walkable bool
}

type meshCollision struct {
	cellSize  float32
	bounds    math32.Box3
	triangles []collisionTriangle
	cells     map[collisionCell][]int
}

type meshHit struct {
	distance float32
	point    math32.Vector3
	triangle *collisionTriangle
}

func buildCollisionMesh(root core.INode) (*meshCollision, error) {

	root.UpdateMatrixWorld()

	collision := &meshCollision{
		cellSize: collisionCellSize,
		cells:    make(map[collisionCell][]int),
	}

	collectCollisionGeometry(root, collision)
	if len(collision.triangles) == 0 {
		return nil, nil
	}

	return collision, nil
}

func collectCollisionGeometry(node core.INode, collision *meshCollision) {

	if gr, ok := node.(graphic.IGraphic); ok {
		geom := gr.GetGeometry()
		matrixWorld := node.GetNode().MatrixWorld()
		geom.ReadFaces(func(vA, vB, vC math32.Vector3) bool {
			vA.ApplyMatrix4(&matrixWorld)
			vB.ApplyMatrix4(&matrixWorld)
			vC.ApplyMatrix4(&matrixWorld)
			collision.addTriangle(node.Name(), vA, vB, vC)
			return false
		})
	}

	for _, child := range node.Children() {
		collectCollisionGeometry(child, collision)
	}
}

func (c *meshCollision) addTriangle(name string, a, b, d math32.Vector3) {

	normal := math32.Normal(&a, &b, &d, nil)
	if normal.LengthSq() < 0.000001 {
		return
	}

	bounds := boxFromTriangle(a, b, d)
	if !boxIsFinite(bounds) {
		return
	}

	tri := collisionTriangle{
		name:     name,
		a:        a,
		b:        b,
		c:        d,
		normal:   *normal,
		bounds:   bounds,
		walkable: normal.Y >= walkableSurfaceMinNormal,
	}

	index := len(c.triangles)
	c.triangles = append(c.triangles, tri)
	if index == 0 {
		c.bounds = bounds
	} else {
		unionBox(&c.bounds, bounds)
	}

	minCellX := c.cellCoord(bounds.Min.X)
	maxCellX := c.cellCoord(bounds.Max.X)
	minCellZ := c.cellCoord(bounds.Min.Z)
	maxCellZ := c.cellCoord(bounds.Max.Z)

	for cellX := minCellX; cellX <= maxCellX; cellX++ {
		for cellZ := minCellZ; cellZ <= maxCellZ; cellZ++ {
			cell := collisionCell{x: cellX, z: cellZ}
			c.cells[cell] = append(c.cells[cell], index)
		}
	}
}

func (c *meshCollision) cellCoord(value float32) int {

	return int(math.Floor(float64(value / c.cellSize)))
}

func (c *meshCollision) raycast(origin, direction math32.Vector3, maxDistance float32, filter func(*collisionTriangle) bool) (meshHit, bool) {

	if c == nil || maxDistance <= 0 || direction.LengthSq() == 0 {
		return meshHit{}, false
	}

	rayDirection := direction.Clone().Normalize()
	ray := math32.NewRay(&origin, rayDirection)
	best := meshHit{}
	found := false
	dir := ray.Direction()

	for _, index := range c.collectRayCandidates(origin, dir, maxDistance) {
		tri := &c.triangles[index]
		if filter != nil && !filter(tri) {
			continue
		}

		var point math32.Vector3
		if !ray.IntersectTriangle(&tri.a, &tri.b, &tri.c, false, &point) {
			continue
		}

		distance := origin.DistanceTo(&point)
		if distance < collisionHitEpsilon || distance > maxDistance {
			continue
		}
		if !found || distance < best.distance {
			best = meshHit{
				distance: distance,
				point:    point,
				triangle: tri,
			}
			found = true
		}
	}

	return best, found
}

func (c *meshCollision) supportAt(x, z, startY, minY float32) (float32, bool) {

	if c == nil || startY <= minY {
		return 0, false
	}

	origin := math32.Vector3{X: x, Y: startY, Z: z}
	direction := math32.Vector3{Y: -1}
	hit, ok := c.raycast(origin, direction, startY-minY, func(tri *collisionTriangle) bool {
		return tri.walkable
	})
	if !ok {
		return 0, false
	}
	return hit.point.Y, true
}

func (c *meshCollision) collectRayCandidates(origin, direction math32.Vector3, maxDistance float32) []int {

	seenCells := make(map[collisionCell]struct{})
	seenTriangles := make(map[int]struct{})
	candidates := make([]int, 0, 256)

	addCell := func(cell collisionCell) {
		if _, ok := seenCells[cell]; ok {
			return
		}
		seenCells[cell] = struct{}{}
		for _, index := range c.cells[cell] {
			if _, ok := seenTriangles[index]; ok {
				continue
			}
			seenTriangles[index] = struct{}{}
			candidates = append(candidates, index)
		}
	}

	if math32.Abs(direction.X) < 0.0001 && math32.Abs(direction.Z) < 0.0001 {
		baseCell := collisionCell{x: c.cellCoord(origin.X), z: c.cellCoord(origin.Z)}
		for dx := -1; dx <= 1; dx++ {
			for dz := -1; dz <= 1; dz++ {
				addCell(collisionCell{x: baseCell.x + dx, z: baseCell.z + dz})
			}
		}
		return candidates
	}

	steps := int(math.Ceil(float64(maxDistance / (c.cellSize * 0.5))))
	if steps < 1 {
		steps = 1
	}

	for step := 0; step <= steps; step++ {
		t := maxDistance * float32(step) / float32(steps)
		x := origin.X + direction.X*t
		z := origin.Z + direction.Z*t
		baseCell := collisionCell{x: c.cellCoord(x), z: c.cellCoord(z)}
		for dx := -1; dx <= 1; dx++ {
			for dz := -1; dz <= 1; dz++ {
				addCell(collisionCell{x: baseCell.x + dx, z: baseCell.z + dz})
			}
		}
	}

	return candidates
}

func boxFromTriangle(a, b, c math32.Vector3) math32.Box3 {

	return math32.Box3{
		Min: math32.Vector3{
			X: min3(a.X, b.X, c.X),
			Y: min3(a.Y, b.Y, c.Y),
			Z: min3(a.Z, b.Z, c.Z),
		},
		Max: math32.Vector3{
			X: max3(a.X, b.X, c.X),
			Y: max3(a.Y, b.Y, c.Y),
			Z: max3(a.Z, b.Z, c.Z),
		},
	}
}

func unionBox(target *math32.Box3, other math32.Box3) {

	if other.Min.X < target.Min.X {
		target.Min.X = other.Min.X
	}
	if other.Min.Y < target.Min.Y {
		target.Min.Y = other.Min.Y
	}
	if other.Min.Z < target.Min.Z {
		target.Min.Z = other.Min.Z
	}
	if other.Max.X > target.Max.X {
		target.Max.X = other.Max.X
	}
	if other.Max.Y > target.Max.Y {
		target.Max.Y = other.Max.Y
	}
	if other.Max.Z > target.Max.Z {
		target.Max.Z = other.Max.Z
	}
}

func min3(a, b, c float32) float32 {

	return math32.Min(a, math32.Min(b, c))
}

func max3(a, b, c float32) float32 {

	return math32.Max(a, math32.Max(b, c))
}
