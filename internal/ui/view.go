package ui

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/secato/slom/internal/catalog"
	"github.com/secato/slom/internal/parse"
	"github.com/secato/slom/internal/render"
	"github.com/secato/slom/internal/steam"
	"github.com/secato/slom/internal/store"
)

// View builds the whole window.
func (a *App) View(c *ui.Context) {
	setup()
	c.SetTheme(gamerTheme())
	a.ctx = c
	t := c.Theme()
	ui.Column(c).Fill().Children(func() {
		if a.loadErr != "" && len(a.games) == 0 {
			ui.Column(c).Fill().Center().Gap(8).Children(func() {
				ui.Text(c, "slom can't find your Steam games").Bold().FontSize(18)
				ui.Text(c, a.loadErr).TextColor(t.TextMuted)
			})
			return
		}
		ui.Split(c, &a.sidebarSize, func() { a.sidebar(c) }, func() { a.editor(c) }).Grow(1)
		a.dialogs(c)
	})
}

// ---- small parts of the look ----

// display is the title font: condensed, spaced out.
func display(e ui.Element, size float32) ui.Element {
	return e.Font(fontDisplay + ", system-ui").Bold().FontSize(size).LetterSpacing(size / 14)
}

// mono is the command font.
func mono(e ui.Element, size float32) ui.Element {
	return e.Font(fontMono + ", monospace").FontSize(size)
}

// pill is a small rounded label, such as "live preview".
func pill(c *ui.Context, text string, bg, fg ui.Color) {
	ui.Box(c).Padding(1, 8).Radius(99).Background(bg).Children(func() {
		mono(ui.Text(c, text), 11).TextColor(fg)
	})
}

// dot is a small round status light, with a glow when lit.
func dot(c *ui.Context, color ui.Color, glow bool) {
	b := ui.Box(c).Width(8).Height(8).Radius(4).Background(color)
	if glow {
		b.Shadow(0, 0, 8, 0, color)
	}
}

// ghost is a quiet button, for what is not the main action.
func ghost(c *ui.Context, label, tip string, disabled bool) bool {
	b := ui.ButtonBase(c.Key("ghost-"+label)).Height(38).Padding(0, 12).Radius(9).Gap(8).Tooltip(tip).Disabled(disabled)
	if b.Hovered() && !disabled {
		b.Background(c.Theme().SurfaceHover)
	}
	b.Children(func() {
		if label == "Copy" {
			icon(c, icons.copy, 14, pal.muted)
		}
		ui.Text(c, label).TextColor(ternary(disabled, pal.faint, pal.text))
	})
	return b.Clicked()
}

// applyColors are the colors of the apply buttons: green when applying is direct, amber when
// Steam runs and applying means closing and reopening it.
func (a *App) applyColors() (bg, glow ui.Color) {
	if a.steamRunning {
		return pal.amber, pal.amberGlow
	}
	return pal.accent, pal.accentGlow
}

// applyTip says what pressing an apply button does right now.
func (a *App) applyTip(what string) string {
	if a.steamRunning {
		return "Force apply " + what + ": close Steam, write the launch options, open Steam again. Anything open in Steam, including games, will be closed. You are asked to confirm first."
	}
	return "Apply " + what + " to Steam. A backup of its configuration is made first."
}

func ternary[T any](ok bool, a, b T) T {
	if ok {
		return a
	}
	return b
}

// ---- left: the game list ----

