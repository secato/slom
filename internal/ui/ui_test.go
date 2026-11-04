package ui

import (
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/secato/slom/internal/catalog"
	"github.com/secato/slom/internal/parse"
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
	tt := ui.NewTester(a.View, 1180, 860)
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
	openTab(a, tt, "Gamescope")
	if !tt.HasText("GAMESCOPE") || !tt.HasText("Output resolution") || !tt.HasText("Refresh rate") {
		t.Error("gamescope options not shown after choosing its tab")
	}
	// selecting a game must not create an assignment until the user edits
	if _, ok := a.store.Assignment("570"); ok {
		t.Error("assignment created by merely selecting")
	}
}

func TestToggleOptionUpdatesPreviewAndStore(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.View, 1180, 860)
	tt.Click("Native Thing")
	// switching on an Int option takes its usual starting value and turns its wrapper on
	a.setOption("gs.refresh", a.cat.Initial(a.cat.Get("gs.refresh")))
	tt.Frame()
	if got := a.preview(); got != "gamescope -r 60 -- %command%" {
		t.Fatalf("preview %q", got)
	}
	if as, ok := a.store.Assignment("2"); !ok || as.Overrides["gs.refresh"] != "60" || as.Overrides["gamescope"] != "1" {
		t.Fatalf("assignment %+v", as)
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
	tt := ui.NewTester(a.View, 1180, 860)
	tt.Click("Dota 2")
	a.saveAsPreset("Native 1440p")
	if a.presetID != "native-1440p" {
		t.Fatalf("preset id %q", a.presetID)
	}
	tt.Frame() // the list reorders: Dota 2 is now pending
	// another game takes the preset
	if err := tt.Click("Native Thing"); err != nil {
		t.Fatal(err)
	}
	if a.cur.AppID != "2" {
		t.Fatalf("clicked game is %s", a.cur.AppID)
	}
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

func TestPresetBarDoesNotRelinkGames(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.View, 1180, 860)
	tt.Click("Dota 2")
	a.saveAsPreset("Alpha")
	if err := a.store.SetAssignment("570", store.Assignment{PresetID: "alpha", Overrides: map[string]string{"gs.refresh": "60"}}); err != nil {
		t.Fatal(err)
	}
	a.selectGame(a.cur)
	// a second preset can no longer share the name, so the select's label is unambiguous
	if _, err := a.store.SavePreset(store.Preset{Name: "alpha"}); err == nil {
		t.Fatal("duplicate preset name accepted")
	}
	for range 5 {
		tt.Frame()
	}
	as, _ := a.store.Assignment("570")
	if a.presetID != "alpha" || as.PresetID != "alpha" || as.Overrides["gs.refresh"] != "60" {
		t.Fatalf("frames relinked the game: preset %q, assignment %+v", a.presetID, as)
	}
}

