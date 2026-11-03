package steam

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Game is an installed Steam app that is a game (not a tool or runtime).
type Game struct {
	AppID      string
	Name       string
	InstallDir string
	Library    string
	LastPlayed int64
	// HasProtonPrefix is true when a Proton prefix exists for the app. When false the game
	// is probably native Linux, or has not been run through Proton yet.
	HasProtonPrefix bool
}

const stateFullyInstalled = 4

// toolPrefixes name apps that are Steam tools, runtimes or compatibility tools.
var toolPrefixes = []string{
	"Proton", "Steam Linux Runtime", "Steamworks Common Redistributables", "Steam Play",
}

// IsTool reports whether an installed app's name marks it as a tool rather than a game.
func IsTool(name string) bool {
	for _, p := range toolPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// Games lists the installed games of every library, sorted by name.
func (in Install) Games() ([]Game, error) {
	libs, err := in.Libraries()
	if err != nil {
		return nil, err
	}
	var games []Game
	seen := map[string]bool{}
	for _, lib := range libs {
		manifests, _ := filepath.Glob(filepath.Join(lib, "steamapps", "appmanifest_*.acf"))
		for _, m := range manifests {
			g, ok := readManifest(m, lib)
			if !ok || seen[g.AppID] {
				continue
			}
			seen[g.AppID] = true
			games = append(games, g)
		}
	}
	sort.Slice(games, func(i, j int) bool {
		a, b := strings.ToLower(games[i].Name), strings.ToLower(games[j].Name)
		if a != b {
			return a < b
		}
		return games[i].AppID < games[j].AppID
	})
	return games, nil
}

func readManifest(path, lib string) (Game, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Game{}, false
	}
	root, err := ParseVDF(data)
	if err != nil {
		return Game{}, false
	}
	st := root.Child("AppState")
	if st == nil {
		return Game{}, false
	}
	flags, _ := strconv.Atoi(st.Str("StateFlags"))
	name := st.Str("name")
	id := st.Str("appid")
	if id == "" || name == "" || flags&stateFullyInstalled == 0 || IsTool(name) {
		return Game{}, false
	}
	played, _ := strconv.ParseInt(st.Str("LastPlayed"), 10, 64)
	_, perr := os.Stat(filepath.Join(lib, "steamapps", "compatdata", id))
	return Game{
		AppID:           id,
		Name:            name,
		InstallDir:      filepath.Join(lib, "steamapps", "common", st.Str("installdir")),
		Library:         lib,
		LastPlayed:      played,
		HasProtonPrefix: perr == nil,
	}, true
}
