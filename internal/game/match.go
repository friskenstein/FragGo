package game

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/math32"
)

type gamePhase int

const (
	phaseMenu gamePhase = iota
	phaseMatch
	phaseResults
)

type sessionMode int

const (
	sessionModeHost sessionMode = iota
	sessionModeJoin
)

type matchConfig struct {
	RoundDuration time.Duration
	StartSpeed    float32
	EndSpeed      float32
	PlayerSlots   int
	FillBots      bool
}

type combatant struct {
	name         string
	root         *core.Node
	spawn        math32.Vector3
	positionVec  math32.Vector3
	yaw          float32
	radius       float32
	health       int
	alive        bool
	bot          bool
	respawnTimer time.Duration
	moveRadius   float32
	moveSpeed    float32
	phase        float32
	score        int
	deaths       int
}

var defaultCombatantSpawnPoints = []math32.Vector3{
	{X: -30, Y: 1, Z: -26},
	{X: 30, Y: 2, Z: -18},
	{X: 0, Y: 4, Z: -26},
	{X: 30, Y: 2, Z: 24},
	{X: -30, Y: 1, Z: 24},
	{X: 0, Y: 2, Z: 30},
	{X: -18, Y: 0, Z: 34},
}

func defaultMatchConfig() matchConfig {

	return matchConfig{
		RoundDuration: 5 * time.Minute,
		StartSpeed:    1.0,
		EndSpeed:      1.8,
		PlayerSlots:   6,
		FillBots:      true,
	}
}

func (c *combatant) position() math32.Vector3 {

	return c.positionVec
}

func (c *combatant) applyDamage(amount int) bool {

	if !c.alive {
		return false
	}

	c.health -= amount
	if c.health > 0 {
		return false
	}

	c.health = 0
	c.alive = false
	c.deaths++
	c.respawnTimer = 1600 * time.Millisecond
	if c.root != nil {
		c.root.SetVisible(false)
	}
	return true
}

func (g *Game) configureMenuDefaults() {

	g.phase = phaseMenu
	g.sessionMode = sessionModeHost
	g.matchConfig = defaultMatchConfig()
	g.menuSelection = 0
}

func (g *Game) currentSpeedMultiplier() float32 {

	if g.matchConfig.RoundDuration <= 0 {
		return g.matchConfig.EndSpeed
	}

	progress := math32.Clamp(float32(g.roundElapsed)/float32(g.matchConfig.RoundDuration), 0, 1)
	return g.matchConfig.StartSpeed + (g.matchConfig.EndSpeed-g.matchConfig.StartSpeed)*progress
}

func scaleDuration(delta time.Duration, multiplier float32) time.Duration {

	if multiplier <= 0 {
		return 0
	}
	return time.Duration(float64(delta) * float64(multiplier))
}

func (g *Game) desiredBotCount() int {

	if !g.matchConfig.FillBots {
		return 0
	}

	if g.matchConfig.PlayerSlots <= 1 {
		return 0
	}

	return g.matchConfig.PlayerSlots - 1
}

func (g *Game) clearCombatants() {

	for _, combatant := range g.combatants {
		if combatant.root != nil {
			g.scene.Remove(combatant.root)
		}
	}
	g.combatants = nil
}

func (g *Game) spawnCombatants() error {

	g.clearCombatants()

	botCount := g.desiredBotCount()
	spawnPoints := g.combatantSpawns
	if len(spawnPoints) == 0 {
		spawnPoints = defaultCombatantSpawnPoints
	}

	for idx := 0; idx < botCount; idx++ {
		root, err := newGopherRoot(fmt.Sprintf("bot-%d", idx+1))
		if err != nil {
			return err
		}

		combatant := &combatant{
			name:       fmt.Sprintf("Bot %d", idx+1),
			root:       root,
			spawn:      spawnPoints[idx%len(spawnPoints)],
			radius:     0.9,
			health:     100,
			alive:      true,
			bot:        true,
			moveRadius: 2.2 + float32(idx%3)*0.7,
			moveSpeed:  0.45 + float32((idx+1)%4)*0.15,
			phase:      float32(idx) * 0.9,
		}
		combatant.positionVec = combatant.spawn
		combatant.root.SetPositionVec(&combatant.positionVec)
		g.scene.Add(combatant.root)
		g.combatants = append(g.combatants, combatant)
	}

	return nil
}

