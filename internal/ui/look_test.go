package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/secato/slom/internal/desktop"
)

func TestLookFollowsTheSystemAppearance(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.View, 900, 600)
	tt.SetDark(true)
	tt.Frame()
	if !pal.dark || pal.ground != darkPalette.ground {
		t.Fatal("a dark desktop gets the dark palette")
	}
	tt.SetDark(false)
	tt.Frame()
	if pal.dark || pal.ground != lightPalette.ground {
		t.Fatal("a light desktop gets the light palette, even after slom set its own theme before")
	}
	tt.SetDark(true)
	tt.Frame()
	if !pal.dark {
		t.Fatal("and back to dark")
	}
}

func TestSlomThemeOverridesTheSystem(t *testing.T) {
	a := testApp(t)
	a.themeMode = "light"
	tt := ui.NewTester(a.View, 900, 600)
	tt.SetDark(true)
	tt.Frame()
	if pal.dark {
		t.Error("SLOM_THEME=light wins over a dark desktop")
	}
	a.themeMode = "dark"
	tt.SetDark(false)
	tt.Frame()
	if !pal.dark {
		t.Error("SLOM_THEME=dark wins over a light desktop")
	}
	t.Setenv("SLOM_THEME", " Light ")
	if themeModeFromEnv() != "light" {
		t.Error("the value is case-insensitive")
	}
	t.Setenv("SLOM_THEME", "system")
	if themeModeFromEnv() != "" {
		t.Error("anything else means the system's")
	}
}

func TestOmarchyDecidesDarkAndAccent(t *testing.T) {
	a := testApp(t)
	a.omarchy = &desktop.Look{Dark: false, HasMode: true, Accent: [3]uint8{0x33, 0x66, 0x99}, HasAccent: true}
	tt := ui.NewTester(a.View, 900, 600)
	tt.SetDark(true) // the system says dark, Omarchy says light: Omarchy knows first
	tt.Frame()
	if pal.dark {
		t.Error("Omarchy's mode wins over the system's")
	}
	if pal.accent == lightPalette.accent {
		t.Error("Omarchy's accent replaces slom's")
	}
	a.omarchy = &desktop.Look{Dark: true, HasMode: true}
	tt.Frame()
	if !pal.dark || pal.accent != darkPalette.accent {
		t.Error("a theme without an accent keeps slom's lime")
	}
}

func TestPortalAccentIsUsedWhenThereIsNoOmarchy(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.View, 900, 600)
	tt.SetDark(true)
	tt.SetPreferences(ui.Preferences{Accent: ui.Hex("#3584E4"), TextScale: 1})
	tt.Frame()
	if pal.accent == darkPalette.accent {
		t.Error("the desktop's accent replaces slom's")
	}
	if got, want := pal.cardOn, darkPalette.cardOn; got == want {
		t.Error("the colors around the accent follow its hue")
	}
	tt.SetPreferences(ui.Preferences{TextScale: 1})
	tt.Frame()
	if pal.accent != darkPalette.accent {
		t.Error("no desktop accent, back to lime")
	}
}

func TestGreyAccentGetsColorAndAchromaticFallsBack(t *testing.T) {
	// Last Horizon's muted rose
	rose := ui.Hex("#b59790")
	_, before, _ := oklch(rose)
	got, ok := usableAccent(rose, true)
	if !ok {
		t.Fatal("a muted color is raised, not dropped")
	}
	_, after, h := oklch(got)
	_, _, wantH := oklch(rose)
	if after < accentMinChroma-0.01 || after <= before {
		t.Errorf("chroma %.3f -> %.3f, want at least %.2f", before, after, accentMinChroma)
	}
	if d := h - wantH; d > 6 || d < -6 {
		t.Errorf("hue moved from %.0f to %.0f", wantH, h)
	}
	if _, ok := usableAccent(ui.Hex("#808080"), true); ok {
		t.Error("a pure grey has no hue: slom keeps its own accent")
	}
	p := paletteFor(true, ui.Hex("#808080"), true)
	if p.accent != darkPalette.accent {
		t.Error("so the palette stays lime")
	}
}