func (a *App) sidebar(c *ui.Context) {
	ui.Column(c).Fill().Background(pal.panel).Padding(18, 14).Gap(14).Children(func() {
		ui.Row(c).Gap(10).Padding(0, 4).Children(func() {
			ui.Box(c).Width(30).Height(30).Radius(8).Background(pal.accent).Center().Shadow(0, 0, 18, 0, pal.accentGlow).Children(func() {
				icon(c, icons.bolt, 18, pal.onAccent)
			})
			display(ui.Text(c, "SLOM"), 26)
		})
		if ui.SearchField(c.Key("search"), &a.query).Placeholder("Search games").Label("Search games").Changed() {
			a.refilter()
		}
		if len(a.users) > 1 {
			names := make([]string, len(a.users))
			for i, u := range a.users {
				names[i] = u.Name
				if names[i] == "" {
					names[i] = "Account " + u.AccountID
				}
			}
			cur := names[a.userIdx]
			if ui.Select(c.Key("user"), &cur, names).Label("Steam account").Changed() {
				for i, n := range names {
					if n == cur {
						a.userIdx = i
					}
				}
				a.reloadGames()
			}
		}

		var pend, rest []*steam.Game
		for _, gi := range a.visible {
			g := &a.games[gi]
			if a.pending(g.AppID) {
				pend = append(pend, g)
			} else {
				rest = append(rest, g)
			}
		}
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Gap(4).Children(func() {
				if len(pend) > 0 {
					a.section(c, fmt.Sprintf("PENDING APPLY · %d", len(pend)), pal.amber,
						"Configured in slom, but different from what Steam has. Select one and apply it.")
					for _, g := range pend {
						a.gameRow(c, g, true)
					}
					if len(pend) > 1 {
						a.applyAll(c, pend)
					}
				}
				if len(rest) > 0 {
					title := fmt.Sprintf("ALL GAMES · %d", len(rest))
					if len(pend) > 0 {
						title = fmt.Sprintf("OTHER GAMES · %d", len(rest))
					}
					a.section(c, title, pal.faint, "")
					for _, g := range rest {
						a.gameRow(c, g, false)
					}
				}
				if len(pend)+len(rest) == 0 {
					ui.Text(c, "No installed games match.").TextColor(pal.muted)
				}
			})
		})
		if a.busy != "" {
			ui.Text(c, a.busy).TextColor(pal.muted).FontSize(12.5)
		}
		a.steamStatus(c)
	})
}

func (a *App) section(c *ui.Context, title string, color ui.Color, tip string) {
	ui.Row(c.Key("sec-"+title)).Gap(8).Padding(10, 6, 2).Tooltip(tip).Children(func() {
		dot(c, color, false)
		display(ui.Text(c, title), 12).TextColor(pal.muted)
	})
}

// gameRow is one game of the list: a tile of its own color, its name, and what it has set.
// A pending game also has a button to apply just that game.
func (a *App) gameRow(c *ui.Context, g *steam.Game, withApply bool) {
	on := a.cur != nil && a.cur.AppID == g.AppID
	outer := ui.Row(c.Key("row-" + g.AppID)).Height(56).Radius(12).Gap(4)
	if on {
		outer.Background(pal.cardOn).Border(1, pal.accentDim).Shadow(0, 0, 22, 0, ui.RGBA(198, 255, 61, 0.12))
	}
	outer.Children(func() {
		row := ui.ButtonBase(c.Key("pick-"+g.AppID)).Grow(1).Height(54).Padding(0, 10).Gap(12).Radius(12).Label(g.Name)
		if row.Hovered() && !on {
			row.Background(c.Theme().SurfaceHover)
		}
		row.Children(func() {
			from, to := tileColors(g.Name)
			ui.Box(c).Width(36).Height(36).Radius(9).Gradient(from, to, 135).Center().Children(func() {
				display(ui.Text(c, initials(g.Name)), 15).TextColor(ui.Hex("#FFFFFF"))
			})
			ui.Column(c).Grow(1).Children(func() {
				ui.Text(c, g.Name).Bold().SingleLine()
				ui.Text(c, a.gameSummary(g)).FontSize(12).TextColor(pal.muted).SingleLine()
			})
			if !withApply && a.steamLines[g.AppID] != "" {
				dot(c, pal.cyan, false)
			}
		})
		if row.Clicked() {
			a.selectGame(g)
		}
		if withApply {
			bg, glow := a.applyColors()
			verb := ternary(a.steamRunning, "Force apply ", "Apply ")
			btn := ui.ButtonBase(c.Key("apply-"+g.AppID)).Width(32).Height(32).Radius(8).Margin(0, 10, 0, 0).Center().
				Background(ternary(a.busy != "", pal.line, bg)).Label(verb + g.Name).Tooltip(a.applyTip(g.Name)).Disabled(a.busy != "")
			if a.busy == "" {
				btn.Shadow(0, 0, 12, 0, glow)
			}
			btn.Children(func() { icon(c, icons.bolt, 16, ternary(a.busy != "", pal.faint, pal.onAccent)) })
			if btn.Clicked() {
				a.askApply([]string{g.AppID})
			}
		}
	})
}

