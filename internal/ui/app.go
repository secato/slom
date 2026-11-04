// Package ui is slom's window: the only package that imports MyGo.
package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"

	"github.com/secato/slom/internal/catalog"
	"github.com/secato/slom/internal/parse"
	"github.com/secato/slom/internal/render"
	"github.com/secato/slom/internal/steam"
	"github.com/secato/slom/internal/store"
)

const backupsKept = 10

// App is the state of the window.
type App struct {
	win *mygo.Window
	ctx *ui.Context

	cat     *catalog.Catalog
	tools   catalog.Tools
	store   *store.Store
	backups string

	install steam.Install
	hasInst bool
	users   []steam.User
	userIdx int
	games   []steam.Game
	// steamLines holds the launch options currently in localconfig.vdf, by app id.
	steamLines   map[string]string
	steamRunning bool

	// list
	query       string
	visible     []int // indices into games
	sidebarSize float32

	// the game being edited
	cur       *steam.Game
	sel       catalog.Selection
	presetID  string
	notes     []string
	tab       int                 // the option group shown
	chips     map[string][]string // chip editor state, by game and field
	chipDraft map[string]string   // the text typed into each "Add…" field
	customRes map[string]bool     // resolution cards showing their width and height fields

	// dialogs
	rawOpen        bool
	raw            string
	savingPreset   bool
	presetName     string
	deletingPreset bool
	closingSteam   bool
	applyIDs       []string    // the games the open "Force apply?" dialog is about
	steamCheck     func() bool // asked just before writing; nil asks the system (tests set it)
	busy           string      // what a background task is doing, "" if idle
	loadErr        string
}

// New builds the app state: it finds Steam, reads the games and opens the store. Problems
// are kept in the state and shown in the window rather than failing the start.
func New(st *store.Store, backups string) *App {
	a := &App{
		cat: catalog.Default(), store: st, backups: backups,
		steamLines: map[string]string{}, sidebarSize: 300,
		chips: map[string][]string{}, chipDraft: map[string]string{}, customRes: map[string]bool{},
	}
	a.tools = catalog.DetectTools(a.cat)
	return a
}

// Start is called once the window exists.
func (a *App) Start(win *mygo.Window) {
	a.win = win
	a.loadSteam()
	go a.pollSteam()
}

func (a *App) loadSteam() {
	home, err := homeDir()
	if err != nil {
		a.loadErr = err.Error()
		return
	}
	installs := steam.FindInstalls(home)
	if len(installs) == 0 {
		a.loadErr = "Steam was not found. slom looks in ~/.local/share/Steam and the Flatpak location."
		return
	}
	a.install, a.hasInst = installs[0], true
	a.reloadUsers()
}

func (a *App) reloadUsers() {
	users, err := a.install.Users()
	if err != nil || len(users) == 0 {
		a.loadErr = "No Steam account with a local configuration was found."
		return
	}
	a.users = users
	a.reloadGames()
}

func (a *App) reloadGames() {
	games, err := a.install.Games()
	if err != nil {
		a.loadErr = err.Error()
		return
	}
	a.games = games
	a.readSteamLines()
	a.refilter()
	a.cur = nil
}

func (a *App) user() steam.User { return a.users[a.userIdx] }

func (a *App) readSteamLines() {
	a.steamLines = map[string]string{}
	if len(a.users) == 0 {
		return
	}
	src, err := readFile(a.user().ConfigPath)
	if err != nil {
		a.loadErr = err.Error()
		return
	}
	if lines, err := steam.LaunchOptions(src); err == nil {
		a.steamLines = lines
	}
}

func (a *App) refilter() {
	q := strings.ToLower(strings.TrimSpace(a.query))
	a.visible = a.visible[:0]
	for i, g := range a.games {
		if q == "" || strings.Contains(strings.ToLower(g.Name), q) || g.AppID == q {
			a.visible = append(a.visible, i)
		}
	}
}

// pollSteam watches whether Steam runs; the apply buttons depend on it.
func (a *App) pollSteam() {
	for {
		running := steam.Running()
		mygo.RunOnMain(func() {
			if running != a.steamRunning {
				a.steamRunning = running
				a.win.Invalidate()
			}
		})
		time.Sleep(2 * time.Second)
	}
}

// ---- the game being edited ----

func (a *App) preset(id string) *store.Preset {
	if id == "" {
		return nil
	}
	if p, ok := a.store.Preset(id); ok {
		return &p
	}
	return nil
}

