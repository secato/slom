package ui

import (
	_ "embed"
	"hash/fnv"
	"sync"

	"github.com/egoist/mygo/ui"
)

// Fonts: both are SIL OFL, licenses next to the files. Embedded so the look does not depend
// on what the system has; the families fall back to the system's when registering fails.
var (
	//go:embed assets/Rajdhani-Bold.ttf
	displayTTF []byte
	//go:embed assets/JetBrainsMono-Regular.ttf
	monoTTF []byte
)

const (
	fontDisplay = "Rajdhani"
	fontMono    = "JetBrains Mono"
)

// Icons, parsed once. They take the color of the text around them.
var (
	//go:embed assets/bolt.svg
	boltSVG []byte
	//go:embed assets/search.svg
	searchSVG []byte
	//go:embed assets/info.svg
	infoSVG []byte
	//go:embed assets/copy.svg
	copySVG []byte

	icons struct {
		bolt, search, info, copy *ui.SVG
	}
	setupOnce sync.Once
)

// setup registers the fonts and parses the icons; safe to call every frame.
func setup() {
	setupOnce.Do(func() {
		_ = ui.RegisterFont(displayTTF, fontDisplay)
		_ = ui.RegisterFont(monoTTF, fontMono)
		for svg, dst := range map[*[]byte]**ui.SVG{
			&boltSVG: &icons.bolt, &searchSVG: &icons.search, &infoSVG: &icons.info, &copySVG: &icons.copy,
		} {
			if s, err := ui.ParseSVG(*svg); err == nil {
				*dst = s
			}
		}
	})
}

// icon draws an icon if it parsed; a missing icon is just absent.
func icon(c *ui.Context, s *ui.SVG, size float32, color ui.Color) {
	if s != nil {
		ui.Icon(c, s).Width(size).Height(size).TextColor(color)
	}
}

// palette is every color of the look, by what it means. The accent always means "this will be
// applied", amber is for Steam being in the way, cyan is information. There is one dark and one
// light palette; the accent group comes from the desktop when it has one (see paletteFor).
type palette struct {
	dark                                               bool
	ground, panel, card, cardOn, line, lineStrong      ui.Color
	surface, surfaceHover, surfacePressed              ui.Color
	text, muted, faint                                 ui.Color
	accent, accentDim, accentGlow, onAccent            ui.Color
	accentHover, accentPressed                         ui.Color
	amber, amberDim, amberBorder, amberText, amberGlow ui.Color
	amberPill                                          ui.Color
	cyan, cyanDim, cyanBorder                          ui.Color
	code                                               ui.Color
	danger, dangerDim, dangerBorder                    ui.Color
	violet, violetDim, violetBorder                    ui.Color
	shadow                                             float32 // opacity of the dock's shadow
}

var darkPalette = palette{
	dark:   true,
	ground: ui.Hex("#0A0C11"), panel: ui.Hex("#0E1118"), card: ui.Hex("#0F131C"), cardOn: ui.Hex("#141F0A"),
	line: ui.Hex("#1B2130"), lineStrong: ui.Hex("#2A3248"),
	surface: ui.Hex("#161C2B"), surfaceHover: ui.Hex("#1D2437"), surfacePressed: ui.Hex("#232B3F"),
	text: ui.Hex("#E8ECF4"), muted: ui.Hex("#8E99B3"), faint: ui.Hex("#6D7893"),
	accent: ui.Hex("#C6FF3D"), accentDim: ui.Hex("#3A5415"), accentGlow: ui.RGBA(198, 255, 61, 0.45), onAccent: ui.Hex("#0A0C11"),
	accentHover: ui.Hex("#D6FF70"), accentPressed: ui.Hex("#A8DE2A"),
	amber: ui.Hex("#FFB454"), amberDim: ui.Hex("#2A1D07"), amberBorder: ui.Hex("#6B4A12"), amberText: ui.Hex("#FFD9A0"), amberGlow: ui.RGBA(255, 180, 84, 0.35),
	amberPill: ui.Hex("#3A2A0B"),
	cyan:      ui.Hex("#3DD9FF"), cyanDim: ui.Hex("#102A33"), cyanBorder: ui.Hex("#1B4452"),
	code:   ui.Hex("#080A0F"),
	violet: ui.Hex("#B79BFF"), violetDim: ui.Hex("#1E1733"), violetBorder: ui.Hex("#4A3A7A"),
	danger: ui.Hex("#FF7A7A"), dangerDim: ui.Hex("#2A1114"), dangerBorder: ui.Hex("#5C2329"),
	shadow: 0.4,
}