// applyAll is the button under the pending games that applies all of them at once.
func (a *App) applyAll(c *ui.Context, pend []*steam.Game) {
	bg, glow := a.applyColors()
	ids := make([]string, len(pend))
	for i, g := range pend {
		ids[i] = g.AppID
	}
	label := fmt.Sprintf("%s all · %d", ternary(a.steamRunning, "Force apply", "Apply"), len(pend))
	btn := ui.ButtonBase(c.Key("apply-all")).Height(38).Radius(9).Gap(8).Margin(4, 0, 0, 0).Label(label).
		Background(ternary(a.busy != "", pal.line, bg)).Tooltip(a.applyTip(fmt.Sprintf("all %d pending games", len(pend)))).Disabled(a.busy != "")
	if a.busy == "" {
		btn.Shadow(0, 0, 14, 0, glow)
	}
	btn.Children(func() {
		fg := ternary(a.busy != "", pal.faint, pal.onAccent)
		icon(c, icons.bolt, 15, fg)
		display(ui.Text(c, label), 15).TextColor(fg)
	})
	if btn.Clicked() {
		a.askApply(ids)
	}
}

// gameSummary lists the groups the game sets something in: "DLSS, Gamescope". Where the game
// runs (native or Proton) is not shown: the Proton prefix says only that the game ran through
// Proton once, not how it runs now.
func (a *App) gameSummary(g *steam.Game) string {
	sel := a.selectionOf(g.AppID)
	used := map[string]bool{}
	for id := range sel.Values {
		if o := a.cat.Get(id); o != nil {
			used[o.Group] = true
		}
	}
	var names []string
	for _, grp := range a.cat.Groups() {
		if used[grp] {
			names = append(names, grp)
		}
	}
	if len(sel.CustomEnv)+len(sel.CustomWrap)+len(sel.CustomArgs)+len(sel.ExtraArgs) > 0 {
		names = append(names, moreTab)
	}
	if len(names) == 0 {
		return "Nothing set"
	}
	return strings.Join(names, ", ")
}

// selectionOf is what the game's line is: slom's configuration if it has one, else Steam's line.
func (a *App) selectionOf(appID string) catalog.Selection {
	if as, ok := a.store.Assignment(appID); ok {
		return store.Effective(a.cat, a.preset(as.PresetID), as)
	}
	return parse.Parse(a.cat, a.steamLines[appID])
}

func initials(name string) string {
	var out []rune
	for _, w := range strings.Fields(name) {
		out = append(out, []rune(strings.ToUpper(w))[0])
		if len(out) == 2 {
			break
		}
	}
	return string(out)
}

// steamStatus says whether Apply writes right away or has to wait for Steam.
func (a *App) steamStatus(c *ui.Context) {
	bg, border, fg, name, note := pal.cyanDim, pal.cyanBorder, pal.cyan, "Steam is closed", "Apply saves right away"
	if a.steamRunning {
		bg, border, fg, name, note = pal.amberDim, pal.amberBorder, pal.amber, "Steam is running", "Changes wait for Apply"
	}
	ui.Row(c).Gap(10).Padding(10, 12).Radius(10).Background(bg).Border(1, border).Children(func() {
		dot(c, fg, a.steamRunning)
		ui.Column(c).Grow(1).Children(func() {
			ui.Text(c, name).Bold().FontSize(12.5).TextColor(fg)
			ui.Text(c, note).FontSize(12.5).TextColor(pal.muted)
		})
	})
}

// ---- right: the editor ----

