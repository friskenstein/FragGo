package game

import (
	"fmt"
	"time"

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

	logoPath, err := fragGoLogoPath()
	if err == nil {
		g.logoImage, err = gui.NewImage(logoPath)
		if err == nil {
			if g.logoImage.Height() > 0 {
				g.logoAspect = g.logoImage.Width() / g.logoImage.Height()
			}
			if g.logoAspect <= 0 {
				g.logoAspect = 3
			}
			g.scene.Add(g.logoImage)
		}
	}

	g.menuTitleLabel = gui.NewLabel("")
	g.menuTitleLabel.SetFontSize(24)
	g.menuTitleLabel.SetColor(&math32.Color{R: 0.97, G: 0.98, B: 1.0})
	g.scene.Add(g.menuTitleLabel)

	g.menuBodyLabel = gui.NewLabel("")
	g.menuBodyLabel.SetFontSize(22)
	g.menuBodyLabel.SetColor(&math32.Color{R: 0.87, G: 0.91, B: 0.98})
	g.scene.Add(g.menuBodyLabel)

	g.rosterLabel = gui.NewLabel("")
	g.rosterLabel.SetFontSize(18)
	g.rosterLabel.SetColor(&math32.Color{R: 0.79, G: 0.84, B: 0.92})
	g.scene.Add(g.rosterLabel)
}

func (g *Game) refreshHUD() {

	if g.phase != phaseMatch {
		g.infoLabel.SetText("")
		g.crosshair.SetText("")
		if g.logoImage != nil {
			g.logoImage.SetVisible(true)
		}

		if g.phase == phaseResults {
			g.menuTitleLabel.SetText(g.resultsTitle())
			g.menuBodyLabel.SetText(g.resultsSummary())
			g.rosterLabel.SetText(g.resultsScoreboard())
			g.controlsLabel.SetText("Enter or Esc return to host lobby")
		} else {
			g.menuTitleLabel.SetText(g.menuTitle())
			g.menuBodyLabel.SetText(g.menuBody())
			g.rosterLabel.SetText(g.rosterPreview())
			g.controlsLabel.SetText(g.menuControls())
		}
	} else {
		if g.logoImage != nil {
			g.logoImage.SetVisible(false)
		}
		g.menuTitleLabel.SetText("")
		g.menuBodyLabel.SetText("")
		g.rosterLabel.SetText("")
		g.crosshair.SetText("+")

		accuracy := 0.0
		if g.shotsFired > 0 {
			accuracy = float64(g.shotsHit) * 100 / float64(g.shotsFired)
		}

		height := "ground"
		if g.playerPos.Y > 0.1 {
			height = fmt.Sprintf("%.1fm", g.playerPos.Y)
		}

		timeLeft := g.matchConfig.RoundDuration - g.roundElapsed
		if timeLeft < 0 {
			timeLeft = 0
		}

		g.infoLabel.SetText(fmt.Sprintf(
			"Hosted Match\nTime Left: %s  Speed: %.2fx  Seats: %d\nFrags: %d  Accuracy: %.0f%%  Velocity: %.1f  Height: %s",
			formatClock(timeLeft),
			g.currentSpeedMultiplier(),
			g.matchConfig.PlayerSlots,
			g.frags,
			accuracy,
			g.playerVelocity.Length(),
			height,
		))
		g.controlsLabel.SetText("WASD move  Space jump  Shift boost  LMB fire  Esc release mouse  F2 lobby  R reset")
	}

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
	if g.logoImage != nil {
		logoWidth := float32(420)
		maxWidth := width * 0.34
		if maxWidth < logoWidth {
			logoWidth = maxWidth
		}
		if logoWidth < 220 {
			logoWidth = 220
		}
		logoHeight := logoWidth / g.logoAspect
		g.logoImage.SetSize(logoWidth, logoHeight)
		g.logoImage.SetPosition(width*0.5-g.logoImage.Width()*0.5, 36)
	}
	g.menuTitleLabel.SetPosition(width*0.5-g.menuTitleLabel.Width()*0.5, 182)
	g.menuBodyLabel.SetPosition(110, 240)
	g.rosterLabel.SetPosition(width-430, 240)
	g.controlsLabel.SetPosition(24, height-g.controlsLabel.Height()-24)
	g.crosshair.SetPosition(width*0.5-g.crosshair.Width()*0.5, height*0.5-g.crosshair.Height()*0.62)
	g.statusLabel.SetPosition(width*0.5-g.statusLabel.Width()*0.5, 28)
}

func formatClock(remaining time.Duration) string {

	totalSeconds := int(remaining.Seconds())
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
