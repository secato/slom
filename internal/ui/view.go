package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/secato/slom/internal/catalog"
	"github.com/secato/slom/internal/parse"
	"github.com/secato/slom/internal/store"
)

// View builds the whole window.
func (a *App) View(c *ui.Context) {
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

// ---- left: the game list ----

func (a *App) sidebar(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(12).Gap(8).Children(func() {
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

		sel := -1
		for row, gi := range a.visible {
			if a.cur != nil && a.games[gi].AppID == a.cur.AppID {
				sel = row
			}
		}
		a.list.Key = func(i int) any { return a.games[a.visible[i]].AppID }
		a.list.Label = func(i int) string { return a.games[a.visible[i]].Name }
		a.list.Selected = &sel
		list := ui.List(c, &a.list, len(a.visible), func(i int) {
			g := a.games[a.visible[i]]
			ui.Row(c).Gap(8).Padding(6, 8).Children(func() {
				ui.Text(c, g.Name).SingleLine().Grow(1)
				switch {
				case a.pending(g.AppID):
					ui.Badge(c, "pending").Tooltip("Configured in slom, but different from what Steam has. Apply it.")
				case a.steamLines[g.AppID] != "":
					ui.Text(c, "●").TextColor(t.Accent).Tooltip("Has launch options in Steam")
				}
			})
		}).Grow(1)
		if list.Changed() && sel >= 0 && sel < len(a.visible) {
			a.selectGame(&a.games[a.visible[sel]])
		}
		if len(a.visible) == 0 {
			ui.Text(c, "No installed games match.").TextColor(t.TextMuted)
		}
		if n := a.pendingCount(); n > 0 {
			ui.Text(c, fmt.Sprintf("%d game(s) out of date", n)).TextColor(t.Warning).FontSize(12).
				Tooltip("Their configuration in slom differs from the launch options in Steam. Select each and Apply.")
		}
	})
}

// ---- right: the editor ----

func (a *App) editor(c *ui.Context) {
	t := c.Theme()
	if a.cur == nil {
		ui.Column(c).Fill().Center().Gap(6).Children(func() {
			ui.Text(c, "Pick a game").Bold().FontSize(18)
			ui.Text(c, "Choose a game on the left to set its Proton, DLSS, gamescope and MangoHud options.").TextColor(t.TextMuted)
		})
		return
	}
	g := a.cur
	// the key scopes every control to the game, so no editing state leaks between games
	ui.Column(c.Key("game-" + g.AppID)).Fill().Padding(16).Gap(12).Children(func() {
		ui.Row(c).Gap(10).Children(func() {
			ui.Text(c, g.Name).Bold().FontSize(20).SingleLine()
			ui.Text(c, "App "+g.AppID).TextColor(t.TextMuted)
			ui.Spacer(c)
			if !g.HasProtonPrefix {
				ui.Badge(c, "no Proton prefix").Tooltip("No Proton prefix was found for this game: it is probably a native Linux game, or has not run through Proton yet. Proton and DLSS options only apply to Windows games run through Proton.")
			}
		})
		if a.steamRunning {
			a.steamBanner(c)
		}
		a.presetBar(c)
		groups := a.cat.Groups()
		tabs := append(a.tabLabels(groups), "Everything else")
		ui.Segmented(c.Key("tabs"), &a.tab, tabs...).Label("Option groups")
		ui.Scroll(c).Grow(1).Gap(14).Padding(0, 8, 0, 0).Children(func() {
			ui.Column(c).MaxWidth(820).Gap(14).Children(func() {
				if a.tab >= 0 && a.tab < len(groups) {
					a.group(c, groups[a.tab])
				} else {
					a.extras(c)
				}
			})
		})
		a.footer(c)
	})
}

func (a *App) steamBanner(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Gap(10).Padding(10, 12).Radius(8).Border(1, t.Warning).Children(func() {
		ui.Text(c, "Steam is running, so slom can't save to it. Close Steam and press Apply, or let slom do it.").Grow(1)
		if ui.Button(c, "Close Steam, apply, reopen").Disabled(a.busy != "").Clicked() {
			a.closingSteam = true
		}
	})
}

func (a *App) presetBar(c *ui.Context) {
	t := c.Theme()
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
		ui.Text(c, "Preset").Tooltip("A preset is a reusable set of options. A game linked to a preset follows it; the changes you make to this game are kept as its own overrides (marked •).")
		if ui.Select(c.Key("preset"), &cur, labels).Label("Preset").Width(220).Changed() {
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
		ui.Spacer(c)
		for _, n := range a.notes {
			ui.Text(c, n).TextColor(t.TextMuted).FontSize(12)
		}
	})
}

// group builds the options of one catalog group.
func (a *App) group(c *ui.Context, name string) {
	t := c.Theme()
	ui.Column(c.Key("g-" + name)).Gap(2).Children(func() {
		ui.Text(c, name).Bold().FontSize(14).TextColor(t.TextMuted).Margin(0, 0, 4, 0)
		ui.Divider(c)
		as, _ := a.store.Assignment(a.cur.AppID)
		over := as.Overridden()
		for i := range a.cat.Options {
			o := &a.cat.Options[i]
			if o.Group == name {
				a.optionRow(c, o, over[o.ID] && a.presetID != "")
			}
		}
	})
}