func (a *App) editor(c *ui.Context) {
	if a.cur == nil {
		ui.Column(c).Fill().Center().Gap(6).Children(func() {
			display(ui.Text(c, "PICK A GAME"), 28)
			ui.Text(c, "Choose a game on the left to set its Proton, DLSS, gamescope and MangoHud options.").TextColor(pal.muted).MaxWidth(420).TextAlign(ui.Center)
		})
		return
	}
	g := a.cur
	// the key scopes every control to the game, so no editing state leaks between games
	ui.Column(c.Key("game-" + g.AppID)).Fill().Children(func() {
		a.header(c, g)
		ui.Column(c).Grow(1).Padding(16, 32, 18).Gap(12).Children(func() {
			groups := a.cat.Groups()
			a.tabs(c, groups)
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Gap(10).Padding(0, 8, 0, 0).MaxWidth(1000).Children(func() {
					if a.tab >= 0 && a.tab < len(groups) {
						a.group(c, groups[a.tab])
					} else {
						a.extras(c)
					}
				})
			})
			a.dock(c)
		})
	})
}

// header is the game's name over a green-to-dark wash, with its facts and its preset.
func (a *App) header(c *ui.Context, g *steam.Game) {
	ui.Column(c).Padding(22, 32, 18).Gap(2).Gradient(washColor(g.Name), pal.ground, 110).Children(func() {
		meta := "APP " + g.AppID
		if g.LastPlayed > 0 {
			meta += " · LAST PLAYED " + ago(time.Unix(g.LastPlayed, 0), time.Now())
		}
		ui.Row(c).Gap(16).Children(func() {
			mono(ui.Text(c, meta), 12).TextColor(pal.muted).LetterSpacing(0.6).Grow(1)
			a.presetBar(c)
		})
		display(ui.Text(c, strings.ToUpper(g.Name)), 40).SingleLine()
	})
}

// ago says how long ago, in the unit a person would use.
func ago(then, now time.Time) string {
	d := now.Sub(then)
	switch {
	case d < 24*time.Hour:
		return "TODAY"
	case d < 48*time.Hour:
		return "YESTERDAY"
	case d < 60*24*time.Hour:
		return fmt.Sprintf("%d DAYS AGO", int(d.Hours()/24))
	}
	return fmt.Sprintf("%d MONTHS AGO", int(d.Hours()/24/30))
}

func (a *App) presetBar(c *ui.Context) {
	presets := a.store.Presets()
	labels := []string{store.NoPresetLabel}
	byLabel := map[string]string{store.NoPresetLabel: ""}
	cur := store.NoPresetLabel
	for _, p := range presets {
		labels = append(labels, p.Name)
		byLabel[p.Name] = p.ID
		if p.ID == a.presetID {
			cur = p.Name
		}
	}
	ui.Row(c).Gap(8).Children(func() {
		ui.Text(c, "Preset").TextColor(pal.muted).Tooltip("A preset is a reusable set of options. A game linked to a preset follows it; the changes you make to this game are kept as its own overrides (marked •).")
		if ui.Select(c.Key("preset"), &cur, labels).Label("Preset").Width(200).Changed() {
			a.choosePreset(byLabel[cur])
		}
		ui.Button(c, "Save as preset…").Tooltip("Save the current settings as a new preset and link this game to it.").OnClick(func() {
			a.presetName, a.savingPreset = "", true
		})
		if a.presetID != "" {
			n := len(a.store.GamesUsing(a.presetID))
			ui.Button(c, "Update preset").Tooltip(fmt.Sprintf("Write this game's settings into the preset. It is used by %d game(s), which will then be out of date until applied.", n)).OnClick(a.updatePreset)
			ui.Button(c, "Delete preset").OnClick(func() { a.deletingPreset = true })
		}
	})
}