func TestMissingToolDisablesOption(t *testing.T) {
	a := testApp(t)
	a.tools = catalog.Tools{"gamescope": true, "mangohud": true, "gamemoderun": false}
	if r := a.cat.Reason(a.sel, a.tools, catalog.IDGameMode); !strings.Contains(r, "not installed") {
		t.Fatalf("reason %q", r)
	}
	tt := ui.NewTester(a.View, 1180, 860)
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
	tt := ui.NewTester(a.View, 1180, 860)
	tt.Click("ELDEN RING")
	a.saveAsPreset("RTX sharp")
	a.tab = 2 // Gamescope
	a.steamRunning = true
	a.sel.CustomWrap = []string{"taskset", "-c", "0-7"}
	for _, kv := range [][2]string{{"gs.out_w", "2560"}, {"gs.out_h", "1440"}, {"gs.in_w", "1920"}, {"gs.in_h", "1080"}, {"gs.refresh", "165"}, {"gs.filter", "fsr"}, {"gs.force_grab", "1"}, {"mh.position", "top-right"}, {"mh.frametime", "1"}, {"gamemode", "1"}} {
		a.setOption(kv[0], kv[1])
	}
	tt.Frame()
	// rest the pointer on a label so the screenshot shows a tooltip
	if r, ok := tt.Find("Refresh rate"); ok {
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

func TestChipsGroupAndNeverSplitOnCommas(t *testing.T) {
	a := testApp(t)
	a.steamLines["2"] = "FOO=bar taskset -c 0-3 %command% -a 'b c' -windowed"
	tt := ui.NewTester(a.View, 1180, 860)
	tt.Click("Native Thing")
	openTab(a, tt, "More")
	if !tt.HasText("Commands") {
		t.Fatalf("extras not shown: %q", tt.Texts())
	}
	// the wrapper tokens are one chip, and so is each flag with its value
	if got := parse.Group(a.sel.CustomWrap); len(got) != 1 || got[0] != "taskset -c 0-3" {
		t.Fatalf("wrap chips %q", got)
	}
	if got := parse.Group(a.sel.CustomArgs); len(got) != 2 || got[0] != "-a 'b c'" || got[1] != "-windowed" {
		t.Fatalf("args chips %q", got)
	}
	// adding an env chip with commas keeps it as one token
	if err := tt.Click("Add to Env variables"); err != nil {
		t.Fatal(err)
	}
	tt.Type("WINEDLLOVERRIDES=dxgi=n,b")
	tt.Key(0, ui.KeyEnter)
	want := []string{"FOO=bar", "WINEDLLOVERRIDES=dxgi=n,b"}
	if !slices.Equal(a.sel.CustomEnv, want) {
		t.Fatalf("env tokens %q, want %q", a.sel.CustomEnv, want)
	}
	if got := a.preview(); !strings.HasPrefix(got, "FOO=bar WINEDLLOVERRIDES=dxgi=n,b taskset -c 0-3 %command% -a 'b c' -windowed") {
		t.Fatalf("preview %q", got)
	}
	// and it is saved
	as, _ := a.store.Assignment("2")
	if !slices.Equal(as.Custom.CustomEnv, want) {
		t.Fatalf("not saved: %+v", as.Custom)
	}
}

func TestSwitchingSteamAccount(t *testing.T) {
	root := t.TempDir()
	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := func(opt string) string {
		return `"UserLocalConfigStore" { "Software" { "Valve" { "Steam" { "apps" { "1" { "LaunchOptions" "` + opt + `" } } } } } }`
	}
	write(filepath.Join(root, "steamapps/appmanifest_1.acf"), `"AppState" { "appid" "1" "name" "Shared Game" "StateFlags" "4" "installdir" "x" }`)
	write(filepath.Join(root, "userdata/100/config/localconfig.vdf"), cfg("A=1 %command%"))
	write(filepath.Join(root, "userdata/200/config/localconfig.vdf"), cfg("B=2 %command%"))
	write(filepath.Join(root, "config/loginusers.vdf"),
		`"users" { "76561197960265828" { "PersonaName" "Alice" "Timestamp" "20" } "76561197960265928" { "PersonaName" "Bob" "Timestamp" "10" } }`)

	a := testApp(t)
	a.install, a.hasInst = steam.Install{Root: root}, true
	a.reloadUsers()
	if len(a.users) != 2 || a.users[0].Name != "Alice" {
		t.Fatalf("users %+v", a.users)
	}
	if a.steamLines["1"] != "A=1 %command%" {
		t.Fatalf("alice lines %v", a.steamLines)
	}
	tt := ui.NewTester(a.View, 1180, 860)
	if err := tt.Click("Alice"); err != nil { // the account select
		t.Fatal(err)
	}
	if err := tt.Click("Bob"); err != nil {
		t.Fatal(err)
	}
	if a.userIdx != 1 || a.steamLines["1"] != "B=2 %command%" {
		t.Fatalf("after switching: idx %d lines %v", a.userIdx, a.steamLines)
	}
	if a.cur != nil {
		t.Error("the open game should be closed when the account changes")
	}
}

// twoPendingGames gives Dota 2 and ELDEN RING a change each, over a real localconfig.vdf copy.
func twoPendingGames(t *testing.T) (*App, *ui.Tester) {
	t.Helper()
	a := testApp(t)
	src, err := os.ReadFile("../steam/testdata/localconfig.vdf")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(a.users[0].ConfigPath, src, 0o644); err != nil {
		t.Fatal(err)
	}
	a.steamCheck = func() bool { return false }
	tt := ui.NewTester(a.View, 1180, 860)
	tt.Click("Dota 2")
	a.setOption("gs.refresh", "60")
	tt.Frame() // the list reorders
	tt.Click("ELDEN RING")
	a.setOption("gs.refresh", "90")
	a.setOption("gamescope", "1")
	tt.Frame()
	return a, tt
}

func TestApplyLivesInTheSidebar(t *testing.T) {
	a, tt := twoPendingGames(t)
	for _, s := range tt.Texts() {
		if s == "Apply" || s == "Force apply" {
			t.Fatalf("the dock still has its own %s button", s)
		}
	}
	if !tt.HasText("Apply all · 2") {
		t.Fatalf("no apply-all under the pending games: %q", tt.Texts())
	}
	// one game
	if err := tt.Click("Apply Dota 2"); err != nil {
		t.Fatal(err)
	}
	if a.pending("570") || !a.pending("1245620") {
		t.Fatalf("applying one game: pending dota=%v elden=%v", a.pending("570"), a.pending("1245620"))
	}
	if got := a.steamLines["570"]; got != "gamescope -r 60 -- %command%" && !strings.Contains(got, "-r 60") {
		t.Errorf("Steam's line for Dota 2: %q", got)
	}
}

func TestApplyAllWritesEveryPendingGameWithOneBackup(t *testing.T) {
	a, tt := twoPendingGames(t)
	if err := tt.Click("Apply all · 2"); err != nil {
		t.Fatal(err)
	}
	if n := len(a.pendingGames()); n != 0 {
		t.Fatalf("%d games still pending", n)
	}
	baks, _ := filepath.Glob(filepath.Join(a.backups, "*.bak"))
	if len(baks) != 1 {
		t.Errorf("%d backups, want 1 for the whole batch", len(baks))
	}
	tt.Frame()
	if tt.HasText("Apply all · 2") || tt.HasText("PENDING APPLY · 2") {
		t.Error("the pending section did not empty")
	}
}

func TestForceApplyAsksBeforeClosingSteam(t *testing.T) {
	a, tt := twoPendingGames(t)
	a.steamRunning = true
	tt.Frame()
	if !tt.HasText("Force apply all · 2") {
		t.Fatalf("no force apply all: %q", tt.Texts())
	}
	if err := tt.Click("Force apply Dota 2"); err != nil {
		t.Fatal(err)
	}
	if !a.closingSteam || len(a.applyIDs) != 1 {
		t.Fatalf("no confirmation for one game: %v %v", a.closingSteam, a.applyIDs)
	}
	tt.Frame()
	if !tt.HasText("Steam will be closed, the launch options saved, and Steam opened again. Anything running in Steam, including games, will be closed.") {
		t.Fatalf("confirmation does not say what happens: %q", tt.Texts())
	}
	a.closingSteam = false
	tt.Frame()
	tt.Click("Force apply all · 2")
	if !a.closingSteam || len(a.applyIDs) != 2 {
		t.Fatalf("no confirmation for all: %v %v", a.closingSteam, a.applyIDs)
	}
	tt.Frame()
	if !tt.HasText("Steam will be closed, the launch options of 2 games saved, and Steam opened again. Anything running in Steam, including games, will be closed.") {
		t.Fatalf("confirmation for two games: %q", tt.Texts())
	}
	if a.steamLines["570"] != "gamescope --force-grab-cursor -W 2560 -H 1440 -r 180 -- %command%" {
		t.Error("a force apply that was not confirmed changed Steam's data")
	}
}

func TestRemoveAllOptionsIsAnIntentionalChange(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.View, 1180, 860)
	tt.Click("Dota 2") // Steam has a gamescope line
	if err := tt.Click("Remove all options"); err != nil {
		t.Fatal(err)
	}
	if a.line() != "" || !a.pending("570") {
		t.Fatalf("not an empty pending change: line %q pending %v", a.line(), a.pending("570"))
	}
	tt.Frame()
	if !tt.HasText("Apply Dota 2") && !tt.HasText("Force apply Dota 2") {
		t.Error("the game is not offered for apply")
	}
	tt.Move(5, 5)
	tt.Advance(10 * time.Second) // the toast and the tooltip of the button just pressed cover the buttons until they are gone
	if err := tt.Click("Discard changes"); err != nil {
		t.Fatal(err)
	}
	if a.line() == "" || a.pending("570") {
		t.Error("discard did not bring the options back")
	}
}

func TestPendingGamesAreGroupedInTheSidebar(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.View, 1180, 860)
	if tt.HasText("PENDING APPLY · 1") {
		t.Fatal("nothing is configured yet")
	}
	tt.Click("Dota 2")
	a.saveAsPreset("Native 1440p")
	a.setOption("gs.refresh", "144") // now differs from the line Steam has
	tt.Frame()
	for _, want := range []string{"PENDING APPLY · 1", "OTHER GAMES · 2", "Nothing set"} {
		if !tt.HasText(want) {
			t.Errorf("sidebar lacks %q: %q", want, tt.Texts())
		}
	}
	for _, bad := range []string{"out of date", "game(s)"} {
		for _, s := range tt.Texts() {
			if strings.Contains(s, bad) {
				t.Errorf("old wording %q is back: %q", bad, s)
			}
		}
	}
}