// optionRow is one option: label with its tooltip, and the control for its kind.
func (a *App) optionRow(c *ui.Context, o *catalog.Option, overridden bool) {
	t := c.Theme()
	reason := a.cat.Reason(a.sel, a.tools, o.ID)
	_, on := a.sel.Values[o.ID]
	indent := float32(0)
	if o.Host != "" {
		indent = 16
	}
	ui.Row(c.Key(o.ID)).Gap(12).Padding(5, 4).Children(func() {
		tip := o.Help
		if o.Support != "" {
			tip += "\n\n" + o.Support
		}
		if reason != "" {
			tip += "\n\n" + reason
		}
		ui.Row(c).Gap(6).Grow(1).Margin(0, 0, 0, indent).Tooltip(tip).Children(func() {
			label := ui.Text(c, o.Label)
			if reason != "" && !on {
				label.TextColor(t.TextMuted)
			}
			if overridden {
				ui.Text(c, "•").TextColor(t.Accent).Tooltip("This game overrides its preset here.")
			}
			ui.Text(c, "ⓘ").TextColor(t.TextMuted).FontSize(12)
		})
		a.control(c, o, reason != "" && !on)
	})
}

// control is the widget for the option's kind; changing it goes through setOption, which
// switches on what the option requires and tells the user.
func (a *App) control(c *ui.Context, o *catalog.Option, disabled bool) {
	cur, on := a.sel.Values[o.ID]
	switch o.Kind {
	case catalog.Bool:
		v := on
		if ui.Switch(c.Key("c-"+o.ID), &v).Label(o.Label).Disabled(disabled).Changed() {
			a.setOption(o.ID, boolValue(v))
		}
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
					a.setOption(o.ID, a.cat.Initial(o, a.defaults))
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
	t := c.Theme()
	ui.Column(c.Key("extras")).Gap(6).Children(func() {
		ui.Text(c, "Everything else").Bold().FontSize(14).TextColor(t.TextMuted).Margin(0, 0, 4, 0)
		ui.Divider(c)
		ui.Text(c, "Anything slom does not have a control for stays here, exactly as it was, and is kept in the command. Add your own too.").TextColor(t.TextMuted).FontSize(12)
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

// ---- bottom: preview and actions ----

func (a *App) footer(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, "Launch options").Bold().FontSize(13).TextColor(t.TextMuted)
		pv := a.preview()
		ui.TextArea(c.Key("preview"), &pv).ReadOnly(true).Font("monospace").Lines(2, 6).Label("Launch options preview")
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "Copy").Tooltip("Copy the whole line, to paste into the game's Launch Options in Steam.").Clicked() {
				a.copyLine()
			}
			ui.Button(c, "Edit as text…").Tooltip("Type or paste a full launch line; slom reads what it recognizes.").OnClick(a.openRaw)
			ui.Button(c, "Import from Steam").Tooltip("Replace these settings with what Steam currently has for this game.").
				Disabled(a.steamLines[a.cur.AppID] == "").OnClick(a.importFromSteam)
			ui.Spacer(c)
			if a.busy != "" {
				ui.Text(c, a.busy).TextColor(t.TextMuted)
			}
			switch {
			case a.steamRunning:
				ui.PrimaryButton(c, "Apply").Disabled(true).Tooltip("Steam is running. Close it first, or use “Close Steam, apply, reopen” above.")
			default:
				ui.PrimaryButton(c, "Apply").Disabled(a.busy != "").Tooltip("Write these launch options into Steam's configuration. A backup is made first.").OnClick(a.apply)
			}
		})
	})
}

// ---- dialogs ----

func (a *App) dialogs(c *ui.Context) {
	ui.Modal(c, &a.rawOpen, func() {
		ui.Column(c).Width(560).Gap(10).Children(func() {
			ui.Text(c, "Edit launch options as text").Bold()
			ui.Text(c, "Recognized parts become controls; the rest stays under “Everything else”.").TextColor(c.Theme().TextMuted)
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
	if ui.AlertDialog(c, &a.closingSteam, "Close Steam and apply?",
		"Steam will be closed, the launch options written, and Steam opened again. Anything running in Steam, including games, will be closed.",
		"Cancel", "Close Steam and apply") == 1 {
		a.closeSteamAndApply()
	}
}

// tabLabels names each group's tab, with a dot where the game sets something in it.
func (a *App) tabLabels(groups []string) []string {
	used := map[string]bool{}
	for id := range a.sel.Values {
		if o := a.cat.Get(id); o != nil {
			used[o.Group] = true
		}
	}
	out := make([]string, len(groups))
	for i, g := range groups {
		out[i] = g
		if used[g] {
			out[i] += " •"
		}
	}
	return out
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
		ui.Text(c, label).Width(190).Tooltip(tip)
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