// tabs is one tab per option group, with a count of what the game sets in it, and "More".
func (a *App) tabs(c *ui.Context, groups []string) {
	used := map[string]int{}
	for id := range a.sel.Values {
		if o := a.cat.Get(id); o != nil {
			used[o.Group]++
		}
	}
	names := append(slices.Clone(groups), moreTab)
	tabs := ui.TabsBase(c.Key("tabs"), &a.tab, len(names))
	tabs.List.Gap(4).Label("Option groups")
	tabs.List.Children(func() {
		for i, name := range names {
			on := i == a.tab
			tab := tabs.Tab(i).Padding(10, 14).Gap(8).Margin(0, 0, -1, 0)
			tab.BorderWidth(0, 0, 2, 0).BorderColor(ternary(on, pal.accent, ui.Transparent))
			tab.Children(func() {
				display(ui.Text(c, name), 16).TextColor(ternary(on, pal.text, pal.muted))
				if n := used[name]; n > 0 {
					ui.Box(c).Padding(0, 7).Radius(99).Background(pal.accent).Children(func() {
						mono(ui.Text(c, fmt.Sprint(n)), 11).TextColor(pal.onAccent)
					}).Tooltip(fmt.Sprintf("%d option(s) set in %s", n, name))
				}
			})
		}
	})
	ui.Divider(c)
}

const moreTab = "More"

// optionRow is one option as a card: its label with the tooltip, and the control for its kind.
func (a *App) optionRow(c *ui.Context, o *catalog.Option, overridden bool) {
	reason := a.cat.Reason(a.sel, a.tools, o.ID)
	_, on := a.sel.Values[o.ID]
	off := reason != "" && !on
	card := ui.Row(c.Key(o.ID)).Gap(12).Padding(0, 14).Height(48).Radius(12).
		Background(ternary(on, pal.cardOn, pal.card)).Border(1, ternary(on, pal.accentDim, pal.line))
	if o.Kind == catalog.Text {
		card.ColumnSpan(2)
	}
	card.Children(func() {
		tip := o.Help
		if o.Support != "" {
			tip += "\n\n" + o.Support
		}
		if reason != "" {
			tip += "\n\n" + reason
		}
		ui.Row(c).Gap(8).Grow(1).Margin(0, 0, 0, ternary[float32](o.Host != "", 14, 0)).Tooltip(tip).Children(func() {
			ui.Text(c, o.Label).TextColor(ternary(off, pal.faint, pal.text))
			if overridden {
				ui.Text(c, "•").TextColor(pal.accent).Tooltip("This game overrides its preset here.")
			}
			icon(c, icons.info, 15, pal.faint)
		})
		a.control(c, o, off, on)
	})
}

// control is the widget for the option's kind; changing it goes through setOption, which
// switches on what the option requires and tells the user.
func (a *App) control(c *ui.Context, o *catalog.Option, disabled, _ bool) {
	cur, on := a.sel.Values[o.ID]
	switch o.Kind {
	case catalog.Bool:
		v := on
		glow := ui.Box(c.Key("glow-" + o.ID)).Radius(99)
		if on {
			glow.Shadow(0, 0, 12, 0, pal.accentGlow)
		}
		glow.Children(func() {
			if ui.Switch(c.Key("c-"+o.ID), &v).Label(o.Label).Disabled(disabled).Changed() {
				a.setOption(o.ID, boolValue(v))
			}
		})
	case catalog.Select:
		labels := []string{"Not set"}
		byLabel := map[string]string{"Not set": ""}
		shown := "Not set"
		for _, ch := range o.Choices {
			labels = append(labels, ch.Label)
			byLabel[ch.Label] = ch.Value
			if on && ch.Value == cur {
				shown = ch.Label
			}
		}
		if ui.Select(c.Key("c-"+o.ID), &shown, labels).Label(o.Label).Disabled(disabled).Width(220).Changed() {
			a.setOption(o.ID, byLabel[shown])
		}
	case catalog.Int:
		ui.Row(c).Gap(8).Children(func() {
			v := on
			if ui.Switch(c.Key("s-"+o.ID), &v).Label("Set " + o.Label).Disabled(disabled).Changed() {
				if v {
					a.setOption(o.ID, a.cat.Initial(o))
				} else {
					a.setOption(o.ID, "")
				}
			}
			if on {
				n := atof(cur)
				if ui.NumberInput(c.Key("n-"+o.ID), &n, float64(o.Min), float64(o.Max), 1).Label(o.Label).Width(120).Changed() {
					a.setOption(o.ID, itoa(int(n)))
				}
				if o.Unit != "" {
					ui.Text(c, o.Unit)
				}
			}
		})
	case catalog.Text:
		s := cur
		if ui.TextInput(c.Key("c-"+o.ID), &s).Label(o.Label).Disabled(disabled).Width(220).Changed() {
			a.setOption(o.ID, s)
		}
	}
}

