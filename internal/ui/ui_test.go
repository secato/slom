package ui

import (
	"image/png"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/secato/slom/internal/catalog"
	"github.com/secato/slom/internal/steam"
	"github.com/secato/slom/internal/store"
)

// testApp builds an App without a window, with games and Steam's current lines set directly.
func testApp(t *testing.T) *App {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := New(st, t.TempDir())
	a.tools = catalog.AllTools(a.cat)
	a.defaults = catalog.Defaults{"screen.width": 2560, "screen.height": 1440, "screen.refresh": 165}
	a.users = []steam.User{{AccountID: "1", Name: "me", ConfigPath: t.TempDir() + "/localconfig.vdf"}}
	a.games = []steam.Game{
		{AppID: "570", Name: "Dota 2", HasProtonPrefix: true},
		{AppID: "1245620", Name: "ELDEN RING", HasProtonPrefix: true},
		{AppID: "2", Name: "Native Thing"},
	}
	a.steamLines = map[string]string{
		"570":     "gamescope --force-grab-cursor -W 2560 -H 1440 -r 180 -- %command%",
		"1245620": "PROTON_ENABLE_NGX_UPDATER=1 DXVK_NVAPI_DRS_NGX_DLSS_SR_OVERRIDE=on DXVK_NVAPI_DRS_NGX_DLSS_SR_OVERRIDE_RENDER_PRESET_SELECTION=render_preset_m %command%",
	}
	a.refilter()
	return a
}

func TestSelectGameImportsSteamLine(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.View, 1180, 780)
	if err := tt.Click("Dota 2"); err != nil {
		t.Fatal(err)
	}
	if a.cur == nil || a.cur.AppID != "570" {
		t.Fatalf("not selected: %+v", a.cur)
	}
	if got := a.preview(); got != "gamescope -W 2560 -H 1440 -r 180 --force-grab-cursor -- %command%" {
		t.Fatalf("preview %q", got)
	}
	if !tt.HasText("Proton") || !tt.HasText("Write a Proton log") {
		t.Error("the first group is not shown")
	}
	if err := tt.Click("Gamescope •"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Run inside gamescope") || !tt.HasText("Output width") {
		t.Error("gamescope options not shown after choosing its tab")
	}
	// selecting a game must not create an assignment until the user edits
	if _, ok := a.store.Assignment("570"); ok {
		t.Error("assignment created by merely selecting")
	}
}

func TestToggleOptionUpdatesPreviewAndStore(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.View, 1180, 780)
	tt.Click("Native Thing")
	// switching on an Int option takes the system-derived default and turns its wrapper on
	a.tab = 2 // the Gamescope tab, so the announcement of the automatic change shows
	a.setOption("gs.refresh", a.cat.Initial(a.cat.Get("gs.refresh"), a.defaults))
	tt.Frame()
	if got := a.preview(); got != "gamescope -r 165 -- %command%" {
		t.Fatalf("preview %q", got)
	}
	if as, ok := a.store.Assignment("2"); !ok || as.Overrides["gs.refresh"] != "165" || as.Overrides["gamescope"] != "1" {
		t.Fatalf("assignment %+v", as)
	}
	if !tt.HasText("Run inside gamescope turned on because Refresh rate needs it") {
		t.Error("the automatic change was not announced")
	}
}

func TestLineIsEmptyForBareCommand(t *testing.T) {
	a := testApp(t)
	if a.line() != "" {
		t.Fatalf("line %q", a.line())
	}
	if a.preview() != "%command%" {
		t.Fatalf("preview %q", a.preview())
	}
}

func TestPresetFlow(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.View, 1180, 780)
	tt.Click("Dota 2")
	a.saveAsPreset("Native 1440p")
	if a.presetID != "native-1440p" {
		t.Fatalf("preset id %q", a.presetID)
	}
	// another game takes the preset
	tt.Click("Native Thing")
	a.choosePreset("native-1440p")
	if got := a.preview(); got != "gamescope -W 2560 -H 1440 -r 180 --force-grab-cursor -- %command%" {
		t.Fatalf("preview %q", got)
	}
	// override one value on the second game, then change the preset: the override survives
	a.setOption("gs.refresh", "60")
	a.cur = &a.games[0]
	a.selectGame(a.cur)
	a.setOption("gs.refresh", "144")
	a.updatePreset()
	a.selectGame(&a.games[2])
	if got := a.preview(); got != "gamescope -W 2560 -H 1440 -r 60 --force-grab-cursor -- %command%" {
		t.Fatalf("override lost: %q", got)
	}
	if !a.pending("2") {
		t.Error("a configured game that differs from Steam should be pending")
	}
}

func TestMissingToolDisablesOption(t *testing.T) {
	a := testApp(t)
	a.tools = catalog.Tools{"gamescope": true, "mangohud": true, "gamemoderun": false}
	if r := a.cat.Reason(a.sel, a.tools, catalog.IDGameMode); !strings.Contains(r, "not installed") {
		t.Fatalf("reason %q", r)
	}
	tt := ui.NewTester(a.View, 1180, 780)
	tt.Click("Native Thing")
	tt.Click("Other")
	if !tt.HasText("Use GameMode") {
		t.Error("disabled option should still be shown")
	}
}

// TestScreenshot writes the README screenshot when SLOM_SCREENSHOT is set to a path.
func TestScreenshot(t *testing.T) {
	path := os.Getenv("SLOM_SCREENSHOT")
	if path == "" {
		t.Skip("set SLOM_SCREENSHOT=$PWD/docs/screenshot.png (absolute path)")
	}
	a := testApp(t)
	a.tools = catalog.Tools{"gamescope": true, "mangohud": true, "gamemoderun": true}
	tt := ui.NewTester(a.View, 1180, 780)
	tt.Click("ELDEN RING")
	a.saveAsPreset("RTX sharp")
	a.tab = 1
	for _, kv := range [][2]string{{"gs.out_w", "2560"}, {"gs.out_h", "1440"}, {"gs.in_w", "1920"}, {"gs.in_h", "1080"}, {"gs.refresh", "165"}, {"gs.filter", "fsr"}, {"gs.force_grab", "1"}, {"mh.position", "top-right"}, {"mh.frametime", "1"}, {"gamemode", "1"}} {
		a.setOption(kv[0], kv[1])
	}
	tt.Frame()
	// rest the pointer on a label so the screenshot shows a tooltip
	if r, ok := tt.Find("DLSS Super Resolution preset"); ok {
		tt.Move(r.X+r.W/2, r.Y+r.H/2)
		tt.Advance(time.Second)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
}
