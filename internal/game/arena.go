package game

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type arenaDefinition struct {
	ID        string
	Label     string
	AssetName string
	Scale     float32
}

var arenaScaleByID = map[string]float32{
	"arena":    1.0,
	"blockout": 1.0,
	"dust2":    2.0,
	"turbine":  1.0,
}

func loadArenaDefinitions() ([]arenaDefinition, error) {

	levelsPath, err := assetPath("levels")
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(levelsPath)
	if err != nil {
		return nil, err
	}

	arenas := make([]arenaDefinition, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".glb" {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		arenas = append(arenas, arenaDefinition{
			ID:        id,
			Label:     formatArenaLabel(id),
			AssetName: entry.Name(),
			Scale:     arenaScaleForID(id),
		})
	}

	sort.Slice(arenas, func(i, j int) bool {
		return arenas[i].Label < arenas[j].Label
	})

	if len(arenas) == 0 {
		return nil, fmt.Errorf("no arena assets found in %s", levelsPath)
	}

	return arenas, nil
}

func arenaScaleForID(id string) float32 {

	if scale, ok := arenaScaleByID[id]; ok {
		return scale
	}
	return 1.0
}

func defaultArenaID(arenas []arenaDefinition) string {

	for _, arena := range arenas {
		if arena.ID == "dust2" {
			return arena.ID
		}
	}
	if len(arenas) == 0 {
		return ""
	}
	return arenas[0].ID
}

func formatArenaLabel(id string) string {

	parts := strings.FieldsFunc(id, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	if len(parts) == 0 {
		return id
	}

	for idx, part := range parts {
		if part == "" {
			continue
		}
		parts[idx] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func (g *Game) selectedArena() (arenaDefinition, bool) {

	for _, arena := range g.arenas {
		if arena.ID == g.matchConfig.ArenaID {
			return arena, true
		}
	}
	if len(g.arenas) == 0 {
		return arenaDefinition{}, false
	}
	return g.arenas[0], true
}

func (g *Game) selectedArenaIndex() int {

	for idx, arena := range g.arenas {
		if arena.ID == g.matchConfig.ArenaID {
			return idx
		}
	}
	return 0
}
