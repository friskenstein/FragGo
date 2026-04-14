package game

import (
	"fmt"

	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/math32"
)

func (g *Game) buildHUD() {

	g.infoLabel = gui.NewLabel("")
	g.infoLabel.SetFontSize(18)
	g.infoLabel.SetColor(&math32.Color{R: 0.95, G: 0.97, B: 1.0})
	g.scene.Add(g.infoLabel)

	g.controlsLabel = gui.NewLabel("WASD move  Space jump  Shift boost  LMB fire  Esc release mouse  R reset")
	g.controlsLabel.SetFontSize(14)
	g.controlsLabel.SetColor(&math32.Color{R: 0.73, G: 0.79, B: 0.88})
	g.scene.Add(g.controlsLabel)

	g.crosshair = gui.NewLabel("+")
	g.crosshair.SetFontSize(36)
	g.crosshair.SetColor(&math32.Color{R: 1.0, G: 0.46, B: 0.24})
	g.scene.Add(g.crosshair)

	g.statusLabel = gui.NewLabel("")
	g.statusLabel.SetFontSize(22)
	g.statusLabel.SetColor(&math32.Color{R: 1.0, G: 0.88, B: 0.55})
	g.scene.Add(g.statusLabel)
}

func (g *Game) refreshHUD() {

	accuracy := 0.0
	if g.shotsFired > 0 {
		accuracy = float64(g.shotsHit) * 100 / float64(g.shotsFired)
	}

	height := "ground"
	if g.playerPos.Y > 0.1 {
		height = fmt.Sprintf("%.1fm", g.playerPos.Y)
	}

	g.infoLabel.SetText(fmt.Sprintf(
		"Prototype Arena\nFrags: %d  Accuracy: %.0f%%\nHeight: %s  Velocity: %.1f",
		g.frags,
		accuracy,
		height,
		g.playerVelocity.Length(),
	))

	if g.statusText == "" {
		g.statusLabel.SetText("")
	} else {
		g.statusLabel.SetText(fmt.Sprintf("%s", g.statusText))
	}

	width, heightPx := g.win.GetSize()
	g.layoutHUD(float32(width), float32(heightPx))
}

func (g *Game) layoutHUD(width, height float32) {

	if g.infoLabel == nil {
		return
	}

	g.infoLabel.SetPosition(24, 22)
	g.controlsLabel.SetPosition(24, height-g.controlsLabel.Height()-24)
	g.crosshair.SetPosition(width*0.5-g.crosshair.Width()*0.5, height*0.5-g.crosshair.Height()*0.62)
	g.statusLabel.SetPosition(width*0.5-g.statusLabel.Width()*0.5, 28)
}