// selectGame loads a game into the editor: its stored assignment, or else what Steam has.
func (a *App) selectGame(g *steam.Game) {
	a.cur, a.notes = g, nil
	if as, ok := a.store.Assignment(g.AppID); ok {
		a.presetID = as.PresetID
		if a.preset(a.presetID) == nil {
			a.presetID = ""
		}
		a.sel = store.Effective(a.cat, a.preset(a.presetID), as)
		return
	}
	a.presetID = ""
	a.sel = parse.Parse(a.cat, a.steamLines[g.AppID])
}

// discardChanges drops what slom has for the game, including its preset link, and goes back
// to what Steam has now.
func (a *App) discardChanges() {
	if a.cur == nil {
		return
	}
	if err := a.store.ClearAssignment(a.cur.AppID); err != nil {
		a.toast("Could not discard: " + err.Error())
		return
	}
	g := a.cur
	a.selectGame(g)
	a.notify("Changes discarded: back to what Steam has.")
}

// commit saves the editor's state as the game's assignment.
func (a *App) commit() {
	if a.cur == nil {
		return
	}
	as := store.Derive(a.cat, a.preset(a.presetID), a.sel)
	if err := a.store.SetAssignment(a.cur.AppID, as); err != nil {
		a.toast("Could not save: " + err.Error())
	}
}

// notify tells the user what slom did on its own, such as turning a wrapper on.
func (a *App) notify(notes ...string) {
	a.notes = notes
	for _, n := range notes {
		a.toast(n)
	}
}

func (a *App) toast(msg string) {
	if a.ctx != nil {
		a.ctx.Toast(msg)
	}
}

// setOption changes one option and reports automatic changes.
func (a *App) setOption(id, v string) {
	notes, err := a.cat.Set(&a.sel, id, v)
	if err != nil {
		a.toast(err.Error())
		return
	}
	a.notify(notes...)
	a.commit()
}

// ---- command line ----

// line is the launch-options string for the editor's state; "" when it would be a bare
// %command%, which Steam stores as no launch options.
func (a *App) line() string { return a.lineOf(a.sel) }

func (a *App) lineOf(sel catalog.Selection) string {
	s := render.Render(a.cat, sel, a.tools)
	if s == render.Command {
		return ""
	}
	return s
}

// preview is the line as shown to the user, always with %command%.
func (a *App) preview() string { return render.Render(a.cat, a.sel, a.tools) }

// steamNormalized is the line Steam has, read and written back the way slom writes it, so
// that a line which only differs in order or spacing is not a change.
func (a *App) steamNormalized(appID string) string {
	return a.lineOf(parse.Parse(a.cat, a.steamLines[appID]))
}

// differs reports whether the editor's line is not what Steam has.
func (a *App) differs() bool { return a.line() != a.steamNormalized(a.cur.AppID) }

// pending reports whether a game's configured line differs from what Steam has.
func (a *App) pending(appID string) bool {
	as, ok := a.store.Assignment(appID)
	if !ok {
		return false
	}
	p := a.preset(as.PresetID)
	return a.lineOf(store.Effective(a.cat, p, as)) != a.steamNormalized(appID)
}

func (a *App) pendingCount() int {
	n := 0
	for _, g := range a.games {
		if a.pending(g.AppID) {
			n++
		}
	}
	return n
}

// ---- presets ----

func (a *App) choosePreset(id string) {
	a.presetID = id
	if id == "" {
		a.notify("Preset removed; the game keeps its current settings.")
		a.commit()
		return
	}
	p := a.preset(id)
	if p == nil {
		return
	}
	custom := a.sel.Clone()
	a.sel = store.Effective(a.cat, p, store.Assignment{PresetID: id, Custom: catalog.Selection{
		CustomEnv: custom.CustomEnv, CustomWrap: custom.CustomWrap, CustomArgs: custom.CustomArgs, ExtraArgs: custom.ExtraArgs,
	}})
	a.notify("Preset applied; this game's own overrides were cleared.")
	a.commit()
}

func (a *App) saveAsPreset(name string) {
	p, err := a.store.SavePreset(store.Preset{Name: name, Selection: a.sel})
	if err != nil {
		a.toast(err.Error())
		return
	}
	a.presetID = p.ID
	a.commit()
	a.toast(fmt.Sprintf("Preset “%s” saved", p.Name))
}

func (a *App) updatePreset() {
	p := a.preset(a.presetID)
	if p == nil {
		return
	}
	p.Selection = a.sel
	if _, err := a.store.SavePreset(*p); err != nil {
		a.toast(err.Error())
		return
	}
	a.commit()
	n := len(a.store.GamesUsing(p.ID))
	a.toast(fmt.Sprintf("Preset “%s” updated for %d game(s)", p.Name, n))
}