// Steam's line may only differ from slom's in order or spacing; that is not a change, so a
// game must not stay pending after it was read from Steam.
func TestSameLineInAnotherOrderIsNotPending(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.View, 1180, 860)
	tt.Click("Dota 2") // Steam has "--force-grab-cursor -W 2560 ..." and slom renders another order
	a.commit()
	if a.pending("570") {
		t.Fatalf("pending although nothing changed: slom %q, Steam %q", a.line(), a.steamLines["570"])
	}
	if a.differs() {
		t.Error("the editor says it differs from Steam")
	}
}

func TestDiscardChangesGoesBackToSteam(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.View, 1180, 860)
	tt.Click("Dota 2")
	a.saveAsPreset("Mine")
	a.setOption("gs.refresh", "60")
	tt.Frame()
	if !a.pending("570") || !tt.HasText("not saved yet") {
		t.Fatal("an edit should be pending")
	}
	if err := tt.Click("Discard changes"); err != nil {
		t.Fatal(err)
	}
	if a.pending("570") || a.presetID != "" {
		t.Errorf("still pending or linked: pending=%v preset=%q", a.pending("570"), a.presetID)
	}
	if _, ok := a.store.Assignment("570"); ok {
		t.Error("the assignment survived the discard")
	}
	if got := a.preview(); got != "gamescope -W 2560 -H 1440 -r 180 --force-grab-cursor -- %command%" {
		t.Errorf("not back to Steam's line: %q", got)
	}
	tt.Frame()
	if !tt.HasText("matches Steam") {
		t.Error("the dock still says there are unsaved changes")
	}
}

