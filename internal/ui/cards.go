package ui

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/secato/slom/internal/catalog"
)

// The tab of one option group: an intro that says what the group is (with the master switch
// of the group, if it has one), then its options under small headings. Switches are cards in
// two columns; sizes, refresh rates and other numbers are cards with the control they deserve.

// resolutionCard says which option ids are the width and height of a resolution, and how the
// card is titled. The height options are not drawn on their own.
type resolutionCard struct{ heightID, title, help string }

var resolutionCards = map[string]resolutionCard{
	"gs.out_w": {"gs.out_h", "Output resolution", "The size of the gamescope window on your display."},
	"gs.in_w":  {"gs.in_h", "Game resolution", "What the game renders at. Lower is faster; gamescope scales it up to the output."},
}

func (a *App) group(c *ui.Context, name string) {
	master := a.cat.Master(name)
	as, _ := a.store.Assignment(a.cur.AppID)
	over := as.Overridden()
	skip := map[string]bool{}
	for _, r := range resolutionCards {
		skip[r.heightID] = true
	}
	ui.Column(c.Key("g-" + name)).Gap(14).Children(func() {
		a.intro(c, name, master)
		for _, sec := range a.cat.Sections(name) {
			ui.Column(c.Key("sec-" + sec)).Gap(10).Children(func() {
				if sec != "" {
					sectionTitle(c, sec)
				}
				ui.Grid(c).Columns(2).Gap(10).Children(func() {
					for i := range a.cat.Options {
						o := &a.cat.Options[i]
						if o.Group != name || o.Section != sec || (master != nil && o.ID == master.ID) || skip[o.ID] {
							continue
						}
						switch {
						case resolutionCards[o.ID].heightID != "":
							a.resolutionCard(c, o, a.cat.Get(resolutionCards[o.ID].heightID))
						case o.Kind == catalog.Int:
							a.sliderCard(c, o, over[o.ID] && a.presetID != "")
						default:
							a.optionRow(c, o, over[o.ID] && a.presetID != "")
						}
					}
				})
			})
		}
	})
}

// intro is the box on top of a tab: what the group is, and its master switch.
func (a *App) intro(c *ui.Context, name string, master *catalog.Option) {
	on := false
	reason := ""
	if master != nil {
		_, on = a.sel.Values[master.ID]
		reason = a.cat.Reason(a.sel, a.tools, master.ID)
	}
	card := ui.Row(c.Key("intro-"+name)).Gap(16).Padding(14, 16).Radius(14).
		Background(ternary(on, pal.cardOn, pal.card)).Border(1, ternary(on, pal.accentDim, pal.line))
	if on {
		card.Shadow(0, 0, 26, 0, ui.RGBA(198, 255, 61, 0.08))
	}
	card.Children(func() {
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			display(ui.Text(c, strings.ToUpper(name)), 19)
			ui.Text(c, a.cat.Summary(name)).FontSize(13).TextColor(pal.muted)
			if reason != "" && !on {
				ui.Text(c, reason).FontSize(12.5).TextColor(pal.amber)
			}
		})
		if master != nil {
			tip := master.Help
			if master.Support != "" {
				tip += "\n\n" + master.Support
			}
			ui.Row(c).Gap(10).Tooltip(tip).Children(func() {
				ui.Text(c, ternary(on, "On", "Off")).TextColor(ternary(on, pal.accent, pal.muted))
				a.control(c, master, reason != "" && !on, on)
			})
		}
	})
}

func sectionTitle(c *ui.Context, title string) {
	ui.Row(c).Gap(10).Padding(4, 2, 0).Children(func() {
		display(ui.Text(c, strings.ToUpper(title)), 12).TextColor(pal.muted)
		ui.Box(c).Grow(1).Height(1).Background(pal.line)
	})
}

// card is the frame of an option that needs more than a switch.
func card(c *ui.Context, key string, on bool, span int) ui.Element {
	e := ui.Column(c.Key(key)).Gap(10).Padding(12, 16).Radius(12).
		Background(ternary(on, pal.cardOn, pal.card)).Border(1, ternary(on, pal.accentDim, pal.line))
	if span > 1 {
		e.ColumnSpan(span)
	}
	return e
}

// cardHead is the top line of a card: the label with its info mark, and the value on the right.
func cardHead(c *ui.Context, label, tip, value string, on bool) {
	ui.Row(c).Gap(8).Children(func() {
		ui.Row(c).Gap(8).Grow(1).Tooltip(tip).Children(func() {
			ui.Text(c, label)
			icon(c, icons.info, 15, pal.faint)
		})
		mono(ui.Text(c, value), 13).TextColor(ternary(on, pal.accent, pal.faint))
	})
}

// choice is a pill that is either the chosen one or not.
func choice(c *ui.Context, label, tip string, on bool) bool {
	b := ui.ButtonBase(c.Key("pill-"+label)).Height(34).Padding(0, 12).Radius(9).Tooltip(tip).
		Background(ternary(on, pal.cardOn, pal.ground)).Border(1, ternary(on, pal.accent, pal.lineStrong))
	if b.Hovered() && !on {
		b.Background(c.Theme().SurfaceHover)
	}
	b.Children(func() { mono(ui.Text(c, label), 12.5).TextColor(ternary(on, pal.accent, pal.text)) })
	return b.Clicked()
}

// ---- resolution ----

type size struct{ w, h int }