// lightPalette keeps the same meanings on a pale ground: the bright colors of the dark look
// become deep ones that read as text, and the dark tints become pale ones.
var lightPalette = palette{
	ground: ui.Hex("#EDF0F6"), panel: ui.Hex("#F6F8FB"), card: ui.Hex("#FFFFFF"), cardOn: ui.Hex("#F0F8DF"),
	line: ui.Hex("#DCE1EB"), lineStrong: ui.Hex("#BCC5D6"),
	surface: ui.Hex("#FFFFFF"), surfaceHover: ui.Hex("#F0F3F9"), surfacePressed: ui.Hex("#E4E9F2"),
	text: ui.Hex("#151A26"), muted: ui.Hex("#4F5A74"), faint: ui.Hex("#66718B"),
	accent: ui.Hex("#4C7A00"), accentDim: ui.Hex("#A9D45A"), accentGlow: ui.RGBA(76, 122, 0, 0.30), onAccent: ui.Hex("#FFFFFF"),
	accentHover: ui.Hex("#3F6700"), accentPressed: ui.Hex("#335500"),
	amber: ui.Hex("#A85F00"), amberDim: ui.Hex("#FFF1DA"), amberBorder: ui.Hex("#E6B567"), amberText: ui.Hex("#6B3C00"), amberGlow: ui.RGBA(168, 95, 0, 0.25),
	amberPill: ui.Hex("#FFE7BD"),
	cyan:      ui.Hex("#00728F"), cyanDim: ui.Hex("#DDF3F9"), cyanBorder: ui.Hex("#8CCFE0"),
	code:   ui.Hex("#E6EAF2"),
	violet: ui.Hex("#6444D0"), violetDim: ui.Hex("#EEE9FF"), violetBorder: ui.Hex("#B9A8F0"),
	danger: ui.Hex("#C0272F"), dangerDim: ui.Hex("#FCEBEC"), dangerBorder: ui.Hex("#EDA5A9"),
	shadow: 0.12,
}

// pal is the palette of the frame being drawn; View sets it at the top of every frame, from the
// desktop's appearance, so the rest of the code just reads it.
var pal = darkPalette

// paletteFor is the palette for a dark or light window. A desktop accent (given as ok) replaces
// slom's own, after usableAccent made it fit; the colors around it are derived from its hue.
func paletteFor(dark bool, accent ui.Color, ok bool) palette {
	p := lightPalette
	if dark {
		p = darkPalette
	}
	if !ok {
		return p
	}
	a, ok := usableAccent(accent, dark)
	if !ok {
		return p
	}
	_, ch, h := oklch(a)
	p.accent, p.accentGlow = a, a.Alpha(ternary[float32](dark, 0.45, 0.30))
	white, black := ui.RGB(255, 255, 255), ui.RGB(0, 0, 0)
	if dark {
		p.accentDim = ui.Oklch(0.40, min(ch, 0.09), h)
		p.cardOn = ui.Oklch(0.24, min(ch, 0.045), h)
		p.accentHover, p.accentPressed = a.Mix(white, 0.2), a.Mix(black, 0.15)
		p.onAccent = readableOn(a, darkPalette.ground, white)
	} else {
		p.accentDim = ui.Oklch(0.80, min(ch, 0.09), h)
		p.cardOn = ui.Oklch(0.96, min(ch, 0.03), h)
		p.accentHover, p.accentPressed = a.Mix(black, 0.12), a.Mix(black, 0.25)
		p.onAccent = readableOn(a, lightPalette.text, white)
	}
	return p
}

// gamerTheme is the theme every widget inherits, built on pal: dark or light as the desktop is.
func gamerTheme() *ui.Theme {
	t := ui.LightTheme()
	if pal.dark {
		t = ui.DarkTheme()
	}
	t.Background = pal.ground
	t.Surface, t.SurfaceHover, t.SurfacePressed = pal.surface, pal.surfaceHover, pal.surfacePressed
	t.Border = pal.lineStrong
	t.Text, t.TextMuted = pal.text, pal.muted
	t.Accent, t.AccentHover, t.AccentPressed = pal.accent, pal.accentHover, pal.accentPressed
	t.AccentText = pal.onAccent
	t.Warning, t.Success = pal.amber, pal.cyan
	t.Focus = pal.accent
	t.Radius = 10
	return t
}

// washColor is the tint of a game's header: its tile's color, faded toward the window's.
func washColor(name string) ui.Color {
	if pal.dark {
		return ui.Oklch(0.27, 0.07, gameHue(name))
	}
	return ui.Oklch(0.90, 0.05, gameHue(name))
}

func gameHue(name string) float32 {
	h := fnv.New32a()
	h.Write([]byte(name))
	return float32(h.Sum32() % 360)
}

// tileColors gives each game its own tile gradient, from its name, so the list is easy to scan.
func tileColors(name string) (from, to ui.Color) {
	hue := gameHue(name)
	return ui.Oklch(0.52, 0.13, hue), ui.Oklch(0.30, 0.09, hue)
}