func TestLimeIsUntouchedByTheDefaultPalette(t *testing.T) {
	if p := paletteFor(true, ui.Color{}, false); p != darkPalette {
		t.Error("no accent, the dark palette is exactly slom's own")
	}
	if p := paletteFor(false, ui.Color{}, false); p != lightPalette {
		t.Error("no accent, the light palette is exactly slom's own")
	}
}

// The text on each surface must be readable, in both looks and with any accent a desktop gives.
func TestPalettesAreReadable(t *testing.T) {
	accents := map[string]ui.Color{
		"none": {}, "gnome blue": ui.Hex("#3584E4"), "gnome yellow": ui.Hex("#C88800"), "gnome slate": ui.Hex("#6F8396"),
		"gnome red": ui.Hex("#E62D42"), "omarchy rose": ui.Hex("#b59790"), "white": ui.Hex("#FFFFFF"), "near black": ui.Hex("#101820"),
	}
	for name, ac := range accents {
		for _, dark := range []bool{true, false} {
			p := paletteFor(dark, ac, ac.A != 0)
			look := map[bool]string{true: "dark", false: "light"}[dark]
			check := func(what string, fg, bg ui.Color, min float64) {
				t.Helper()
				if r := contrast(fg, bg); r < min {
					t.Errorf("%s %s: %s on its background is %.2f:1, want %.1f", name, look, what, r, min)
				}
			}
			for _, bg := range []ui.Color{p.ground, p.panel, p.card} {
				check("text", p.text, bg, 7)
				check("muted", p.muted, bg, 4.5)
				check("faint", p.faint, bg, 3.5)
				check("accent", p.accent, bg, 3)
				check("amber", p.amber, bg, 3)
				check("cyan", p.cyan, bg, 3)
				check("violet", p.violet, bg, 3)
				check("danger", p.danger, bg, 3)
			}
			check("accent on a lit card", p.accent, p.cardOn, 3)
			check("on-accent text", p.onAccent, p.accent, 4.5)
			check("amber text on its tint", p.amberText, p.amberDim, 4.5)
			check("amber on its pill", p.amber, p.amberPill, 3)
			check("cyan on its tint", p.cyan, p.cyanDim, 3)
			check("violet on its tint", p.violet, p.violetDim, 3)
			check("danger on its tint", p.danger, p.dangerDim, 3)
		}
	}
}

func TestOklchRoundTrips(t *testing.T) {
	for _, h := range []string{"#C6FF3D", "#3584E4", "#b59790", "#FFB454"} {
		c := ui.Hex(h)
		l, ch, hue := oklch(c)
		back := ui.Oklch(l, ch, hue)
		d := func(a, b uint8) int { return int(a) - int(b) }
		if d(c.R, back.R) > 2 || d(c.R, back.R) < -2 || d(c.G, back.G) > 2 || d(c.G, back.G) < -2 || d(c.B, back.B) > 2 || d(c.B, back.B) < -2 {
			t.Errorf("%s -> %v", h, back)
		}
	}
}

func TestThemeToggleChoosesAndSaves(t *testing.T) {
	a := testApp(t)
	tt := ui.NewTester(a.View, 1000, 700)
	tt.SetDark(true)
	tt.Frame()
	if a.themeChoice() != "system" || !pal.dark {
		t.Fatalf("starts on System, following the dark desktop: %q", a.themeChoice())
	}
	tt.Click("Light")
	tt.Frame()
	if a.store.Theme() != "light" || pal.dark {
		t.Errorf("Light: saved %q, dark %v", a.store.Theme(), pal.dark)
	}
	tt.Click("Dark")
	tt.SetDark(false)
	tt.Frame()
	if a.store.Theme() != "dark" || !pal.dark {
		t.Errorf("Dark wins over a light desktop: saved %q, dark %v", a.store.Theme(), pal.dark)
	}
	tt.Click("System")
	tt.Frame()
	if a.store.Theme() != "system" || pal.dark {
		t.Errorf("System follows the light desktop again: saved %q, dark %v", a.store.Theme(), pal.dark)
	}
}

func TestThemeToggleIsLockedByTheEnvironment(t *testing.T) {
	a := testApp(t)
	a.themeMode = "dark"
	tt := ui.NewTester(a.View, 1000, 700)
	tt.Frame()
	tt.Click("Light")
	tt.Frame()
	if a.store.Theme() != "system" || !pal.dark {
		t.Error("SLOM_THEME decides; the toggle changes nothing")
	}
}
