package ui

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// Oklch helpers for the accent: the desktop (or the Omarchy theme) chooses it, so slom has to
// make any color usable: not too grey to stand out, not too dark or too light to read.

// accentMinChroma is how colorful an accent must be. A theme's accent can be a muted rose or a
// grey; as "this will be applied" it still has to stand out from the neutral surfaces.
const accentMinChroma = 0.09

// greyChroma is below which a color has no hue worth keeping (slate, white, black): slom's own
// accent is used instead of inventing one.
const greyChroma = 0.015

func linear(v uint8) float64 {
	x := float64(v) / 255
	if x <= 0.04045 {
		return x / 12.92
	}
	return math.Pow((x+0.055)/1.055, 2.4)
}

// oklch converts a color to Oklch: lightness 0..1, chroma, hue in degrees.
func oklch(c ui.Color) (l, ch, h float32) {
	r, g, b := linear(c.R), linear(c.G), linear(c.B)
	lp := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	mp := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	sp := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	L := 0.2104542553*lp + 0.7936177850*mp - 0.0040720468*sp
	a := 1.9779984951*lp - 2.4285922050*mp + 0.4505937099*sp
	bb := 0.0259040371*lp + 0.7827717662*mp - 0.8086757660*sp
	hue := math.Atan2(bb, a) * 180 / math.Pi
	if hue < 0 {
		hue += 360
	}
	return float32(L), float32(math.Hypot(a, bb)), float32(hue)
}

// luminance is the WCAG relative luminance.
func luminance(c ui.Color) float64 {
	return 0.2126*linear(c.R) + 0.7152*linear(c.G) + 0.0722*linear(c.B)
}

// contrast is the WCAG contrast ratio of two colors, 1..21.
func contrast(a, b ui.Color) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// readableOn picks the text color that reads best on a fill.
func readableOn(fill, dark, light ui.Color) ui.Color {
	if contrast(fill, dark) >= contrast(fill, light) {
		return dark
	}
	return light
}

func clamp(v, lo, hi float32) float32 { return min(max(v, lo), hi) }

// usableAccent turns the accent the desktop gave into one slom can draw with: colorful enough,
// and light enough (dark mode) or dark enough (light mode) to read on its ground. ok is false
// for a color with no hue (a grey accent), where slom keeps its own.
func usableAccent(c ui.Color, dark bool) (ui.Color, bool) {
	l, ch, h := oklch(c)
	if ch < greyChroma {
		return ui.Color{}, false
	}
	ch = max(ch, accentMinChroma)
	if dark {
		l = clamp(l, 0.70, 0.95)
	} else {
		l = clamp(l, 0.40, 0.55)
	}
	return ui.Oklch(l, ch, h), true
}