func (a *App) deletePreset() {
	if a.presetID == "" {
		return
	}
	name := a.presetName0()
	if err := a.store.DeletePreset(a.cat, a.presetID); err != nil {
		a.toast(err.Error())
		return
	}
	a.presetID = ""
	a.commit()
	a.toast(fmt.Sprintf("Preset “%s” deleted; its games keep their settings", name))
}

func (a *App) presetName0() string {
	if p := a.preset(a.presetID); p != nil {
		return p.Name
	}
	return ""
}

// ---- raw editing ----

func (a *App) openRaw() {
	a.raw = a.preview()
	a.rawOpen = true
}

func (a *App) applyRaw() {
	a.sel = parse.Parse(a.cat, a.raw)
	a.notify("Imported from the text you entered.")
	a.commit()
}

// ---- applying to Steam ----

func (a *App) copyLine() {
	if err := mygo.Clipboard.Write(transfer.TextData(a.preview())); err != nil {
		a.toast("Could not copy: " + err.Error())
		return
	}
	a.toast("Launch options copied")
}

// pendingGames are the games whose configured line differs from Steam's, in list order.
func (a *App) pendingGames() []*steam.Game {
	var out []*steam.Game
	for i := range a.games {
		if a.pending(a.games[i].AppID) {
			out = append(out, &a.games[i])
		}
	}
	return out
}

// assignedLine is the line slom has configured for a game: what applying it writes.
func (a *App) assignedLine(appID string) (string, bool) {
	as, ok := a.store.Assignment(appID)
	if !ok {
		return "", false
	}
	return a.lineOf(store.Effective(a.cat, a.preset(as.PresetID), as)), true
}

// linesOf collects what applying the games writes, by app id.
func (a *App) linesOf(ids []string) map[string]string {
	out := map[string]string{}
	for _, id := range ids {
		if line, ok := a.assignedLine(id); ok {
			out[id] = line
		}
	}
	return out
}

// askApply applies the games now if Steam is closed; if it runs, asks first, because applying
// then means closing Steam.
func (a *App) askApply(ids []string) {
	if a.busy != "" || len(ids) == 0 {
		return
	}
	if a.steamRunning {
		a.applyIDs, a.closingSteam = ids, true
		return
	}
	a.applyGames(ids)
}

// applyGames writes the games' lines into Steam's configuration (Steam must be closed).
func (a *App) applyGames(ids []string) {
	err := steam.WriteManyLaunchOptions(a.user().ConfigPath, a.linesOf(ids), steam.WriteOptions{
		BackupDir: a.backups, Keep: backupsKept, SteamRunning: a.steamCheck,
	})
	if err != nil {
		if err == steam.ErrSteamRunning {
			a.toast("Steam is running. Close it first.")
		} else {
			a.toast("Could not apply: " + err.Error())
		}
		return
	}
	a.readSteamLines()
	a.toast(appliedMessage(len(ids)))
}

func appliedMessage(n int) string {
	if n == 1 {
		return "Applied. Start Steam to use the new launch options."
	}
	return fmt.Sprintf("Applied to %d games. Start Steam to use the new launch options.", n)
}

// forceApply quits Steam, writes the games' lines, and starts Steam again.
func (a *App) forceApply(ids []string) {
	if a.busy != "" || len(ids) == 0 {
		return
	}
	lines, path := a.linesOf(ids), a.user().ConfigPath
	a.busy = "Closing Steam…"
	go func() {
		err := a.install.Shutdown(45 * time.Second)
		if err == nil {
			err = steam.WriteManyLaunchOptions(path, lines, steam.WriteOptions{BackupDir: a.backups, Keep: backupsKept})
		}
		startErr := a.install.Start() // reopen Steam whatever happened
		a.win.Update(func() {
			a.busy = ""
			a.readSteamLines()
			switch {
			case err != nil:
				a.toast("Not applied: " + err.Error())
			case startErr != nil:
				a.toast("Applied, but Steam did not restart: " + startErr.Error())
			default:
				a.toast(strings.Replace(appliedMessage(len(ids)), "Start Steam to use the new launch options.", "Steam is starting again.", 1))
			}
		})
	}()
}

// removeAll empties the game's launch options. It is a change like any other: it waits to be
// applied, and Discard changes undoes it.
func (a *App) removeAll() {
	if a.cur == nil {
		return
	}
	a.sel = parse.Parse(a.cat, "")
	a.presetID = ""
	a.notify("All options removed. Apply to save that to Steam, or Discard changes to undo.")
	a.commit()
}

// ---- helpers for the view ----

func itoa(n int) string { return strconv.Itoa(n) }

func atof(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