// resolutions are the sizes offered: your screen's first among the usual ones.
func (a *App) resolutions() []size {
	out := []size{{1280, 720}, {1600, 900}, {1920, 1080}, {2560, 1440}, {3200, 1800}, {3840, 2160}}
	if w, h := a.defaults["screen.width"], a.defaults["screen.height"]; w > 0 && h > 0 && !slices.Contains(out, size{w, h}) {
		out = append(out, size{w, h})
	}
	slices.SortFunc(out, func(x, y size) int { return x.w*x.h - y.w*y.h })
	return out
}

func (s size) String() string { return fmt.Sprintf("%d×%d", s.w, s.h) }

// resolutionCard picks a width and height together, from pills or typed.
func (a *App) resolutionCard(c *ui.Context, wo, ho *catalog.Option) {
	r := resolutionCards[wo.ID]
	wv, wOn := a.sel.Values[wo.ID]
	hv, hOn := a.sel.Values[ho.ID]
	cur := size{int(atof(wv)), int(atof(hv))}
	on := wOn || hOn
	presets := a.resolutions()
	matches := wOn && hOn && slices.Contains(presets, cur)
	custom := a.customRes[a.cur.AppID+"/"+wo.ID] || (on && !matches)
	value := "Not set"
	if on {
		value = strings.Replace(cur.String(), "×", " × ", 1)
	}
	card(c, wo.ID, on, 2).Children(func() {
		cardHead(c, r.title, r.help+"\n\n"+wo.Help, value, on)
		ui.Row(c).Wrap().Gap(6).Children(func() {
			for _, p := range presets {
				tip := ""
				if p.w == a.defaults["screen.width"] && p.h == a.defaults["screen.height"] {
					tip = "Your screen's size."
				}
				if choice(c, p.String(), tip, matches && p == cur) {
					a.customRes[a.cur.AppID+"/"+wo.ID] = false
					if matches && p == cur {
						a.setOption(wo.ID, "")
						a.setOption(ho.ID, "")
					} else {
						a.setOption(wo.ID, itoa(p.w))
						a.setOption(ho.ID, itoa(p.h))
					}
				}
			}
			if choice(c, "Custom", "Type a width and a height.", custom) {
				if custom && !on {
					a.customRes[a.cur.AppID+"/"+wo.ID] = false
				} else {
					a.customRes[a.cur.AppID+"/"+wo.ID] = true
					if !on {
						a.setOption(wo.ID, a.cat.Initial(wo, a.defaults))
						a.setOption(ho.ID, a.cat.Initial(ho, a.defaults))
					}
				}
			}
		})
		if custom {
			ui.Row(c).Gap(8).Children(func() {
				w, h := atof(wv), atof(hv)
				if ui.NumberInput(c.Key("w-"+wo.ID), &w, float64(wo.Min), float64(wo.Max), 1).Label(wo.Label).Width(120).Changed() {
					a.setOption(wo.ID, itoa(int(w)))
				}
				ui.Text(c, "×").TextColor(pal.muted)
				if ui.NumberInput(c.Key("h-"+ho.ID), &h, float64(ho.Min), float64(ho.Max), 1).Label(ho.Label).Width(120).Changed() {
					a.setOption(ho.ID, itoa(int(h)))
				}
			})
		}
		if w, h := a.defaults["screen.width"], a.defaults["screen.height"]; w > 0 && h > 0 {
			ui.Text(c, "Your screen is "+size{w, h}.String()+".").FontSize(12.5).TextColor(pal.muted)
		}
	})
}

// ---- numbers ----

// sliderRange is the span of a slider: the option's own for small ranges, and 30 up to the
// screen's refresh rate (at least 240) for rates.
func (a *App) sliderRange(o *catalog.Option) (lo, hi float64) {
	lo, hi = float64(o.Min), float64(o.Max)
	if hi-lo > 200 {
		lo = math.Max(lo, 30)
		hi = math.Max(240, math.Ceil(float64(a.defaults["screen.refresh"])/30)*30)
	}
	return lo, hi
}

// sliderCard sets a number with a slider. Until it is touched the option is not set.
func (a *App) sliderCard(c *ui.Context, o *catalog.Option, overridden bool) {
	cur, on := a.sel.Values[o.ID]
	lo, hi := a.sliderRange(o)
	v := atof(cur)
	if !on {
		v = atof(a.cat.Initial(o, a.defaults))
	}
	v = math.Min(math.Max(v, lo), hi)
	value := "Not set"
	if on {
		value = itoa(int(v)) + ternary(o.Unit != "", " "+o.Unit, "")
	}
	tip := o.Help
	if o.Support != "" {
		tip += "\n\n" + o.Support
	}
	card(c, o.ID, on, 1).Children(func() {
		cardHead(c, o.Label+ternary(overridden, " •", ""), tip, value, on)
		sl := ui.Slider(c.Key("sl-"+o.ID), &v, lo, hi).Label(o.Label)
		if !on {
			sl.Opacity(0.4) // untouched: the thumb shows the default, nothing is set
		}
		if sl.Changed() {
			a.setOption(o.ID, itoa(int(math.Round(v))))
		}
		ui.Row(c).Justify(ui.SpaceBetween).Children(func() {
			for i := 0; i <= 4; i++ {
				mono(ui.Text(c, itoa(int(math.Round(lo+(hi-lo)*float64(i)/4)))), 12).TextColor(pal.faint)
			}
		})
		if on && ghost(c, "Clear", "Stop setting this option.", false) {
			a.setOption(o.ID, "")
		}
	})
}