func boolValue(on bool) string {
	if on {
		return "1"
	}
	return ""
}

// extras shows the pieces of an imported line the catalog does not know, as chips.
func (a *App) extras(c *ui.Context) {
	ui.Column(c.Key("extras")).Gap(14).Children(func() {
		a.intro(c, moreTab, nil)
		a.placeExample(c)
		a.chipEditor(c, "env", "Before the game: variables", "Environment variables such as FOO=1, placed first in the command.",
			a.sel.CustomEnv, false, func(t []string) { a.sel.CustomEnv = t })
		a.chipEditor(c, "wrap", "Before the game: commands", "Other programs that wrap the game, such as taskset -c 0-3.",
			a.sel.CustomWrap, true, func(t []string) { a.sel.CustomWrap = t })
		a.chipEditor(c, "args", "After the game: arguments", "Arguments passed to the game itself, after %command%, such as -windowed or -novid.",
			a.sel.CustomArgs, true, func(t []string) { a.sel.CustomArgs = t })
		for _, w := range a.cat.Options {
			if w.Slot != catalog.SlotWrapper || len(a.sel.ExtraArgs[w.ID]) == 0 {
				continue
			}
			host, items := w.ID, w.ConfigEnv != ""
			a.chipEditor(c, "extra-"+host, "Extra "+w.Name+" settings", "Flags or items of "+w.Name+" that slom has no control for.",
				a.sel.ExtraArgs[host], !items, func(t []string) {
					if len(t) == 0 {
						delete(a.sel.ExtraArgs, host)
					} else {
						a.sel.ExtraArgs[host] = t
					}
				})
		}
	})
}

// ---- bottom: the command and the actions ----

// dock shows the command that will be written, the one Steam has now, and the difference,
// with the actions on the line. It is green-edged while there is something to apply.
func (a *App) dock(c *ui.Context) {
	steamLine := a.steamLines[a.cur.AppID]
	differs := a.differs()
	ui.Column(c).Radius(14).Background(pal.panel).Border(1, ternary(differs, pal.accentDim, pal.line)).
		Shadow(0, -10, 30, 0, ui.RGBA(0, 0, 0, 0.4)).Padding(12, 14, 14).Gap(10).Children(func() {
		ui.Row(c).Gap(10).Children(func() {
			display(ui.Text(c, "LAUNCH COMMAND"), 13).TextColor(pal.muted)
			if differs {
				pill(c, "not saved yet", ui.Hex("#3A2A0B"), pal.amber)
			} else {
				pill(c, "matches Steam", pal.cyanDim, pal.cyan)
			}
		})
		ui.Scroll(c.Key("dock-scroll")).MaxHeight(280).Children(func() {
			ui.Column(c).Gap(10).Padding(0, 8, 0, 0).Children(func() {
				dockRow(c, "PREVIEW", "The line slom will write into the game's Launch Options in Steam.", func() {
					a.commandBox(c, "preview", a.preview(), "Launch options preview: ")
				})
				dockRow(c, "CURRENT ON STEAM", "The launch options Steam has for this game right now.", func() {
					if steamLine == "" {
						mono(ui.Text(c, "no launch options"), 12.5).TextColor(pal.faint).Margin(10, 0, 0, 12)
						return
					}
					a.commandBox(c, "steam", steamLine, "Launch options on Steam: ")
				})
				dockRow(c, "DIFFERENCE", "What applying changes in Steam. Green is added, red is removed. A piece that only moved is not a change. Apply with the bolt button in the list on the left.", func() {
					added, removed := parse.Diff(steamLine, a.line())
					ui.Row(c.Key("diff")).Wrap().Gap(6).Padding(6, 0, 0).Children(func() {
						if len(added)+len(removed) == 0 {
							mono(ui.Text(c, "no difference"), 12.5).TextColor(pal.faint).Margin(4, 0, 0, 12)
						}
						for _, u := range removed {
							diffChip(c, "−", u, pal.danger, pal.dangerDim, pal.dangerBorder)
						}
						for _, u := range added {
							diffChip(c, "+", u, pal.accent, pal.cardOn, pal.accentDim)
						}
					})
				})
			})
		})
		ui.Row(c).Gap(6).Children(func() {
			if ghost(c, "Copy", "Copy the whole line, to paste into the game's Launch Options in Steam.", false) {
				a.copyLine()
			}
			if ghost(c, "Edit as text…", "Type or paste a full launch line; slom reads what it recognizes.", false) {
				a.openRaw()
			}
			if ghost(c, "Discard changes", "Drop what is set here, and the preset link, and go back to what Steam has for this game.", !differs) {
				a.discardChanges()
			}
			if ghost(c, "Remove all options", "Empty every launch option of this game, including the extra arguments. It is a change like any other: apply it from the list on the left, or Discard changes to undo it.", a.sel.IsEmpty()) {
				a.removeAll()
			}
		})
	})
}

