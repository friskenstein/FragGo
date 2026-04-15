package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/window"
)

const (
	menuItemMode = iota
	menuItemArena
	menuItemRoundTime
	menuItemSpeedStart
	menuItemSpeedEnd
	menuItemSlots
	menuItemFillBots
	menuItemStart
	menuItemCount
)

func (g *Game) handleMenuInput(key window.Key) {

	switch key {
	case window.KeyUp:
		g.menuSelection = (g.menuSelection - 1 + menuItemCount) % menuItemCount
	case window.KeyDown:
		g.menuSelection = (g.menuSelection + 1) % menuItemCount
	case window.KeyLeft:
		g.adjustMenuSetting(-1)
	case window.KeyRight:
		g.adjustMenuSetting(1)
	case window.KeyEnter:
		g.activateMenuSelection()
	case window.KeyEscape:
		if g.sessionMode == sessionModeJoin {
			g.sessionMode = sessionModeHost
			g.setStatus("Back to hosted match setup", 1200*time.Millisecond)
		}
	}
}

func (g *Game) handleResultsInput(key window.Key) {

	switch key {
	case window.KeyEnter, window.KeyEscape:
		g.returnToMenu("Back in host lobby")
	}
}

func (g *Game) adjustMenuSetting(direction int) {

	switch g.menuSelection {
	case menuItemMode:
		if direction != 0 {
			if g.sessionMode == sessionModeHost {
				g.sessionMode = sessionModeJoin
			} else {
				g.sessionMode = sessionModeHost
			}
		}
	case menuItemArena:
		if direction != 0 {
			if err := g.stepArenaSelection(direction); err != nil {
				g.setStatus(fmt.Sprintf("Failed to load map: %v", err), 2*time.Second)
			}
		}
	case menuItemRoundTime:
		next := g.matchConfig.RoundDuration + time.Duration(direction)*time.Minute
		if next < 2*time.Minute {
			next = 2 * time.Minute
		}
		if next > 15*time.Minute {
			next = 15 * time.Minute
		}
		g.matchConfig.RoundDuration = next
	case menuItemSpeedStart:
		g.matchConfig.StartSpeed = math32.Clamp(g.matchConfig.StartSpeed+0.25*float32(direction), 0.5, 4.0)
		if g.matchConfig.EndSpeed < g.matchConfig.StartSpeed {
			g.matchConfig.EndSpeed = g.matchConfig.StartSpeed
		}
	case menuItemSpeedEnd:
		g.matchConfig.EndSpeed = math32.Clamp(g.matchConfig.EndSpeed+0.25*float32(direction), g.matchConfig.StartSpeed, 4.5)
	case menuItemSlots:
		next := g.matchConfig.PlayerSlots + direction
		if next < 2 {
			next = 2
		}
		if next > 8 {
			next = 8
		}
		g.matchConfig.PlayerSlots = next
	case menuItemFillBots:
		if direction != 0 {
			g.matchConfig.FillBots = !g.matchConfig.FillBots
		}
	}
}

func (g *Game) activateMenuSelection() {

	if g.menuSelection == menuItemFillBots {
		g.matchConfig.FillBots = !g.matchConfig.FillBots
		return
	}

	if g.menuSelection != menuItemStart {
		g.adjustMenuSetting(1)
		return
	}

	if g.sessionMode == sessionModeJoin {
		g.setStatus("Remote join is not wired yet; host locally for now", 2*time.Second)
		return
	}

	if err := g.startHostedMatch(); err != nil {
		g.setStatus(fmt.Sprintf("Failed to start match: %v", err), 2*time.Second)
	}
}

func (g *Game) menuTitle() string {

	if g.sessionMode == sessionModeJoin {
		return "Join Flow"
	}
	return "Host Lobby"
}

func (g *Game) menuBody() string {

	mode := "Host local match"
	if g.sessionMode == sessionModeJoin {
		mode = "Join remote host"
	}

	rows := []string{
		g.formatMenuRow(menuItemMode, "Session", mode),
		g.formatMenuRow(menuItemArena, "Map", g.menuArenaLabel()),
		g.formatMenuRow(menuItemRoundTime, "Round Time", fmt.Sprintf("%dm", int(g.matchConfig.RoundDuration.Minutes()))),
		g.formatMenuRow(menuItemSpeedStart, "Start Speed", fmt.Sprintf("%.2fx", g.matchConfig.StartSpeed)),
		g.formatMenuRow(menuItemSpeedEnd, "End Speed", fmt.Sprintf("%.2fx", g.matchConfig.EndSpeed)),
		g.formatMenuRow(menuItemSlots, "Seats", fmt.Sprintf("%d", g.matchConfig.PlayerSlots)),
		g.formatMenuRow(menuItemFillBots, "Fill Bots", onOffLabel(g.matchConfig.FillBots)),
	}

	if g.sessionMode == sessionModeJoin {
		rows = append(rows, g.formatMenuRow(menuItemStart, "Connect", "Not implemented yet"))
	} else {
		rows = append(rows, g.formatMenuRow(menuItemStart, "Start Match", "Enter"))
	}

	return strings.Join(rows, "\n")
}

func (g *Game) rosterPreview() string {

	slots := []string{"You (Host gopher)"}
	for idx := 1; idx < g.matchConfig.PlayerSlots; idx++ {
		if g.matchConfig.FillBots {
			slots = append(slots, fmt.Sprintf("Bot %d", idx))
			continue
		}
		slots = append(slots, fmt.Sprintf("Open Seat %d", idx))
	}

	header := "Lobby Seats"
	if g.sessionMode == sessionModeJoin {
		header = "Join Notes"
		slots = []string{
			"Remote host/join transport is the next step.",
			"This pass wires the host-side menu, timing,",
			"bot seats, and match-state flow first.",
		}
	}

	return header + "\n" + strings.Join(slots, "\n")
}

func (g *Game) menuControls() string {

	if g.sessionMode == sessionModeJoin {
		return "Up/Down select  Left/Right adjust  Enter attempt join  Esc back"
	}
	return "Up/Down select  Left/Right adjust  Enter apply/start"
}

func (g *Game) formatMenuRow(idx int, label, value string) string {

	prefix := "  "
	if g.menuSelection == idx {
		prefix = "> "
	}
	return fmt.Sprintf("%s%-12s %s", prefix, label, value)
}

func onOffLabel(enabled bool) string {

	if enabled {
		return "On"
	}
	return "Off"
}

func (g *Game) menuArenaLabel() string {

	arena, ok := g.selectedArena()
	if !ok {
		return "Unavailable"
	}
	return fmt.Sprintf("%s (%.2fx)", arena.Label, arena.Scale)
}

func (g *Game) stepArenaSelection(direction int) error {

	if len(g.arenas) == 0 || direction == 0 {
		return nil
	}

	currentIndex := g.selectedArenaIndex()
	nextIndex := (currentIndex + direction + len(g.arenas)) % len(g.arenas)
	nextArena := g.arenas[nextIndex]
	if nextArena.ID == g.matchConfig.ArenaID {
		return nil
	}

	previousArenaID := g.matchConfig.ArenaID
	g.matchConfig.ArenaID = nextArena.ID
	if err := g.reloadArena(); err != nil {
		g.matchConfig.ArenaID = previousArenaID
		return err
	}

	g.setStatus(fmt.Sprintf("Selected map: %s", nextArena.Label), 1200*time.Millisecond)
	return nil
}