func (g *Game) startHostedMatch() error {

	g.matchTime = 0
	g.roundElapsed = 0
	g.frags = 0
	g.playerDeaths = 0
	g.shotsFired = 0
	g.shotsHit = 0
	g.fireCooldown = 0
	g.fireQueued = false

	g.resetPlayer()
	if err := g.spawnCombatants(); err != nil {
		return err
	}

	g.phase = phaseMatch
	g.captureMouse()
	g.setStatus("Hosted match live", 1200*time.Millisecond)
	return nil
}

func (g *Game) endMatch() {

	g.phase = phaseResults
	g.releaseMouse()
	g.setStatus(fmt.Sprintf("Round complete: %d frags", g.frags), 3*time.Second)
}

func (g *Game) returnToMenu(status string) {

	g.phase = phaseMenu
	g.releaseMouse()
	g.clearCombatants()
	if status != "" {
		g.setStatus(status, 2*time.Second)
	}
}

func (g *Game) updateCombatants(delta time.Duration) {

	seconds := float32(g.matchTime.Seconds())
	for _, combatant := range g.combatants {
		if !combatant.alive {
			combatant.respawnTimer -= delta
			if combatant.respawnTimer > 0 {
				continue
			}

			combatant.alive = true
			combatant.health = 100
			combatant.positionVec = combatant.spawn
			combatant.root.SetVisible(true)
		}

		if combatant.bot {
			angle := seconds*combatant.moveSpeed + combatant.phase
			combatant.positionVec = combatant.spawn
			combatant.positionVec.X += math32.Cos(angle) * combatant.moveRadius
			combatant.positionVec.Z += math32.Sin(angle) * combatant.moveRadius * 0.85
			combatant.yaw = angle + math32.Pi*0.5
		}

		combatant.root.SetPositionVec(&combatant.positionVec)
		combatant.root.SetRotation(0, math32.Pi/2-combatant.yaw, 0)
	}
}

func (g *Game) resultsTitle() string {

	return "Round Results"
}

func (g *Game) resultsSummary() string {

	timeLabel := formatClock(g.matchConfig.RoundDuration)
	if g.roundElapsed > 0 && g.roundElapsed < g.matchConfig.RoundDuration {
		timeLabel = formatClock(g.roundElapsed)
	}

	return fmt.Sprintf(
		"Hosted Match Complete\nDuration: %s  Final Speed: %.2fx\nYour Accuracy: %.0f%%  Press Enter to return to lobby",
		timeLabel,
		g.currentSpeedMultiplier(),
		g.localAccuracy(),
	)
}

func (g *Game) resultsScoreboard() string {

	type resultRow struct {
		name     string
		score    int
		deaths   int
		accuracy string
	}

	rows := []resultRow{
		{
			name:     "You",
			score:    g.frags,
			deaths:   g.playerDeaths,
			accuracy: fmt.Sprintf("%.0f%%", g.localAccuracy()),
		},
	}

	for _, combatant := range g.combatants {
		rows = append(rows, resultRow{
			name:     combatant.name,
			score:    combatant.score,
			deaths:   combatant.deaths,
			accuracy: "-",
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].score != rows[j].score {
			return rows[i].score > rows[j].score
		}
		if rows[i].deaths != rows[j].deaths {
			return rows[i].deaths < rows[j].deaths
		}
		return rows[i].name < rows[j].name
	})

	lines := []string{"Player           Frags  Deaths  Accuracy"}
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("%-15s %5d  %6d  %s", row.name, row.score, row.deaths, row.accuracy))
	}
	return strings.Join(lines, "\n")
}

func (g *Game) localAccuracy() float64 {

	if g.shotsFired == 0 {
		return 0
	}
	return float64(g.shotsHit) * 100 / float64(g.shotsFired)
}