// dockRow is one labeled part of the dock: a caption with an info mark on the left, the content
// on the right.
func dockRow(c *ui.Context, label, tip string, content func()) {
	ui.Row(c.Key("dock-" + label)).Gap(14).AlignItems(ui.Start).Children(func() {
		ui.Row(c).Gap(6).Width(150).Padding(10, 0, 0).Tooltip(tip).Children(func() {
			display(ui.Text(c, label), 12).TextColor(pal.muted)
			icon(c, icons.info, 14, pal.faint)
		})
		ui.Column(c).Grow(1).MinWidth(0).Children(content)
	})
}

// diffChip is one added or removed piece of the command.
func diffChip(c *ui.Context, sign, unit string, fg, bg, border ui.Color) {
	ui.Row(c).Gap(6).Padding(3, 9).Radius(7).Background(bg).Border(1, border).Children(func() {
		mono(ui.Text(c, sign), 12.5).TextColor(fg)
		mono(ui.Text(c, unit), 12.5).TextColor(fg)
	})
}

var envRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// commandBox draws the line with its parts in colors: variables cyan, programs amber,
// %command% green. It reads like one line of text and wraps like one.
func (a *App) commandBox(c *ui.Context, name, line, label string) {
	ui.Row(c.Key("cmd-"+name)).Wrap().Gap(8).Background(pal.code).Border(1, pal.line).Radius(10).Padding(10, 12).Label(label + line).Children(func() {
		afterGame, wantProgram := false, true
		for i, tk := range parse.Tokenize(line) {
			v := tk.Raw
			key := fmt.Sprint("tok-", i)
			switch {
			case v == render.Command:
				mono(ui.Text(c.Key(key), v), 12).TextColor(pal.accent)
				afterGame = true
			case afterGame:
				mono(ui.Text(c.Key(key), v), 12)
			case envRe.MatchString(v):
				name, val, _ := strings.Cut(v, "=")
				mono(ui.RichText(c.Key(key),
					ui.Span{Text: name + "=", Color: pal.cyan},
					ui.Span{Text: val, Color: pal.text}), 12)
			case v == "--":
				mono(ui.Text(c.Key(key), v), 12).TextColor(pal.muted)
				wantProgram = true
				continue
			case wantProgram:
				mono(ui.Text(c.Key(key), v), 12).TextColor(pal.amber)
			default:
				mono(ui.Text(c.Key(key), v), 12)
			}
			wantProgram = false
		}
	})
}

// ---- dialogs ----

