package ui

import (
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/secato/slom/internal/desktop"
)

// The look follows the desktop. Dark or light, in order of say:
//  1. SLOM_THEME=dark|light, for when detection is wrong (the default is "system");
//  2. Omarchy's active theme, when Omarchy is there;
//  3. the system's appearance, as MyGo reads it (GTK setting, desktop portal).
//
// The accent comes from Omarchy's theme, else from the desktop portal (GNOME 47+, KDE Plasma),
// else slom's own lime. See usableAccent for how a chosen accent is made to fit.

// themeModeFromEnv reads SLOM_THEME: "dark", "light", or "" for the system's.
func themeModeFromEnv() string {
	switch v := strings.ToLower(strings.TrimSpace(os.Getenv("SLOM_THEME"))); v {
	case "dark", "light":
		return v
	}
	return ""
}

// themeChoice is "dark", "light" or "system": SLOM_THEME when set, else what the toggle saved.
func (a *App) themeChoice() string {
	if a.themeMode != "" {
		return a.themeMode
	}
	return a.store.Theme()
}

// themeToggle is the Dark / Light / System switch at the bottom of the sidebar.
func (a *App) themeToggle(c *ui.Context) {
	cur := a.themeChoice()
	locked := a.themeMode != ""
	ui.Row(c).Gap(6).Children(func() {
		for _, o := range []struct{ id, label, tip string }{
			{"dark", "Dark", "Always use the dark look"},
			{"light", "Light", "Always use the light look"},
			{"system", "System", "Follow the desktop's dark or light setting and accent"},
		} {
			tip := o.tip
			if locked {
				tip = "SLOM_THEME is set, so it decides the look"
			}
			if segment(c, o.label, tip, cur == o.id, locked) && !locked && cur != o.id {
				if err := a.store.SetTheme(o.id); err != nil {
					a.notify("Couldn't save the look: " + err.Error())
				}
			}
		}
	})
}

// segment is one choice of a row of equal choices.
func segment(c *ui.Context, label, tip string, on, disabled bool) bool {
	b := ui.ButtonBase(c.Key("seg-"+label)).Grow(1).Height(30).Padding(0, 8).Radius(8).Tooltip(tip).Disabled(disabled).
		Background(ternary(on, pal.cardOn, pal.ground)).Border(1, ternary(on, pal.accent, pal.lineStrong))
	if b.Hovered() && !on && !disabled {
		b.Background(c.Theme().SurfaceHover)
	}
	b.Children(func() {
		ui.Row(c).Grow(1).Justify(ui.Center).Children(func() {
			ui.Text(c, label).FontSize(12.5).TextColor(ternary(on, pal.accent, pal.muted))
		})
	})
	return b.Clicked()
}

// look is the palette for this frame.
func (a *App) look(c *ui.Context) palette {
	dark := c.Theme().Dark // the system's, before slom sets its own theme
	if a.omarchy != nil && a.omarchy.HasMode {
		dark = a.omarchy.Dark
	}
	switch a.themeChoice() {
	case "dark":
		dark = true
	case "light":
		dark = false
	}
	var accent ui.Color
	var ok bool
	if o := a.omarchy; o != nil && o.HasAccent {
		accent, ok = ui.RGB(o.Accent[0], o.Accent[1], o.Accent[2]), true
	} else if p := c.Preferences(); p.Accent.A > 0 {
		accent, ok = p.Accent, true
	}
	return paletteFor(dark, accent, ok)
}

// readOmarchy loads Omarchy's theme, if it is there.
func (a *App) readOmarchy() {
	if l, stamp, ok := desktop.ReadOmarchy(); ok {
		a.omarchy, a.omarchyStamp = &l, stamp
	}
}

// pollOmarchy notices a theme switch. Omarchy replaces the theme directory rather than editing
// it, and a switch between two dark themes changes nothing the desktop portal reports, so the
// stamp Omarchy writes after each switch is what slom watches. While a switch is half done the
// read fails and the last look stays.
func (a *App) pollOmarchy() {
	for {
		time.Sleep(time.Second)
		l, stamp, ok := desktop.ReadOmarchy()
		if !ok {
			continue
		}
		mygo.RunOnMain(func() {
			if stamp != a.omarchyStamp {
				a.omarchy, a.omarchyStamp = &l, stamp
				a.win.Invalidate()
			}
		})
	}
}