func TestResolutionPillsSetWidthAndHeightTogether(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.View, 1180, 860)
	tt.Click("Native Thing")
	openTab(a, tt, "Gamescope")
	if err := tt.Click("1920×1080"); err != nil {
		t.Fatal(err)
	}
	if a.sel.Values["gs.out_w"] != "1920" || a.sel.Values["gs.out_h"] != "1080" {
		t.Fatalf("pill did not set both: %v", a.sel.Values)
	}
	if _, on := a.sel.Values["gamescope"]; !on {
		t.Error("choosing a size should switch gamescope on")
	}
}

func TestSlidersSnapToUsualValuesAndKeepCustomOnes(t *testing.T) {
	a := testApp(t)
	o := a.cat.Get("gs.refresh")
	got := a.stopsOf(o)
	want := []int{30, 60, 90, 120, 144, 180, 240}
	if !slices.Equal(got, want) {
		t.Fatalf("stops %v, want %v", got, want)
	}
	if i := nearest(got, 143); got[i] != 144 {
		t.Errorf("143 Hz snaps to %d", got[i])
	}
	// a value of your own is kept as typed, not rounded to a stop
	a.cur = &a.games[2]
	a.setOption("gs.refresh", "100")
	if a.sel.Values["gs.refresh"] != "100" {
		t.Errorf("custom value lost: %q", a.sel.Values["gs.refresh"])
	}
}

