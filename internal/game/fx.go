package game

import (
	"time"

	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
)

func (g *Game) buildEffects() {

	g.cameraTrace = newTraceLine(
		&math32.Color{R: 0.16, G: 0.82, B: 1.0},
		&math32.Color{R: 0.65, G: 0.96, B: 1.0},
	)
	g.muzzleTrace = newTraceLine(
		&math32.Color{R: 1.0, G: 0.52, B: 0.18},
		&math32.Color{R: 1.0, G: 0.9, B: 0.42},
	)
	g.scene.Add(g.cameraTrace)
	g.scene.Add(g.muzzleTrace)
	g.cameraTrace.SetVisible(false)
	g.muzzleTrace.SetVisible(false)

	impactMat := material.NewStandard(&math32.Color{R: 1.0, G: 0.78, B: 0.3})
	impactMat.SetEmissiveColor(&math32.Color{R: 0.9, G: 0.5, B: 0.18})
	g.impactFlash = graphic.NewMesh(geometry.NewSphere(0.12, 10, 8), impactMat)
	g.impactFlash.SetVisible(false)
	g.scene.Add(g.impactFlash)
}

func newTraceLine(startColor, endColor *math32.Color) *graphic.Lines {

	positions := math32.NewArrayF32(0, 12)
	positions.Append(
		0, 0, 0, startColor.R, startColor.G, startColor.B,
		0, 0, 0, endColor.R, endColor.G, endColor.B,
	)

	geom := geometry.NewGeometry()
	geom.AddVBO(
		gls.NewVBO(positions).
			AddAttrib(gls.VertexPosition).
			AddAttrib(gls.VertexColor),
	)

	line := graphic.NewLines(geom, material.NewBasic())
	return line
}

func (g *Game) showTrace(line *graphic.Lines, start, end math32.Vector3) {

	vbo := line.GetGeometry().VBO(gls.VertexPosition)
	base := *vbo.Buffer()
	buffer := math32.NewArrayF32(0, 12)
	buffer.Append(
		start.X, start.Y, start.Z,
		base[3], base[4], base[5],
		end.X, end.Y, end.Z,
		base[9], base[10], base[11],
	)
	vbo.SetBuffer(buffer)
	line.SetVisible(true)
	g.traceTTL = 110 * time.Millisecond
}

func (g *Game) showImpact(point math32.Vector3) {

	g.impactFlash.SetPositionVec(&point)
	g.impactFlash.SetScale(1, 1, 1)
	g.impactFlash.SetVisible(true)
	g.impactTTL = 130 * time.Millisecond
}

func (g *Game) hideImpact() {

	g.impactTTL = 0
	g.impactFlash.SetVisible(false)
}

func (g *Game) updateEffects(delta time.Duration) {

	if g.traceTTL > 0 {
		g.traceTTL -= delta
		if g.traceTTL <= 0 {
			g.traceTTL = 0
			g.cameraTrace.SetVisible(false)
			g.muzzleTrace.SetVisible(false)
		}
	}

	if g.impactTTL > 0 {
		g.impactTTL -= delta
		if g.impactTTL <= 0 {
			g.impactTTL = 0
			g.impactFlash.SetVisible(false)
			return
		}

		scale := 1 + 1.7*(1-float32(g.impactTTL)/float32(130*time.Millisecond))
		g.impactFlash.SetScale(scale, scale, scale)
	}
}
