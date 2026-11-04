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
// applied", amber is for Steam being in the way, cyan is information.
type palette struct {
	ground, panel, card, cardOn, line, lineStrong      ui.Color
	text, muted, faint                                 ui.Color
	accent, accentDim, accentGlow, onAccent            ui.Color
	amber, amberDim, amberBorder, amberText, amberGlow ui.Color
	cyan, cyanDim, cyanBorder                          ui.Color
	code                                               ui.Color
}

var pal = palette{
	ground: ui.Hex("#0A0C11"), panel: ui.Hex("#0E1118"), card: ui.Hex("#0F131C"), cardOn: ui.Hex("#141F0A"),
	line: ui.Hex("#1B2130"), lineStrong: ui.Hex("#2A3248"),
	text: ui.Hex("#E8ECF4"), muted: ui.Hex("#8E99B3"), faint: ui.Hex("#6D7893"),
	accent: ui.Hex("#C6FF3D"), accentDim: ui.Hex("#3A5415"), accentGlow: ui.RGBA(198, 255, 61, 0.45), onAccent: ui.Hex("#0A0C11"),
	amber: ui.Hex("#FFB454"), amberDim: ui.Hex("#2A1D07"), amberBorder: ui.Hex("#6B4A12"), amberText: ui.Hex("#FFD9A0"), amberGlow: ui.RGBA(255, 180, 84, 0.35),
	cyan: ui.Hex("#3DD9FF"), cyanDim: ui.Hex("#102A33"), cyanBorder: ui.Hex("#1B4452"),
	code: ui.Hex("#080A0F"),
}

// gamerTheme is the dark theme every widget inherits; the window is always dark, whatever the
// desktop's appearance, because the look is built on it.
func gamerTheme() *ui.Theme {
	t := ui.DarkTheme()
	t.Background = pal.ground
	t.Surface = ui.Hex("#161C2B")
	t.SurfaceHover = ui.Hex("#1D2437")
	t.SurfacePressed = ui.Hex("#232B3F")
	t.Border = pal.lineStrong
	t.Text, t.TextMuted = pal.text, pal.muted
	t.Accent, t.AccentHover, t.AccentPressed = pal.accent, ui.Hex("#D6FF70"), ui.Hex("#A8DE2A")
	t.AccentText = pal.onAccent
	t.Warning, t.Success = pal.amber, pal.cyan
	t.Focus = pal.accent
	t.Radius = 10
	return t
}

// tileColors gives each game its own tile gradient, from its name, so the list is easy to scan.
func tileColors(name string) (from, to ui.Color) {
	h := fnv.New32a()
	h.Write([]byte(name))
	hue := float32(h.Sum32() % 360)
	return ui.Oklch(0.52, 0.13, hue), ui.Oklch(0.30, 0.09, hue)
}