func TestNoScreenIsAssumed(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.View, 1180, 860)
	tt.Click("Native Thing")
	openTab(a, tt, "Gamescope")
	for _, s := range tt.Texts() {
		if strings.Contains(s, "Your screen") {
			t.Errorf("the window guesses the screen: %q", s)
		}
	}
	if got := a.cat.Initial(a.cat.Get("gs.out_w")) + "x" + a.cat.Initial(a.cat.Get("gs.out_h")); got != "1920x1080" {
		t.Errorf("sizes start at %s, not a standard value", got)
	}
}

func TestDockShowsPreviewSteamAndDifference(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.View, 1180, 860)
	tt.Click("Dota 2")
	tt.Frame()
	for _, want := range []string{"PREVIEW", "DIFFERENCE", "no difference", "matches Steam"} {
		if !tt.HasText(want) {
			t.Errorf("dock lacks %q", want)
		}
	}
	a.setOption("gs.refresh", "90")
	a.setOption("gs.borderless", "1")
	tt.Frame()
	added, removed := parse.Diff(a.steamLines["570"], a.line())
	if !slices.Equal(added, []string{"-r 90", "-b"}) || !slices.Equal(removed, []string{"-r 180"}) {
		t.Fatalf("added %q removed %q", added, removed)
	}
	for _, want := range []string{"-r 90", "-b", "-r 180", "not saved yet"} {
		if !tt.HasText(want) {
			t.Errorf("the difference does not show %q: %q", want, tt.Texts())
		}
	}
	for _, gone := range []string{"Nothing to apply: this matches Steam.", "live preview", "CURRENT ON STEAM", "Steam currently has:"} {
		if tt.HasText(gone) {
			t.Errorf("useless wording is back: %q", gone)
		}
	}
}

func TestSidebarListsChangedCategoriesNotHowTheGameRuns(t *testing.T) {
	a := testApp(t)
	a.steamLines["2"] = "FOO=bar %command%"
	tt := ui.NewTester(a.View, 1180, 860)
	tt.Frame()
	want := map[string]string{
		"570":     "Gamescope",
		"1245620": "DLSS, Gamescope",
		"2":       "More",
	}
	// ELDEN RING's line has DLSS options only in the fixture; Dota 2's has gamescope only
	want["1245620"] = "DLSS"
	for id, w := range want {
		var g *steam.Game
		for i := range a.games {
			if a.games[i].AppID == id {
				g = &a.games[i]
			}
		}
		if got := a.gameSummary(g); got != w {
			t.Errorf("%s: %q, want %q", g.Name, got, w)
		}
	}
	for _, s := range tt.Texts() {
		if strings.Contains(s, "Proton ·") || strings.Contains(s, "Native ·") {
			t.Errorf("the sidebar still says how the game runs: %q", s)
		}
	}
	a.setOption("mh.frametime", "1") // cur is nil: nothing happens, no panic
}

func TestMoreTabShowsWhereTheFieldsLand(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.View, 1180, 860)
	tt.Click("Native Thing")
	openTab(a, tt, "More")
	for _, want := range []string{"WHERE THEY GO", "ENV VARIABLES", "COMMANDS", "ARGUMENTS", "MY_VAR=1", "taskset -c 0-3", "-windowed"} {
		if !tt.HasText(want) {
			t.Errorf("example lacks %q", want)
		}
	}
}

// openTab shows a tab by name. Clicking its text is ambiguous: the sidebar lists the same
// category names under the games.
func openTab(a *App, tt *ui.Tester, name string) {
	for i, g := range append(a.cat.Groups(), moreTab) {
		if g == name {
			a.tab = i
		}
	}
	tt.Frame()
}