func (a *App) dialogs(c *ui.Context) {
	ui.Modal(c, &a.rawOpen, func() {
		ui.Column(c).Width(560).Gap(10).Children(func() {
			ui.Text(c, "Edit launch options as text").Bold()
			ui.Text(c, "Recognized parts become controls; the rest stays under “More”.").TextColor(c.Theme().TextMuted)
			ui.TextArea(c, &a.raw).Font("monospace").Lines(4, 10).AutoFocus().Label("Launch options")
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				ui.Button(c, "Cancel").OnClick(func() { a.rawOpen = false })
				ui.PrimaryButton(c, "Use this").OnClick(func() { a.applyRaw(); a.rawOpen = false })
			})
		})
	})
	ui.Modal(c, &a.savingPreset, func() {
		ui.Column(c).Width(380).Gap(10).Children(func() {
			ui.Text(c, "Save as preset").Bold()
			submitted := ui.TextInput(c, &a.presetName).Placeholder("Preset name, e.g. Native 1440p").AutoFocus().Label("Preset name").Submitted()
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				ui.Button(c, "Cancel").OnClick(func() { a.savingPreset = false })
				if ui.PrimaryButton(c, "Save").Disabled(a.presetName == "").Clicked() || (submitted && a.presetName != "") {
					a.saveAsPreset(a.presetName)
					a.savingPreset = false
				}
			})
		})
	})
	if ui.AlertDialog(c, &a.deletingPreset, "Delete preset “"+a.presetName0()+"”?",
		"Games that use it keep their current settings.", "Cancel", "Delete") == 1 {
		a.deletePreset()
	}
	if ui.AlertDialog(c, &a.closingSteam, ternary(len(a.applyIDs) > 1, fmt.Sprintf("Force apply to %d games?", len(a.applyIDs)), "Force apply?"),
		forceMessage(len(a.applyIDs)), "Cancel", "Force apply") == 1 {
		a.forceApply(a.applyIDs)
	}
}

// chipEditor edits a list of verbatim tokens as chips: one chip per command or flag with its
// value (group), each editable in place, plus a field that adds one on Enter. The chips are
// kept in the app while the user types, so a chip never re-splits under the cursor; they are
// rebuilt from the tokens when those change from elsewhere (another game, an import).
func (a *App) chipEditor(c *ui.Context, id, label, tip string, tokens []string, group bool, set func([]string)) {
	flatten := func(chips []string) []string {
		if group || id == "env" {
			return parse.Ungroup(chips)
		}
		var out []string // MangoHud items: one item per chip, kept as typed
		for _, ch := range chips {
			if ch = strings.TrimSpace(ch); ch != "" {
				out = append(out, ch)
			}
		}
		return out
	}
	key := a.cur.AppID + "/" + id
	st := a.chips[key]
	if st == nil || !slices.Equal(flatten(st), tokens) {
		if group {
			st = parse.Group(tokens)
		} else {
			st = append([]string(nil), tokens...)
		}
	}
	ui.Row(c.Key("x-" + id)).Gap(12).Children(func() {
		ui.Row(c).Gap(8).Width(200).Tooltip(tip).Children(func() {
			if col, ok := slotColor(id); ok {
				dot(c, col, false)
			}
			ui.Text(c, label)
		})
		ui.Column(c).Grow(1).Gap(4).Children(func() {
			changed := false
			for i := range st {
				txt := st[i]
				if ui.TextInput(c.Key(fmt.Sprintf("chip-%s-%d", id, i)), &txt).Font("monospace").Label(fmt.Sprintf("%s %d", label, i+1)).Changed() {
					st[i], changed = txt, true
				}
			}
			add := a.chipDraft[key]
			if ui.TextInput(c.Key("add-"+id), &add).Placeholder("Add…  (press Enter)").Label("Add to "+label).Submitted() && strings.TrimSpace(add) != "" {
				st, add, changed = append(st, add), "", true
			}
			a.chipDraft[key] = add
			if changed {
				set(flatten(st))
				a.commit()
			}
		})
	})
	a.chips[key] = st
}

func forceMessage(n int) string {
	what := "the launch options saved"
	if n > 1 {
		what = fmt.Sprintf("the launch options of %d games saved", n)
	}
	return "Steam will be closed, " + what + ", and Steam opened again. Anything running in Steam, including games, will be closed."
}
